// Package generator renders the embedded service templates, which are derived
// from iconpay-qoin-integrator, into a new or existing service repository.
package generator

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"go/format"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

//go:embed all:templates
var templateFS embed.FS

// renderTree renders every template under templates/<root> into dir. keep
// decides, per output path (slash-separated, relative to dir), whether a file
// is generated at all; target maps a template path to its output path.
func renderTree(root, dir string, data any, keep func(rel string) bool, target func(rel string) string) ([]string, error) {
	base := path.Join("templates", root)
	var written []string

	err := fs.WalkDir(templateFS, base, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		rel := strings.TrimSuffix(strings.TrimPrefix(p, base+"/"), ".tmpl")
		if target != nil {
			rel = target(rel)
		}
		if keep != nil && !keep(rel) {
			return nil
		}

		content, err := render(p, data)
		if err != nil {
			return err
		}

		out := filepath.Join(dir, filepath.FromSlash(rel))
		if err := writeNewFile(out, content); err != nil {
			return err
		}
		written = append(written, rel)
		return nil
	})
	return written, err
}

func render(name string, data any) ([]byte, error) {
	raw, err := templateFS.ReadFile(name)
	if err != nil {
		return nil, err
	}

	tmpl, err := template.New(path.Base(name)).Option("missingkey=error").Parse(string(raw))
	if err != nil {
		return nil, fmt.Errorf("parse template %s: %w", name, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("execute template %s: %w", name, err)
	}

	if strings.HasSuffix(strings.TrimSuffix(name, ".tmpl"), ".go") {
		formatted, err := format.Source(buf.Bytes())
		if err != nil {
			return nil, fmt.Errorf("gofmt %s: %w\n%s", name, err, buf.String())
		}
		return formatted, nil
	}
	return buf.Bytes(), nil
}

func writeNewFile(out string, content []byte) error {
	if _, err := os.Stat(out); err == nil {
		return fmt.Errorf("%s already exists", out)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	return os.WriteFile(out, content, 0o644)
}

var modulePattern = regexp.MustCompile(`(?m)^module\s+(\S+)`)

// readModule returns the module path of the service in dir, and checks that
// dir looks like a service this tool generated.
func readModule(dir string) (string, error) {
	raw, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return "", fmt.Errorf("go.mod not found in %s — run this from the service root", dir)
	}
	m := modulePattern.FindSubmatch(raw)
	if m == nil {
		return "", errors.New("module path not found in go.mod")
	}
	if _, err := os.Stat(filepath.Join(dir, "appservice", "appservice.go")); err != nil {
		return "", fmt.Errorf("%s does not look like a generated service (appservice/appservice.go missing)", dir)
	}
	return string(m[1]), nil
}

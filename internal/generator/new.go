package generator

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type NewOptions struct {
	Name        string // repository / service name, e.g. iconpay-dana-integrator
	Module      string // Go module path; defaults to Name
	AppName     string // human readable name, e.g. "Integrator Dana"
	OutputDir   string // parent directory the service directory is created in
	Manifest    string // k8s-manifest-ni path used by CI; defaults to iconpay/<Name>
	Port        int
	WithGateway bool // gateway client + psp-id/signature request filter
	Postgres    bool
	Redis       bool
	Example     bool // generate an example feature
	Tidy        bool // run `go mod tidy` afterwards
	GitInit     bool
}

type projectData struct {
	Name        string
	Module      string
	AppName     string
	Manifest    string
	RedisPrefix string
	Port        int
	WithGateway bool
	Postgres    bool
	Redis       bool
}

func New(opts NewOptions) (string, error) {
	name, err := NewName(opts.Name)
	if err != nil {
		return "", err
	}

	data := projectData{
		Name:        name.Kebab,
		Module:      opts.Module,
		AppName:     opts.AppName,
		Manifest:    opts.Manifest,
		RedisPrefix: name.Snake + ":",
		Port:        opts.Port,
		WithGateway: opts.WithGateway,
		Postgres:    opts.Postgres,
		Redis:       opts.Redis,
	}
	if data.Module == "" {
		data.Module = name.Kebab
	}
	if data.AppName == "" {
		data.AppName = name.Title
	}
	if data.Manifest == "" {
		data.Manifest = "iconpay/" + name.Kebab
	}
	if data.Port == 0 {
		data.Port = 6001
	}

	dir := filepath.Join(opts.OutputDir, name.Kebab)
	if entries, err := os.ReadDir(dir); err == nil && len(entries) > 0 {
		return "", fmt.Errorf("%s already exists and is not empty", dir)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}

	keep := func(rel string) bool {
		if !opts.WithGateway && (strings.HasPrefix(rel, "external/gateway/") || rel == "internal/router/filter/request_filter.go") {
			return false
		}
		return true
	}
	// Dotfiles are stored without the dot so they are not hidden inside the
	// template tree.
	target := func(rel string) string {
		switch rel {
		case "gitignore", "gitlab-ci.yml":
			return "." + rel
		}
		return rel
	}

	if _, err := renderTree("project", dir, data, keep, target); err != nil {
		return dir, err
	}

	if opts.Example {
		if _, err := AddFeature(FeatureOptions{Dir: dir, Name: "example", Action: "ping", Method: "POST"}); err != nil {
			return dir, fmt.Errorf("example feature: %w", err)
		}
	}

	if opts.Tidy {
		if err := run(dir, "go", "mod", "tidy"); err != nil {
			fmt.Fprintf(os.Stderr, "warning: go mod tidy failed (%v); run it manually once repo-iconx.air.id is reachable\n", err)
		}
	}

	if opts.GitInit {
		if err := run(dir, "git", "init", "-q", "-b", "development"); err != nil {
			fmt.Fprintf(os.Stderr, "warning: git init failed: %v\n", err)
		}
	}

	return dir, nil
}

func run(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

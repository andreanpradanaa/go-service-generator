package generator

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ExternalOptions struct {
	Dir  string // service root
	Name string // partner, e.g. dana → external/dana, app.external.dana
}

type externalData struct {
	Module string
	Name   Name
}

func AddExternal(opts ExternalOptions) ([]string, error) {
	module, err := readModule(opts.Dir)
	if err != nil {
		return nil, err
	}
	name, err := NewName(opts.Name)
	if err != nil {
		return nil, err
	}
	data := externalData{Module: module, Name: name}

	if err := ensureExternalRoot(opts.Dir, module); err != nil {
		return nil, err
	}

	target := func(rel string) string {
		rel = strings.ReplaceAll(rel, "__pkg__", name.Lower)
		return strings.ReplaceAll(rel, "__file__", name.Lower)
	}
	written, err := renderTree("external", opts.Dir, data, nil, target)
	if err != nil {
		return written, err
	}

	externalGo := filepath.Join(opts.Dir, "external", "external.go")
	if err := insertBeforeMarker(externalGo, "// svcgen:imports", fmt.Sprintf("%q", module+"/external/"+name.Lower)); err != nil {
		return written, err
	}
	if err := insertBeforeMarker(externalGo, "// svcgen:modules", name.Lower+".Module,"); err != nil {
		return written, err
	}

	env := "${ APP_EXTERNAL_" + name.Upper + "_%s | %s }"
	configYml := filepath.Join(opts.Dir, "internal", "resources", "config.yml")
	err = insertBeforeMarker(configYml, "# svcgen:external",
		name.Kebab+":",
		"  base-url: "+fmt.Sprintf(env, "BASE_URL", "CHANGE_ME"),
		"  client-id: "+fmt.Sprintf(env, "CLIENT_ID", "CHANGE_ME"),
		"  client-secret: "+fmt.Sprintf(env, "CLIENT_SECRET", "CHANGE_ME"),
		"  timeout: "+fmt.Sprintf(env, "TIMEOUT", "30s"),
	)
	return written, err
}

// ensureExternalRoot creates external/external.go, registers external.Module
// in appservice and adds app.external to config.yml, for services generated
// without any external package.
func ensureExternalRoot(dir, module string) error {
	externalGo := filepath.Join(dir, "external", "external.go")
	if _, err := os.Stat(externalGo); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	content, err := render("templates/project/external/external.go.tmpl", projectData{Module: module})
	if err != nil {
		return err
	}
	if err := writeNewFile(externalGo, content); err != nil {
		return err
	}

	appservice := filepath.Join(dir, "appservice", "appservice.go")
	if err := insertBeforeMarker(appservice, "// svcgen:imports", fmt.Sprintf("%q", module+"/external")); err != nil {
		return err
	}
	if err := insertBeforeMarker(appservice, "// svcgen:modules", "external.Module,"); err != nil {
		return err
	}

	configYml := filepath.Join(dir, "internal", "resources", "config.yml")
	return insertBeforeMarker(configYml, "# svcgen:app", "external:", "  # svcgen:external")
}

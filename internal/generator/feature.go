package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type FeatureOptions struct {
	Dir    string // service root
	Name   string // feature, e.g. balance → /api/v1/balance
	Action string // first endpoint, e.g. account → POST /api/v1/balance/account; empty → POST /api/v1/balance
	Method string // HTTP method
	Public bool   // register in Public() instead of Authenticated()
}

type featureData struct {
	Module       string
	Feature      Name
	Action       Name   // handler method name
	Type         string // DTO name prefix: BalanceAccount, or Balance without an action
	Method       string // fiber router method: Post, Get, ...
	Public       bool
	GroupPath    string
	ActionPath   string
	ServiceConst string
}

func AddFeature(opts FeatureOptions) ([]string, error) {
	module, err := readModule(opts.Dir)
	if err != nil {
		return nil, err
	}
	feature, err := NewName(opts.Name)
	if err != nil {
		return nil, err
	}
	action := feature
	if opts.Action != "" {
		if action, err = NewName(opts.Action); err != nil {
			return nil, err
		}
	}
	method, err := fiberMethod(opts.Method)
	if err != nil {
		return nil, err
	}

	data := featureData{
		Module:    module,
		Feature:   feature,
		Action:    action,
		Type:      feature.Pascal,
		Method:    method,
		Public:    opts.Public,
		GroupPath: "/api/v1/" + feature.Kebab,
	}
	if opts.Action != "" {
		data.Type += action.Pascal
		data.ActionPath = "/" + action.Kebab
	}
	data.ServiceConst = "Service" + data.Type

	rcFile := filepath.Join(opts.Dir, "internal", "model", "rc", "rc.go")
	serviceCode, err := nextServiceCode(rcFile)
	if err != nil {
		return nil, err
	}

	target := func(rel string) string {
		return strings.ReplaceAll(rel, "__feature__", feature.Snake)
	}
	written, err := renderTree("feature", opts.Dir, data, nil, target)
	if err != nil {
		return written, err
	}

	internal := filepath.Join(opts.Dir, "internal")
	steps := []struct {
		file, marker string
		lines        []string
	}{
		{filepath.Join(internal, "router", "router.go"), "// svcgen:routes",
			[]string{fmt.Sprintf("fx.Annotate(new%sRouter, fx.ResultTags(`group:\"routes\"`)),", feature.Pascal)}},
		{filepath.Join(internal, "controller", "controller.go"), "// svcgen:controllers",
			[]string{fmt.Sprintf("new%sController,", feature.Pascal)}},
		{filepath.Join(internal, "usecase", "usecase.go"), "// svcgen:usecases",
			[]string{fmt.Sprintf("new%sUsecase,", feature.Pascal)}},
		{rcFile, "// svcgen:service-codes",
			[]string{fmt.Sprintf("%s = %q", data.ServiceConst, serviceCode)}},
		{rcFile, "// svcgen:service-paths",
			[]string{fmt.Sprintf("%q: %s,", data.GroupPath+data.ActionPath, data.ServiceConst)}},
	}
	for _, s := range steps {
		if err := insertBeforeMarker(s.file, s.marker, s.lines...); err != nil {
			return written, err
		}
	}

	return written, nil
}

func fiberMethod(m string) (string, error) {
	switch strings.ToUpper(m) {
	case "", "POST":
		return "Post", nil
	case "GET":
		return "Get", nil
	case "PUT":
		return "Put", nil
	case "PATCH":
		return "Patch", nil
	case "DELETE":
		return "Delete", nil
	}
	return "", fmt.Errorf("unsupported method %q", m)
}

var serviceCodePattern = regexp.MustCompile(`(?m)^\s*Service\w+\s*=\s*"(\d{2})"`)

// nextServiceCode returns the next free 2-digit service code in rc.go.
func nextServiceCode(rcFile string) (string, error) {
	raw, err := os.ReadFile(rcFile)
	if err != nil {
		return "", err
	}

	highest := 0
	for _, m := range serviceCodePattern.FindAllSubmatch(raw, -1) {
		n, _ := strconv.Atoi(string(m[1]))
		highest = max(highest, n)
	}
	if highest >= 99 {
		return "", fmt.Errorf("no free service code left in %s", rcFile)
	}
	return fmt.Sprintf("%02d", highest+1), nil
}

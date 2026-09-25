package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewName(t *testing.T) {
	tests := map[string]Name{
		"transfer-out": {Pascal: "TransferOut", Camel: "transferOut", Snake: "transfer_out", Kebab: "transfer-out", Upper: "TRANSFER_OUT", Lower: "transferout"},
		"transferOut":  {Pascal: "TransferOut", Camel: "transferOut", Snake: "transfer_out", Kebab: "transfer-out", Upper: "TRANSFER_OUT", Lower: "transferout"},
		"sso_url":      {Pascal: "SSOURL", Camel: "ssoURL", Snake: "sso_url", Kebab: "sso-url", Upper: "SSO_URL", Lower: "ssourl"},
	}
	for raw, want := range tests {
		got, err := NewName(raw)
		if err != nil {
			t.Fatalf("NewName(%q): %v", raw, err)
		}
		if got.Pascal != want.Pascal || got.Camel != want.Camel || got.Snake != want.Snake ||
			got.Kebab != want.Kebab || got.Upper != want.Upper || got.Lower != want.Lower {
			t.Errorf("NewName(%q) = %+v, want %+v", raw, got, want)
		}
	}

	if _, err := NewName("1bad"); err == nil {
		t.Error("NewName(\"1bad\") should fail")
	}
}

// TestGenerate renders a full service and checks the registrations svcgen
// injects. Building the result needs repo-iconx.air.id, so that is left to
// the manual check described in the README.
func TestGenerate(t *testing.T) {
	out := t.TempDir()
	dir, err := New(NewOptions{Name: "iconpay-test-integrator", OutputDir: out, WithGateway: true, Example: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := AddExternal(ExternalOptions{Dir: dir, Name: "dana"}); err != nil {
		t.Fatal(err)
	}
	if _, err := AddFeature(FeatureOptions{Dir: dir, Name: "balance", Action: "account"}); err != nil {
		t.Fatal(err)
	}
	if _, err := AddFeature(FeatureOptions{Dir: dir, Name: "balance", Action: "account"}); err == nil {
		t.Error("adding the same feature twice should fail")
	}

	expect := map[string][]string{
		".gitlab-ci.yml":                    {"MANIFEST: iconpay/iconpay-test-integrator"},
		"go.mod":                            {"module iconpay-test-integrator"},
		"internal/router/router.go":         {"newHealthRouter", "newExampleRouter", "newBalanceRouter"},
		"internal/controller/controller.go": {"newBalanceController,"},
		"internal/usecase/usecase.go":       {"newBalanceUsecase,"},
		"internal/model/rc/rc.go":           {`ServiceExamplePing    = "01"`, `ServiceBalanceAccount = "02"`, `"/api/v1/balance/account": ServiceBalanceAccount`},
		"external/external.go":              {`"iconpay-test-integrator/external/dana"`, "dana.Module,"},
		"internal/resources/config.yml":     {"    dana:\n      base-url: ${ APP_EXTERNAL_DANA_BASE_URL | CHANGE_ME }"},
	}
	for file, wants := range expect {
		raw, err := os.ReadFile(filepath.Join(dir, file))
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range wants {
			if !strings.Contains(string(raw), want) {
				t.Errorf("%s: missing %q\n%s", file, want, raw)
			}
		}
	}
}

// TestGenerateWithoutExternal checks that a service without third-party
// connections has no external/ package, and that add external creates and
// wires it afterwards.
func TestGenerateWithoutExternal(t *testing.T) {
	dir, err := New(NewOptions{Name: "plain-svc", OutputDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"external", "internal/router/filter/request_filter.go"} {
		if _, err := os.Stat(filepath.Join(dir, p)); !os.IsNotExist(err) {
			t.Errorf("%s should not exist without external connections", p)
		}
	}

	if _, err := AddExternal(ExternalOptions{Dir: dir, Name: "dana"}); err != nil {
		t.Fatal(err)
	}
	expect := map[string][]string{
		"appservice/appservice.go":      {`"plain-svc/external"`, "external.Module,"},
		"external/external.go":          {`"plain-svc/external/dana"`, "dana.Module,"},
		"internal/resources/config.yml": {"  external:\n    dana:\n"},
	}
	for file, wants := range expect {
		raw, err := os.ReadFile(filepath.Join(dir, file))
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range wants {
			if !strings.Contains(string(raw), want) {
				t.Errorf("%s: missing %q\n%s", file, want, raw)
			}
		}
	}
}

// TestNoGeneratorTraces checks that generated services, including files
// edited by add feature / add external, never mention the generator.
func TestNoGeneratorTraces(t *testing.T) {
	dir, err := New(NewOptions{Name: "trace-svc", OutputDir: t.TempDir(), WithGateway: true, Example: true, Externals: []string{"ovo"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := AddFeature(FeatureOptions{Dir: dir, Name: "balance", Action: "account"}); err != nil {
		t.Fatal(err)
	}
	if _, err := AddExternal(ExternalOptions{Dir: dir, Name: "dana"}); err != nil {
		t.Fatal(err)
	}

	err = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if strings.Contains(strings.ToLower(string(raw)), "svcgen") {
			t.Errorf("%s mentions svcgen", p)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

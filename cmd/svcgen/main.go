// Command svcgen generates Go service repositories with the same structure as
// iconpay-qoin-integrator (fx + fiber + icon-digital-library/common), and adds
// features and external partner clients to them afterwards.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/andreanpradana/go-service-generator/internal/generator"
)

var version = "dev"

const usage = `svcgen — generator boilerplate service Go (struktur iconpay-qoin-integrator)

Usage:
  svcgen new <name> [flags]                  buat repo service baru
  svcgen add feature <name> [flags]          tambah router/controller/usecase/dto
  svcgen add external <name> [flags]         tambah client partner di external/<name>
  svcgen version

Jalankan "svcgen <command> -h" untuk daftar flag.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "new":
		err = cmdNew(os.Args[2:])
	case "add":
		err = cmdAdd(os.Args[2:])
	case "version", "-v", "--version":
		fmt.Println(version)
	case "help", "-h", "--help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", os.Args[1], usage)
		os.Exit(2)
	}

	if errors.Is(err, flag.ErrHelp) {
		return
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// parse lets flags appear before or after the positional name, which the
// standard flag package does not do on its own.
func parse(fs *flag.FlagSet, args []string) (string, error) {
	var positional []string
	for len(args) > 0 {
		if err := fs.Parse(args); err != nil {
			return "", err
		}
		args = fs.Args()
		if len(args) > 0 {
			positional = append(positional, args[0])
			args = args[1:]
		}
	}
	if len(positional) != 1 {
		fs.Usage()
		return "", fmt.Errorf("expected exactly one name, got %d", len(positional))
	}
	return positional[0], nil
}

func cmdNew(args []string) error {
	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: svcgen new <name> [flags]\n\nContoh: svcgen new iconpay-dana-integrator --port 6002 --example\n\nFlags:")
		fs.PrintDefaults()
	}

	var opts generator.NewOptions
	fs.StringVar(&opts.Module, "module", "", "Go module path (default: <name>)")
	fs.StringVar(&opts.AppName, "app-name", "", "nama aplikasi di config (default: <name> dalam Title Case)")
	fs.StringVar(&opts.OutputDir, "out", ".", "parent directory tempat repo dibuat")
	fs.StringVar(&opts.Manifest, "manifest", "", "path k8s-manifest-ni untuk CI (default: iconpay/<name>)")
	fs.IntVar(&opts.Port, "port", 6001, "port HTTP service")
	fs.BoolVar(&opts.WithGateway, "gateway", true, "sertakan client gateway + request filter psp-id/signature")
	fs.BoolVar(&opts.Postgres, "postgres", true, "aktifkan data source postgres di config")
	fs.BoolVar(&opts.Redis, "redis", true, "aktifkan data source redis di config")
	fs.BoolVar(&opts.Example, "example", false, "buat contoh feature (POST /api/v1/example/ping)")
	fs.BoolVar(&opts.Tidy, "tidy", true, "jalankan go mod tidy setelah generate")
	fs.BoolVar(&opts.GitInit, "git", true, "git init dengan branch development")

	name, err := parse(fs, args)
	if err != nil {
		return err
	}
	opts.Name = name

	dir, err := generator.New(opts)
	if err != nil {
		return err
	}

	fmt.Printf("\n✔ Service dibuat di %s\n\nLangkah selanjutnya:\n  cd %s\n  svcgen add external <partner>\n  svcgen add feature <feature> --action <action>\n  go run ./cmd\n", dir, dir)
	return nil
}

func cmdAdd(args []string) error {
	if len(args) < 1 {
		fmt.Fprint(os.Stderr, usage)
		return fmt.Errorf("missing add target: feature | external")
	}

	switch args[0] {
	case "feature":
		return cmdAddFeature(args[1:])
	case "external":
		return cmdAddExternal(args[1:])
	}
	return fmt.Errorf("unknown add target %q: use feature | external", args[0])
}

func cmdAddFeature(args []string) error {
	fs := flag.NewFlagSet("add feature", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: svcgen add feature <name> [flags]\n\nContoh: svcgen add feature balance --action account   → POST /api/v1/balance/account\n\nFlags:")
		fs.PrintDefaults()
	}

	opts := generator.FeatureOptions{}
	fs.StringVar(&opts.Dir, "dir", ".", "root service")
	fs.StringVar(&opts.Action, "action", "", "nama endpoint pertama; kosong = route di root group /api/v1/<name>")
	fs.StringVar(&opts.Method, "method", "POST", "HTTP method: GET, POST, PUT, PATCH, DELETE")
	fs.BoolVar(&opts.Public, "public", false, "daftarkan di Public() router, bukan Authenticated() (request filter tetap berlaku sesuai PrefixPaths)")

	name, err := parse(fs, args)
	if err != nil {
		return err
	}
	opts.Name = name

	written, err := generator.AddFeature(opts)
	printWritten(written)
	return err
}

func cmdAddExternal(args []string) error {
	fs := flag.NewFlagSet("add external", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: svcgen add external <name> [flags]\n\nContoh: svcgen add external dana\n\nFlags:")
		fs.PrintDefaults()
	}

	opts := generator.ExternalOptions{}
	fs.StringVar(&opts.Dir, "dir", ".", "root service")

	name, err := parse(fs, args)
	if err != nil {
		return err
	}
	opts.Name = name

	written, err := generator.AddExternal(opts)
	printWritten(written)
	return err
}

func printWritten(files []string) {
	for _, f := range files {
		fmt.Println("  + " + f)
	}
}

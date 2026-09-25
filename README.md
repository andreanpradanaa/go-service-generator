# svcgen — Go service generator

CLI untuk generate boilerplate repo service Go dengan struktur yang sama seperti `iconpay-qoin-integrator`:
fx + fiber + `repo-iconx.air.id/icon-digital-library/common`, lengkap dengan Dockerfile, `.gitlab-ci.yml`, dan `sonar-project.properties`.

## Install

Repo ini private, jadi Go perlu tahu module-nya private:

```bash
go env -w GOPRIVATE='repo-iconx.air.id/*,github.com/andreanpradana/*'
go install github.com/andreanpradana/go-service-generator/cmd/svcgen@latest
```

Atau dari source: `go build -o svcgen ./cmd/svcgen`.

## Pemakaian

### Buat service baru

```bash
svcgen new iconpay-dana-integrator --port 6002
```

| Flag | Default | Keterangan |
|---|---|---|
| `--module` | `<name>` | Go module path |
| `--app-name` | `<name>` Title Case | `app.name` di config |
| `--out` | `.` | parent directory |
| `--manifest` | `iconpay/<name>` | path di `k8s-manifest-ni` untuk job deploy CI |
| `--port` | `6001` | port HTTP |
| `--gateway` | `true` | client `external/gateway` + request filter `psp-id`/`timestamp`/`signature` |
| `--postgres` / `--redis` | `true` | nilai default `enable` data source di config |
| `--example` | `false` | contoh feature `POST /api/v1/example/ping` |
| `--tidy` | `true` | jalankan `go mod tidy` |
| `--git` | `true` | `git init` dengan branch `development` |

### Tambah partner (external client)

```bash
cd iconpay-dana-integrator
svcgen add external dana
```

Membuat `external/dana/` (config, http client, `Client.do` dengan klasifikasi SUCCESS/PENDING/FAILED dan timeout → `cerror.ErrTimeout`), mendaftarkan `dana.Module` di `external/external.go`, dan menambah `app.external.dana` di `config.yml`.

### Tambah feature / endpoint

```bash
svcgen add feature balance --action account          # POST /api/v1/balance/account
svcgen add feature transfer-out --method GET --public # GET  /api/v1/transfer-out
```

Membuat router, controller, usecase, dan dto, lalu mendaftarkannya ke `fx.Provide` di router/controller/usecase, dan menambah service code berikutnya di `internal/model/rc/rc.go`.

> Komentar `// svcgen:*` dan `# svcgen:*` di repo hasil generate adalah penanda tempat svcgen menyisipkan kode. Jangan dihapus.

## Mengubah template

Template ada di `internal/generator/templates/` dan di-embed ke binary:

- `project/` — `svcgen new` (dotfile disimpan tanpa titik: `gitignore.tmpl`, `gitlab-ci.yml.tmpl`)
- `feature/` — `svcgen add feature` (`__feature__` diganti snake_case nama feature)
- `external/` — `svcgen add external` (`__pkg__` / `__file__` diganti nama package)

Semua file `.tmpl` memakai `text/template`; file `.go` otomatis di-gofmt. Setelah mengubah template:

```bash
go test ./...
go build -o svcgen ./cmd/svcgen
./svcgen new tmp-svc --out /tmp --example && (cd /tmp/tmp-svc && go build ./...)
```

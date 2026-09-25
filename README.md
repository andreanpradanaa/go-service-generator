# svcgen — Go service generator

CLI untuk generate boilerplate repo service Go dengan struktur yang sama seperti `iconpay-qoin-integrator`:
fx + fiber + `repo-iconx.air.id/icon-digital-library/common`, lengkap dengan Dockerfile, `.gitlab-ci.yml`, dan `sonar-project.properties`.

## Install

```bash
go env -w GOPRIVATE='repo-iconx.air.id/*,github.com/andreanpradanaa/*'
go install github.com/andreanpradanaa/go-service-generator/cmd/svcgen@latest
```

- `repo-iconx.air.id/*` dibutuhkan oleh service hasil generate (dependency `icon-digital-library/common`), bukan oleh svcgen.
- `github.com/andreanpradanaa/*` hanya perlu kalau repo ini private.
- `go env -w GOPRIVATE=...` menimpa nilai lama, jadi tulis semua pola sekaligus.

`go install` menaruh binary di `$(go env GOPATH)/bin` (biasanya `~/go/bin`). Kalau muncul `command not found: svcgen`, tambahkan folder itu ke PATH (sekali saja):

```bash
# zsh (default macOS)
echo 'export PATH="$(go env GOPATH)/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc

# bash
echo 'export PATH="$(go env GOPATH)/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

Cek dengan `svcgen version`.

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
| `--gateway` | `false` *(ditanya)* | client `external/gateway` + request filter `psp-id`/`timestamp`/`signature` |
| `--postgres` / `--redis` | `true` | nilai default `enable` data source di config |
| `--external` | kosong *(ditanya)* | package external partner, pisahkan dengan koma (`dana,ovo`) |
| `--example` | `false` | contoh feature `POST /api/v1/example/ping` |
| `--tidy` | `true` | jalankan `go mod tidy` |
| `--git` | `true` | `git init` dengan branch `development` |

Secara default service **tidak terhubung ke pihak ketiga**: tidak ada folder `external/`, gateway, maupun request filter.
Kalau `--gateway` dan `--external` tidak diisi, svcgen bertanya dulu:

```
Buat package external (koneksi ke pihak ketiga)? (y/N): y
Sertakan gateway iconpay (client + request filter psp-id/signature)? (y/N): y
Nama partner (pisahkan dengan koma, kosongkan jika tidak ada): dana
```

- Jawab `N` / Enter di pertanyaan pertama → tanpa external sama sekali.
- Tanpa pertanyaan (script/CI): isi flag-nya, misalnya `--gateway --external dana,ovo`, atau `--external=""` untuk tanpa external.
- Pertanyaan juga dilewati kalau stdin bukan terminal; hasilnya tanpa external.
- External tetap bisa ditambah nanti dengan `svcgen add external`.

Setiap service punya `GET /healthz` (di `internal/router/health_router.go`) untuk probe Kubernetes:

```json
{"status":"UP","version":"<commit id>","uptime":"1m5s"}
```

Route ini di luar `/api/v1`, jadi tidak kena request filter. Isinya hanya menandakan proses hidup dan tidak mengecek postgres/redis/partner.

### Tambah partner (external client)

```bash
cd iconpay-dana-integrator
svcgen add external dana
```

Kalau belum ada folder `external/`, svcgen membuatnya dan mendaftarkan `external.Module` di `appservice`. Lalu membuat `external/dana/` (config, http client, `Client.do` dengan klasifikasi SUCCESS/PENDING/FAILED dan timeout → `cerror.ErrTimeout`), mendaftarkan `dana.Module` di `external/external.go`, dan menambah `app.external.dana` di `config.yml`.

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

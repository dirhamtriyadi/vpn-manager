# VPN Manager

Monorepo aplikasi panel web untuk mengelola VPN server dan user dari browser. Saat ini runtime yang aktif adalah WireGuard; fondasi multi-protocol sudah disiapkan untuk OpenVPN, L2TP/IPsec, SSTP, dan PPTP legacy/insecure. Backend membuat interface WireGuard, generate key, generate config/QR client, membaca status handshake, dan menerapkan perubahan langsung ke kernel Linux via netlink. Frontend menyediakan dashboard React untuk membuat instance WireGuard, menambah peer, melihat status, copy/download config, script MikroTik RouterOS, serta Trash/Restore.

## Fitur

- Fondasi VPN Manager multi-protocol dengan daftar protocol: WireGuard, OpenVPN, L2TP/IPsec, SSTP, dan PPTP legacy/insecure.
- Endpoint generic awal `/api/v1/vpn/*` untuk protocol, instance, user, dan status.
- Semua protocol (WireGuard, OpenVPN, L2TP/IPsec, SSTP, PPTP) berstatus `available`/fungsional, bukan roadmap. WireGuard berjalan langsung via kernel netlink; protocol lain menulis config host dan menjalankan command provisioning saat di-apply.
- OpenVPN: metadata model, `.ovpn` generator, container manifest, encrypted secret store untuk CA/cert/key, persisted runtime manifest per instance, lifecycle plan, status parser, firewall/NAT ownership, dan apply endpoint (`POST /api/v1/vpn/openvpn/instances/{id}/apply`) yang benar-benar menjalankan `docker compose up` + iptables saat eksekusi diaktifkan.
- L2TP/IPsec, SSTP, dan PPTP: roadmap/service-plan/production-plan endpoints (`/api/v1/vpn/roadmaps/{protocol}`, `/api/v1/vpn/service-plans/{protocol}`, `/api/v1/vpn/production-plans/{protocol}`), config preview (`POST /api/v1/vpn/config-preview`), draft instance endpoints (`GET/POST /api/v1/vpn/{protocol}/instances`), apply endpoint (`POST /api/v1/vpn/{protocol}/instances/{id}/apply`) yang menulis config (ipsec/xl2tpd/sstpd/pptpd) dan menjalankan command + iptables, auto-migrated metadata models, plus halaman UI `/vpn/{protocol}`.
- Eksekusi runtime non-WireGuard dilindungi satu toggle keamanan tunggal `VPN_EXECUTION_ENABLED` (default `false`). Saat `false`, apply melakukan validasi lalu mengembalikan `412 Precondition Failed` tanpa menyentuh host. Aktifkan hanya pada host yang sudah memasang daemon yang dibutuhkan (docker; strongswan + xl2tpd; sstpd; pptpd).
- Response API memakai envelope konsisten: sukses `{"success":true,"code":"OK","message":...,"data":...}` (plus `meta` untuk paginasi dan `warning` saat apply ke kernel/runtime gagal sebagian); error `{"success":false,"code":"CONFLICT","message":...}` dengan `code` machine-readable dan HTTP status yang tepat (400/401/404/409/412/422/500). Error validasi menambahkan `errors:{field:[...]}`.
- Autentikasi multi-user berbasis database dengan password di-hash Argon2id. Bootstrap super-admin pertama dari `AUTH_USERNAME`/`AUTH_PASSWORD`, selanjutnya kelola user/role/permission via API.
- RBAC granular (Spatie-style): permission per aksi (`interfaces.create`, `peers.delete`, `vpn.apply`, dst), role yang membundel permission, dan permission langsung ke user; permission efektif = gabungan role ∪ direct. Endpoint `/users`, `/roles`, `/permissions` dengan guard `RequirePermission` di tiap route, plus role default `super-admin`/`operator`/`viewer`.
- Resource ownership: tiap interface/OpenVPN/instance VPN punya `owner_id`; user non-admin hanya melihat & mengelola VPN miliknya sendiri, super-admin (`*`) melihat semua. Inilah yang membuat tiap user bisa menangani VPN yang berbeda-beda.
- Roadmap protocol terdokumentasi di `docs/vpn-protocol-roadmap.md`.
- Buat WireGuard interface/server dari UI.
- Generate keypair server/client otomatis.
- Auto-assign IP tunnel peer.
- Generate client `.conf` dan QR code.
- Dashboard status live: online/offline, latest handshake, RX/TX.
- Enable/disable peer.
- Add/edit/delete peer secara incremental agar peer lain tidak ikut disconnect.
- Soft delete / Trash:
  - Delete biasa masuk Trash.
  - Restore peer/interface.
  - Delete permanently untuk membersihkan Trash.
- Script MikroTik RouterOS 7 untuk peer:
  - setup WireGuard client di MikroTik.
  - teardown script untuk membersihkan konfigurasi.
  - route otomatis berdasarkan allowed-address.
- Swagger UI untuk API backend.

## Struktur monorepo

```text
wireguard/
├── backend/              # Go API: Gin, GORM, Postgres, wgctrl/netlink
│   ├── main.go
│   ├── config/
│   ├── database/
│   ├── dto/
│   ├── handlers/
│   ├── middleware/
│   ├── models/
│   ├── routes/
│   ├── vpn/              # Protocol abstraction + WireGuard adapter/mappers
│   ├── wg/
│   ├── Dockerfile
│   ├── docker-compose.yml
│   ├── .env.example
│   └── README.md
├── frontend/             # React UI: Vite, TypeScript, Tailwind, shadcn-style UI
│   ├── src/
│   ├── package.json
│   ├── .env.example
│   └── README.md
└── README.md             # File ini
```

## Roadmap multi-protocol

Phase 1 menambahkan fondasi tanpa mematikan fitur WireGuard lama:

- `wireguard`: available sekarang.
- `openvpn`: fungsional via container OpenVPN + generation `.ovpn`; apply menjalankan `docker compose` saat `VPN_EXECUTION_ENABLED=true`.
- `l2tp_ipsec`: fungsional via strongSwan + xl2tpd; apply menulis config ipsec/xl2tpd dan menjalankan systemctl + iptables.
- `sstp`: fungsional via sstpd dengan TLS certificate; cocok untuk client Windows.
- `pptp`: fungsional tetapi legacy/insecure; hanya untuk kompatibilitas perangkat lama dan tidak direkomendasikan untuk deployment baru.

Endpoint generic awal:

```text
GET /api/v1/vpn/protocols
GET /api/v1/vpn/instances
GET /api/v1/vpn/instances/:id
GET /api/v1/vpn/instances/:id/users
GET /api/v1/vpn/instances/:id/status
```

Endpoint WireGuard lama (`/interfaces` dan `/peers`) tetap dipertahankan agar fitur existing tidak breaking.

## Stack

Backend:

- Go 1.22
- Gin
- GORM
- PostgreSQL
- WireGuard `wgctrl`
- Linux netlink via `vishvananda/netlink`
- Swagger / swaggo

Frontend:

- React 18
- TypeScript
- Vite
- Tailwind CSS
- react-hook-form
- Zod
- Axios
- lucide-react

## Kebutuhan host

Backend harus berjalan di Linux yang punya dukungan WireGuard kernel.

Install/cek di host:

```bash
sudo modprobe wireguard
lsmod | grep wireguard
```

Backend perlu akses netlink untuk membuat interface WireGuard, jadi proses/container perlu:

```text
CAP_NET_ADMIN
```

Jika ingin client VPN mengakses LAN/internet lewat server, aktifkan IP forwarding dan NAT di host WireGuard:

```bash
sudo sysctl -w net.ipv4.ip_forward=1
```

Contoh NAT dengan iptables, sesuaikan interface LAN/WAN host:

```bash
sudo iptables -t nat -A POSTROUTING -s 10.8.0.0/24 -o eth0 -j MASQUERADE
```

## Quick start development

Jika memakai root `Makefile`, kamu tidak perlu masuk ke folder `backend/` atau `frontend/` untuk command umum.

Setup awal:

```bash
make setup
```

Jalankan backend + frontend sekaligus:

```bash
make dev
```

Jika ingin sekalian start database Docker Compose:

```bash
make dev-db
```

Jalankan salah satu saja:

```bash
make backend
make frontend
```

Command lain:

```bash
make help
make db-up
make docker-up
make lint
make build
```

Catatan: backend lokal tetap perlu permission `CAP_NET_ADMIN` untuk membuat interface WireGuard. Jika `make backend` atau `make dev` gagal karena permission netlink, jalankan backend dengan user/root yang punya permission tersebut, atau gunakan mode Docker Compose.

### 1. Backend manual

```bash
cd backend
cp .env.example .env
```

Edit `.env` sesuai environment:

```env
SERVER_PORT=8080
GIN_MODE=debug
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=wg_panel
DB_SSLMODE=disable
DEFAULT_ENDPOINT=192.168.10.10
CORS_ALLOW_ORIGINS=http://localhost:5173

# Login admin (wajib diatur saat setup pertama)
AUTH_USERNAME=admin
AUTH_PASSWORD=change-me
AUTH_TOKEN_SECRET=          # isi string acak panjang di production: openssl rand -hex 32
AUTH_TOKEN_TTL_HOURS=24
```

> **Auth.** Panel dilindungi login admin tunggal. Kredensial dibaca dari env
> (`AUTH_USERNAME` / `AUTH_PASSWORD`) — ubah sebelum panel diekspos. Jika
> `AUTH_PASSWORD` kosong, login dimatikan dan seluruh API dikunci. Login lewat
> `POST /api/v1/auth/login` mengembalikan token bearer yang dikirim sebagai
> header `Authorization: Bearer <token>` pada tiap request. Kosongkan
> `AUTH_TOKEN_SECRET` hanya untuk dev (sesi reset tiap restart).

Jalankan PostgreSQL sendiri, atau pakai compose backend:

```bash
cd backend
docker compose up -d db
```

Jalankan API secara lokal:

```bash
cd backend
go mod download
sudo go run main.go
```

Kenapa `sudo`? Karena backend perlu membuat/mengubah interface WireGuard di kernel.

Swagger:

```text
http://localhost:8080/swagger/index.html
```

API base:

```text
http://localhost:8080/api/v1
```

### 2. Frontend

```bash
cd frontend
cp .env.example .env
npm install
npm run dev -- --host 0.0.0.0
```

Default `.env` frontend:

```env
VITE_API_BASE_URL=http://localhost:8080/api/v1
```

Buka:

```text
http://localhost:5173
```

## Menjalankan full-stack dengan Docker Compose

`backend/docker-compose.yml` menjalankan **tiga** service sekaligus: `db`
(PostgreSQL), `api` (backend Go), dan `web` (frontend di-build lalu disajikan
Nginx pada port `5173`).

Dari folder backend:

```bash
cd backend
docker compose up --build
```

Buka panel di `http://localhost:5173`, lalu login dengan `AUTH_USERNAME` /
`AUTH_PASSWORD`.

Konfigurasi lewat environment (atau buat file `backend/.env`) — semua punya
default, ubah seperlunya:

```env
# Login admin (WAJIB diubah sebelum diekspos)
AUTH_USERNAME=admin
AUTH_PASSWORD=change-me
AUTH_TOKEN_SECRET=            # production: openssl rand -hex 32
AUTH_TOKEN_TTL_HOURS=24

# URL API yang dipanggil browser — di-bake ke frontend saat build.
# Pakai IP yang bisa dijangkau browser, jangan "localhost" bila diakses dari mesin lain.
VITE_API_BASE_URL=http://localhost:8083/api/v1

# Origin frontend untuk CORS (harus cocok dengan alamat saat panel dibuka)
CORS_ALLOW_ORIGINS=http://localhost:5173
DEFAULT_ENDPOINT=vpn.example.com
```

> Mengubah `VITE_API_BASE_URL` perlu **rebuild** service `web`
> (`docker compose up --build web`) karena nilainya di-bake saat build, bukan
> runtime.

Catatan penting:

- Service `api` memakai `network_mode: host` + capability `NET_ADMIN`.
- Service `web` (Nginx) hanya menyajikan static; browser memanggil API langsung,
  jadi `VITE_API_BASE_URL` harus alamat API yang bisa dijangkau dari browser.
- Host tetap harus punya module WireGuard.
- Compose menjalankan PostgreSQL di port host `5434`.
- `iptables` sudah disertakan di image `api` untuk fitur internet client (masquerade).
- `AUTH_PASSWORD` kosong = login mati & API terkunci. Default-nya `change-me` —
  **ganti sebelum produksi.**

## Internet untuk client (NAT / Masquerade)

WireGuard hanya mendorong paket ke dalam tunnel; agar client mendapat **internet**,
server harus mem-forward + NAT paket client keluar lewat uplink. Saat membuat
interface, centang **Internet access (NAT/masquerade)**:

- Backend otomatis mengaktifkan `net.ipv4.ip_forward` dan memasang aturan
  `iptables` (MASQUERADE + FORWARD) dari subnet tunnel ke interface uplink.
- **Egress interface** boleh dikosongkan → server mendeteksi interface
  default-route dan menyimpannya, sehingga aturan bisa **dihapus kembali** secara
  tepat.
- Aturan dilepas otomatis saat interface **dimatikan, dihapus, atau di-purge**
  (`ip_forward` sengaja dibiarkan menyala karena bisa dipakai tunnel lain).
- Client juga perlu `AllowedIPs = 0.0.0.0/0` (default di form peer) dan `DNS`
  terisi agar internet jalan penuh. Untuk akses internal saja (split tunnel),
  matikan masquerade dan set `AllowedIPs` client ke subnet tertentu.

Catatan: butuh `CAP_NET_ADMIN`/root dan biner `iptables` di host/kontainer.

Jika frontend berjalan dari host lain, sesuaikan:

```env
CORS_ALLOW_ORIGINS=http://IP_FRONTEND:5173
```

Dan di frontend:

```env
VITE_API_BASE_URL=http://IP_BACKEND:8083/api/v1
```

## Build production

Backend:

```bash
cd backend
go test ./...
go build -o wg-panel ./main.go
```

Frontend:

```bash
cd frontend
npm run lint
npm run build
```

Preview frontend build:

```bash
npm run preview -- --host 0.0.0.0
```

## Topologi contoh lokal

Contoh topologi yang cocok untuk simulasi lokal:

```text
ONT ISP/router
  |
MikroTik hAP ac2
  | LAN 192.168.10.0/24
Mini PC / WireGuard server / WG Panel
  IP: 192.168.10.10
  UDP WireGuard: 51820
```

Untuk simulasi lokal, endpoint WireGuard di panel bisa diisi:

```text
192.168.10.10:51820
```

atau jika field host dan port terpisah:

```text
Endpoint/Host: 192.168.10.10
Listen Port: 51820
```

Untuk akses dari internet, endpoint harus IP publik/DDNS, bukan IP private seperti `192.168.x.x`.

## Alur penggunaan

1. Buka frontend.
2. Klik `New interface`.
3. Isi contoh:

```text
Name: wg0
Listen Port: 51820
Address: 10.8.0.1/24
Endpoint: 192.168.10.10
DNS: 1.1.1.1
MTU: 1420
```

4. Klik `Add peer`.
5. Buka tombol QR/config pada peer.
6. Import config ke client WireGuard, atau copy script MikroTik RouterOS jika peer akan dipakai di MikroTik.
7. Cek status handshake di dashboard.

## MikroTik RouterOS peer script

Panel menyediakan script RouterOS 7 untuk membuat MikroTik sebagai WireGuard client.

Contoh hasil yang diharapkan:

```routeros
/interface/wireguard/add name="wg-router" mtu=1420 private-key="..." comment="wg-panel wg-router"
/ip/address/add address=10.8.0.2/32 interface="wg-router" comment="wg-panel wg-router"
/interface/wireguard/peers/add interface="wg-router" public-key="..." endpoint-address=192.168.10.10 endpoint-port=51820 allowed-address=10.8.0.0/24 persistent-keepalive=25s comment="wg-panel wg-router" preshared-key="..."
/ip/route/add dst-address=10.8.0.0/24 gateway="wg-router" comment="wg-panel wg-router"
/ping 10.8.0.1 count=5
```

Catatan:

- Jangan commit private key/preshared key asli ke GitHub.
- `allowed-address` RouterOS harus tanpa spasi setelah koma jika berisi banyak CIDR, contoh:

```routeros
allowed-address=0.0.0.0/0,::/0
```

- Untuk test lokal, mulai dari:

```routeros
allowed-address=10.8.0.0/24
```

Cek di MikroTik:

```routeros
/interface/wireguard/peers/print detail
/ping 10.8.0.1 count=5
```

Handshake berhasil jika muncul `last-handshake`, `rx`, dan `tx`.

## Trash / soft delete

Delete biasa tidak langsung menghapus permanen.

Peer:

```text
Delete peer -> masuk Trash -> peer dicabut dari kernel WireGuard
Restore peer -> peer aktif lagi -> peer dimasukkan kembali ke kernel
Delete permanently -> record peer dihapus permanen dari database
```

Interface:

```text
Delete interface -> masuk Trash -> WireGuard device dihapus dari kernel -> peers ikut masuk Trash
Restore interface -> interface dan peers direstore -> apply ulang ke kernel
Delete permanently -> interface dan peers dihapus permanen dari database
```

Di dashboard akan muncul section `Trash` jika ada item terhapus.

## API ringkas

Base path:

```text
/api/v1
```

Interface:

| Method | Path | Keterangan |
|---|---|---|
| GET | `/interfaces` | List interface aktif |
| POST | `/interfaces` | Buat interface |
| GET | `/interfaces/:id` | Detail interface |
| PUT | `/interfaces/:id` | Update interface |
| DELETE | `/interfaces/:id` | Soft delete interface |
| POST | `/interfaces/:id/sync` | Full apply/reconcile ke kernel |
| GET | `/interfaces/:id/status` | Status live interface + peers |
| GET | `/interfaces/trash` | List interface di Trash |
| POST | `/interfaces/:id/restore` | Restore interface |
| DELETE | `/interfaces/:id/purge` | Delete permanen interface |

Peer:

| Method | Path | Keterangan |
|---|---|---|
| GET | `/interfaces/:id/peers` | List peers aktif di interface |
| POST | `/interfaces/:id/peers` | Tambah peer |
| PUT | `/peers/:peerId` | Update peer |
| DELETE | `/peers/:peerId` | Soft delete peer |
| GET | `/peers/:peerId/config` | Download config `.conf` |
| GET | `/peers/:peerId/qrcode` | QR code config |
| GET | `/peers/trash` | List peer di Trash |
| POST | `/peers/:peerId/restore` | Restore peer |
| DELETE | `/peers/:peerId/purge` | Delete permanen peer |

## Catatan koneksi peer

Panel sekarang menerapkan perubahan peer secara incremental:

```text
Tambah peer     -> hanya peer baru yang ditambahkan
Edit peer       -> hanya peer itu yang di-update
Delete peer     -> hanya peer itu yang dicabut dari kernel
Full sync/apply -> replace seluruh daftar peer sebagai recovery/manual reconcile
```

Artinya tambah/hapus peer tidak seharusnya membuat peer lain disconnect.

## Troubleshooting

### Peer tidak handshake

Cek backend/server:

```bash
sudo wg show
```

Cek port listen:

```bash
sudo ss -lunp | grep 51820
```

Cek firewall host mengizinkan UDP WireGuard.

### MikroTik tidak bisa ping server tunnel

Dari MikroTik:

```routeros
/ping 192.168.10.10 count=5
/interface/wireguard/peers/print detail
/ping 10.8.0.1 count=5
```

Jika `last-handshake` kosong:

- endpoint salah.
- public key server/client tidak cocok.
- preshared key tidak cocok.
- UDP 51820 terblokir.
- peer belum terdaftar di panel.

### Akses dari luar rumah tidak jalan

Jika ISP memakai CGNAT atau WAN router mendapat IP private, port forward langsung tidak cukup. Opsi:

- minta public IP ke ISP.
- bridge ONT agar router mendapat public IP.
- pakai VPS relay WireGuard.
- pakai overlay VPN seperti Tailscale/ZeroTier/NetBird.

## Keamanan

- Jangan commit file `.env` asli.
- Jangan commit private key WireGuard atau preshared key asli.
- Batasi CORS di production.
- Jalankan backend hanya di host terpercaya karena punya `CAP_NET_ADMIN`.
- Backup database sebelum purge/delete permanen.

## Development notes

Command yang sering dipakai:

```bash
# frontend
cd frontend
npm run lint
npm run build

# backend
cd backend
go test ./...
go build ./...
```

Jika Swagger perlu digenerate ulang:

```bash
cd backend
swag init -g main.go -o docs
```

## License

Belum ditentukan. Tambahkan file `LICENSE` jika repository akan dipublikasikan/open-source.

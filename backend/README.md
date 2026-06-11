# WireGuard Panel — Backend (Gin + GORM + Postgres + swaggo + validator)

A management API for a WireGuard VPN concentrator. You manage **interfaces**
(the server side) and **peers** (clients) over HTTP — no `wg` / `wg-quick` CLI.

Keys, client `.conf` files and QR codes are generated server-side; peer changes
are pushed straight to the kernel WireGuard device via **netlink** (`wgctrl` +
`vishvananda/netlink`), so there is nothing to type on the box.

## Stack
- **Gin** — HTTP framework
- **GORM** + **Postgres** — store interface/peer metadata
- **wgctrl-go** — configure the WireGuard device (keys, port, peers) via netlink
- **vishvananda/netlink** — create the link & assign its address
- **skip2/go-qrcode** — client-config QR codes
- **swaggo/gin-swagger** — Swagger UI
- **go-playground/validator** — request validation

## How "no CLI" works
- `POST /interfaces` generates a key pair (or accepts yours), stores metadata,
  then creates the `wgX` link, assigns its address and brings it up.
- `POST /interfaces/:id/peers` generates the client key pair + preshared key,
  auto-allocates the next free tunnel IP, saves it, and reconciles the kernel
  device so the peer is live immediately.
- `GET /peers/:id/config` and `GET /peers/:id/qrcode` hand the user a ready
  config / QR to import into the WireGuard app.
- `GET /interfaces/:id/status` reads live handshake & transfer counters.

## Project structure
```
backend/
├── main.go                 # entrypoint + swagger info
├── config/                 # env config
├── database/               # GORM connection + auto migrate
├── models/                 # WGInterface, Peer
├── dto/                    # request/response structs (validator tags)
├── middleware/validator.go # validation helper
├── wg/service.go           # keygen, netlink link, wgctrl device, stats, configs, IP alloc
├── handlers/               # interface + peer handlers, reconcile()
├── routes/                 # router + CORS + swagger
└── docs/                   # generated swagger
```

## Requirements (host)
- Linux with the **wireguard** kernel module (`modprobe wireguard`).
- The process needs **CAP_NET_ADMIN** (run as root or grant the capability).
- For full internet egress for clients, enable forwarding and add NAT once on the host:
  ```bash
  sysctl -w net.ipv4.ip_forward=1
  # NAT example (replace eth0 with your WAN interface):
  nft add table ip nat 2>/dev/null
  nft add chain ip nat postrouting '{ type nat hook postrouting priority 100; }' 2>/dev/null
  nft add rule ip nat postrouting oifname "eth0" masquerade
  ```
  (This NAT step is the only one-time host setup; everything else is via the API.)

## Quick start (local)
```bash
cp .env.example .env        # set DB creds + DEFAULT_ENDPOINT
go mod tidy
sudo go run main.go         # sudo: needs CAP_NET_ADMIN to touch the device
```
Swagger UI: http://localhost:8080/swagger/index.html

> Note: if you run without the wireguard module or without NET_ADMIN, the API
> still works for metadata, config and QR generation — kernel apply steps return
> a clear message (`saved but not applied to kernel: ...`) instead of crashing.

## Quick start (Docker)
```bash
docker compose up --build
```
The `api` service uses host networking + `NET_ADMIN`; the host kernel must
provide the wireguard module.

## Regenerate Swagger
```bash
swag init -g main.go -o docs
```

## Endpoints (base `/api/v1`)
| Method | Path                         | Description                         |
|--------|------------------------------|-------------------------------------|
| GET    | `/interfaces`                | List interfaces                     |
| POST   | `/interfaces`                | Create interface (auto keygen)      |
| GET    | `/interfaces/:id`            | Get interface + peers               |
| PUT    | `/interfaces/:id`            | Update interface                    |
| DELETE | `/interfaces/:id`            | Delete interface (tears down link)  |
| POST   | `/interfaces/:id/sync`       | Re-apply interface to the kernel    |
| GET    | `/interfaces/:id/status`     | Live handshake/transfer per peer    |
| GET    | `/interfaces/:id/peers`      | List peers                          |
| POST   | `/interfaces/:id/peers`      | Add peer (auto key/IP/PSK)          |
| PUT    | `/peers/:peerId`             | Update peer                         |
| DELETE | `/peers/:peerId`             | Delete peer                         |
| GET    | `/peers/:peerId/config`      | Download client `.conf`             |
| GET    | `/peers/:peerId/qrcode`      | Client config QR (PNG)              |

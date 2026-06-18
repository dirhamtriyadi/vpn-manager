# VPN Protocol Roadmap

Status implementasi multi-protocol saat ini bersifat bertahap. WireGuard tetap menjadi runtime aktif; protocol lain sekarang punya service-plan scaffold dan readiness/enablement gate, tetapi belum diaktifkan sampai driver runtime/service diuji di host.

## Status protocol

| Protocol | Status | Runtime strategy | Config download | QR | Certificates | Catatan |
| --- | --- | --- | --- | --- | --- | --- |
| WireGuard | Available | `host_kernel_netlink` | Ya | Ya | Tidak | Menggunakan model/interface/peer yang sudah ada. |
| OpenVPN | Roadmap | `container_openvpn_preview` | Ya (`.ovpn`) | Tidak | Ya | Butuh CA, sertifikat server/client, template config, manifest container, dan service status. |
| L2TP/IPsec | Roadmap | `host_ipsec_ppp` | Tidak | Tidak | Opsional/PSK | Butuh IPsec/IKE daemon, PPP users, secrets, firewall/NAT. |
| SSTP | Roadmap | `container_or_host_sstp` | Tidak | Tidak | Ya | Butuh daemon SSTP, TLS cert, users, dan status integration. |
| PPTP | Legacy roadmap | `legacy_host_pptpd` | Tidak | Tidak | Tidak | Legacy/insecure; hanya untuk kompatibilitas client lama. |

## Prinsip implementasi

1. Jangan menandai protocol sebagai `available` sebelum driver runtime benar-benar bisa create/sync/status/config.
2. Endpoint `/api/v1/vpn/protocols` mengambil metadata dari protocol specs dan availability dari driver registry.
3. UI boleh menampilkan roadmap/capability, tetapi tombol create harus disabled untuk protocol tanpa driver.
4. OpenVPN/L2TP/SSTP/PPTP perlu model/config sendiri; jangan dipaksakan ke tabel WireGuard `interfaces` dan `peers`.
5. PPTP harus selalu diberi warning legacy/insecure.

## Generic service-plan endpoints

Semua protocol sekarang punya endpoint generic untuk roadmap dan dry-run service plan:

- `GET /api/v1/vpn/roadmaps/{protocol}` untuk readiness, enablement blockers, dan component list.
- `GET /api/v1/vpn/service-plans/{protocol}` untuk dry-run runtime/firewall/user plan.
- `GET /api/v1/vpn/production-plans/{protocol}` untuk production command checklist, config-file targets, firewall commands, status commands, dan blocker gate.
- `POST /api/v1/vpn/openvpn/instances/{id}/apply` untuk menulis manifest OpenVPN dan menjalankan firewall/docker compose saat semua gate aktif.
- `POST /api/v1/vpn/{protocol}/instances/{id}/apply` untuk L2TP/IPsec, SSTP, atau PPTP; endpoint ini menulis file `/etc/*` dan menjalankan service/firewall command hanya saat semua gate aktif.
- `POST /api/v1/vpn/config-preview` untuk preview file konfigurasi L2TP/IPsec, SSTP, atau PPTP tanpa menulis ke host.
- `GET/POST /api/v1/vpn/{protocol}/instances` untuk draft instance L2TP/IPsec, SSTP, atau PPTP; default `enabled=false` dan runtime tetap disabled.

Protocol values: `wireguard`, `openvpn`, `l2tp_ipsec`, `sstp`, `pptp`.

Untuk L2TP/IPsec, SSTP, dan PPTP, UI tidak lagi berhenti di tombol `Coming soon`; halaman `/vpn/{protocol}` menampilkan service plan lengkap. Plan ini tetap tidak menginstall daemon, tidak menjalankan service, dan tidak apply firewall.

Global enablement gates untuk protocol non-WireGuard:

- `VPN_RUNTIME_EXECUTION_ENABLED=true`
- `VPN_FIREWALL_APPLY_ENABLED=true`
- `VPN_HOST_VERIFICATION_PASSED=true`

Gate ini hanya readiness signal; protocol tetap baru boleh `available=true` setelah ada driver runtime nyata yang didaftarkan di registry. Production-plan endpoint sudah menampilkan command/config checklist untuk tiap protocol dan mode `blocked`/`manual`/`executor_enabled`, tetapi tidak menjalankan command dari UI.

## Next implementation candidate

Protocol paling masuk akal setelah WireGuard adalah OpenVPN karena output client bisa berupa file `.ovpn` dan runtime bisa diisolasi lewat container. Scaffold awal sudah tersedia:

- Model metadata: `OpenVPNInstance` dan `OpenVPNUser`.
- Secret storage scaffold: `EncryptedSecret` + `backend/secrets` envelope encryption/helper refs.
- Endpoint roadmap: `GET /api/v1/vpn/openvpn/roadmap`.
- Endpoint draft instance: `GET/POST /api/v1/vpn/openvpn/instances`.
- Endpoint persisted runtime manifest: `GET/POST /api/v1/vpn/openvpn/instances/{id}/runtime-manifest`.
- Endpoint draft users/clients: `GET/POST /api/v1/vpn/openvpn/instances/{id}/users`.
- Endpoint dry-run lifecycle plan: `POST /api/v1/vpn/openvpn/instances/{id}/lifecycle/{action}`.
- Endpoint firewall/NAT ownership plan: `POST /api/v1/vpn/openvpn/instances/{id}/firewall-plan`.
- Endpoint status parser preview: `POST /api/v1/vpn/openvpn/status/parse`.
- Endpoint preview profil: `POST /api/v1/vpn/openvpn/client-profile/preview`.
- Endpoint preview runtime container: `POST /api/v1/vpn/openvpn/runtime/preview`.
- Generator `.ovpn` inline certificate: `backend/openvpn.BuildClientProfile`.
- Generator preview `server.conf` + `docker-compose.yml`: `backend/openvpn.BuildContainerRuntimeManifest`.
- UI roadmap + manifest preview: `/vpn/openvpn`.

Endpoint draft instance membutuhkan environment `OPENVPN_SECRET_MASTER_KEY` sebelum menerima CA/cert/private-key material. Endpoint menyimpan ciphertext di `EncryptedSecret` dan mengembalikan reference/status saja; OpenVPN tetap disabled dan tidak menjalankan container. Endpoint runtime manifest per instance menyimpan hasil generator `server.conf` dan `docker-compose.yml` untuk draft tersebut. Endpoint user draft menyimpan client cert/key terenkripsi. Endpoint lifecycle dan firewall saat ini menghasilkan dry-run plan saja; perintah dan rule tidak dieksekusi/diterapkan otomatis.

Endpoint preview hanya untuk validasi/generator sementara dan tidak berarti OpenVPN sudah enabled. Secret storage, manifest persistence, lifecycle dry-run, status parser, firewall plan, dan user draft scaffold sudah ada. OpenVPN belum boleh menjadi `available` sampai eksekusi runtime/firewall benar-benar diverifikasi di host dan diaktifkan secara eksplisit.

Operational enablement gates:

- `OPENVPN_RUNTIME_EXECUTION_ENABLED=true` untuk mengizinkan eksekusi command container di phase berikutnya.
- `OPENVPN_FIREWALL_APPLY_ENABLED=true` untuk mengizinkan apply firewall/NAT rule di phase berikutnya.
- `OPENVPN_HOST_VERIFICATION_PASSED=true` hanya setelah `go test`, `go build`, dan review dry-run plan lulus di host deployment.

Selama gate belum lengkap, endpoint lifecycle/firewall tetap dry-run plan dan OpenVPN tetap `available=false`.

Sebelum OpenVPN bisa dibuat sebagai instance aktif, lanjutkan dengan:

- Jalankan Go test/build di host.
- Review hasil dry-run lifecycle/firewall plan pada environment deployment.
- Tambahkan feature flag/setting eksplisit untuk mengizinkan eksekusi container dan apply firewall rule.

# VPN Protocol Roadmap

Status implementasi multi-protocol saat ini bersifat bertahap. WireGuard tetap menjadi runtime aktif; protocol lain belum diaktifkan sampai runtime/service strategy dipilih dan diuji.

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

## Next implementation candidate

Protocol paling masuk akal setelah WireGuard adalah OpenVPN karena output client bisa berupa file `.ovpn` dan runtime bisa diisolasi lewat container. Scaffold awal sudah tersedia:

- Model metadata: `OpenVPNInstance` dan `OpenVPNUser`.
- Secret storage scaffold: `EncryptedSecret` + `backend/secrets` envelope encryption/helper refs.
- Endpoint roadmap: `GET /api/v1/vpn/openvpn/roadmap`.
- Endpoint draft instance: `GET/POST /api/v1/vpn/openvpn/instances`.
- Endpoint persisted runtime manifest: `GET/POST /api/v1/vpn/openvpn/instances/{id}/runtime-manifest`.
- Endpoint preview profil: `POST /api/v1/vpn/openvpn/client-profile/preview`.
- Endpoint preview runtime container: `POST /api/v1/vpn/openvpn/runtime/preview`.
- Generator `.ovpn` inline certificate: `backend/openvpn.BuildClientProfile`.
- Generator preview `server.conf` + `docker-compose.yml`: `backend/openvpn.BuildContainerRuntimeManifest`.
- UI roadmap + manifest preview: `/vpn/openvpn`.

Endpoint draft instance membutuhkan environment `OPENVPN_SECRET_MASTER_KEY` sebelum menerima CA/cert/private-key material. Endpoint menyimpan ciphertext di `EncryptedSecret` dan mengembalikan reference/status saja; OpenVPN tetap disabled dan tidak menjalankan container. Endpoint runtime manifest per instance menyimpan hasil generator `server.conf` dan `docker-compose.yml` untuk draft tersebut, tetapi tetap tidak menjalankan container.

Endpoint preview hanya untuk validasi/generator sementara dan tidak berarti OpenVPN sudah enabled. Secret storage dan manifest persistence scaffold sudah ada, tetapi runtime/service OpenVPN belum boleh diaktifkan sebelum lifecycle dan firewall ownership selesai.

Sebelum OpenVPN bisa dibuat sebagai instance aktif, lanjutkan dengan:

- Persist generated runtime manifests per instance.
- Cara start/stop/reload container.
- Cara membaca connected clients/status.
- Firewall/NAT ownership.

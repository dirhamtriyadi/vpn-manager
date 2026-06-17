# VPN Protocol Roadmap

Status implementasi multi-protocol saat ini bersifat bertahap. WireGuard tetap menjadi runtime aktif; protocol lain belum diaktifkan sampai runtime/service strategy dipilih dan diuji.

## Status protocol

| Protocol | Status | Runtime strategy | Config download | QR | Certificates | Catatan |
| --- | --- | --- | --- | --- | --- | --- |
| WireGuard | Available | `host_kernel_netlink` | Ya | Ya | Tidak | Menggunakan model/interface/peer yang sudah ada. |
| OpenVPN | Roadmap | `container_or_host_openvpn` | Ya (`.ovpn`) | Tidak | Ya | Butuh CA, sertifikat server/client, template config, dan service status. |
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

Protocol paling masuk akal setelah WireGuard adalah OpenVPN karena output client bisa berupa file `.ovpn` dan runtime bisa diisolasi lewat container. Sebelum implementasi penuh, tentukan dulu:

- Host-native atau container OpenVPN.
- Lokasi penyimpanan CA/cert/key.
- Template server/client config.
- Cara start/stop/reload service.
- Cara membaca connected clients/status.
- Firewall/NAT ownership.

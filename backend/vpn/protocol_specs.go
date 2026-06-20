package vpn

import "github.com/example/wg-panel/models"

const (
	ProtocolStatusAvailable       = "available"
	ProtocolStatusLegacyAvailable = "legacy_available"
)

type ProtocolSpec struct {
	Protocol       models.VPNProtocol
	Label          string
	Status         string
	Description    string
	LegacyInsecure bool
	Capabilities   ProtocolCapabilities
}

func AllProtocolSpecs() []ProtocolSpec {
	return []ProtocolSpec{
		{
			Protocol:       models.ProtocolWireGuard,
			Label:          "WireGuard",
			Status:         ProtocolStatusAvailable,
			Description:    "Fast kernel-backed VPN using the existing WireGuard interface and peer workflow.",
			LegacyInsecure: false,
			Capabilities: ProtocolCapabilities{
				RuntimeStrategy:      "host_kernel_netlink",
				ConfigDownload:       true,
				QRCode:               true,
				RequiresCertificates: false,
			},
		},
		{
			Protocol:       models.ProtocolOpenVPN,
			Label:          "OpenVPN",
			Status:         ProtocolStatusAvailable,
			Description:    "Containerized OpenVPN with certificate-authority secret storage, server config + .ovpn generation, and gated runtime apply.",
			LegacyInsecure: false,
			Capabilities: ProtocolCapabilities{
				RuntimeStrategy:      "container_openvpn",
				ConfigDownload:       true,
				QRCode:               false,
				RequiresCertificates: true,
			},
		},
		{
			Protocol:       models.ProtocolL2TPIPsec,
			Label:          "L2TP/IPsec",
			Status:         ProtocolStatusAvailable,
			Description:    "Host IPsec/IKE (strongSwan) + xl2tpd with PPP users, PSK handling, firewall/NAT rules, and gated runtime apply.",
			LegacyInsecure: false,
			Capabilities: ProtocolCapabilities{
				RuntimeStrategy:      "host_ipsec_ppp",
				ConfigDownload:       false,
				QRCode:               false,
				RequiresCertificates: false,
			},
		},
		{
			Protocol:       models.ProtocolSSTP,
			Label:          "SSTP",
			Status:         ProtocolStatusAvailable,
			Description:    "Host SSTP daemon with TLS certificate material, PPP users, service status integration, and gated runtime apply.",
			LegacyInsecure: false,
			Capabilities: ProtocolCapabilities{
				RuntimeStrategy:      "host_sstp",
				ConfigDownload:       false,
				QRCode:               false,
				RequiresCertificates: true,
			},
		},
		{
			Protocol:       models.ProtocolPPTP,
			Label:          "PPTP",
			Status:         ProtocolStatusLegacyAvailable,
			Description:    "Legacy/insecure compatibility protocol (pptpd); functional but enable only for old clients that cannot use safer VPNs.",
			LegacyInsecure: true,
			Capabilities: ProtocolCapabilities{
				RuntimeStrategy:      "host_pptpd",
				ConfigDownload:       false,
				QRCode:               false,
				RequiresCertificates: false,
			},
		},
	}
}

func GetProtocolSpec(protocol models.VPNProtocol) (ProtocolSpec, bool) {
	for _, spec := range AllProtocolSpecs() {
		if spec.Protocol == protocol {
			return spec, true
		}
	}
	return ProtocolSpec{}, false
}

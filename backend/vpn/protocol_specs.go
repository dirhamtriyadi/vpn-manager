package vpn

import "github.com/example/wg-panel/models"

const (
	ProtocolStatusAvailable     = "available"
	ProtocolStatusRoadmap       = "roadmap"
	ProtocolStatusLegacyRoadmap = "legacy_roadmap"
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
			Status:         ProtocolStatusRoadmap,
			Description:    "Roadmap protocol; needs OpenVPN runtime, certificate authority, server config, and .ovpn generation.",
			LegacyInsecure: false,
			Capabilities: ProtocolCapabilities{
				RuntimeStrategy:      "container_or_host_openvpn",
				ConfigDownload:       true,
				QRCode:               false,
				RequiresCertificates: true,
			},
		},
		{
			Protocol:       models.ProtocolL2TPIPsec,
			Label:          "L2TP/IPsec",
			Status:         ProtocolStatusRoadmap,
			Description:    "Roadmap protocol; needs IPsec/IKE daemon, PPP users, PSK/certificate handling, and firewall/NAT rules.",
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
			Status:         ProtocolStatusRoadmap,
			Description:    "Roadmap protocol; needs SSTP daemon, TLS certificate management, users, and service status integration.",
			LegacyInsecure: false,
			Capabilities: ProtocolCapabilities{
				RuntimeStrategy:      "container_or_host_sstp",
				ConfigDownload:       false,
				QRCode:               false,
				RequiresCertificates: true,
			},
		},
		{
			Protocol:       models.ProtocolPPTP,
			Label:          "PPTP",
			Status:         ProtocolStatusLegacyRoadmap,
			Description:    "Legacy/insecure compatibility protocol; only consider for old clients that cannot use safer VPNs.",
			LegacyInsecure: true,
			Capabilities: ProtocolCapabilities{
				RuntimeStrategy:      "legacy_host_pptpd",
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

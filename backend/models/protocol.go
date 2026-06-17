package models

import "strings"

// VPNProtocol identifies a supported VPN protocol family. Only WireGuard is
// available in Phase 1; the other values are exposed so the API/UI can present a
// stable roadmap without pretending runtime support exists yet.
type VPNProtocol string

const (
	ProtocolWireGuard VPNProtocol = "wireguard"
	ProtocolOpenVPN   VPNProtocol = "openvpn"
	ProtocolL2TPIPsec VPNProtocol = "l2tp_ipsec"
	ProtocolSSTP      VPNProtocol = "sstp"
	ProtocolPPTP      VPNProtocol = "pptp"
)

func (p VPNProtocol) String() string {
	return string(p)
}

func ParseVPNProtocol(value string) (VPNProtocol, bool) {
	switch VPNProtocol(strings.ToLower(strings.TrimSpace(value))) {
	case ProtocolWireGuard:
		return ProtocolWireGuard, true
	case ProtocolOpenVPN:
		return ProtocolOpenVPN, true
	case ProtocolL2TPIPsec:
		return ProtocolL2TPIPsec, true
	case ProtocolSSTP:
		return ProtocolSSTP, true
	case ProtocolPPTP:
		return ProtocolPPTP, true
	default:
		return "", false
	}
}

func (p VPNProtocol) IsLegacyInsecure() bool {
	return p == ProtocolPPTP
}

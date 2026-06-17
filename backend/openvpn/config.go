package openvpn

import (
	"fmt"
	"strings"
)

type ClientProfileInput struct {
	ClientName    string
	RemoteHost    string
	RemotePort    int
	Protocol      string
	CACertPEM     string
	ClientCertPEM string
	ClientKeyPEM  string
	TLSAuthPEM    string
}

func BuildClientProfile(input ClientProfileInput) (string, error) {
	protocol := normalizeProtocol(input.Protocol)
	if strings.TrimSpace(input.ClientName) == "" {
		return "", fmt.Errorf("client name is required")
	}
	if strings.TrimSpace(input.RemoteHost) == "" {
		return "", fmt.Errorf("remote host is required")
	}
	if input.RemotePort <= 0 || input.RemotePort > 65535 {
		return "", fmt.Errorf("remote port must be between 1 and 65535")
	}
	if strings.TrimSpace(input.CACertPEM) == "" {
		return "", fmt.Errorf("ca certificate is required")
	}
	if strings.TrimSpace(input.ClientCertPEM) == "" {
		return "", fmt.Errorf("client certificate is required")
	}
	if strings.TrimSpace(input.ClientKeyPEM) == "" {
		return "", fmt.Errorf("client private key is required")
	}

	var b strings.Builder
	b.WriteString("client\n")
	b.WriteString("dev tun\n")
	b.WriteString("proto " + protocol + "\n")
	b.WriteString(fmt.Sprintf("remote %s %d\n", strings.TrimSpace(input.RemoteHost), input.RemotePort))
	b.WriteString("resolv-retry infinite\n")
	b.WriteString("nobind\n")
	b.WriteString("persist-key\n")
	b.WriteString("persist-tun\n")
	b.WriteString("remote-cert-tls server\n")
	b.WriteString("cipher AES-256-GCM\n")
	b.WriteString("auth SHA256\n")
	b.WriteString("verb 3\n")
	b.WriteString("\n<ca>\n")
	b.WriteString(strings.TrimSpace(input.CACertPEM))
	b.WriteString("\n</ca>\n\n<cert>\n")
	b.WriteString(strings.TrimSpace(input.ClientCertPEM))
	b.WriteString("\n</cert>\n\n<key>\n")
	b.WriteString(strings.TrimSpace(input.ClientKeyPEM))
	b.WriteString("\n</key>\n")
	if strings.TrimSpace(input.TLSAuthPEM) != "" {
		b.WriteString("\n<tls-crypt>\n")
		b.WriteString(strings.TrimSpace(input.TLSAuthPEM))
		b.WriteString("\n</tls-crypt>\n")
	}
	return b.String(), nil
}

func normalizeProtocol(protocol string) string {
	switch strings.ToLower(strings.TrimSpace(protocol)) {
	case "tcp", "tcp4", "tcp6":
		return "tcp"
	case "udp", "udp4", "udp6", "":
		return "udp"
	default:
		return "udp"
	}
}

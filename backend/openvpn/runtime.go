package openvpn

import (
	"fmt"
	"net"
	"regexp"
	"strings"
)

type RuntimeManifestInput struct {
	InstanceName string
	RemoteHost   string
	ListenPort   int
	Protocol     string
	TunnelCIDR   string
	DNS          string
}

type RuntimeManifest struct {
	RuntimeMode string            `json:"runtime_mode"`
	Files       map[string]string `json:"files"`
	Warnings    []string          `json:"warnings"`
}

func BuildContainerRuntimeManifest(input RuntimeManifestInput) (RuntimeManifest, error) {
	instanceName := sanitizeInstanceName(input.InstanceName)
	if instanceName == "" {
		return RuntimeManifest{}, fmt.Errorf("instance name is required")
	}
	if strings.TrimSpace(input.RemoteHost) == "" {
		return RuntimeManifest{}, fmt.Errorf("remote host is required")
	}
	port := input.ListenPort
	if port == 0 {
		port = 1194
	}
	if port < 1 || port > 65535 {
		return RuntimeManifest{}, fmt.Errorf("listen port must be between 1 and 65535")
	}
	protocol := normalizeProtocol(input.Protocol)
	networkIP, netmask, err := openVPNServerNetwork(input.TunnelCIDR)
	if err != nil {
		return RuntimeManifest{}, err
	}

	serverConf := buildServerConf(serverConfInput{
		Port:       port,
		Protocol:   protocol,
		NetworkIP:  networkIP,
		Netmask:    netmask,
		DNS:        input.DNS,
		RemoteHost: input.RemoteHost,
	})
	compose := buildDockerCompose(composeInput{
		InstanceName: instanceName,
		Port:         port,
		Protocol:     protocol,
	})

	return RuntimeManifest{
		RuntimeMode: "container",
		Files: map[string]string{
			"server.conf":        serverConf,
			"docker-compose.yml": compose,
		},
		Warnings: []string{
			"Preview only: do not deploy until CA/certificate/key storage and lifecycle management are implemented.",
			"Container needs NET_ADMIN and /dev/net/tun access on the host.",
		},
	}, nil
}

type serverConfInput struct {
	Port      int
	Protocol  string
	NetworkIP string
	Netmask   string
	DNS        string
	RemoteHost string
}

func buildServerConf(input serverConfInput) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("port %d\n", input.Port))
	b.WriteString("proto " + input.Protocol + "\n")
	b.WriteString("dev tun\n")
	b.WriteString("topology subnet\n")
	b.WriteString(fmt.Sprintf("server %s %s\n", input.NetworkIP, input.Netmask))
	b.WriteString("keepalive 10 120\n")
	b.WriteString("persist-key\n")
	b.WriteString("persist-tun\n")
	b.WriteString("user nobody\n")
	b.WriteString("group nogroup\n")
	b.WriteString("cipher AES-256-GCM\n")
	b.WriteString("auth SHA256\n")
	b.WriteString("verb 3\n")
	b.WriteString("explicit-exit-notify 1\n")
	if strings.TrimSpace(input.DNS) != "" {
		b.WriteString(fmt.Sprintf("push \"dhcp-option DNS %s\"\n", strings.TrimSpace(input.DNS)))
	}
	b.WriteString("# Remote host for generated client profiles: " + strings.TrimSpace(input.RemoteHost) + "\n")
	b.WriteString("# Certificate/key paths are intentionally externalized for a future secret store.\n")
	b.WriteString("ca /etc/openvpn/pki/ca.crt\n")
	b.WriteString("cert /etc/openvpn/pki/server.crt\n")
	b.WriteString("key /etc/openvpn/pki/server.key\n")
	b.WriteString("dh none\n")
	return b.String()
}

type composeInput struct {
	InstanceName string
	Port         int
	Protocol     string
}

func buildDockerCompose(input composeInput) string {
	containerName := "vpn-manager-openvpn-" + input.InstanceName
	return fmt.Sprintf(`services:
  openvpn:
    image: kylemanna/openvpn:latest
    container_name: %s
    restart: unless-stopped
    cap_add:
      - NET_ADMIN
    devices:
      - /dev/net/tun:/dev/net/tun
    ports:
      - "%d:%d/%s"
    volumes:
      - ./server.conf:/etc/openvpn/server.conf:ro
      - ./pki:/etc/openvpn/pki:ro
    command: ["openvpn", "--config", "/etc/openvpn/server.conf"]
`, containerName, input.Port, input.Port, input.Protocol)
}

func openVPNServerNetwork(cidr string) (string, string, error) {
	ip, ipNet, err := net.ParseCIDR(strings.TrimSpace(cidr))
	if err != nil || ip.To4() == nil {
		return "", "", fmt.Errorf("tunnel cidr must be a valid IPv4 CIDR")
	}
	ones, bits := ipNet.Mask.Size()
	if bits != 32 || ones <= 0 || ones >= 32 {
		return "", "", fmt.Errorf("tunnel cidr must be an IPv4 network with usable host addresses")
	}
	mask := net.IP(ipNet.Mask).String()
	return ipNet.IP.String(), mask, nil
}

var unsafeInstanceNameChars = regexp.MustCompile(`[^a-z0-9-]+`)

func sanitizeInstanceName(name string) string {
	clean := strings.ToLower(strings.TrimSpace(name))
	clean = strings.ReplaceAll(clean, "_", "-")
	clean = unsafeInstanceNameChars.ReplaceAllString(clean, "-")
	clean = strings.Trim(clean, "-")
	return clean
}

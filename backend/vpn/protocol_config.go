package vpn

import (
	"fmt"
	"net"
	"strings"

	"github.com/example/wg-panel/models"
)

type ProtocolConfigInput struct {
	Protocol   models.VPNProtocol `json:"protocol"`
	Name       string             `json:"name"`
	RemoteHost string             `json:"remote_host"`
	ListenPort int                `json:"listen_port"`
	PoolCIDR   string             `json:"pool_cidr"`
	DNS        string             `json:"dns"`
}

type ProtocolConfigPreview struct {
	Protocol       models.VPNProtocol `json:"protocol"`
	Label          string             `json:"label"`
	RuntimeMode    string             `json:"runtime_mode"`
	Files          map[string]string  `json:"files"`
	SecretRefs     map[string]string  `json:"secret_refs"`
	Warnings       []string           `json:"warnings"`
	LegacyInsecure bool               `json:"legacy_insecure"`
}

func BuildProtocolConfigPreview(input ProtocolConfigInput) (ProtocolConfigPreview, error) {
	spec, ok := GetProtocolSpec(input.Protocol)
	if !ok {
		return ProtocolConfigPreview{}, fmt.Errorf("unsupported vpn protocol: %s", input.Protocol)
	}
	if input.Protocol == models.ProtocolWireGuard || input.Protocol == models.ProtocolOpenVPN {
		return ProtocolConfigPreview{}, fmt.Errorf("generic config preview is only for L2TP/IPsec, SSTP, and PPTP")
	}
	name := sanitizeConfigName(input.Name)
	if name == "" {
		return ProtocolConfigPreview{}, fmt.Errorf("name is required")
	}
	pool := strings.TrimSpace(input.PoolCIDR)
	if _, _, err := net.ParseCIDR(pool); err != nil {
		return ProtocolConfigPreview{}, fmt.Errorf("pool_cidr must be a valid CIDR")
	}
	dns := strings.TrimSpace(input.DNS)
	if dns == "" {
		dns = "1.1.1.1"
	}
	port := input.ListenPort
	if port == 0 {
		port = defaultProtocolPort(input.Protocol)
	}
	preview := ProtocolConfigPreview{
		Protocol:       input.Protocol,
		Label:          spec.Label,
		RuntimeMode:    spec.Capabilities.RuntimeStrategy,
		Files:          map[string]string{},
		SecretRefs:     map[string]string{"credentials": "[ENCRYPTED_SECRET_REF]"},
		LegacyInsecure: spec.LegacyInsecure,
		Warnings: []string{
			"Preview only; do not write these files until host verification and explicit production gates pass.",
			"Credential values are represented as [ENCRYPTED_SECRET_REF] and must be resolved through the encrypted secret store at render time.",
		},
	}
	switch input.Protocol {
	case models.ProtocolL2TPIPsec:
		preview.SecretRefs["ipsec_psk"] = "[ENCRYPTED_SECRET_REF]"
		preview.Files["ipsec.conf"] = fmt.Sprintf("config setup\n  uniqueids=no\n\nconn %s\n  auto=add\n  type=transport\n  keyexchange=ikev1\n  authby=secret\n  left=%%defaultroute\n  leftprotoport=17/1701\n  right=%%any\n  rightprotoport=17/0\n  ike=aes256-sha1-modp2048\n  esp=aes256-sha1\n", name)
		preview.Files["ipsec.secrets"] = ": PSK [ENCRYPTED_SECRET_REF]\n"
		preview.Files["xl2tpd.conf"] = fmt.Sprintf("[global]\nport = %d\n\n[lns default]\nip range = %s\nlocal ip = %s\nrequire chap = yes\nrefuse pap = yes\nrequire authentication = yes\nname = %s\npppoptfile = /etc/ppp/options.xl2tpd\nlength bit = yes\n", port, pool, firstUsableIP(pool), name)
		preview.Files["options.xl2tpd"] = fmt.Sprintf("require-mschap-v2\nms-dns %s\nasyncmap 0\nauth\ncrtscts\nlock\nhide-password\nmodem\nmtu 1280\nmru 1280\n", dns)
		preview.Files["chap-secrets"] = "# client server secret ip\n[ENCRYPTED_SECRET_REF] * [ENCRYPTED_SECRET_REF] *\n"
	case models.ProtocolSSTP:
		preview.SecretRefs["tls_cert"] = "[ENCRYPTED_SECRET_REF]"
		preview.SecretRefs["tls_key"] = "[ENCRYPTED_SECRET_REF]"
		preview.Files["sstpd.conf"] = fmt.Sprintf("listen = 0.0.0.0\nport = %d\ncert = /var/lib/wg-panel/sstp/%s/tls.crt\nkey = /var/lib/wg-panel/sstp/%s/tls.key\nlocal = %s\nremote = %s\nppp = /usr/sbin/pppd\n", port, name, name, firstUsableIP(pool), pool)
		preview.Files["options.sstpd"] = fmt.Sprintf("require-mschap-v2\nms-dns %s\nproxyarp\nlock\nmtu 1400\nmru 1400\n", dns)
		preview.Files["chap-secrets"] = "# client server secret ip\n[ENCRYPTED_SECRET_REF] sstpd [ENCRYPTED_SECRET_REF] *\n"
	case models.ProtocolPPTP:
		preview.Files["pptpd.conf"] = fmt.Sprintf("option /etc/ppp/options.pptpd\nlogwtmp\nlocalip %s\nremoteip %s\n", firstUsableIP(pool), pool)
		preview.Files["options.pptpd"] = fmt.Sprintf("name pptpd\nrequire-mschap-v2\nrequire-mppe-128\nms-dns %s\nproxyarp\nlock\nnobsdcomp\nnovj\nnovjccomp\nnologfd\n", dns)
		preview.Files["chap-secrets"] = "# client server secret ip\n[ENCRYPTED_SECRET_REF] pptpd [ENCRYPTED_SECRET_REF] *\n"
		preview.Warnings = append(preview.Warnings, "PPTP is insecure and should be enabled only for legacy clients after explicit administrator confirmation.")
	default:
		return ProtocolConfigPreview{}, fmt.Errorf("unsupported vpn protocol: %s", input.Protocol)
	}
	return preview, nil
}

func defaultProtocolPort(protocol models.VPNProtocol) int {
	switch protocol {
	case models.ProtocolL2TPIPsec:
		return 1701
	case models.ProtocolSSTP:
		return 443
	case models.ProtocolPPTP:
		return 1723
	default:
		return 0
	}
}

func sanitizeConfigName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		} else if r == ' ' || r == '.' {
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-_")
}

func replaceRuntimePlaceholders(commands []string, input ProtocolConfigInput, instanceID uint) []string {
	name := sanitizeConfigName(input.Name)
	replacements := map[string]string{
		"{instance_id}": fmt.Sprintf("%d", instanceID),
		"{name}":        name,
		"{listen_port}": fmt.Sprintf("%d", input.ListenPort),
		"{pool_cidr}":   strings.TrimSpace(input.PoolCIDR),
		"{tunnel_cidr}": strings.TrimSpace(input.PoolCIDR),
	}
	out := make([]string, 0, len(commands))
	for _, command := range commands {
		for old, next := range replacements {
			command = strings.ReplaceAll(command, old, next)
		}
		out = append(out, command)
	}
	return out
}

func BuildLegacyRuntimeApplyPlan(input ProtocolConfigInput, instanceID uint, gates ProductionGates, executorEnabled bool) (map[string]string, []string, error) {
	preview, err := BuildProtocolConfigPreview(input)
	if err != nil {
		return nil, nil, err
	}
	prod, err := BuildProductionPlan(input.Protocol, gates, executorEnabled)
	if err != nil {
		return nil, nil, err
	}
	if !prod.Ready || !executorEnabled {
		return nil, nil, fmt.Errorf("production gates and VPN_COMMAND_EXECUTOR_ENABLED must be enabled before applying %s", input.Protocol)
	}
	files := map[string]string{}
	switch input.Protocol {
	case models.ProtocolL2TPIPsec:
		files["/etc/ipsec.conf"] = preview.Files["ipsec.conf"]
		files["/etc/ipsec.secrets"] = preview.Files["ipsec.secrets"]
		files["/etc/xl2tpd/xl2tpd.conf"] = preview.Files["xl2tpd.conf"]
		files["/etc/ppp/options.xl2tpd"] = preview.Files["options.xl2tpd"]
		files["/etc/ppp/chap-secrets"] = preview.Files["chap-secrets"]
	case models.ProtocolSSTP:
		files["/etc/sstpd/sstpd.conf"] = preview.Files["sstpd.conf"]
		files["/etc/ppp/options.sstpd"] = preview.Files["options.sstpd"]
		files["/etc/ppp/chap-secrets"] = preview.Files["chap-secrets"]
	case models.ProtocolPPTP:
		files["/etc/pptpd.conf"] = preview.Files["pptpd.conf"]
		files["/etc/ppp/options.pptpd"] = preview.Files["options.pptpd"]
		files["/etc/ppp/chap-secrets"] = preview.Files["chap-secrets"]
	default:
		return nil, nil, fmt.Errorf("unsupported legacy runtime protocol: %s", input.Protocol)
	}
	commands := replaceRuntimePlaceholders(append(prod.FirewallCommands, prod.RuntimeCommands...), input, instanceID)
	return files, commands, nil
}

func firstUsableIP(cidr string) string {
	ip, ipNet, err := net.ParseCIDR(strings.TrimSpace(cidr))
	if err != nil || ipNet == nil {
		return ""
	}
	ip = ip.To4()
	if ip == nil {
		return ipNet.IP.String()
	}
	first := append(net.IP(nil), ip...)
	first[3]++
	return first.String()
}

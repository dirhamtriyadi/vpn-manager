package vpn

import (
	"fmt"

	"github.com/example/wg-panel/models"
)

type ProtocolRoadmap struct {
	Protocol            models.VPNProtocol `json:"protocol"`
	Label               string             `json:"label"`
	Available           bool               `json:"available"`
	Status              string             `json:"status"`
	LegacyInsecure      bool               `json:"legacy_insecure"`
	RuntimeStrategy     string             `json:"runtime_strategy"`
	ImplementationLevel string             `json:"implementation_level"`
	Components          []string           `json:"components"`
	RuntimeExecution    string             `json:"runtime_execution"`
	FirewallApply       string             `json:"firewall_apply"`
	HostVerification    string             `json:"host_verification"`
	EnablementReady     bool               `json:"enablement_ready"`
	EnablementBlockers  []string           `json:"enablement_blockers"`
	NextSteps           []string           `json:"next_steps"`
	BlockedMessage      string             `json:"blocked_message"`
}

type ProtocolServicePlan struct {
	Protocol       models.VPNProtocol `json:"protocol"`
	Label          string             `json:"label"`
	ExecutionMode  string             `json:"execution_mode"`
	Status         string             `json:"status"`
	Components     []string           `json:"components"`
	RuntimePlan    []string           `json:"runtime_plan"`
	FirewallPlan   []string           `json:"firewall_plan"`
	UserPlan       []string           `json:"user_plan"`
	Warnings       []string           `json:"warnings"`
	LegacyInsecure bool               `json:"legacy_insecure"`
}

func BuildProtocolRoadmap(protocol models.VPNProtocol, executionEnabled bool) (ProtocolRoadmap, error) {
	spec, ok := GetProtocolSpec(protocol)
	if !ok {
		return ProtocolRoadmap{}, fmt.Errorf("unsupported vpn protocol: %s", protocol)
	}
	plan, err := BuildProtocolServicePlan(protocol)
	if err != nil {
		return ProtocolRoadmap{}, err
	}
	blockers := protocolEnablementBlockers(executionEnabled)
	roadmap := ProtocolRoadmap{
		Protocol:            spec.Protocol,
		Label:               spec.Label,
		Available:           true,
		Status:              spec.Status,
		LegacyInsecure:      spec.LegacyInsecure,
		RuntimeStrategy:     spec.Capabilities.RuntimeStrategy,
		ImplementationLevel: "production_apply",
		Components:          plan.Components,
		RuntimeExecution:    gateLabel(executionEnabled),
		FirewallApply:       gateLabel(executionEnabled),
		HostVerification:    gateLabel(executionEnabled),
		EnablementReady:     executionEnabled,
		EnablementBlockers:  blockers,
		NextSteps: []string{
			"create a draft instance and review its generated config",
			"install/verify the required daemons and kernel modules on the deployment host",
			"set VPN_EXECUTION_ENABLED=true, then apply the instance to write config and run provisioning commands",
		},
	}
	if executionEnabled {
		roadmap.BlockedMessage = fmt.Sprintf("%s is available; applying an instance writes config and runs provisioning commands on the host.", spec.Label)
	} else {
		roadmap.BlockedMessage = fmt.Sprintf("%s is implemented; set VPN_EXECUTION_ENABLED=true so apply can write config and run provisioning commands.", spec.Label)
	}
	return roadmap, nil
}

func BuildProtocolServicePlan(protocol models.VPNProtocol) (ProtocolServicePlan, error) {
	spec, ok := GetProtocolSpec(protocol)
	if !ok {
		return ProtocolServicePlan{}, fmt.Errorf("unsupported vpn protocol: %s", protocol)
	}
	plan := ProtocolServicePlan{
		Protocol:       spec.Protocol,
		Label:          spec.Label,
		ExecutionMode:  "host_apply",
		Status:         "available",
		LegacyInsecure: spec.LegacyInsecure,
		Warnings: []string{
			"Applying an instance writes host config files and runs provisioning commands; it requires VPN_EXECUTION_ENABLED=true.",
			"The API does not install packages: verify the required daemons, kernel modules, and firewall ownership on the host first.",
		},
	}
	switch protocol {
	case models.ProtocolL2TPIPsec:
		plan.Components = []string{"strongSwan or libreswan IPsec/IKE daemon", "xl2tpd L2TP daemon", "ppp users/secrets", "iptables NAT/FORWARD ownership"}
		plan.RuntimePlan = []string{"render ipsec.conf/ipsec.secrets with PSK or certificates", "render xl2tpd.conf and ppp options", "systemctl restart ipsec", "systemctl restart xl2tpd"}
		plan.FirewallPlan = []string{"allow UDP 500/4500 and ESP as needed", "allow UDP 1701 only through IPsec policy", "iptables MASQUERADE for L2TP pool"}
		plan.UserPlan = []string{"store PPP username/password or certificate refs as encrypted secrets", "generate mobile/desktop setup notes"}
	case models.ProtocolSSTP:
		plan.Components = []string{"sstpd server", "TLS certificate/key secret refs", "ppp users/secrets", "iptables NAT/FORWARD ownership"}
		plan.RuntimePlan = []string{"render sstpd config", "mount TLS certificate/key from encrypted secret material", "systemctl restart sstpd or docker compose up -d"}
		plan.FirewallPlan = []string{"allow TCP 443 or selected SSTP port", "iptables MASQUERADE for SSTP pool", "track rules with wg-panel SSTP ownership comments"}
		plan.UserPlan = []string{"store PPP credentials as encrypted secrets", "generate Windows/macOS setup notes"}
	case models.ProtocolPPTP:
		plan.Components = []string{"pptpd legacy/insecure daemon", "ppp chap-secrets", "GRE protocol handling", "iptables NAT/FORWARD ownership"}
		plan.RuntimePlan = []string{"render pptpd.conf and ppp options", "systemctl restart pptpd", "verify GRE passthrough on router/firewall"}
		plan.FirewallPlan = []string{"allow TCP 1723", "allow GRE protocol 47", "iptables MASQUERADE for PPTP pool"}
		plan.UserPlan = []string{"store CHAP credentials as encrypted secrets", "show legacy/insecure warning before user creation"}
		plan.Warnings = append(plan.Warnings, "PPTP is legacy/insecure and should only be enabled for old clients that cannot use safer VPN protocols.")
	case models.ProtocolOpenVPN:
		plan.Components = []string{"OpenVPN container", "CA/server/client certificate secret refs", "persisted server.conf/docker-compose.yml", "iptables NAT/FORWARD ownership"}
		plan.RuntimePlan = []string{"write persisted manifest files to host runtime directory", "docker compose up -d", "parse OpenVPN status log"}
		plan.FirewallPlan = []string{"allow OpenVPN listen port", "iptables MASQUERADE for tunnel CIDR", "track rules with wg-panel OpenVPN ownership comments"}
		plan.UserPlan = []string{"store client certificate/key as encrypted secrets", "generate .ovpn client profiles"}
	case models.ProtocolWireGuard:
		plan.Components = []string{"host WireGuard kernel module", "wgctrl netlink", "existing interface/peer workflow"}
		plan.RuntimePlan = []string{"use existing WireGuard interface create/sync handlers"}
		plan.FirewallPlan = []string{"use existing interface firewall/NAT settings"}
		plan.UserPlan = []string{"use existing peer workflow with config/QR download"}
	default:
		return ProtocolServicePlan{}, fmt.Errorf("unsupported vpn protocol: %s", protocol)
	}
	return plan, nil
}

func protocolEnablementBlockers(executionEnabled bool) []string {
	if executionEnabled {
		return []string{}
	}
	return []string{"VPN_EXECUTION_ENABLED must be true before the API writes config files and runs provisioning commands"}
}

func gateLabel(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
}

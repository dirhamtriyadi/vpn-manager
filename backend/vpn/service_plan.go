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

func BuildProtocolRoadmap(protocol models.VPNProtocol, runtimeExecution, firewallApply, hostVerification bool) (ProtocolRoadmap, error) {
	spec, ok := GetProtocolSpec(protocol)
	if !ok {
		return ProtocolRoadmap{}, fmt.Errorf("unsupported vpn protocol: %s", protocol)
	}
	plan, err := BuildProtocolServicePlan(protocol)
	if err != nil {
		return ProtocolRoadmap{}, err
	}
	blockers := protocolEnablementBlockers(runtimeExecution, firewallApply, hostVerification)
	return ProtocolRoadmap{
		Protocol:            spec.Protocol,
		Label:               spec.Label,
		Available:           false,
		Status:              spec.Status,
		LegacyInsecure:      spec.LegacyInsecure,
		RuntimeStrategy:     spec.Capabilities.RuntimeStrategy,
		ImplementationLevel: "service_plan_scaffold",
		Components:          plan.Components,
		RuntimeExecution:    gateLabel(runtimeExecution),
		FirewallApply:       gateLabel(firewallApply),
		HostVerification:    gateLabel(hostVerification),
		EnablementReady:     len(blockers) == 0,
		EnablementBlockers:  blockers,
		NextSteps: []string{
			"review dry-run service plan on the deployment host",
			"install/verify required daemons and kernel modules outside the app container",
			"register a real protocol driver only after host verification passes",
		},
		BlockedMessage: fmt.Sprintf("%s has a complete dry-run service plan but remains unavailable until a real host/runtime driver is implemented and verified.", spec.Label),
	}, nil
}

func BuildProtocolServicePlan(protocol models.VPNProtocol) (ProtocolServicePlan, error) {
	spec, ok := GetProtocolSpec(protocol)
	if !ok {
		return ProtocolServicePlan{}, fmt.Errorf("unsupported vpn protocol: %s", protocol)
	}
	plan := ProtocolServicePlan{
		Protocol:       spec.Protocol,
		Label:          spec.Label,
		ExecutionMode:  "dry_run",
		Status:         "planned",
		LegacyInsecure: spec.LegacyInsecure,
		Warnings: []string{
			"Dry-run plan only; the API does not install packages, start services, or apply firewall rules.",
			"Execute only after host-side verification and explicit feature gates are enabled.",
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
		plan.Components = []string{"pptpd legacy daemon", "ppp chap-secrets", "GRE protocol handling", "iptables NAT/FORWARD ownership"}
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

func protocolEnablementBlockers(runtimeExecution, firewallApply, hostVerification bool) []string {
	blockers := []string{}
	if !runtimeExecution {
		blockers = append(blockers, "VPN_RUNTIME_EXECUTION_ENABLED must be true before service/container commands can run")
	}
	if !firewallApply {
		blockers = append(blockers, "VPN_FIREWALL_APPLY_ENABLED must be true before firewall/NAT rules can be applied")
	}
	if !hostVerification {
		blockers = append(blockers, "VPN_HOST_VERIFICATION_PASSED must be true after host-side tests/build and dry-run plan review")
	}
	return blockers
}

func gateLabel(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
}

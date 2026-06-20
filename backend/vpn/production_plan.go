package vpn

import (
	"fmt"

	"github.com/example/vpn-manager/models"
)

type ProductionPlan struct {
	Protocol         models.VPNProtocol `json:"protocol"`
	Label            string             `json:"label"`
	Ready            bool               `json:"ready"`
	ExecutionMode    string             `json:"execution_mode"`
	RuntimeCommands  []string           `json:"runtime_commands"`
	FirewallCommands []string           `json:"firewall_commands"`
	StatusCommands   []string           `json:"status_commands"`
	ConfigFiles      []string           `json:"config_files"`
	Blockers         []string           `json:"blockers"`
	Warnings         []string           `json:"warnings"`
	LegacyInsecure   bool               `json:"legacy_insecure"`
}

func BuildProductionPlan(protocol models.VPNProtocol, executionEnabled bool) (ProductionPlan, error) {
	spec, ok := GetProtocolSpec(protocol)
	if !ok {
		return ProductionPlan{}, fmt.Errorf("unsupported vpn protocol: %s", protocol)
	}
	blockers := protocolEnablementBlockers(executionEnabled)
	plan := ProductionPlan{
		Protocol:       spec.Protocol,
		Label:          spec.Label,
		Ready:          executionEnabled,
		ExecutionMode:  "requires_execution_enabled",
		Blockers:       blockers,
		LegacyInsecure: spec.LegacyInsecure,
		Warnings: []string{
			"Review generated config paths, ports, subnets, and firewall ownership on the deployment host before enabling execution.",
			"The app should run with the least host privileges needed for the selected protocol.",
		},
	}
	if executionEnabled {
		plan.ExecutionMode = "execution_enabled"
	}

	switch protocol {
	case models.ProtocolOpenVPN:
		plan.ConfigFiles = []string{"/var/lib/vpn-manager/openvpn/{instance_id}/server.conf", "/var/lib/vpn-manager/openvpn/{instance_id}/docker-compose.yml", "/var/lib/vpn-manager/openvpn/{instance_id}/pki/*"}
		plan.RuntimeCommands = []string{"install -d -m 0700 /var/lib/vpn-manager/openvpn/{instance_id}", "docker compose -p vpn-manager-openvpn-{name} -f /var/lib/vpn-manager/openvpn/{instance_id}/docker-compose.yml up -d"}
		plan.FirewallCommands = []string{"iptables -A INPUT -p udp --dport {listen_port} -m comment --comment 'vpn-manager openvpn {name}' -j ACCEPT", "iptables -A FORWARD -s {tunnel_cidr} -m comment --comment 'vpn-manager openvpn {name}' -j ACCEPT", "iptables -t nat -A POSTROUTING -s {tunnel_cidr} -m comment --comment 'vpn-manager openvpn {name}' -j MASQUERADE"}
		plan.StatusCommands = []string{"docker compose -p vpn-manager-openvpn-{name} -f /var/lib/vpn-manager/openvpn/{instance_id}/docker-compose.yml ps", "docker compose -p vpn-manager-openvpn-{name} -f /var/lib/vpn-manager/openvpn/{instance_id}/docker-compose.yml logs --tail=100 openvpn"}
	case models.ProtocolL2TPIPsec:
		plan.ConfigFiles = []string{"/etc/ipsec.conf", "/etc/ipsec.secrets", "/etc/xl2tpd/xl2tpd.conf", "/etc/ppp/options.xl2tpd", "/etc/ppp/chap-secrets"}
		plan.RuntimeCommands = []string{"systemctl enable --now ipsec || systemctl enable --now strongswan", "systemctl enable --now xl2tpd", "systemctl restart ipsec || systemctl restart strongswan", "systemctl restart xl2tpd"}
		plan.FirewallCommands = []string{"iptables -A INPUT -p udp -m multiport --dports 500,4500 -m comment --comment 'vpn-manager l2tp-ipsec {name}' -j ACCEPT", "iptables -A INPUT -p udp --dport 1701 -m policy --dir in --pol ipsec -m comment --comment 'vpn-manager l2tp-ipsec {name}' -j ACCEPT", "iptables -t nat -A POSTROUTING -s {pool_cidr} -m comment --comment 'vpn-manager l2tp-ipsec {name}' -j MASQUERADE"}
		plan.StatusCommands = []string{"ipsec statusall || strongswan statusall", "systemctl status xl2tpd --no-pager"}
	case models.ProtocolSSTP:
		plan.ConfigFiles = []string{"/etc/sstpd/sstpd.conf", "/etc/ppp/options.sstpd", "/etc/ppp/chap-secrets", "/var/lib/vpn-manager/sstp/{instance_id}/tls.crt", "/var/lib/vpn-manager/sstp/{instance_id}/tls.key"}
		plan.RuntimeCommands = []string{"install -d -m 0700 /var/lib/vpn-manager/sstp/{instance_id}", "systemctl enable --now sstpd", "systemctl restart sstpd"}
		plan.FirewallCommands = []string{"iptables -A INPUT -p tcp --dport {listen_port} -m comment --comment 'vpn-manager sstp {name}' -j ACCEPT", "iptables -A FORWARD -s {pool_cidr} -m comment --comment 'vpn-manager sstp {name}' -j ACCEPT", "iptables -t nat -A POSTROUTING -s {pool_cidr} -m comment --comment 'vpn-manager sstp {name}' -j MASQUERADE"}
		plan.StatusCommands = []string{"systemctl status sstpd --no-pager", "journalctl -u sstpd -n 100 --no-pager"}
	case models.ProtocolPPTP:
		plan.ConfigFiles = []string{"/etc/pptpd.conf", "/etc/ppp/options.pptpd", "/etc/ppp/chap-secrets"}
		plan.RuntimeCommands = []string{"systemctl enable --now pptpd", "systemctl restart pptpd", "modprobe nf_conntrack_pptp || true"}
		plan.FirewallCommands = []string{"iptables -A INPUT -p tcp --dport 1723 -m comment --comment 'vpn-manager pptp {name}' -j ACCEPT", "iptables -A INPUT -p 47 -m comment --comment 'vpn-manager pptp {name} GRE' -j ACCEPT", "iptables -t nat -A POSTROUTING -s {pool_cidr} -m comment --comment 'vpn-manager pptp {name}' -j MASQUERADE"}
		plan.StatusCommands = []string{"systemctl status pptpd --no-pager", "journalctl -u pptpd -n 100 --no-pager"}
		plan.Warnings = append(plan.Warnings, "PPTP is legacy/insecure; keep it disabled unless old clients have no safer alternative.")
	case models.ProtocolWireGuard:
		plan.Ready = true
		plan.ExecutionMode = "existing_driver"
		plan.ConfigFiles = []string{"existing database-backed WireGuard interface/peer configuration"}
		plan.RuntimeCommands = []string{"use existing WireGuard create/sync endpoints"}
		plan.FirewallCommands = []string{"use existing interface masquerade/firewall behavior"}
		plan.StatusCommands = []string{"use wgctrl status endpoint"}
	default:
		return ProductionPlan{}, fmt.Errorf("unsupported vpn protocol: %s", protocol)
	}
	return plan, nil
}

package openvpn

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/example/vpn-manager/models"
	"github.com/example/vpn-manager/secrets"
)

type LifecyclePlan struct {
	Action        string   `json:"action"`
	ExecutionMode string   `json:"execution_mode"`
	Status        string   `json:"status"`
	ProjectName   string   `json:"project_name"`
	ContainerName string   `json:"container_name"`
	Commands      []string `json:"commands"`
	Warnings      []string `json:"warnings"`
}

type FirewallPlan struct {
	Status        string   `json:"status"`
	OwnershipKey  string   `json:"ownership_key"`
	Rules         []string `json:"rules"`
	TeardownRules []string `json:"teardown_rules"`
	Warnings      []string `json:"warnings"`
}

type StatusSnapshot struct {
	State   string         `json:"state"`
	Clients []StatusClient `json:"clients"`
	Raw     string         `json:"raw,omitempty"`
}

type StatusClient struct {
	CommonName     string `json:"common_name"`
	RealAddress    string `json:"real_address"`
	VirtualAddress string `json:"virtual_address"`
	BytesReceived  int64  `json:"bytes_received"`
	BytesSent      int64  `json:"bytes_sent"`
	ConnectedSince string `json:"connected_since"`
}

type UserDraftInput struct {
	InstanceID     uint
	Name           string
	AssignedIP     string
	ClientCertPEM  string
	ClientKeyPEM   string
}

type UserDraft struct {
	User    models.OpenVPNUser
	Secrets []models.EncryptedSecret
}

type EnablementGates struct {
	Ready                   bool     `json:"ready"`
	RuntimeExecutionEnabled bool     `json:"runtime_execution_enabled"`
	FirewallApplyEnabled    bool     `json:"firewall_apply_enabled"`
	HostVerificationPassed  bool     `json:"host_verification_passed"`
	Blockers                []string `json:"blockers"`
}

// BuildEnablementGates reports whether OpenVPN apply will execute, driven by the
// single VPN_EXECUTION_ENABLED toggle shared by every protocol.
func BuildEnablementGates(executionEnabled bool) EnablementGates {
	blockers := []string{}
	if !executionEnabled {
		blockers = append(blockers, "VPN_EXECUTION_ENABLED must be true before the API writes config and runs container/firewall commands")
	}
	return EnablementGates{Ready: executionEnabled, RuntimeExecutionEnabled: executionEnabled, FirewallApplyEnabled: executionEnabled, HostVerificationPassed: executionEnabled, Blockers: blockers}
}

func BuildLifecyclePlan(instance models.OpenVPNInstance, action string) (LifecyclePlan, error) {
	if instance.ID == 0 {
		return LifecyclePlan{}, fmt.Errorf("saved OpenVPN instance is required")
	}
	action = strings.ToLower(strings.TrimSpace(action))
	if action == "" {
		action = "status"
	}
	allowed := map[string]bool{"start": true, "stop": true, "restart": true, "reload": true, "status": true}
	if !allowed[action] {
		return LifecyclePlan{}, fmt.Errorf("unsupported OpenVPN lifecycle action")
	}
	name := sanitizeInstanceName(instance.Name)
	if name == "" {
		return LifecyclePlan{}, fmt.Errorf("instance name is required")
	}
	project := "vpn-manager-openvpn-" + name
	base := fmt.Sprintf("docker compose -p %s -f /var/lib/vpn-manager/openvpn/%d/docker-compose.yml", project, instance.ID)
	commands := map[string][]string{
		"start":   {base + " up -d"},
		"stop":    {base + " down"},
		"restart": {base + " down", base + " up -d"},
		"reload":  {base + " exec openvpn sh -c 'kill -HUP 1'"},
		"status":  {base + " ps", base + " logs --tail=100 openvpn"},
	}
	return LifecyclePlan{
		Action:        action,
		ExecutionMode: "host_apply",
		Status:        "planned",
		ProjectName:   project,
		ContainerName: project,
		Commands:      commands[action],
		Warnings: []string{
			"These are the exact commands the apply endpoint runs; it executes them only when VPN_EXECUTION_ENABLED=true.",
			"Verify manifest files, secret material, firewall ownership, and Docker availability on the host before enabling execution.",
		},
	}, nil
}

func BuildFirewallPlan(instance models.OpenVPNInstance) (FirewallPlan, error) {
	if instance.ID == 0 {
		return FirewallPlan{}, fmt.Errorf("saved OpenVPN instance is required")
	}
	_, ipNet, err := net.ParseCIDR(strings.TrimSpace(instance.TunnelCIDR))
	if err != nil || ipNet == nil {
		return FirewallPlan{}, fmt.Errorf("tunnel cidr must be a valid CIDR")
	}
	name := sanitizeInstanceName(instance.Name)
	comment := "vpn-manager openvpn " + name
	cidr := ipNet.String()
	return FirewallPlan{
		Status:       "planned",
		OwnershipKey: comment,
		Rules: []string{
			fmt.Sprintf("iptables -A FORWARD -s %s -m comment --comment %q -j ACCEPT", cidr, comment),
			fmt.Sprintf("iptables -A FORWARD -d %s -m comment --comment %q -j ACCEPT", cidr, comment),
			fmt.Sprintf("iptables -t nat -A POSTROUTING -s %s -m comment --comment %q -j MASQUERADE", cidr, comment),
		},
		TeardownRules: []string{
			fmt.Sprintf("iptables-save | grep -v %q | iptables-restore", comment),
		},
		Warnings: []string{
			"These are the exact rules the apply endpoint installs; it applies them only when VPN_EXECUTION_ENABLED=true.",
			"Review WAN/LAN interface selection before enabling automatic firewall ownership.",
		},
	}, nil
}

func ParseStatusLog(raw string) StatusSnapshot {
	clients := []StatusClient{}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "CLIENT_LIST,") {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) < 7 {
			continue
		}
		received, _ := strconv.ParseInt(parts[4], 10, 64)
		sent, _ := strconv.ParseInt(parts[5], 10, 64)
		clients = append(clients, StatusClient{
			CommonName:     parts[1],
			RealAddress:    parts[2],
			VirtualAddress: parts[3],
			BytesReceived:  received,
			BytesSent:      sent,
			ConnectedSince: parts[6],
		})
	}
	state := "not_running"
	if len(clients) > 0 || strings.Contains(raw, "OpenVPN CLIENT LIST") {
		state = "running"
	}
	return StatusSnapshot{State: state, Clients: clients, Raw: raw}
}

func BuildUserDraft(input UserDraftInput, envelope *secrets.Envelope) (UserDraft, error) {
	if envelope == nil {
		return UserDraft{}, fmt.Errorf("secret envelope is required")
	}
	if input.InstanceID == 0 {
		return UserDraft{}, fmt.Errorf("saved OpenVPN instance is required")
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return UserDraft{}, fmt.Errorf("client name is required")
	}
	if strings.TrimSpace(input.ClientCertPEM) == "" {
		return UserDraft{}, fmt.Errorf("client certificate is required")
	}
	if strings.TrimSpace(input.ClientKeyPEM) == "" {
		return UserDraft{}, fmt.Errorf("client private key is required")
	}
	user := models.OpenVPNUser{InstanceID: input.InstanceID, Name: name, AssignedIP: strings.TrimSpace(input.AssignedIP), Enabled: true}
	scope := fmt.Sprintf("openvpn-client-%d-%s", input.InstanceID, name)
	inputs := []struct {
		Name      string
		Plaintext string
		AssignRef func(string)
	}{
		{Name: "client-cert-pem", Plaintext: input.ClientCertPEM, AssignRef: func(ref string) { user.CertRef = ref }},
		{Name: "client-key-pem", Plaintext: input.ClientKeyPEM, AssignRef: func(ref string) { user.KeyRef = ref }},
	}
	secretsOut := make([]models.EncryptedSecret, 0, len(inputs))
	for _, item := range inputs {
		sealed, err := envelope.Encrypt(strings.TrimSpace(item.Plaintext))
		if err != nil {
			return UserDraft{}, err
		}
		ref := secrets.BuildRef(scope, 0, item.Name)
		item.AssignRef(ref)
		secretsOut = append(secretsOut, models.EncryptedSecret{Ref: ref, Scope: "openvpn", OwnerType: "openvpn_user", Name: item.Name, Algorithm: sealed.Algorithm, Nonce: sealed.Nonce, Ciphertext: sealed.Ciphertext, KeyVersion: "v1"})
	}
	return UserDraft{User: user, Secrets: secretsOut}, nil
}

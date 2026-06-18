package openvpn

import (
	"strings"
	"testing"

	"github.com/example/wg-panel/models"
	"github.com/example/wg-panel/secrets"
)

func testInstance() models.OpenVPNInstance {
	return models.OpenVPNInstance{ID: 7, Name: "office vpn", RemoteHost: "vpn.example.com", ListenPort: 1194, Protocol: "udp", TunnelCIDR: "10.20.0.0/24", DNS: "1.1.1.1", RuntimeMode: "container_openvpn_preview"}
}

func TestBuildLifecyclePlanIsDryRunAndDeterministic(t *testing.T) {
	plan, err := BuildLifecyclePlan(testInstance(), "start")
	if err != nil {
		t.Fatalf("BuildLifecyclePlan returned error: %v", err)
	}
	if plan.Action != "start" || plan.ExecutionMode != "dry_run" || plan.Status != "planned" {
		t.Fatalf("unexpected lifecycle plan: %#v", plan)
	}
	joined := strings.Join(plan.Commands, "\n")
	if !strings.Contains(joined, "vpn-manager-openvpn-office-vpn") || !strings.Contains(joined, "docker compose") {
		t.Fatalf("commands should reference deterministic compose project/container: %s", joined)
	}
	if len(plan.Warnings) == 0 || !strings.Contains(plan.Warnings[0], "not executed") {
		t.Fatalf("expected dry-run warning, got %#v", plan.Warnings)
	}
}

func TestBuildFirewallPlanIncludesApplyAndTeardownRules(t *testing.T) {
	plan, err := BuildFirewallPlan(testInstance())
	if err != nil {
		t.Fatalf("BuildFirewallPlan returned error: %v", err)
	}
	if plan.Status != "planned" || len(plan.Rules) == 0 {
		t.Fatalf("unexpected firewall plan: %#v", plan)
	}
	joined := strings.Join(plan.Rules, "\n")
	teardown := strings.Join(plan.TeardownRules, "\n")
	if !strings.Contains(joined, "10.20.0.0/24") || !strings.Contains(joined, "wg-panel openvpn office-vpn") {
		t.Fatalf("rules missing CIDR/comment: %s", joined)
	}
	if !strings.Contains(teardown, "wg-panel openvpn office-vpn") {
		t.Fatalf("teardown missing ownership comment: %s", teardown)
	}
}

func TestParseStatusLogParsesClientList(t *testing.T) {
	raw := "OpenVPN CLIENT LIST\nCLIENT_LIST,alice,203.0.113.5:51111,10.20.0.2,1234,5678,2026-06-17 10:00:00\nEND\n"
	status := ParseStatusLog(raw)
	if status.State != "running" || len(status.Clients) != 1 {
		t.Fatalf("unexpected status: %#v", status)
	}
	client := status.Clients[0]
	if client.CommonName != "alice" || client.VirtualAddress != "10.20.0.2" || client.BytesReceived != 1234 || client.BytesSent != 5678 {
		t.Fatalf("unexpected client: %#v", client)
	}
}

func TestBuildEnablementGatesRequireExplicitFlags(t *testing.T) {
	gates := BuildEnablementGates(false, false, false)
	if gates.Ready {
		t.Fatal("OpenVPN should not be ready without explicit runtime, firewall, and verification gates")
	}
	if len(gates.Blockers) != 3 {
		t.Fatalf("expected 3 blockers, got %#v", gates.Blockers)
	}
	gates = BuildEnablementGates(true, true, true)
	if !gates.Ready || len(gates.Blockers) != 0 {
		t.Fatalf("expected ready gates, got %#v", gates)
	}
}

func TestBuildUserDraftStoresClientKeyAsEncryptedSecret(t *testing.T) {
	envelope, err := secrets.NewEnvelope("dev-master-key")
	if err != nil {
		t.Fatalf("NewEnvelope returned error: %v", err)
	}
	draft, err := BuildUserDraft(UserDraftInput{
		InstanceID:     7,
		Name:           "alice",
		AssignedIP:     "10.20.0.2",
		ClientCertPEM:  "-----BEGIN CERTIFICATE-----\nclient\n-----END CERTIFICATE-----",
		ClientKeyPEM:   "-----BEGIN PRIVATE KEY-----\nsecret\n-----END PRIVATE KEY-----",
	}, envelope)
	if err != nil {
		t.Fatalf("BuildUserDraft returned error: %v", err)
	}
	if draft.User.KeyRef == "" || draft.User.CertRef == "" {
		t.Fatalf("expected secret refs on user: %#v", draft.User)
	}
	for _, secret := range draft.Secrets {
		if strings.Contains(secret.Ciphertext, "secret") || strings.Contains(secret.Ciphertext, "client") {
			t.Fatalf("ciphertext leaked plaintext: %#v", secret)
		}
	}
}

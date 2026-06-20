package vpn

import (
	"strings"
	"testing"

	"github.com/example/vpn-manager/models"
)

func TestBuildProductionPlanRequiresExecutionEnabled(t *testing.T) {
	plan, err := BuildProductionPlan(models.ProtocolOpenVPN, false)
	if err != nil {
		t.Fatalf("BuildProductionPlan returned error: %v", err)
	}
	if plan.Ready {
		t.Fatal("production plan must not be ready when execution is disabled")
	}
	if plan.ExecutionMode != "requires_execution_enabled" {
		t.Fatalf("expected requires_execution_enabled mode, got %s", plan.ExecutionMode)
	}
	if len(plan.Blockers) == 0 || !strings.Contains(strings.Join(plan.Blockers, "\n"), "VPN_EXECUTION_ENABLED") {
		t.Fatalf("expected VPN_EXECUTION_ENABLED blocker, got %#v", plan.Blockers)
	}
}

func TestBuildProductionPlanProvidesProtocolCommandsWhenEnabled(t *testing.T) {
	tests := []struct {
		protocol models.VPNProtocol
		wants    []string
	}{
		{models.ProtocolOpenVPN, []string{"docker compose", "openvpn", "iptables"}},
		{models.ProtocolL2TPIPsec, []string{"ipsec", "xl2tpd", "iptables"}},
		{models.ProtocolSSTP, []string{"sstpd", "iptables"}},
		{models.ProtocolPPTP, []string{"pptpd", "GRE", "iptables"}},
	}
	for _, tt := range tests {
		plan, err := BuildProductionPlan(tt.protocol, true)
		if err != nil {
			t.Fatalf("BuildProductionPlan(%s) returned error: %v", tt.protocol, err)
		}
		if !plan.Ready {
			t.Fatalf("expected %s production plan to be ready when execution is enabled: %#v", tt.protocol, plan.Blockers)
		}
		if plan.ExecutionMode != "execution_enabled" {
			t.Fatalf("expected execution_enabled mode, got %s", plan.ExecutionMode)
		}
		blob := strings.Join(append(append(plan.RuntimeCommands, plan.FirewallCommands...), plan.StatusCommands...), "\n")
		for _, want := range tt.wants {
			if !strings.Contains(blob, want) {
				t.Fatalf("expected %s commands to include %q, got %s", tt.protocol, want, blob)
			}
		}
	}
}

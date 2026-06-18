package vpn

import (
	"strings"
	"testing"

	"github.com/example/wg-panel/models"
)

func TestBuildProductionPlanRefusesExecutionWithoutAllGates(t *testing.T) {
	plan, err := BuildProductionPlan(models.ProtocolOpenVPN, ProductionGates{RuntimeExecution: true, FirewallApply: false, HostVerification: true}, false)
	if err != nil {
		t.Fatalf("BuildProductionPlan returned error: %v", err)
	}
	if plan.Ready {
		t.Fatal("production plan must not be ready when firewall gate is disabled")
	}
	if plan.ExecutionMode != "blocked" {
		t.Fatalf("expected blocked execution mode, got %s", plan.ExecutionMode)
	}
	if len(plan.Blockers) == 0 || !strings.Contains(strings.Join(plan.Blockers, "\n"), "VPN_FIREWALL_APPLY_ENABLED") {
		t.Fatalf("expected firewall gate blocker, got %#v", plan.Blockers)
	}
}

func TestBuildProductionPlanProvidesProtocolCommandsWhenGatesAreReady(t *testing.T) {
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
		plan, err := BuildProductionPlan(tt.protocol, ProductionGates{RuntimeExecution: true, FirewallApply: true, HostVerification: true}, false)
		if err != nil {
			t.Fatalf("BuildProductionPlan(%s) returned error: %v", tt.protocol, err)
		}
		if !plan.Ready {
			t.Fatalf("expected %s production plan to be ready when gates are enabled: %#v", tt.protocol, plan.Blockers)
		}
		if plan.ExecutionMode != "manual" {
			t.Fatalf("expected manual execution mode by default, got %s", plan.ExecutionMode)
		}
		blob := strings.Join(append(append(plan.RuntimeCommands, plan.FirewallCommands...), plan.StatusCommands...), "\n")
		for _, want := range tt.wants {
			if !strings.Contains(blob, want) {
				t.Fatalf("expected %s commands to include %q, got %s", tt.protocol, want, blob)
			}
		}
	}
}

func TestBuildProductionPlanMarksExecutorModeOnlyWhenRequested(t *testing.T) {
	plan, err := BuildProductionPlan(models.ProtocolSSTP, ProductionGates{RuntimeExecution: true, FirewallApply: true, HostVerification: true}, true)
	if err != nil {
		t.Fatalf("BuildProductionPlan returned error: %v", err)
	}
	if plan.ExecutionMode != "executor_enabled" {
		t.Fatalf("expected executor_enabled mode, got %s", plan.ExecutionMode)
	}
}

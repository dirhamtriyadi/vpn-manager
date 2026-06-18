package vpn

import (
	"strings"
	"testing"

	"github.com/example/wg-panel/models"
)

func TestBuildProtocolRoadmapRequiresExplicitExecutionGates(t *testing.T) {
	roadmap, err := BuildProtocolRoadmap(models.ProtocolL2TPIPsec, false, false, false)
	if err != nil {
		t.Fatalf("BuildProtocolRoadmap returned error: %v", err)
	}
	if roadmap.Available {
		t.Fatal("L2TP/IPsec should not be available without explicit execution gates")
	}
	if roadmap.EnablementReady {
		t.Fatal("L2TP/IPsec should not be enablement-ready without host verification and apply gates")
	}
	if len(roadmap.EnablementBlockers) != 3 {
		t.Fatalf("expected 3 blockers, got %d: %#v", len(roadmap.EnablementBlockers), roadmap.EnablementBlockers)
	}
}

func TestBuildProtocolServicePlansForAllRoadmapProtocols(t *testing.T) {
	tests := []struct {
		protocol models.VPNProtocol
		contains []string
	}{
		{models.ProtocolL2TPIPsec, []string{"strongSwan", "xl2tpd", "ipsec", "iptables"}},
		{models.ProtocolSSTP, []string{"sstpd", "TLS", "ppp", "iptables"}},
		{models.ProtocolPPTP, []string{"pptpd", "GRE", "legacy/insecure", "iptables"}},
	}
	for _, tt := range tests {
		plan, err := BuildProtocolServicePlan(tt.protocol)
		if err != nil {
			t.Fatalf("BuildProtocolServicePlan(%s) returned error: %v", tt.protocol, err)
		}
		if plan.ExecutionMode != "dry_run" || plan.Status != "planned" {
			t.Fatalf("expected dry-run planned service plan, got mode=%s status=%s", plan.ExecutionMode, plan.Status)
		}
		blob := strings.Join(append(append(plan.Components, plan.RuntimePlan...), plan.FirewallPlan...), "\n")
		for _, want := range tt.contains {
			if !strings.Contains(blob, want) {
				t.Fatalf("plan for %s missing %q in %#v", tt.protocol, want, plan)
			}
		}
	}
}

func TestBuildProtocolRoadmapReadyWhenAllGatesAreEnabled(t *testing.T) {
	roadmap, err := BuildProtocolRoadmap(models.ProtocolSSTP, true, true, true)
	if err != nil {
		t.Fatalf("BuildProtocolRoadmap returned error: %v", err)
	}
	if !roadmap.EnablementReady {
		t.Fatal("expected enablement_ready when runtime, firewall, and host verification gates are true")
	}
	if roadmap.Available {
		t.Fatal("roadmap protocol should still require a real registered driver before available=true")
	}
	if len(roadmap.EnablementBlockers) != 0 {
		t.Fatalf("expected no blockers when gates are enabled, got %#v", roadmap.EnablementBlockers)
	}
}

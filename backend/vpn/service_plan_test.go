package vpn

import (
	"strings"
	"testing"

	"github.com/example/vpn-manager/models"
)

func TestBuildProtocolRoadmapRequiresExecutionEnabled(t *testing.T) {
	roadmap, err := BuildProtocolRoadmap(models.ProtocolL2TPIPsec, false)
	if err != nil {
		t.Fatalf("BuildProtocolRoadmap returned error: %v", err)
	}
	if !roadmap.Available {
		t.Fatal("L2TP/IPsec is implemented and should report available")
	}
	if roadmap.EnablementReady {
		t.Fatal("L2TP/IPsec should not be enablement-ready while VPN_EXECUTION_ENABLED is off")
	}
	if len(roadmap.EnablementBlockers) != 1 {
		t.Fatalf("expected 1 blocker, got %d: %#v", len(roadmap.EnablementBlockers), roadmap.EnablementBlockers)
	}
}

func TestBuildProtocolServicePlansForAllProtocols(t *testing.T) {
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
		if plan.ExecutionMode != "host_apply" || plan.Status != "available" {
			t.Fatalf("expected host_apply available service plan, got mode=%s status=%s", plan.ExecutionMode, plan.Status)
		}
		blob := strings.Join(append(append(plan.Components, plan.RuntimePlan...), plan.FirewallPlan...), "\n")
		for _, want := range tt.contains {
			if !strings.Contains(blob, want) {
				t.Fatalf("plan for %s missing %q in %#v", tt.protocol, want, plan)
			}
		}
	}
}

func TestBuildProtocolRoadmapReadyWhenExecutionEnabled(t *testing.T) {
	roadmap, err := BuildProtocolRoadmap(models.ProtocolSSTP, true)
	if err != nil {
		t.Fatalf("BuildProtocolRoadmap returned error: %v", err)
	}
	if !roadmap.EnablementReady {
		t.Fatal("expected enablement_ready when VPN_EXECUTION_ENABLED is true")
	}
	if !roadmap.Available {
		t.Fatal("expected available=true for an implemented protocol")
	}
	if len(roadmap.EnablementBlockers) != 0 {
		t.Fatalf("expected no blockers when execution is enabled, got %#v", roadmap.EnablementBlockers)
	}
}

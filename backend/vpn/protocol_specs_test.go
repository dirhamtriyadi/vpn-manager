package vpn

import (
	"testing"

	"github.com/example/vpn-manager/models"
)

func TestAllProtocolSpecsDefinesSupportedRoadmap(t *testing.T) {
	specs := AllProtocolSpecs()
	if len(specs) != 5 {
		t.Fatalf("expected 5 protocol specs, got %d", len(specs))
	}

	wantOrder := []models.VPNProtocol{
		models.ProtocolWireGuard,
		models.ProtocolOpenVPN,
		models.ProtocolL2TPIPsec,
		models.ProtocolSSTP,
		models.ProtocolPPTP,
	}
	for i, want := range wantOrder {
		if specs[i].Protocol != want {
			t.Fatalf("spec[%d] protocol = %s, want %s", i, specs[i].Protocol, want)
		}
	}

	wireGuard, ok := GetProtocolSpec(models.ProtocolWireGuard)
	if !ok {
		t.Fatal("expected WireGuard spec")
	}
	if wireGuard.Status != ProtocolStatusAvailable {
		t.Fatalf("WireGuard status = %s, want %s", wireGuard.Status, ProtocolStatusAvailable)
	}
	if wireGuard.Capabilities.RuntimeStrategy != "host_kernel_netlink" || !wireGuard.Capabilities.ConfigDownload || !wireGuard.Capabilities.QRCode {
		t.Fatalf("unexpected WireGuard capabilities: %+v", wireGuard.Capabilities)
	}

	pptp, ok := GetProtocolSpec(models.ProtocolPPTP)
	if !ok {
		t.Fatal("expected PPTP spec")
	}
	if pptp.Status != ProtocolStatusLegacyAvailable {
		t.Fatalf("PPTP status = %s, want %s", pptp.Status, ProtocolStatusLegacyAvailable)
	}
	if !pptp.LegacyInsecure {
		t.Fatal("expected PPTP spec to be marked legacy/insecure")
	}
}

func TestGetProtocolSpecRejectsUnknownProtocol(t *testing.T) {
	if _, ok := GetProtocolSpec(models.VPNProtocol("unknown")); ok {
		t.Fatal("expected unknown protocol to be rejected")
	}
}

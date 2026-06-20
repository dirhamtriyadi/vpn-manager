package wg

import (
	"testing"
	"time"

	"github.com/example/vpn-manager/models"
)

func TestPeerDeviceConfigIsIncremental(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}
	psk, err := GeneratePresharedKey()
	if err != nil {
		t.Fatalf("GeneratePresharedKey() error = %v", err)
	}

	cfg, err := PeerDeviceConfig(models.Peer{
		Name:                "router",
		PublicKey:           kp.PublicKey,
		AllowedIPs:          "10.8.0.2/32",
		PresharedKey:        psk,
		PersistentKeepalive: 25,
		Enabled:             true,
	})
	if err != nil {
		t.Fatalf("PeerDeviceConfig() error = %v", err)
	}

	if cfg.ReplacePeers {
		t.Fatalf("ReplacePeers = true, want false so existing peers keep their handshake state")
	}
	if len(cfg.Peers) != 1 {
		t.Fatalf("len(Peers) = %d, want 1", len(cfg.Peers))
	}
	peer := cfg.Peers[0]
	if peer.Remove {
		t.Fatalf("peer.Remove = true, want false for add/update")
	}
	if !peer.ReplaceAllowedIPs {
		t.Fatalf("peer.ReplaceAllowedIPs = false, want true")
	}
	if got, want := len(peer.AllowedIPs), 1; got != want {
		t.Fatalf("len(peer.AllowedIPs) = %d, want %d", got, want)
	}
	if peer.PersistentKeepaliveInterval == nil || *peer.PersistentKeepaliveInterval != 25*time.Second {
		t.Fatalf("PersistentKeepaliveInterval = %v, want 25s", peer.PersistentKeepaliveInterval)
	}
	if peer.PresharedKey == nil {
		t.Fatalf("PresharedKey is nil, want configured")
	}
}

func TestRemovePeerDeviceConfigIsIncremental(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	cfg, err := RemovePeerDeviceConfig(kp.PublicKey)
	if err != nil {
		t.Fatalf("RemovePeerDeviceConfig() error = %v", err)
	}

	if cfg.ReplacePeers {
		t.Fatalf("ReplacePeers = true, want false so deleting one peer does not reset others")
	}
	if len(cfg.Peers) != 1 {
		t.Fatalf("len(Peers) = %d, want 1", len(cfg.Peers))
	}
	if !cfg.Peers[0].Remove {
		t.Fatalf("peer.Remove = false, want true")
	}
}

func TestValidatePublicKeyRejectsInvalidKey(t *testing.T) {
	if err := ValidatePublicKey("not-a-wireguard-key"); err == nil {
		t.Fatalf("ValidatePublicKey() error = nil, want invalid key error")
	}
}

func TestValidateCIDRListRejectsInvalidCIDR(t *testing.T) {
	if err := ValidateCIDRList("0.0.0.0/0, 999.999.999.999/99"); err == nil {
		t.Fatalf("ValidateCIDRList() error = nil, want invalid CIDR error")
	}
}

func TestIPInCIDRRequiresAssignedIPInsideInterfaceSubnet(t *testing.T) {
	if err := ValidateIPInCIDR("10.9.0.2", "10.8.0.1/24"); err == nil {
		t.Fatalf("ValidateIPInCIDR() error = nil, want outside subnet error")
	}
	if err := ValidateIPInCIDR("10.8.0.2", "10.8.0.1/24"); err != nil {
		t.Fatalf("ValidateIPInCIDR() error = %v, want nil", err)
	}
}

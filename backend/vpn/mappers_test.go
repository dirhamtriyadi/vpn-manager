package vpn

import (
	"testing"

	"github.com/example/wg-panel/models"
)

func TestMapWGInterfaceToVPNInstance(t *testing.T) {
	iface := models.WGInterface{
		ID:              7,
		Name:            "wg0",
		PublicKey:       "public-key",
		ListenPort:      51820,
		Address:         "10.8.0.1/24",
		DNS:             "1.1.1.1",
		MTU:             1420,
		Endpoint:        "vpn.example.com",
		Enabled:         true,
		Masquerade:      true,
		EgressInterface: "eth0",
	}

	instance := MapWGInterfaceToVPNInstance(iface)

	if instance.ID != iface.ID || instance.Name != iface.Name {
		t.Fatalf("mapped identity mismatch: %+v", instance)
	}
	if instance.Protocol != models.ProtocolWireGuard {
		t.Fatalf("protocol = %q, want wireguard", instance.Protocol)
	}
	if instance.LegacyInsecure {
		t.Fatal("wireguard instance must not be marked legacy/insecure")
	}
	if instance.WireGuard == nil {
		t.Fatal("expected wireguard metadata")
	}
	if instance.WireGuard.PublicKey != iface.PublicKey || instance.WireGuard.Masquerade != iface.Masquerade {
		t.Fatalf("wireguard metadata mismatch: %+v", instance.WireGuard)
	}
}

func TestMapPeerToVPNUser(t *testing.T) {
	peer := models.Peer{
		ID:                  9,
		InterfaceID:         7,
		Name:                "client-a",
		PublicKey:           "peer-public-key",
		AllowedIPs:          "10.8.0.2/32",
		AssignedIP:          "10.8.0.2",
		ClientAllowedIPs:    "0.0.0.0/0, ::/0",
		PersistentKeepalive: 25,
		Enabled:             true,
		Online:              true,
		RxBytes:             1024,
		TxBytes:             2048,
	}

	user := MapPeerToVPNUser(peer)

	if user.ID != peer.ID || user.InstanceID != peer.InterfaceID || user.Name != peer.Name {
		t.Fatalf("mapped identity mismatch: %+v", user)
	}
	if user.Protocol != models.ProtocolWireGuard {
		t.Fatalf("protocol = %q, want wireguard", user.Protocol)
	}
	if user.WireGuard == nil {
		t.Fatal("expected wireguard user metadata")
	}
	if user.WireGuard.PublicKey != peer.PublicKey || user.WireGuard.PersistentKeepalive != peer.PersistentKeepalive {
		t.Fatalf("wireguard user metadata mismatch: %+v", user.WireGuard)
	}
}

package models

import "testing"

func TestParseVPNProtocolAcceptsKnownProtocols(t *testing.T) {
	cases := map[string]VPNProtocol{
		"wireguard":  ProtocolWireGuard,
		" WireGuard ": ProtocolWireGuard,
		"openvpn":    ProtocolOpenVPN,
		"l2tp_ipsec": ProtocolL2TPIPsec,
		"sstp":       ProtocolSSTP,
		"pptp":       ProtocolPPTP,
	}

	for input, expected := range cases {
		actual, ok := ParseVPNProtocol(input)
		if !ok {
			t.Fatalf("expected %q to parse", input)
		}
		if actual != expected {
			t.Fatalf("ParseVPNProtocol(%q) = %q, want %q", input, actual, expected)
		}
	}
}

func TestParseVPNProtocolRejectsUnknownProtocol(t *testing.T) {
	if protocol, ok := ParseVPNProtocol("ipsec-only"); ok {
		t.Fatalf("expected unknown protocol to be rejected, got %q", protocol)
	}
}

func TestVPNProtocolLegacyInsecureMarksOnlyPPTP(t *testing.T) {
	if !ProtocolPPTP.IsLegacyInsecure() {
		t.Fatal("expected PPTP to be marked legacy/insecure")
	}
	for _, protocol := range []VPNProtocol{ProtocolWireGuard, ProtocolOpenVPN, ProtocolL2TPIPsec, ProtocolSSTP} {
		if protocol.IsLegacyInsecure() {
			t.Fatalf("expected %q not to be marked legacy/insecure", protocol)
		}
	}
}

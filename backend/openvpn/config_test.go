package openvpn

import (
	"strings"
	"testing"
)

func TestBuildClientProfileUsesSafeInlineCertificates(t *testing.T) {
	profile, err := BuildClientProfile(ClientProfileInput{
		ClientName: "laptop",
		RemoteHost: "vpn.example.com",
		RemotePort: 1194,
		Protocol: "udp",
		CACertPEM: "-----BEGIN CERTIFICATE-----\nCA\n-----END CERTIFICATE-----",
		ClientCertPEM: "-----BEGIN CERTIFICATE-----\nCLIENT\n-----END CERTIFICATE-----",
		ClientKeyPEM: "-----BEGIN PRIVATE KEY-----\nKEY\n-----END PRIVATE KEY-----",
	})
	if err != nil {
		t.Fatalf("BuildClientProfile returned error: %v", err)
	}

	for _, want := range []string{
		"client",
		"dev tun",
		"proto udp",
		"remote vpn.example.com 1194",
		"remote-cert-tls server",
		"<ca>",
		"<cert>",
		"<key>",
	} {
		if !strings.Contains(profile, want) {
			t.Fatalf("profile missing %q:\n%s", want, profile)
		}
	}
}

func TestBuildClientProfileRejectsMissingRuntimeData(t *testing.T) {
	_, err := BuildClientProfile(ClientProfileInput{ClientName: "client"})
	if err == nil {
		t.Fatal("expected missing remote/cert data to fail")
	}
}

func TestNormalizeProtocolDefaultsToUDP(t *testing.T) {
	if got := normalizeProtocol(""); got != "udp" {
		t.Fatalf("empty protocol = %q, want udp", got)
	}
	if got := normalizeProtocol(" TCP "); got != "tcp" {
		t.Fatalf("TCP protocol = %q, want tcp", got)
	}
	if got := normalizeProtocol("bad"); got != "udp" {
		t.Fatalf("bad protocol = %q, want udp", got)
	}
}

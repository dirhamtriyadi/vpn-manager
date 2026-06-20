package openvpn

import (
	"strings"
	"testing"

	"github.com/example/vpn-manager/secrets"
)

func TestBuildInstanceDraftStoresSecretReferencesOnly(t *testing.T) {
	envelope, err := secrets.NewEnvelope("test-master-key")
	if err != nil {
		t.Fatalf("NewEnvelope returned error: %v", err)
	}

	draft, err := BuildInstanceDraft(InstanceDraftInput{
		Name:          "office vpn",
		RemoteHost:    "vpn.example.com",
		ListenPort:    1194,
		Protocol:      "udp4",
		TunnelCIDR:    "10.20.0.0/24",
		DNS:           "1.1.1.1",
		CACertPEM:     "-----BEGIN CERTIFICATE-----\nca\n-----END CERTIFICATE-----",
		ServerCertPEM: "-----BEGIN CERTIFICATE-----\nserver\n-----END CERTIFICATE-----",
		ServerKeyPEM:  "-----BEGIN PRIVATE KEY-----\nserver-key\n-----END PRIVATE KEY-----",
		TLSCryptPEM:   "-----BEGIN OpenVPN Static key V1-----\ntls\n-----END OpenVPN Static key V1-----",
	}, envelope)
	if err != nil {
		t.Fatalf("BuildInstanceDraft returned error: %v", err)
	}

	if draft.Instance.Enabled {
		t.Fatal("draft OpenVPN instance must stay disabled")
	}
	if draft.Instance.Protocol != "udp" {
		t.Fatalf("protocol normalized to %q, want udp", draft.Instance.Protocol)
	}
	if draft.Instance.CARef == "" || draft.Instance.ServerCertRef == "" || draft.Instance.ServerKeyRef == "" || draft.Instance.TLSCryptRef == "" {
		t.Fatalf("missing secret refs: %#v", draft.Instance)
	}
	if len(draft.Secrets) != 4 {
		t.Fatalf("secret count = %d, want 4", len(draft.Secrets))
	}
	for _, secret := range draft.Secrets {
		if strings.Contains(secret.Ciphertext, "PRIVATE KEY") || strings.Contains(secret.Ciphertext, "CERTIFICATE") || strings.Contains(secret.Ciphertext, "server-key") {
			t.Fatalf("secret ciphertext exposes plaintext: %#v", secret)
		}
	}
}

func TestBuildInstanceDraftRejectsMissingRequiredCertificates(t *testing.T) {
	envelope, err := secrets.NewEnvelope("test-master-key")
	if err != nil {
		t.Fatalf("NewEnvelope returned error: %v", err)
	}

	_, err = BuildInstanceDraft(InstanceDraftInput{
		Name:       "office",
		RemoteHost: "vpn.example.com",
		TunnelCIDR: "10.20.0.0/24",
		CACertPEM:  "ca",
	}, envelope)
	if err == nil {
		t.Fatal("expected missing server cert/key to fail")
	}
}

func TestBuildInstanceDraftRejectsNilEnvelope(t *testing.T) {
	_, err := BuildInstanceDraft(InstanceDraftInput{
		Name:          "office",
		RemoteHost:    "vpn.example.com",
		TunnelCIDR:    "10.20.0.0/24",
		CACertPEM:     "ca",
		ServerCertPEM: "cert",
		ServerKeyPEM:  "key",
	}, nil)
	if err == nil {
		t.Fatal("expected nil envelope to fail")
	}
}

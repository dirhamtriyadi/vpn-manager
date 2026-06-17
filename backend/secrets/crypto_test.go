package secrets

import (
	"strings"
	"testing"
)

func TestEncryptDecryptRoundTripDoesNotExposePlaintext(t *testing.T) {
	store, err := NewEnvelope("dev-master-key")
	if err != nil {
		t.Fatalf("NewEnvelope returned error: %v", err)
	}

	sealed, err := store.Encrypt("-----BEGIN PRIVATE KEY-----\nsecret\n-----END PRIVATE KEY-----")
	if err != nil {
		t.Fatalf("Encrypt returned error: %v", err)
	}
	if sealed.Nonce == "" || sealed.Ciphertext == "" || sealed.Algorithm == "" {
		t.Fatalf("sealed secret has empty fields: %#v", sealed)
	}
	if strings.Contains(sealed.Ciphertext, "PRIVATE KEY") || strings.Contains(sealed.Ciphertext, "secret") {
		t.Fatalf("ciphertext exposes plaintext: %s", sealed.Ciphertext)
	}

	plain, err := store.Decrypt(sealed)
	if err != nil {
		t.Fatalf("Decrypt returned error: %v", err)
	}
	if plain != "-----BEGIN PRIVATE KEY-----\nsecret\n-----END PRIVATE KEY-----" {
		t.Fatalf("decrypted plaintext mismatch: %q", plain)
	}
}

func TestNewEnvelopeRejectsEmptyMasterKey(t *testing.T) {
	if _, err := NewEnvelope("   "); err == nil {
		t.Fatal("expected empty master key to fail")
	}
}

func TestBuildRefSanitizesSecretReferences(t *testing.T) {
	ref := BuildRef("openvpn", 42, "Server Key PEM")
	if ref != "openvpn/42/server-key-pem" {
		t.Fatalf("ref = %q", ref)
	}
}

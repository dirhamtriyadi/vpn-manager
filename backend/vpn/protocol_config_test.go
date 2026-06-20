package vpn

import (
	"strings"
	"testing"

	"github.com/example/vpn-manager/models"
)

func TestBuildProtocolConfigPreviewForL2TPIPsec(t *testing.T) {
	preview, err := BuildProtocolConfigPreview(ProtocolConfigInput{
		Protocol:   models.ProtocolL2TPIPsec,
		Name:       "office-l2tp",
		RemoteHost: "vpn.example.com",
		ListenPort: 1701,
		PoolCIDR:   "10.30.0.0/24",
		DNS:        "1.1.1.1",
	})
	if err != nil {
		t.Fatalf("BuildProtocolConfigPreview returned error: %v", err)
	}
	assertFileContains(t, preview.Files, "ipsec.conf", "leftprotoport=17/1701")
	assertFileContains(t, preview.Files, "xl2tpd.conf", "ip range = 10.30.0.0/24")
	assertFileContains(t, preview.Files, "chap-secrets", "[ENCRYPTED_SECRET_REF]")
}

func TestBuildProtocolConfigPreviewForSSTP(t *testing.T) {
	preview, err := BuildProtocolConfigPreview(ProtocolConfigInput{
		Protocol:   models.ProtocolSSTP,
		Name:       "office-sstp",
		RemoteHost: "vpn.example.com",
		ListenPort: 443,
		PoolCIDR:   "10.40.0.0/24",
		DNS:        "1.1.1.1",
	})
	if err != nil {
		t.Fatalf("BuildProtocolConfigPreview returned error: %v", err)
	}
	assertFileContains(t, preview.Files, "sstpd.conf", "listen = 0.0.0.0")
	assertFileContains(t, preview.Files, "sstpd.conf", "port = 443")
	assertFileContains(t, preview.Files, "chap-secrets", "[ENCRYPTED_SECRET_REF]")
}

func TestBuildProtocolConfigPreviewForPPTP(t *testing.T) {
	preview, err := BuildProtocolConfigPreview(ProtocolConfigInput{
		Protocol:   models.ProtocolPPTP,
		Name:       "office-pptp",
		RemoteHost: "vpn.example.com",
		ListenPort: 1723,
		PoolCIDR:   "10.50.0.0/24",
		DNS:        "1.1.1.1",
	})
	if err != nil {
		t.Fatalf("BuildProtocolConfigPreview returned error: %v", err)
	}
	assertFileContains(t, preview.Files, "pptpd.conf", "localip")
	assertFileContains(t, preview.Files, "options.pptpd", "require-mschap-v2")
	if !preview.LegacyInsecure {
		t.Fatal("PPTP preview must be marked legacy insecure")
	}
}

func TestBuildProtocolConfigPreviewRejectsInvalidInput(t *testing.T) {
	_, err := BuildProtocolConfigPreview(ProtocolConfigInput{Protocol: models.ProtocolSSTP, Name: "bad", PoolCIDR: "not-a-cidr"})
	if err == nil || !strings.Contains(err.Error(), "pool_cidr") {
		t.Fatalf("expected pool_cidr validation error, got %v", err)
	}
}

func assertFileContains(t *testing.T, files map[string]string, filename string, want string) {
	t.Helper()
	got, ok := files[filename]
	if !ok {
		t.Fatalf("expected file %s in %#v", filename, files)
	}
	if !strings.Contains(got, want) {
		t.Fatalf("expected %s to contain %q, got:\n%s", filename, want, got)
	}
}

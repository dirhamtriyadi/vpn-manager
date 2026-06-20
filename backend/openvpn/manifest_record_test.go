package openvpn

import (
	"strings"
	"testing"

	"github.com/example/vpn-manager/models"
)

func TestBuildRuntimeManifestRecordFromInstance(t *testing.T) {
	instance := models.OpenVPNInstance{
		ID:          42,
		Name:        "office",
		RemoteHost:  "vpn.example.com",
		ListenPort:  1194,
		Protocol:    "udp",
		TunnelCIDR:  "10.20.0.0/24",
		DNS:         "1.1.1.1",
		RuntimeMode: "container_openvpn_preview",
	}

	record, err := BuildRuntimeManifestRecord(instance)
	if err != nil {
		t.Fatalf("BuildRuntimeManifestRecord returned error: %v", err)
	}

	if record.InstanceID != instance.ID {
		t.Fatalf("InstanceID = %d, want %d", record.InstanceID, instance.ID)
	}
	if record.RuntimeMode != "container" {
		t.Fatalf("RuntimeMode = %q, want container", record.RuntimeMode)
	}
	if !strings.Contains(record.ServerConf, "server 10.20.0.0 255.255.255.0") {
		t.Fatalf("server.conf missing normalized CIDR/netmask: %s", record.ServerConf)
	}
	if !strings.Contains(record.ComposeYAML, "NET_ADMIN") || !strings.Contains(record.ComposeYAML, "/dev/net/tun:/dev/net/tun") {
		t.Fatalf("compose yaml missing required container privileges: %s", record.ComposeYAML)
	}
	if record.GenerationStatus != "generated" {
		t.Fatalf("GenerationStatus = %q, want generated", record.GenerationStatus)
	}
	if !strings.Contains(record.Warnings, "Preview only") {
		t.Fatalf("warnings should preserve runtime preview warnings: %s", record.Warnings)
	}
}

func TestBuildRuntimeManifestRecordRejectsUnsavedInstance(t *testing.T) {
	_, err := BuildRuntimeManifestRecord(models.OpenVPNInstance{
		Name:       "office",
		RemoteHost: "vpn.example.com",
		TunnelCIDR: "10.20.0.0/24",
	})
	if err == nil {
		t.Fatal("expected unsaved instance to fail")
	}
}

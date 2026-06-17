package openvpn

import (
	"strings"
	"testing"
)

func TestBuildContainerRuntimeManifestIncludesSafeRuntimeFiles(t *testing.T) {
	manifest, err := BuildContainerRuntimeManifest(RuntimeManifestInput{
		InstanceName: "office",
		RemoteHost:   "vpn.example.com",
		ListenPort:   1194,
		Protocol:     "udp",
		TunnelCIDR:   "10.20.0.0/24",
		DNS:          "1.1.1.1",
	})
	if err != nil {
		t.Fatalf("BuildContainerRuntimeManifest returned error: %v", err)
	}

	if manifest.RuntimeMode != "container" {
		t.Fatalf("RuntimeMode = %q, want container", manifest.RuntimeMode)
	}
	serverConf := manifest.Files["server.conf"]
	compose := manifest.Files["docker-compose.yml"]
	for _, want := range []string{
		"port 1194",
		"proto udp",
		"server 10.20.0.0 255.255.255.0",
		"push \"dhcp-option DNS 1.1.1.1\"",
	} {
		if !strings.Contains(serverConf, want) {
			t.Fatalf("server.conf missing %q:\n%s", want, serverConf)
		}
	}
	for _, want := range []string{
		"vpn-manager-openvpn-office",
		"NET_ADMIN",
		"/dev/net/tun:/dev/net/tun",
		"1194:1194/udp",
	} {
		if !strings.Contains(compose, want) {
			t.Fatalf("docker-compose.yml missing %q:\n%s", want, compose)
		}
	}
	if strings.Contains(compose, "PRIVATE KEY") || strings.Contains(serverConf, "PRIVATE KEY") {
		t.Fatal("runtime manifest must not inline private keys")
	}
}

func TestBuildContainerRuntimeManifestRejectsInvalidTunnelCIDR(t *testing.T) {
	_, err := BuildContainerRuntimeManifest(RuntimeManifestInput{
		InstanceName: "bad",
		RemoteHost:   "vpn.example.com",
		ListenPort:   1194,
		TunnelCIDR:   "not-a-cidr",
	})
	if err == nil {
		t.Fatal("expected invalid tunnel cidr to fail")
	}
}

func TestBuildContainerRuntimeManifestDefaultsProtocolAndPort(t *testing.T) {
	manifest, err := BuildContainerRuntimeManifest(RuntimeManifestInput{
		InstanceName: "defaulted",
		RemoteHost:   "vpn.example.com",
		TunnelCIDR:   "10.21.0.0/24",
	})
	if err != nil {
		t.Fatalf("BuildContainerRuntimeManifest returned error: %v", err)
	}
	if !strings.Contains(manifest.Files["server.conf"], "proto udp") {
		t.Fatalf("server.conf should default proto udp:\n%s", manifest.Files["server.conf"])
	}
	if !strings.Contains(manifest.Files["docker-compose.yml"], "1194:1194/udp") {
		t.Fatalf("compose should default port/proto mapping:\n%s", manifest.Files["docker-compose.yml"])
	}
}

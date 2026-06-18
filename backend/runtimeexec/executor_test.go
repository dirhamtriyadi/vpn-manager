package runtimeexec

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecutorRefusesApplyWithoutAllGates(t *testing.T) {
	plan := ApplyPlan{Files: map[string]string{"server.conf": "dev tun\n"}, Commands: []string{"true"}}
	result, err := Apply(context.Background(), Options{RootDir: t.TempDir(), Gates: Gates{RuntimeExecution: true, FirewallApply: false, HostVerification: true}, ExecutorEnabled: true}, plan)
	if err == nil || !strings.Contains(err.Error(), "VPN_FIREWALL_APPLY_ENABLED") {
		t.Fatalf("expected firewall gate error, got result=%#v err=%v", result, err)
	}
}

func TestExecutorWritesFilesAndRunsCommandsWhenGated(t *testing.T) {
	root := t.TempDir()
	runs := []string{}
	plan := ApplyPlan{
		Files: map[string]string{
			"openvpn/1/server.conf":         "dev tun\n",
			"openvpn/1/docker-compose.yml": "services: {}\n",
		},
		Commands: []string{"docker compose ps", "iptables -S"},
	}
	result, err := Apply(context.Background(), Options{
		RootDir:         root,
		Gates:           Gates{RuntimeExecution: true, FirewallApply: true, HostVerification: true},
		ExecutorEnabled: true,
		Runner: func(ctx context.Context, command string) CommandResult {
			runs = append(runs, command)
			return CommandResult{Command: command, ExitCode: 0, Output: "ok"}
		},
	}, plan)
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if result.Status != "applied" {
		t.Fatalf("expected applied status, got %s", result.Status)
	}
	if len(runs) != 2 {
		t.Fatalf("expected two command runs, got %#v", runs)
	}
	content, err := os.ReadFile(filepath.Join(root, "openvpn/1/server.conf"))
	if err != nil {
		t.Fatalf("expected rendered file: %v", err)
	}
	if string(content) != "dev tun\n" {
		t.Fatalf("unexpected file content: %q", string(content))
	}
}

func TestExecutorRejectsUnsafeFilePath(t *testing.T) {
	_, err := Apply(context.Background(), Options{RootDir: t.TempDir(), Gates: Gates{RuntimeExecution: true, FirewallApply: true, HostVerification: true}, ExecutorEnabled: true}, ApplyPlan{Files: map[string]string{"../etc/passwd": "bad"}})
	if err == nil || !strings.Contains(err.Error(), "unsafe file path") {
		t.Fatalf("expected unsafe file path error, got %v", err)
	}
}

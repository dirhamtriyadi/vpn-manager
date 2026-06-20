package wg

import (
	"strings"
	"testing"
)

func TestSubnetFor(t *testing.T) {
	cases := map[string]string{
		"10.8.0.1/24":    "10.8.0.0/24",
		"192.168.9.5/16": "192.168.0.0/16",
	}
	for in, want := range cases {
		got, err := subnetFor(in)
		if err != nil {
			t.Fatalf("subnetFor(%q): %v", in, err)
		}
		if got != want {
			t.Fatalf("subnetFor(%q) = %q, want %q", in, got, want)
		}
	}
	if _, err := subnetFor("not-a-cidr"); err == nil {
		t.Fatal("expected error for invalid address")
	}
}

func TestNatRules(t *testing.T) {
	rules := natRules("10.8.0.0/24", "wg0", "eth0")
	if len(rules) != 4 {
		t.Fatalf("expected 4 rules, got %d", len(rules))
	}
	if rules[0].table != "nat" || rules[0].chain != "POSTROUTING" {
		t.Fatalf("first rule should be nat/POSTROUTING, got %s/%s", rules[0].table, rules[0].chain)
	}
	if rules[1].table != "" || rules[1].chain != "FORWARD" {
		t.Fatalf("forward rule should be filter/FORWARD, got %q/%s", rules[1].table, rules[1].chain)
	}
	// The last rule allows peer-to-peer traffic within the tunnel (wg0 -> wg0).
	last := rules[len(rules)-1]
	body := strings.Join(last.args, " ")
	if last.chain != "FORWARD" || body != "-i wg0 -o wg0 -j ACCEPT" {
		t.Fatalf("intra-tunnel rule = %s %q, want FORWARD \"-i wg0 -o wg0 -j ACCEPT\"", last.chain, body)
	}
}

func TestIptablesArgs(t *testing.T) {
	rules := natRules("10.8.0.0/24", "wg0", "eth0")

	got := strings.Join(iptablesArgs("-A", rules[0]), " ")
	want := "-t nat -A POSTROUTING -s 10.8.0.0/24 -o eth0 -j MASQUERADE"
	if got != want {
		t.Fatalf("nat args = %q, want %q", got, want)
	}

	// The check (-C) and delete (-D) actions must produce the same rule body so
	// teardown removes exactly what setup added.
	add := iptablesArgs("-A", rules[1])
	del := iptablesArgs("-D", rules[1])
	if strings.Join(add[1:], " ") != strings.Join(del[1:], " ") {
		t.Fatalf("add/delete rule bodies differ: %v vs %v", add, del)
	}
	if del[0] != "-D" {
		t.Fatalf("expected -D action, got %q", del[0])
	}
}

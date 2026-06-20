package wg

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/example/vpn-manager/models"
)

// This file publishes a peer behind WireGuard on the host's public IP. Inbound
// traffic on the WAN (egress) port is DNAT'd into the tunnel to the peer's
// tunnel IP, then masqueraded so the reply returns through the server. The peer
// (e.g. a MikroTik behind CGNAT) does the final DST-NAT to its own LAN device.
// All rules carry a per-forward comment so they can be removed precisely.

func normalizeProto(p string) string {
	if strings.ToLower(strings.TrimSpace(p)) == "udp" {
		return "udp"
	}
	return "tcp"
}

func pfComment(id uint) string {
	return fmt.Sprintf("vpn-manager-pf-%d", id)
}

// portForwardRules builds the DNAT + FORWARD (both directions) + MASQUERADE
// rules for one port forward. wgDevice is the server WireGuard interface name.
func portForwardRules(pf models.PortForward, wgDevice string) []rule {
	proto := normalizeProto(pf.Protocol)
	dport := strconv.Itoa(pf.PublicPort)
	tport := strconv.Itoa(pf.TargetPort)
	target := pf.TargetIP + ":" + tport
	c := pfComment(pf.ID)
	egress := pf.Egress
	return []rule{
		// DNAT inbound WAN:publicPort -> peer tunnel IP:targetPort
		{table: "nat", chain: "PREROUTING", args: []string{"-i", egress, "-p", proto, "--dport", dport, "-j", "DNAT", "--to-destination", target, "-m", "comment", "--comment", c}},
		// allow the new connection WAN -> tunnel (FORWARD rules are inserted at the
		// top so they beat a default FORWARD DROP policy / Docker's chains)
		{chain: "FORWARD", args: []string{"-i", egress, "-o", wgDevice, "-p", proto, "-d", pf.TargetIP, "--dport", tport, "-j", "ACCEPT", "-m", "comment", "--comment", c}},
		// allow the reply tunnel -> WAN
		{chain: "FORWARD", args: []string{"-i", wgDevice, "-o", egress, "-m", "state", "--state", "RELATED,ESTABLISHED", "-j", "ACCEPT", "-m", "comment", "--comment", c}},
		// masquerade into the tunnel so the peer's reply comes back to the server
		{table: "nat", chain: "POSTROUTING", args: []string{"-o", wgDevice, "-p", proto, "-d", pf.TargetIP, "--dport", tport, "-j", "MASQUERADE", "-m", "comment", "--comment", c}},
	}
}

// ApplyPortForward installs the rules. It is idempotent. The forward must have
// its TargetIP and Egress resolved (snapshot) before calling.
func ApplyPortForward(pf models.PortForward, wgDevice string) error {
	if strings.TrimSpace(pf.TargetIP) == "" || strings.TrimSpace(pf.Egress) == "" {
		return fmt.Errorf("port forward %d is missing a resolved target IP or egress interface", pf.ID)
	}
	if err := EnableForwarding(); err != nil {
		return err
	}
	for _, r := range portForwardRules(pf, wgDevice) {
		var err error
		if r.chain == "FORWARD" {
			err = ensureRuleInsert(r)
		} else {
			err = ensureRule(r)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// RemovePortForward removes the rules. Missing rules are ignored.
func RemovePortForward(pf models.PortForward, wgDevice string) error {
	var firstErr error
	for _, r := range portForwardRules(pf, wgDevice) {
		if err := deleteRule(r); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// ensureRuleInsert inserts a rule at the top of its chain if absent. Inserting
// (vs appending) keeps the rule ahead of a default DROP policy and Docker's own
// FORWARD rules.
func ensureRuleInsert(r rule) error {
	if runIptables(iptablesArgs("-C", r)...) == nil {
		return nil
	}
	var args []string
	if r.table != "" {
		args = append(args, "-t", r.table)
	}
	args = append(args, "-I", r.chain, "1")
	args = append(args, r.args...)
	return runIptables(args...)
}

package wg

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"

	"github.com/example/vpn-manager/models"
	"github.com/vishvananda/netlink"
)

// This file gives an interface's clients internet access. WireGuard only moves
// packets into the tunnel; to forward them on to the internet the host needs
// IPv4 forwarding plus NAT (MASQUERADE) and FORWARD rules from the tunnel
// subnet out the WAN interface. All rules are scoped by the tunnel subnet and
// device name so they can be removed again without touching unrelated rules.

// rule is a single iptables rule in a table/chain.
type rule struct {
	table string // empty means the default "filter" table
	chain string
	args  []string
}

// natRules returns the rules that make a tunnel subnet route to the internet via
// egress (source NAT plus both forwarding directions) and that let peers on the
// same tunnel reach each other.
func natRules(subnet, device, egress string) []rule {
	return []rule{
		{table: "nat", chain: "POSTROUTING", args: []string{"-s", subnet, "-o", egress, "-j", "MASQUERADE"}},
		{chain: "FORWARD", args: []string{"-i", device, "-o", egress, "-j", "ACCEPT"}},
		{chain: "FORWARD", args: []string{"-i", egress, "-o", device, "-m", "state", "--state", "RELATED,ESTABLISHED", "-j", "ACCEPT"}},
		// Peer-to-peer within the same tunnel (hub-and-spoke): the packet enters
		// and leaves on the same wg device. Without this, Docker's default FORWARD
		// DROP policy silently blocks one peer from reaching another even though
		// each peer's handshake with the server is fine.
		{chain: "FORWARD", args: []string{"-i", device, "-o", device, "-j", "ACCEPT"}},
	}
}

// SetupNAT enables forwarding and installs the masquerade/forward rules for the
// interface. It is idempotent (existing rules are not duplicated) and returns
// the egress interface actually used so the caller can persist it.
func SetupNAT(iface *models.WGInterface) (string, error) {
	subnet, err := subnetFor(iface.Address)
	if err != nil {
		return "", err
	}

	egress := strings.TrimSpace(iface.EgressInterface)
	if egress == "" {
		egress, err = DefaultEgressInterface()
		if err != nil {
			return "", fmt.Errorf("auto-detect egress interface: %w", err)
		}
	}

	if err := EnableForwarding(); err != nil {
		return egress, err
	}
	for _, r := range natRules(subnet, iface.Name, egress) {
		if err := ensureRule(r); err != nil {
			return egress, err
		}
	}
	return egress, nil
}

// TeardownNAT removes the masquerade/forward rules for the interface. Rules that
// are not present are skipped, so it is safe to call repeatedly. IPv4
// forwarding is intentionally left enabled because other tunnels may rely on it.
func TeardownNAT(iface *models.WGInterface) error {
	subnet, err := subnetFor(iface.Address)
	if err != nil {
		return err
	}

	egress := strings.TrimSpace(iface.EgressInterface)
	if egress == "" {
		// Best effort: fall back to the current default route.
		if detected, derr := DefaultEgressInterface(); derr == nil {
			egress = detected
		}
	}
	if egress == "" {
		return nil // nothing concrete to target
	}

	var firstErr error
	for _, r := range natRules(subnet, iface.Name, egress) {
		if err := deleteRule(r); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// EnableForwarding turns on IPv4 forwarding. It is never disabled on teardown
// because it is a host-global setting other interfaces may depend on.
//
// Read-first: when forwarding is already on (the usual case on a WireGuard
// server / router host) there is nothing to do. This matters because under
// network_mode: host the container often gets /proc/sys mounted read-only, so a
// blind WriteFile fails with EROFS/EACCES — and since callers treat that as
// fatal, it would abort the whole NAT / port-forward apply and leave ZERO rules
// installed even though the host already had forwarding enabled.
func EnableForwarding() error {
	const path = "/proc/sys/net/ipv4/ip_forward"
	if cur, err := os.ReadFile(path); err == nil && strings.TrimSpace(string(cur)) == "1" {
		return nil
	}
	if err := os.WriteFile(path, []byte("1\n"), 0o644); err != nil {
		return fmt.Errorf("enable ip_forward (run `sysctl -w net.ipv4.ip_forward=1` on the host): %w", err)
	}
	return nil
}

// DefaultEgressInterface returns the interface carrying the IPv4 default route
// (the WAN uplink), used when masquerade is enabled without an explicit egress.
func DefaultEgressInterface() (string, error) {
	routes, err := netlink.RouteList(nil, netlink.FAMILY_V4)
	if err != nil {
		return "", fmt.Errorf("list routes: %w", err)
	}
	for _, r := range routes {
		if r.Dst != nil && !r.Dst.IP.IsUnspecified() {
			continue // not a default route
		}
		link, err := netlink.LinkByIndex(r.LinkIndex)
		if err != nil {
			return "", fmt.Errorf("resolve egress link: %w", err)
		}
		return link.Attrs().Name, nil
	}
	return "", fmt.Errorf("no IPv4 default route found")
}

// subnetFor returns the network CIDR (e.g. 10.8.0.0/24) for an interface
// address such as 10.8.0.1/24.
func subnetFor(address string) (string, error) {
	_, ipNet, err := net.ParseCIDR(strings.TrimSpace(address))
	if err != nil {
		return "", fmt.Errorf("invalid interface address %q: %w", address, err)
	}
	return ipNet.String(), nil
}

// ensureRule appends a rule if it is not already present.
func ensureRule(r rule) error {
	if runIptables(iptablesArgs("-C", r)...) == nil {
		return nil // already present
	}
	return runIptables(iptablesArgs("-A", r)...)
}

// deleteRule removes a rule if present; a missing rule is not an error.
func deleteRule(r rule) error {
	if runIptables(iptablesArgs("-C", r)...) != nil {
		return nil // not present
	}
	return runIptables(iptablesArgs("-D", r)...)
}

func iptablesArgs(action string, r rule) []string {
	var args []string
	if r.table != "" {
		args = append(args, "-t", r.table)
	}
	args = append(args, action, r.chain)
	return append(args, r.args...)
}

func runIptables(args ...string) error {
	// -w makes iptables wait for the xtables lock (/run/xtables.lock) rather than
	// failing immediately when another process holds it. Under network_mode: host
	// the lock is shared with Docker's own firewall churn and with concurrent
	// reconciles, so without -w a boot-time burst of rule calls can spuriously
	// error out. Wait up to 5s per call.
	full := append([]string{"-w", "5"}, args...)
	out, err := exec.Command("iptables", full...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("iptables %s: %v (%s)", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

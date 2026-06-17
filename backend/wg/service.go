package wg

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/example/wg-panel/models"
	"github.com/vishvananda/netlink"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// KeyPair holds a base64-encoded WireGuard key pair.
type KeyPair struct {
	PrivateKey string
	PublicKey  string
}

// GenerateKeyPair creates a new Curve25519 key pair (no CLI needed).
func GenerateKeyPair() (KeyPair, error) {
	priv, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return KeyPair{}, err
	}
	return KeyPair{
		PrivateKey: priv.String(),
		PublicKey:  priv.PublicKey().String(),
	}, nil
}

// PublicKeyFromPrivate derives the public key from a base64 private key.
func PublicKeyFromPrivate(privateKey string) (string, error) {
	k, err := wgtypes.ParseKey(privateKey)
	if err != nil {
		return "", fmt.Errorf("invalid private key: %w", err)
	}
	return k.PublicKey().String(), nil
}

// GeneratePresharedKey creates a new preshared key.
func GeneratePresharedKey() (string, error) {
	k, err := wgtypes.GenerateKey()
	if err != nil {
		return "", err
	}
	return k.String(), nil
}

// PeerStat is a runtime snapshot read from the kernel for a single peer.
type PeerStat struct {
	LastHandshake time.Time
	RxBytes       int64
	TxBytes       int64
}

// EnsureLink creates the WireGuard netlink device if it does not exist, assigns
// the given CIDR address and brings it up. Requires CAP_NET_ADMIN (root).
func EnsureLink(name, address string, mtu int) error {
	link, err := netlink.LinkByName(name)
	if err != nil {
		la := netlink.NewLinkAttrs()
		la.Name = name
		if mtu > 0 {
			la.MTU = mtu
		}
		wglink := &netlink.Wireguard{LinkAttrs: la}
		if err := netlink.LinkAdd(wglink); err != nil {
			return fmt.Errorf("create link %s: %w", name, err)
		}
		link = wglink
	}

	addr, err := netlink.ParseAddr(address)
	if err != nil {
		return fmt.Errorf("parse address %q: %w", address, err)
	}
	// Add address if not already present (ignore "exists" errors).
	if err := netlink.AddrReplace(link, addr); err != nil {
		return fmt.Errorf("assign address: %w", err)
	}

	if err := netlink.LinkSetUp(link); err != nil {
		return fmt.Errorf("bring up link: %w", err)
	}
	return nil
}

// RemoveLink deletes the WireGuard device.
func RemoveLink(name string) error {
	link, err := netlink.LinkByName(name)
	if err != nil {
		return nil // already gone
	}
	return netlink.LinkDel(link)
}

// ConfigureDevice pushes the private key, listen port and the full peer set to
// the kernel device via netlink (wgctrl). ReplacePeers makes it the source of
// truth, so disabled peers are simply omitted from the slice. Use this for full
// sync/recovery; peer CRUD paths should use ConfigurePeer/RemovePeer to avoid
// resetting other peers' handshake state.
func ConfigureDevice(iface *models.WGInterface, peers []models.Peer) error {
	client, err := wgctrl.New()
	if err != nil {
		return fmt.Errorf("open wgctrl: %w", err)
	}
	defer client.Close()

	cfg, err := FullDeviceConfig(iface, peers)
	if err != nil {
		return err
	}
	if err := client.ConfigureDevice(iface.Name, cfg); err != nil {
		return fmt.Errorf("configure device %s: %w", iface.Name, err)
	}
	return nil
}

// ConfigurePeer adds or updates one enabled peer without replacing the rest of
// the device peer set, preserving existing peer handshakes.
func ConfigurePeer(ifaceName string, peer models.Peer) error {
	client, err := wgctrl.New()
	if err != nil {
		return fmt.Errorf("open wgctrl: %w", err)
	}
	defer client.Close()

	cfg, err := PeerDeviceConfig(peer)
	if err != nil {
		return err
	}
	if err := client.ConfigureDevice(ifaceName, cfg); err != nil {
		return fmt.Errorf("configure peer %q on %s: %w", peer.Name, ifaceName, err)
	}
	return nil
}

// RemovePeer removes one peer from a live device without replacing the rest of
// the peer set, preserving existing peer handshakes.
func RemovePeer(ifaceName, publicKey string) error {
	client, err := wgctrl.New()
	if err != nil {
		return fmt.Errorf("open wgctrl: %w", err)
	}
	defer client.Close()

	cfg, err := RemovePeerDeviceConfig(publicKey)
	if err != nil {
		return err
	}
	if err := client.ConfigureDevice(ifaceName, cfg); err != nil {
		return fmt.Errorf("remove peer from %s: %w", ifaceName, err)
	}
	return nil
}

// FullDeviceConfig builds a source-of-truth configuration for full sync.
func FullDeviceConfig(iface *models.WGInterface, peers []models.Peer) (wgtypes.Config, error) {
	priv, err := wgtypes.ParseKey(iface.PrivateKey)
	if err != nil {
		return wgtypes.Config{}, fmt.Errorf("parse interface private key: %w", err)
	}

	peerConfigs := make([]wgtypes.PeerConfig, 0, len(peers))
	for i := range peers {
		p := peers[i]
		if !p.Enabled {
			continue
		}
		pc, err := buildPeerConfig(p)
		if err != nil {
			return wgtypes.Config{}, err
		}
		peerConfigs = append(peerConfigs, pc)
	}

	port := iface.ListenPort
	return wgtypes.Config{
		PrivateKey:   &priv,
		ListenPort:   &port,
		ReplacePeers: true,
		Peers:        peerConfigs,
	}, nil
}

// PeerDeviceConfig builds an incremental add/update config for one peer.
func PeerDeviceConfig(peer models.Peer) (wgtypes.Config, error) {
	pc, err := buildPeerConfig(peer)
	if err != nil {
		return wgtypes.Config{}, err
	}
	return wgtypes.Config{ReplacePeers: false, Peers: []wgtypes.PeerConfig{pc}}, nil
}

// RemovePeerDeviceConfig builds an incremental remove config for one peer.
func RemovePeerDeviceConfig(publicKey string) (wgtypes.Config, error) {
	pub, err := wgtypes.ParseKey(publicKey)
	if err != nil {
		return wgtypes.Config{}, fmt.Errorf("peer public key: %w", err)
	}
	return wgtypes.Config{
		ReplacePeers: false,
		Peers: []wgtypes.PeerConfig{{
			PublicKey: pub,
			Remove:    true,
		}},
	}, nil
}

// ValidatePublicKey checks that a user-supplied peer public key is a valid
// WireGuard base64 key before the peer is persisted.
func ValidatePublicKey(publicKey string) error {
	if strings.TrimSpace(publicKey) == "" {
		return nil
	}
	if _, err := wgtypes.ParseKey(publicKey); err != nil {
		return fmt.Errorf("invalid public key: %w", err)
	}
	return nil
}

// ValidateCIDRList checks comma-separated WireGuard AllowedIPs values.
func ValidateCIDRList(list string) error {
	_, err := parseCIDRList(list)
	return err
}

// ValidateIPInCIDR checks that an assigned peer IP belongs to the parent
// interface subnet. The IP must be a host address, not the server address.
func ValidateIPInCIDR(assignedIP, serverCIDR string) error {
	assigned := net.ParseIP(strings.TrimSpace(assignedIP))
	if assigned == nil {
		return fmt.Errorf("invalid assigned IP %q", assignedIP)
	}
	serverIP, ipNet, err := net.ParseCIDR(strings.TrimSpace(serverCIDR))
	if err != nil {
		return fmt.Errorf("invalid interface address %q: %w", serverCIDR, err)
	}
	if !ipNet.Contains(assigned) {
		return fmt.Errorf("assigned IP %s is outside interface subnet %s", assigned.String(), ipNet.String())
	}
	if assigned.Equal(serverIP) || assigned.Equal(ipNet.IP) {
		return fmt.Errorf("assigned IP %s is reserved", assigned.String())
	}
	return nil
}

func buildPeerConfig(p models.Peer) (wgtypes.PeerConfig, error) {
	pub, err := wgtypes.ParseKey(p.PublicKey)
	if err != nil {
		return wgtypes.PeerConfig{}, fmt.Errorf("peer %q public key: %w", p.Name, err)
	}

	allowed, err := parseCIDRList(p.AllowedIPs)
	if err != nil {
		return wgtypes.PeerConfig{}, fmt.Errorf("peer %q allowed_ips: %w", p.Name, err)
	}

	pc := wgtypes.PeerConfig{
		PublicKey:         pub,
		ReplaceAllowedIPs: true,
		AllowedIPs:        allowed,
	}
	if p.PresharedKey != "" {
		psk, err := wgtypes.ParseKey(p.PresharedKey)
		if err != nil {
			return wgtypes.PeerConfig{}, fmt.Errorf("peer %q preshared key: %w", p.Name, err)
		}
		pc.PresharedKey = &psk
	}
	if p.PersistentKeepalive > 0 {
		ka := time.Duration(p.PersistentKeepalive) * time.Second
		pc.PersistentKeepaliveInterval = &ka
	}
	return pc, nil
}

// DeviceStats returns a map of peer-public-key -> runtime stats read from the kernel.
func DeviceStats(name string) (map[string]PeerStat, error) {
	client, err := wgctrl.New()
	if err != nil {
		return nil, fmt.Errorf("open wgctrl: %w", err)
	}
	defer client.Close()

	dev, err := client.Device(name)
	if err != nil {
		return nil, fmt.Errorf("read device %s: %w", name, err)
	}

	stats := make(map[string]PeerStat, len(dev.Peers))
	for _, p := range dev.Peers {
		stats[p.PublicKey.String()] = PeerStat{
			LastHandshake: p.LastHandshakeTime,
			RxBytes:       p.ReceiveBytes,
			TxBytes:       p.TransmitBytes,
		}
	}
	return stats, nil
}

// BuildClientConfig renders a ready-to-use wg-quick client configuration.
// PrivateKey must have been stored (server-generated) for the [Interface] block.
func BuildClientConfig(iface *models.WGInterface, peer *models.Peer) string {
	var b strings.Builder
	b.WriteString("[Interface]\n")
	if peer.PrivateKey != "" {
		fmt.Fprintf(&b, "PrivateKey = %s\n", peer.PrivateKey)
	} else {
		b.WriteString("PrivateKey = <your-private-key>\n")
	}
	fmt.Fprintf(&b, "Address = %s/32\n", peer.AssignedIP)
	if iface.DNS != "" {
		fmt.Fprintf(&b, "DNS = %s\n", iface.DNS)
	}
	if iface.MTU > 0 {
		fmt.Fprintf(&b, "MTU = %d\n", iface.MTU)
	}

	b.WriteString("\n[Peer]\n")
	fmt.Fprintf(&b, "PublicKey = %s\n", iface.PublicKey)
	if peer.PresharedKey != "" {
		fmt.Fprintf(&b, "PresharedKey = %s\n", peer.PresharedKey)
	}
	fmt.Fprintf(&b, "Endpoint = %s:%d\n", iface.Endpoint, iface.ListenPort)
	fmt.Fprintf(&b, "AllowedIPs = %s\n", peer.ClientAllowedIPs)
	if peer.PersistentKeepalive > 0 {
		fmt.Fprintf(&b, "PersistentKeepalive = %d\n", peer.PersistentKeepalive)
	}
	return b.String()
}

// NextFreeIP returns the next unused host IP within the interface subnet,
// skipping the server address and any IPs already taken by peers.
func NextFreeIP(serverCIDR string, taken []string) (string, error) {
	ip, ipNet, err := net.ParseCIDR(serverCIDR)
	if err != nil {
		return "", fmt.Errorf("invalid interface address %q: %w", serverCIDR, err)
	}

	used := map[string]bool{ip.String(): true}
	for _, t := range taken {
		used[strings.TrimSpace(t)] = true
	}

	candidate := cloneIP(ipNet.IP)
	for {
		incIP(candidate)
		if !ipNet.Contains(candidate) {
			return "", fmt.Errorf("no free addresses left in %s", serverCIDR)
		}
		// skip network and broadcast-ish first address
		if candidate.Equal(ipNet.IP) {
			continue
		}
		if !used[candidate.String()] {
			return candidate.String(), nil
		}
	}
}

func parseCIDRList(list string) ([]net.IPNet, error) {
	parts := strings.Split(list, ",")
	out := make([]net.IPNet, 0, len(parts))
	for _, raw := range parts {
		s := strings.TrimSpace(raw)
		if s == "" {
			continue
		}
		_, ipNet, err := net.ParseCIDR(s)
		if err != nil {
			return nil, fmt.Errorf("invalid CIDR %q: %w", s, err)
		}
		out = append(out, *ipNet)
	}
	return out, nil
}

func cloneIP(ip net.IP) net.IP {
	dup := make(net.IP, len(ip))
	copy(dup, ip)
	return dup
}

func incIP(ip net.IP) {
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] != 0 {
			break
		}
	}
}

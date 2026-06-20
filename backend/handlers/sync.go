package handlers

import (
	"fmt"

	"github.com/example/vpn-manager/database"
	"github.com/example/vpn-manager/models"
	"github.com/example/vpn-manager/wg"
)

// reconcile applies the desired state of an interface (and its enabled peers)
// to the live kernel device. It is best-effort: callers decide whether a
// failure is fatal or just a warning surfaced to the UI.
func reconcile(ifaceID uint) error {
	var iface models.WGInterface
	if err := database.DB.Preload("Peers").First(&iface, ifaceID).Error; err != nil {
		return err
	}

	if !iface.Enabled {
		// Tear the link down so a disabled interface stops serving traffic, and
		// remove its NAT rules so they don't linger.
		_ = wg.TeardownNAT(&iface)
		return wg.RemoveLink(iface.Name)
	}

	if err := wg.EnsureLink(iface.Name, iface.Address, iface.MTU); err != nil {
		return err
	}
	if err := wg.ConfigureDevice(&iface, iface.Peers); err != nil {
		return err
	}

	return applyNAT(&iface)
}

// applyNAT installs or removes the interface's NAT/forwarding rules to match its
// Masquerade setting. When masquerade is on with an auto-detected egress, the
// resolved interface name is persisted so later teardown targets the same rules.
func applyNAT(iface *models.WGInterface) error {
	if !iface.Masquerade {
		_ = wg.TeardownNAT(iface)
		return nil
	}

	egress, err := wg.SetupNAT(iface)
	if err != nil {
		return fmt.Errorf("nat setup: %w", err)
	}
	if egress != "" && egress != iface.EgressInterface {
		if err := database.DB.Model(iface).Update("egress_interface", egress).Error; err != nil {
			return err
		}
		iface.EgressInterface = egress
	}
	return nil
}

// syncPeer applies a single peer add/update/delete to the live kernel device
// without replacing the full peer set. This preserves other peers' handshakes.
func syncPeer(ifaceID uint, peer models.Peer, remove bool) error {
	var iface models.WGInterface
	if err := database.DB.First(&iface, ifaceID).Error; err != nil {
		return err
	}
	if !iface.Enabled {
		return nil
	}
	if err := wg.EnsureLink(iface.Name, iface.Address, iface.MTU); err != nil {
		return err
	}
	if remove || !peer.Enabled {
		return wg.RemovePeer(iface.Name, peer.PublicKey)
	}
	return wg.ConfigurePeer(iface.Name, peer)
}

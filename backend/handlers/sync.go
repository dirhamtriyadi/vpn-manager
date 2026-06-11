package handlers

import (
	"github.com/example/wg-panel/database"
	"github.com/example/wg-panel/models"
	"github.com/example/wg-panel/wg"
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
		// Tear the link down so a disabled interface stops serving traffic.
		return wg.RemoveLink(iface.Name)
	}

	if err := wg.EnsureLink(iface.Name, iface.Address, iface.MTU); err != nil {
		return err
	}
	return wg.ConfigureDevice(&iface, iface.Peers)
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

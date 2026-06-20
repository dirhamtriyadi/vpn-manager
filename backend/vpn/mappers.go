package vpn

import (
	"github.com/example/vpn-manager/dto"
	"github.com/example/vpn-manager/models"
)

func MapWGInterfaceToVPNInstance(iface models.WGInterface) dto.VPNInstanceResponse {
	return dto.VPNInstanceResponse{
		ID:             iface.ID,
		Protocol:       models.ProtocolWireGuard,
		Name:           iface.Name,
		ListenPort:     iface.ListenPort,
		Address:        iface.Address,
		Endpoint:       iface.Endpoint,
		Enabled:        iface.Enabled,
		LegacyInsecure: models.ProtocolWireGuard.IsLegacyInsecure(),
		WireGuard: &dto.WireGuardMetadata{
			PublicKey:       iface.PublicKey,
			DNS:             iface.DNS,
			MTU:             iface.MTU,
			Masquerade:      iface.Masquerade,
			EgressInterface: iface.EgressInterface,
		},
	}
}

func MapWGInterfacesToVPNInstances(ifaces []models.WGInterface) []dto.VPNInstanceResponse {
	instances := make([]dto.VPNInstanceResponse, 0, len(ifaces))
	for _, iface := range ifaces {
		instances = append(instances, MapWGInterfaceToVPNInstance(iface))
	}
	return instances
}

func MapPeerToVPNUser(peer models.Peer) dto.VPNUserResponse {
	return dto.VPNUserResponse{
		ID:         peer.ID,
		InstanceID: peer.InterfaceID,
		Protocol:   models.ProtocolWireGuard,
		Name:       peer.Name,
		AssignedIP: peer.AssignedIP,
		Enabled:    peer.Enabled,
		Online:     peer.Online,
		RxBytes:    peer.RxBytes,
		TxBytes:    peer.TxBytes,
		WireGuard: &dto.WireGuardUserMeta{
			PublicKey:           peer.PublicKey,
			AllowedIPs:          peer.AllowedIPs,
			ClientAllowedIPs:    peer.ClientAllowedIPs,
			PersistentKeepalive: peer.PersistentKeepalive,
		},
	}
}

func MapPeersToVPNUsers(peers []models.Peer) []dto.VPNUserResponse {
	users := make([]dto.VPNUserResponse, 0, len(peers))
	for _, peer := range peers {
		users = append(users, MapPeerToVPNUser(peer))
	}
	return users
}

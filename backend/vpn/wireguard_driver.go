package vpn

import "github.com/example/wg-panel/models"

type WireGuardDriver struct{}

func NewWireGuardDriver() *WireGuardDriver {
	return &WireGuardDriver{}
}

func (d *WireGuardDriver) Protocol() models.VPNProtocol {
	return models.ProtocolWireGuard
}

func (d *WireGuardDriver) Capabilities() ProtocolCapabilities {
	return ProtocolCapabilities{
		RuntimeStrategy:      "host_kernel_netlink",
		ConfigDownload:       true,
		QRCode:               true,
		RequiresCertificates: false,
	}
}

func (d *WireGuardDriver) Status(instanceID uint) (InstanceStatus, error) {
	return InstanceStatus{Protocol: models.ProtocolWireGuard, Up: true}, nil
}

func (d *WireGuardDriver) Sync(instanceID uint) error {
	return nil
}

func (d *WireGuardDriver) GenerateUserConfig(userID uint) ([]byte, string, error) {
	return nil, "", nil
}

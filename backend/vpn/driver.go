package vpn

import "github.com/example/wg-panel/models"

type InstanceStatus struct {
	Protocol models.VPNProtocol `json:"protocol"`
	Up       bool               `json:"up"`
	Message  string             `json:"message,omitempty"`
}

type ProtocolCapabilities struct {
	RuntimeStrategy      string `json:"runtime_strategy"`
	ConfigDownload       bool   `json:"config_download"`
	QRCode               bool   `json:"qr_code"`
	RequiresCertificates bool   `json:"requires_certificates"`
}

type Driver interface {
	Protocol() models.VPNProtocol
	Capabilities() ProtocolCapabilities
	Status(instanceID uint) (InstanceStatus, error)
	Sync(instanceID uint) error
	GenerateUserConfig(userID uint) ([]byte, string, error)
}

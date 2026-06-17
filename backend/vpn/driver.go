package vpn

import "github.com/example/wg-panel/models"

type InstanceStatus struct {
	Protocol models.VPNProtocol `json:"protocol"`
	Up       bool               `json:"up"`
	Message  string             `json:"message,omitempty"`
}

type Driver interface {
	Protocol() models.VPNProtocol
	Status(instanceID uint) (InstanceStatus, error)
	Sync(instanceID uint) error
	GenerateUserConfig(userID uint) ([]byte, string, error)
}

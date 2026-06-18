package dto

import "github.com/example/wg-panel/models"

type ProtocolConfigPreviewRequest struct {
	Protocol   models.VPNProtocol `json:"protocol" validate:"required,oneof=l2tp_ipsec sstp pptp"`
	Name       string             `json:"name" validate:"required"`
	RemoteHost string             `json:"remote_host" validate:"required"`
	ListenPort int                `json:"listen_port" validate:"omitempty,gte=1,lte=65535"`
	PoolCIDR   string             `json:"pool_cidr" validate:"required,cidr"`
	DNS        string             `json:"dns" validate:"omitempty"`
}

type LegacyVPNInstanceDraftRequest struct {
	Protocol   models.VPNProtocol `json:"protocol" validate:"required,oneof=l2tp_ipsec sstp pptp"`
	Name       string             `json:"name" validate:"required"`
	RemoteHost string             `json:"remote_host" validate:"required"`
	ListenPort int                `json:"listen_port" validate:"omitempty,gte=1,lte=65535"`
	PoolCIDR   string             `json:"pool_cidr" validate:"required,cidr"`
	DNS        string             `json:"dns" validate:"omitempty"`
}

type LegacyVPNInstanceDraftResponse struct {
	ID                  uint               `json:"id"`
	Protocol            models.VPNProtocol `json:"protocol"`
	Name                string             `json:"name"`
	RemoteHost          string             `json:"remote_host"`
	ListenPort          int                `json:"listen_port"`
	PoolCIDR            string             `json:"pool_cidr"`
	DNS                 string             `json:"dns"`
	Enabled             bool               `json:"enabled"`
	RuntimeMode         string             `json:"runtime_mode"`
	SecretStorageStatus string             `json:"secret_storage_status"`
	SecretRefs          map[string]string  `json:"secret_refs"`
}

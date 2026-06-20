package dto

import "github.com/example/vpn-manager/models"

type VPNProtocolResponse struct {
	ID                   models.VPNProtocol `json:"id"`
	Label                string             `json:"label"`
	Status               string             `json:"status"`
	Description          string             `json:"description"`
	Available            bool               `json:"available"`
	LegacyInsecure       bool               `json:"legacy_insecure"`
	RuntimeStrategy      string             `json:"runtime_strategy,omitempty"`
	ConfigDownload       bool               `json:"config_download"`
	QRCode               bool               `json:"qr_code"`
	RequiresCertificates bool               `json:"requires_certificates"`
}

type VPNInstanceResponse struct {
	ID             uint                `json:"id"`
	Protocol       models.VPNProtocol  `json:"protocol"`
	Name           string              `json:"name"`
	ListenPort     int                 `json:"listen_port"`
	Address        string              `json:"address"`
	Endpoint       string              `json:"endpoint"`
	Enabled        bool                `json:"enabled"`
	Status         string              `json:"status,omitempty"`
	LegacyInsecure bool                `json:"legacy_insecure"`
	WireGuard      *WireGuardMetadata  `json:"wireguard,omitempty"`
}

type WireGuardMetadata struct {
	PublicKey       string `json:"public_key"`
	DNS             string `json:"dns"`
	MTU             int    `json:"mtu"`
	Masquerade      bool   `json:"masquerade"`
	EgressInterface string `json:"egress_interface"`
}

type VPNUserResponse struct {
	ID         uint               `json:"id"`
	InstanceID uint               `json:"instance_id"`
	Protocol   models.VPNProtocol `json:"protocol"`
	Name       string             `json:"name"`
	AssignedIP string             `json:"assigned_ip"`
	Enabled    bool               `json:"enabled"`
	Online     bool               `json:"online,omitempty"`
	RxBytes    int64              `json:"rx_bytes,omitempty"`
	TxBytes    int64              `json:"tx_bytes,omitempty"`
	WireGuard  *WireGuardUserMeta  `json:"wireguard,omitempty"`
}

type WireGuardUserMeta struct {
	PublicKey           string `json:"public_key"`
	AllowedIPs          string `json:"allowed_ips"`
	ClientAllowedIPs    string `json:"client_allowed_ips"`
	PersistentKeepalive int    `json:"persistent_keepalive"`
}

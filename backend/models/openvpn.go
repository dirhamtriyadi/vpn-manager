package models

import (
	"time"

	"gorm.io/gorm"
)

// OpenVPNInstance stores OpenVPN server configuration metadata. It is not marked
// available until an OpenVPN runtime driver is registered.
type OpenVPNInstance struct {
	ID          uint   `json:"id" gorm:"primaryKey"`
	Name        string `json:"name" gorm:"size:64;uniqueIndex;not null"`
	RemoteHost  string `json:"remote_host" gorm:"size:255;not null"`
	ListenPort  int    `json:"listen_port" gorm:"not null;default:1194"`
	Protocol    string `json:"protocol" gorm:"size:8;not null;default:udp"`
	TunnelCIDR  string `json:"tunnel_cidr" gorm:"size:64;not null"`
	DNS         string `json:"dns" gorm:"size:128"`
	Enabled     bool   `json:"enabled" gorm:"not null;default:false"`
	RuntimeMode string `json:"runtime_mode" gorm:"size:64;not null;default:container_openvpn_preview"`
	CARef       string `json:"ca_ref" gorm:"size:255"`
	ServerCertRef string `json:"server_cert_ref" gorm:"size:255"`
	ServerKeyRef  string `json:"-" gorm:"size:255"`
	TLSCryptRef   string `json:"tls_crypt_ref" gorm:"size:255"`

	Users []OpenVPNUser `json:"users,omitempty" gorm:"foreignKey:InstanceID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

// OpenVPNUser stores user/client metadata. Certificate/key material should live
// in a dedicated secret store or encrypted column before OpenVPN is enabled.
type OpenVPNUser struct {
	ID         uint   `json:"id" gorm:"primaryKey"`
	InstanceID uint   `json:"instance_id" gorm:"index;not null"`
	Name       string `json:"name" gorm:"size:64;not null"`
	AssignedIP string `json:"assigned_ip" gorm:"size:64"`
	Enabled    bool   `json:"enabled" gorm:"not null;default:true"`
	CertRef    string `json:"cert_ref" gorm:"size:255"`
	KeyRef     string `json:"-" gorm:"size:255"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

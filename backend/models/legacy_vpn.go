package models

import (
	"time"

	"gorm.io/gorm"
)

// LegacyVPNInstance stores draft metadata for non-WireGuard, non-OpenVPN protocols.
// Runtime execution remains gated until a protocol-specific driver is registered.
type LegacyVPNInstance struct {
	ID          uint        `json:"id" gorm:"primaryKey"`
	Protocol    VPNProtocol `json:"protocol" gorm:"size:32;index;not null"`
	Name        string      `json:"name" gorm:"size:64;uniqueIndex;not null"`
	RemoteHost  string      `json:"remote_host" gorm:"size:255;not null"`
	ListenPort  int         `json:"listen_port" gorm:"not null"`
	PoolCIDR    string      `json:"pool_cidr" gorm:"size:64;not null"`
	DNS         string      `json:"dns" gorm:"size:128"`
	Enabled     bool        `json:"enabled" gorm:"not null;default:false"`
	RuntimeMode string      `json:"runtime_mode" gorm:"size:64;not null"`
	SecretRef   string      `json:"secret_ref" gorm:"size:255"`
	CertRef     string      `json:"cert_ref" gorm:"size:255"`
	KeyRef      string      `json:"-" gorm:"size:255"`

	Users []LegacyVPNUser `json:"users,omitempty" gorm:"foreignKey:InstanceID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

// LegacyVPNUser stores PPP/CHAP user metadata for L2TP/IPsec, SSTP, and PPTP.
// Password material must be stored only through encrypted secret references.
type LegacyVPNUser struct {
	ID          uint        `json:"id" gorm:"primaryKey"`
	InstanceID  uint        `json:"instance_id" gorm:"index;not null"`
	Protocol    VPNProtocol `json:"protocol" gorm:"size:32;index;not null"`
	Name        string      `json:"name" gorm:"size:64;not null"`
	AssignedIP  string      `json:"assigned_ip" gorm:"size:64"`
	Enabled     bool        `json:"enabled" gorm:"not null;default:true"`
	PasswordRef string      `json:"-" gorm:"size:255"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

package models

import (
	"time"

	"gorm.io/gorm"
)

// Peer represents a WireGuard client attached to an interface.
type Peer struct {
	ID          uint   `json:"id" gorm:"primaryKey"`
	InterfaceID uint   `json:"interface_id" gorm:"index;not null"`
	Name        string `json:"name" gorm:"size:128;not null"`

	// PrivateKey is stored only when the server generated the keypair, so we can
	// render a complete client config / QR. It is never exposed in list/detail JSON.
	PrivateKey   string `json:"-" gorm:"size:64"`
	PublicKey    string `json:"public_key" gorm:"size:64;not null"`
	PresharedKey string `json:"-" gorm:"size:64"`

	// AllowedIPs are the IPs routed TO this peer on the server (usually its /32).
	AllowedIPs string `json:"allowed_ips" gorm:"size:255;not null"` // e.g. 10.8.0.2/32
	// AssignedIP is the tunnel address handed to the client.
	AssignedIP string `json:"assigned_ip" gorm:"size:64;not null"` // e.g. 10.8.0.2
	// ClientAllowedIPs are what the client routes through the tunnel.
	ClientAllowedIPs    string `json:"client_allowed_ips" gorm:"size:255;not null;default:'0.0.0.0/0, ::/0'"`
	PersistentKeepalive int    `json:"persistent_keepalive" gorm:"default:25"`
	Enabled             bool   `json:"enabled" gorm:"not null;default:true"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`

	// Runtime fields (populated from the kernel, not stored).
	LastHandshake string `json:"last_handshake,omitempty" gorm:"-"`
	RxBytes       int64  `json:"rx_bytes" gorm:"-"`
	TxBytes       int64  `json:"tx_bytes" gorm:"-"`
	Online        bool   `json:"online" gorm:"-"`
}

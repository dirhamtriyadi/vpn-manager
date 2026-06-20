package models

import (
	"time"

	"gorm.io/gorm"
)

// WGInterface represents a WireGuard server interface (the concentrator side).
type WGInterface struct {
	ID         uint   `json:"id" gorm:"primaryKey"`
	Name       string `json:"name" gorm:"size:32;uniqueIndex;not null"` // e.g. wg0
	PrivateKey string `json:"-" gorm:"size:64;not null"`                // base64, never exposed via JSON
	PublicKey  string `json:"public_key" gorm:"size:64;not null"`
	ListenPort int    `json:"listen_port" gorm:"not null"`
	Address    string `json:"address" gorm:"size:64;not null"` // server CIDR, e.g. 10.8.0.1/24
	DNS        string `json:"dns" gorm:"size:128"`             // DNS pushed to clients
	MTU        int    `json:"mtu" gorm:"default:1420"`
	Endpoint   string `json:"endpoint" gorm:"size:255;not null"` // public host clients dial, e.g. vpn.example.com
	Enabled    bool   `json:"enabled" gorm:"not null;default:true"`

	// Masquerade installs NAT (MASQUERADE) + forwarding rules so clients of this
	// interface reach the internet through the server's uplink. Off by default
	// because it routes client traffic out the host.
	Masquerade bool `json:"masquerade" gorm:"not null;default:false"`
	// EgressInterface is the WAN/uplink the masquerade rules use. When empty and
	// Masquerade is on, the server auto-detects the default-route interface and
	// stores the resolved name here so teardown targets the same rules.
	EgressInterface string `json:"egress_interface" gorm:"size:32"`

	Peers []Peer `json:"peers,omitempty" gorm:"foreignKey:InterfaceID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`

	// OwnerID scopes the interface to a panel user. Nil means unowned (only
	// super admins see unowned resources). Set to the creator on create.
	OwnerID *uint `json:"owner_id,omitempty" gorm:"index"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

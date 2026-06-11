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

	Peers []Peer `json:"peers,omitempty" gorm:"foreignKey:InterfaceID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

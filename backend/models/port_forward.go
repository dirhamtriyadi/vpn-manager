package models

import "time"

// PortForward exposes a port on the server's public IP and forwards it, through
// the WireGuard tunnel, to a peer's tunnel IP. The peer (e.g. a MikroTik) does
// the final hop to its own LAN device. This is the "rent a public IP" use case
// for clients behind CGNAT.
//
// TargetIP and Egress are snapshots resolved at apply time so the exact same
// iptables rules can be removed later even if the peer IP or default route
// changes.
type PortForward struct {
	ID          uint `json:"id" gorm:"primaryKey"`
	InterfaceID uint `json:"interface_id" gorm:"index;not null"`
	PeerID      uint `json:"peer_id" gorm:"index;not null"`
	// Protocol + PublicPort are unique together: the same WAN port can serve tcp
	// and udp, but not two forwards on the same protocol/port.
	Protocol   string `json:"protocol" gorm:"size:4;not null;default:tcp;uniqueIndex:idx_pf_proto_public"` // tcp | udp
	PublicPort int    `json:"public_port" gorm:"not null;uniqueIndex:idx_pf_proto_public"`
	// TargetPort is the port on the peer's side (what the peer listens on / will
	// itself DST-NAT to a LAN device).
	TargetPort int    `json:"target_port" gorm:"not null"`
	TargetIP   string `json:"target_ip" gorm:"size:64"` // resolved peer tunnel IP (snapshot)
	Egress     string `json:"egress,omitempty" gorm:"size:32"`
	Enabled    bool   `json:"enabled" gorm:"not null;default:true"`
	Comment    string `json:"comment" gorm:"size:128"`

	OwnerID *uint `json:"owner_id,omitempty" gorm:"index"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

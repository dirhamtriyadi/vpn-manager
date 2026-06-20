package dto

import "time"

type CreatePortForwardRequest struct {
	InterfaceID uint   `json:"interface_id" validate:"required"`
	PeerID      uint   `json:"peer_id" validate:"required"`
	Protocol    string `json:"protocol" validate:"required,oneof=tcp udp"`
	PublicPort  int    `json:"public_port" validate:"required,min=1,max=65535"`
	TargetPort  int    `json:"target_port" validate:"required,min=1,max=65535"`
	Comment     string `json:"comment" validate:"max=128"`
}

type UpdatePortForwardRequest struct {
	Enabled    *bool  `json:"enabled"`
	TargetPort int    `json:"target_port" validate:"omitempty,min=1,max=65535"`
	Comment    string `json:"comment" validate:"max=128"`
}

type PortForwardResponse struct {
	ID            uint      `json:"id"`
	InterfaceID   uint      `json:"interface_id"`
	InterfaceName string    `json:"interface_name"`
	PeerID        uint      `json:"peer_id"`
	PeerName      string    `json:"peer_name"`
	Protocol      string    `json:"protocol"`
	PublicPort    int       `json:"public_port"`
	TargetPort    int       `json:"target_port"`
	TargetIP      string    `json:"target_ip"`
	Egress        string    `json:"egress,omitempty"`
	Enabled       bool      `json:"enabled"`
	Comment       string    `json:"comment"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

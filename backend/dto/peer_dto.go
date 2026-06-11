package dto

// CreatePeerRequest is the payload for adding a peer.
// If PublicKey is empty the server generates a key pair (and stores the private
// key so it can render a complete client config / QR code).
// If AssignedIP is empty the server allocates the next free address.
type CreatePeerRequest struct {
	Name                string `json:"name" validate:"required,min=1,max=128" example:"laptop-andi"`
	PublicKey           string `json:"public_key" validate:"omitempty" example:""`
	AssignedIP          string `json:"assigned_ip" validate:"omitempty,ip" example:"10.8.0.2"`
	ClientAllowedIPs    string `json:"client_allowed_ips" validate:"omitempty" example:"0.0.0.0/0, ::/0"`
	PersistentKeepalive int    `json:"persistent_keepalive" validate:"omitempty,min=0,max=65535" example:"25"`
	UsePresharedKey     bool   `json:"use_preshared_key" example:"true"`
	Enabled             *bool  `json:"enabled" example:"true"`
}

// UpdatePeerRequest is the payload for updating a peer.
type UpdatePeerRequest struct {
	Name                string `json:"name" validate:"required,min=1,max=128" example:"laptop-andi"`
	ClientAllowedIPs    string `json:"client_allowed_ips" validate:"omitempty" example:"0.0.0.0/0, ::/0"`
	PersistentKeepalive int    `json:"persistent_keepalive" validate:"omitempty,min=0,max=65535" example:"25"`
	Enabled             *bool  `json:"enabled" example:"true"`
}

package dto

// CreateInterfaceRequest is the payload for creating a WireGuard interface.
// If PrivateKey is empty the server generates a fresh key pair.
type CreateInterfaceRequest struct {
	Name       string `json:"name" validate:"required,min=2,max=32" example:"wg0"`
	PrivateKey string `json:"private_key" validate:"omitempty" example:""`
	ListenPort int    `json:"listen_port" validate:"required,min=1,max=65535" example:"51820"`
	Address    string `json:"address" validate:"required,cidr" example:"10.8.0.1/24"`
	Endpoint   string `json:"endpoint" validate:"required,min=3,max=255" example:"vpn.example.com"`
	DNS        string `json:"dns" validate:"omitempty,max=128" example:"1.1.1.1"`
	MTU        int    `json:"mtu" validate:"omitempty,min=576,max=9000" example:"1420"`
	Enabled    *bool  `json:"enabled" example:"true"`
}

// UpdateInterfaceRequest is the payload for updating an interface.
type UpdateInterfaceRequest struct {
	ListenPort int    `json:"listen_port" validate:"required,min=1,max=65535" example:"51820"`
	Address    string `json:"address" validate:"required,cidr" example:"10.8.0.1/24"`
	Endpoint   string `json:"endpoint" validate:"required,min=3,max=255" example:"vpn.example.com"`
	DNS        string `json:"dns" validate:"omitempty,max=128" example:"1.1.1.1"`
	MTU        int    `json:"mtu" validate:"omitempty,min=576,max=9000" example:"1420"`
	Enabled    *bool  `json:"enabled" example:"true"`
}

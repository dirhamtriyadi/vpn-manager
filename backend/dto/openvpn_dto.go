package dto

type OpenVPNRoadmapResponse struct {
	Available           bool     `json:"available"`
	Status              string   `json:"status"`
	RuntimeMode         string   `json:"runtime_mode"`
	SecretStorageStatus string   `json:"secret_storage_status"`
	ManifestStatus      string   `json:"manifest_status"`
	NextSteps           []string `json:"next_steps"`
	BlockedMessage      string   `json:"blocked_message"`
}

type OpenVPNClientProfilePreviewRequest struct {
	ClientName    string `json:"client_name" validate:"required"`
	RemoteHost    string `json:"remote_host" validate:"required"`
	RemotePort    int    `json:"remote_port" validate:"required,min=1,max=65535"`
	Protocol      string `json:"protocol" validate:"omitempty"`
	CACertPEM     string `json:"ca_cert_pem" validate:"required"`
	ClientCertPEM string `json:"client_cert_pem" validate:"required"`
	ClientKeyPEM  string `json:"client_key_pem" validate:"required"`
	TLSAuthPEM    string `json:"tls_auth_pem" validate:"omitempty"`
}

type OpenVPNClientProfilePreviewResponse struct {
	Filename string `json:"filename"`
	Profile  string `json:"profile"`
}

type CreateOpenVPNInstanceDraftRequest struct {
	Name          string `json:"name" validate:"required"`
	RemoteHost    string `json:"remote_host" validate:"required"`
	ListenPort    int    `json:"listen_port" validate:"omitempty,gte=1,lte=65535"`
	Protocol      string `json:"protocol" validate:"omitempty,oneof=udp udp4 udp6 tcp tcp4 tcp6"`
	TunnelCIDR    string `json:"tunnel_cidr" validate:"required,cidr"`
	DNS           string `json:"dns"`
	CACertPEM     string `json:"ca_cert_pem" validate:"required"`
	ServerCertPEM string `json:"server_cert_pem" validate:"required"`
	ServerKeyPEM  string `json:"server_key_pem" validate:"required"`
	TLSCryptPEM   string `json:"tls_crypt_pem"`
}

type OpenVPNInstanceDraftResponse struct {
	ID                  uint              `json:"id"`
	Name                string            `json:"name"`
	RemoteHost          string            `json:"remote_host"`
	ListenPort          int               `json:"listen_port"`
	Protocol            string            `json:"protocol"`
	TunnelCIDR          string            `json:"tunnel_cidr"`
	DNS                 string            `json:"dns"`
	Enabled             bool              `json:"enabled"`
	RuntimeMode         string            `json:"runtime_mode"`
	SecretStorageStatus string            `json:"secret_storage_status"`
	SecretRefs          map[string]string `json:"secret_refs"`
}

type OpenVPNRuntimeManifestResponse struct {
	ID               uint     `json:"id"`
	InstanceID       uint     `json:"instance_id"`
	RuntimeMode      string   `json:"runtime_mode"`
	ServerConf       string   `json:"server_conf"`
	ComposeYAML      string   `json:"compose_yaml"`
	Warnings         []string `json:"warnings"`
	GenerationStatus string   `json:"generation_status"`
}

type OpenVPNRuntimeManifestPreviewRequest struct {
	InstanceName string `json:"instance_name" validate:"required"`
	RemoteHost   string `json:"remote_host" validate:"required"`
	ListenPort   int    `json:"listen_port" validate:"omitempty,min=1,max=65535"`
	Protocol     string `json:"protocol" validate:"omitempty"`
	TunnelCIDR   string `json:"tunnel_cidr" validate:"required,cidr"`
	DNS          string `json:"dns" validate:"omitempty"`
}

type OpenVPNRuntimeManifestPreviewResponse struct {
	RuntimeMode string            `json:"runtime_mode"`
	Files       map[string]string `json:"files"`
	Warnings    []string          `json:"warnings"`
}

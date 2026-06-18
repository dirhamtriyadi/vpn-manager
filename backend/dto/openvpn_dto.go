package dto

type OpenVPNRoadmapResponse struct {
	Available           bool     `json:"available"`
	Status              string   `json:"status"`
	RuntimeMode         string   `json:"runtime_mode"`
	SecretStorageStatus string   `json:"secret_storage_status"`
	ManifestStatus      string   `json:"manifest_status"`
	LifecycleStatus     string   `json:"lifecycle_status"`
	StatusParserStatus  string   `json:"status_parser_status"`
	FirewallStatus      string   `json:"firewall_status"`
	UserStorageStatus   string   `json:"user_storage_status"`
	RuntimeExecution    string   `json:"runtime_execution"`
	FirewallApply       string   `json:"firewall_apply"`
	HostVerification    string   `json:"host_verification"`
	EnablementReady     bool     `json:"enablement_ready"`
	EnablementBlockers  []string `json:"enablement_blockers"`
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

type OpenVPNUserDraftRequest struct {
	Name          string `json:"name" validate:"required"`
	AssignedIP    string `json:"assigned_ip" validate:"omitempty,ip"`
	ClientCertPEM string `json:"client_cert_pem" validate:"required"`
	ClientKeyPEM  string `json:"client_key_pem" validate:"required"`
}

type OpenVPNUserDraftResponse struct {
	ID                  uint              `json:"id"`
	InstanceID          uint              `json:"instance_id"`
	Name                string            `json:"name"`
	AssignedIP          string            `json:"assigned_ip"`
	Enabled             bool              `json:"enabled"`
	SecretStorageStatus string            `json:"secret_storage_status"`
	SecretRefs          map[string]string `json:"secret_refs"`
}

type OpenVPNLifecyclePlanResponse struct {
	Action        string   `json:"action"`
	ExecutionMode string   `json:"execution_mode"`
	Status        string   `json:"status"`
	ProjectName   string   `json:"project_name"`
	ContainerName string   `json:"container_name"`
	Commands      []string `json:"commands"`
	Warnings      []string `json:"warnings"`
}

type OpenVPNFirewallPlanResponse struct {
	Status        string   `json:"status"`
	OwnershipKey  string   `json:"ownership_key"`
	Rules         []string `json:"rules"`
	TeardownRules []string `json:"teardown_rules"`
	Warnings      []string `json:"warnings"`
}

type OpenVPNStatusPreviewRequest struct {
	RawStatus string `json:"raw_status"`
}

type OpenVPNStatusResponse struct {
	State   string                    `json:"state"`
	Clients []OpenVPNStatusClientInfo `json:"clients"`
	Raw     string                    `json:"raw,omitempty"`
}

type OpenVPNStatusClientInfo struct {
	CommonName     string `json:"common_name"`
	RealAddress    string `json:"real_address"`
	VirtualAddress string `json:"virtual_address"`
	BytesReceived  int64  `json:"bytes_received"`
	BytesSent      int64  `json:"bytes_sent"`
	ConnectedSince string `json:"connected_since"`
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

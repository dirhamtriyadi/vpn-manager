package openvpn

import (
	"fmt"
	"net"
	"strings"

	"github.com/example/vpn-manager/models"
	"github.com/example/vpn-manager/secrets"
)

type InstanceDraftInput struct {
	Name          string
	RemoteHost    string
	ListenPort    int
	Protocol      string
	TunnelCIDR    string
	DNS           string
	CACertPEM     string
	ServerCertPEM string
	ServerKeyPEM  string
	TLSCryptPEM   string
}

type InstanceDraft struct {
	Instance models.OpenVPNInstance
	Secrets  []models.EncryptedSecret
}

func BuildInstanceDraft(input InstanceDraftInput, envelope *secrets.Envelope) (InstanceDraft, error) {
	if envelope == nil {
		return InstanceDraft{}, fmt.Errorf("secret envelope is required")
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return InstanceDraft{}, fmt.Errorf("instance name is required")
	}
	remoteHost := strings.TrimSpace(input.RemoteHost)
	if remoteHost == "" {
		return InstanceDraft{}, fmt.Errorf("remote host is required")
	}
	if _, _, err := net.ParseCIDR(strings.TrimSpace(input.TunnelCIDR)); err != nil {
		return InstanceDraft{}, fmt.Errorf("tunnel cidr must be a valid CIDR")
	}
	if strings.TrimSpace(input.CACertPEM) == "" {
		return InstanceDraft{}, fmt.Errorf("ca certificate is required")
	}
	if strings.TrimSpace(input.ServerCertPEM) == "" {
		return InstanceDraft{}, fmt.Errorf("server certificate is required")
	}
	if strings.TrimSpace(input.ServerKeyPEM) == "" {
		return InstanceDraft{}, fmt.Errorf("server private key is required")
	}

	port := input.ListenPort
	if port == 0 {
		port = 1194
	}
	if port < 1 || port > 65535 {
		return InstanceDraft{}, fmt.Errorf("listen port must be between 1 and 65535")
	}

	instance := models.OpenVPNInstance{
		Name:        name,
		RemoteHost:  remoteHost,
		ListenPort:  port,
		Protocol:    normalizeProtocol(input.Protocol),
		TunnelCIDR:  strings.TrimSpace(input.TunnelCIDR),
		DNS:         strings.TrimSpace(input.DNS),
		Enabled:     false,
		RuntimeMode: "container_openvpn_preview",
	}

	scope := "openvpn-" + name
	secretInputs := []struct {
		Name      string
		Plaintext string
		AssignRef func(string)
	}{
		{Name: "ca-cert-pem", Plaintext: input.CACertPEM, AssignRef: func(ref string) { instance.CARef = ref }},
		{Name: "server-cert-pem", Plaintext: input.ServerCertPEM, AssignRef: func(ref string) { instance.ServerCertRef = ref }},
		{Name: "server-key-pem", Plaintext: input.ServerKeyPEM, AssignRef: func(ref string) { instance.ServerKeyRef = ref }},
	}
	if strings.TrimSpace(input.TLSCryptPEM) != "" {
		secretInputs = append(secretInputs, struct {
			Name      string
			Plaintext string
			AssignRef func(string)
		}{Name: "tls-crypt-pem", Plaintext: input.TLSCryptPEM, AssignRef: func(ref string) { instance.TLSCryptRef = ref }})
	}

	encrypted := make([]models.EncryptedSecret, 0, len(secretInputs))
	for _, secretInput := range secretInputs {
		sealed, err := envelope.Encrypt(strings.TrimSpace(secretInput.Plaintext))
		if err != nil {
			return InstanceDraft{}, err
		}
		ref := secrets.BuildRef(scope, 0, secretInput.Name)
		secretInput.AssignRef(ref)
		encrypted = append(encrypted, models.EncryptedSecret{
			Ref:        ref,
			Scope:      "openvpn",
			OwnerType:  "openvpn_instance",
			OwnerID:    0,
			Name:       secretInput.Name,
			Algorithm:  sealed.Algorithm,
			Nonce:      sealed.Nonce,
			Ciphertext: sealed.Ciphertext,
			KeyVersion: "v1",
		})
	}

	return InstanceDraft{Instance: instance, Secrets: encrypted}, nil
}

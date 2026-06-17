package openvpn

import (
	"fmt"
	"strings"

	"github.com/example/wg-panel/models"
)

func BuildRuntimeManifestRecord(instance models.OpenVPNInstance) (models.OpenVPNRuntimeManifest, error) {
	if instance.ID == 0 {
		return models.OpenVPNRuntimeManifest{}, fmt.Errorf("saved OpenVPN instance is required")
	}
	manifest, err := BuildContainerRuntimeManifest(RuntimeManifestInput{
		InstanceName: instance.Name,
		RemoteHost:   instance.RemoteHost,
		ListenPort:   instance.ListenPort,
		Protocol:     instance.Protocol,
		TunnelCIDR:   instance.TunnelCIDR,
		DNS:          instance.DNS,
	})
	if err != nil {
		return models.OpenVPNRuntimeManifest{}, err
	}

	return models.OpenVPNRuntimeManifest{
		InstanceID:       instance.ID,
		RuntimeMode:      manifest.RuntimeMode,
		ServerConf:       manifest.Files["server.conf"],
		ComposeYAML:      manifest.Files["docker-compose.yml"],
		Warnings:         strings.Join(manifest.Warnings, "\n"),
		GenerationStatus: "generated",
	}, nil
}

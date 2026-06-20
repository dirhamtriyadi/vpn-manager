package vpn

import (
	"errors"
	"testing"

	"github.com/example/vpn-manager/models"
)

type fakeDriver struct {
	protocol models.VPNProtocol
}

func (d fakeDriver) Protocol() models.VPNProtocol { return d.protocol }
func (d fakeDriver) Capabilities() ProtocolCapabilities {
	return ProtocolCapabilities{RuntimeStrategy: "test", ConfigDownload: true}
}
func (d fakeDriver) Status(instanceID uint) (InstanceStatus, error) {
	return InstanceStatus{Protocol: d.protocol, Up: true}, nil
}
func (d fakeDriver) Sync(instanceID uint) error { return nil }
func (d fakeDriver) GenerateUserConfig(userID uint) ([]byte, string, error) {
	return []byte("config"), "client.conf", nil
}

type badDriver struct{}

func (d badDriver) Protocol() models.VPNProtocol { return models.ProtocolWireGuard }
func (d badDriver) Capabilities() ProtocolCapabilities { return ProtocolCapabilities{} }
func (d badDriver) Status(instanceID uint) (InstanceStatus, error) {
	return InstanceStatus{}, errors.New("boom")
}
func (d badDriver) Sync(instanceID uint) error { return errors.New("boom") }
func (d badDriver) GenerateUserConfig(userID uint) ([]byte, string, error) {
	return nil, "", errors.New("boom")
}

func TestRegistryRegistersAndRetrievesDriver(t *testing.T) {
	registry := NewRegistry()
	driver := fakeDriver{protocol: models.ProtocolWireGuard}
	if err := registry.Register(driver); err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	actual, ok := registry.Get(models.ProtocolWireGuard)
	if !ok {
		t.Fatal("expected registered driver to be found")
	}
	if actual.Protocol() != models.ProtocolWireGuard {
		t.Fatalf("driver protocol = %q, want %q", actual.Protocol(), models.ProtocolWireGuard)
	}
}

func TestRegistryRejectsDuplicateDriver(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(fakeDriver{protocol: models.ProtocolWireGuard}); err != nil {
		t.Fatalf("first Register returned error: %v", err)
	}
	if err := registry.Register(badDriver{}); err == nil {
		t.Fatal("expected duplicate driver registration to fail")
	}
}

func TestRegistryRejectsNilDriver(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(nil); err == nil {
		t.Fatal("expected nil driver registration to fail")
	}
}

func TestNewDefaultRegistryIncludesWireGuardDriver(t *testing.T) {
	registry, err := NewDefaultRegistry()
	if err != nil {
		t.Fatalf("NewDefaultRegistry returned error: %v", err)
	}
	if !registry.Supports(models.ProtocolWireGuard) {
		t.Fatal("expected default registry to support WireGuard")
	}
	driver, ok := registry.Get(models.ProtocolWireGuard)
	if !ok {
		t.Fatal("expected default registry to return WireGuard driver")
	}
	capabilities := driver.Capabilities()
	if capabilities.RuntimeStrategy != "host_kernel_netlink" {
		t.Fatalf("runtime strategy = %q, want host_kernel_netlink", capabilities.RuntimeStrategy)
	}
	if !capabilities.ConfigDownload || !capabilities.QRCode {
		t.Fatalf("expected WireGuard config download and QR support: %+v", capabilities)
	}
	if registry.Supports(models.ProtocolOpenVPN) {
		t.Fatal("expected OpenVPN to stay unavailable in Phase 1")
	}
}

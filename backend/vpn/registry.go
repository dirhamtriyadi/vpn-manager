package vpn

import (
	"fmt"

	"github.com/example/vpn-manager/models"
)

type Registry struct {
	drivers map[models.VPNProtocol]Driver
}

func NewRegistry() *Registry {
	return &Registry{drivers: map[models.VPNProtocol]Driver{}}
}

func NewDefaultRegistry() (*Registry, error) {
	registry := NewRegistry()
	if err := registry.Register(NewWireGuardDriver()); err != nil {
		return nil, err
	}
	return registry, nil
}

func (r *Registry) Register(driver Driver) error {
	if driver == nil {
		return fmt.Errorf("driver is nil")
	}
	protocol := driver.Protocol()
	if _, exists := r.drivers[protocol]; exists {
		return fmt.Errorf("driver already registered for protocol %s", protocol)
	}
	r.drivers[protocol] = driver
	return nil
}

func (r *Registry) Get(protocol models.VPNProtocol) (Driver, bool) {
	driver, ok := r.drivers[protocol]
	return driver, ok
}

func (r *Registry) Supports(protocol models.VPNProtocol) bool {
	_, ok := r.Get(protocol)
	return ok
}

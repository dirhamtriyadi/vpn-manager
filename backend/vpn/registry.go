package vpn

import (
	"fmt"

	"github.com/example/wg-panel/models"
)

type Registry struct {
	drivers map[models.VPNProtocol]Driver
}

func NewRegistry() *Registry {
	return &Registry{drivers: map[models.VPNProtocol]Driver{}}
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

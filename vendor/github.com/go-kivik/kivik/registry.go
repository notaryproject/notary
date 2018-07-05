package kivik

import (
	"sync"

	"github.com/go-kivik/kivik/driver"
)

var (
	driversMu sync.RWMutex
	drivers   = make(map[string]driver.Driver)
)

// Register makes a database driver available by the provided name. If Register
// is called twice with the same name or if driver is nil, it panics.
func Register(name string, driver driver.Driver) {
	driversMu.Lock()
	defer driversMu.Unlock()
	if driver == nil {
		panic("kivik: Register driver is nil")
	}
	if _, dup := drivers[name]; dup {
		panic("kivk: Register called twice for driver " + name)
	}
	drivers[name] = driver
}

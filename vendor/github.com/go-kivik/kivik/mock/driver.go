package mock

import (
	"context"

	"github.com/go-kivik/kivik/driver"
)

// Driver mocks a Kivik Driver.
type Driver struct {
	NewClientFunc func(ctx context.Context, name string) (driver.Client, error)
}

var _ driver.Driver = &Driver{}

// NewClient calls d.NewClientFunc
func (d *Driver) NewClient(ctx context.Context, name string) (driver.Client, error) {
	return d.NewClientFunc(ctx, name)
}

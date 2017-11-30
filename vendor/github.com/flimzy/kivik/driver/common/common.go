// Package common contains logic and data structures shared by various kivik
// backends. It is not meant to be used directly.
package common

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/flimzy/kivik/driver"
)

// Client is a shared core between Kivik backends.
type Client struct {
	version *driver.Version
}

// NewClient returns a new client core.
func NewClient(version, vendor string) *Client {
	return &Client{
		version: &driver.Version{
			Version:     version,
			Vendor:      vendor,
			RawResponse: json.RawMessage(fmt.Sprintf(`{"version":"%s","vendor":{"name":"%s"}}`, version, vendor)),
		},
	}
}

// Version returns the configured server info.
func (c *Client) Version(_ context.Context) (*driver.Version, error) {
	return c.version, nil
}

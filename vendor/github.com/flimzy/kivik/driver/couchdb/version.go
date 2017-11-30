package couchdb

import (
	"context"
	"encoding/json"

	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/driver"
)

// Version returns the server's version info.
func (c *client) Version(ctx context.Context) (*driver.Version, error) {
	i := &info{}
	_, err := c.DoJSON(ctx, kivik.MethodGet, "/", nil, i)
	return &driver.Version{
		Version:     i.Version,
		Vendor:      i.Vendor.Name,
		RawResponse: i.Data,
	}, err
}

type info struct {
	Data    json.RawMessage
	Version string `json:"version"`
	Vendor  struct {
		Name string `json:"name"`
	} `json:"vendor"`
}

func (i *info) UnmarshalJSON(data []byte) error {
	type alias info
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	i.Data = data
	i.Version = a.Version
	i.Vendor = a.Vendor
	return nil
}

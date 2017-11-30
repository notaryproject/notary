package couchdb

import (
	"context"
	"encoding/json"
	"io"

	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/driver"
	"github.com/flimzy/kivik/driver/couchdb/chttp"
)

// Changes returns the changes stream for the database.
func (d *db) Changes(ctx context.Context, opts map[string]interface{}) (driver.Changes, error) {
	overrideOpts := map[string]interface{}{
		"feed":      "continuous",
		"since":     "now",
		"heartbeat": 6000,
	}
	options, err := optionsToParams(opts, overrideOpts)
	if err != nil {
		return nil, err
	}
	resp, err := d.Client.DoReq(ctx, kivik.MethodGet, d.path("_changes", options), nil)
	if err != nil {
		return nil, err
	}
	if err = chttp.ResponseError(resp); err != nil {
		return nil, err
	}
	return newChangesRows(resp.Body), nil
}

type changesRows struct {
	body   io.ReadCloser
	dec    *json.Decoder
	closed bool
}

func newChangesRows(r io.ReadCloser) *changesRows {
	return &changesRows{
		body: r,
	}
}

var _ driver.Changes = &changesRows{}

func (r *changesRows) Close() error {
	return r.body.Close()
}

func (r *changesRows) Next(row *driver.Change) error {
	if r.closed {
		return io.EOF
	}
	if r.dec == nil {
		r.dec = json.NewDecoder(r.body)
	}
	if !r.dec.More() {
		return io.EOF
	}

	return r.dec.Decode(row)
}

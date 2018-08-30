package couchdb

import (
	"context"
	"encoding/json"
	"io"

	"github.com/go-kivik/couchdb/chttp"
	"github.com/go-kivik/kivik"
	"github.com/go-kivik/kivik/driver"
	"github.com/go-kivik/kivik/errors"
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
	body io.ReadCloser
	dec  *json.Decoder
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
	if r.dec == nil {
		r.dec = json.NewDecoder(r.body)
	}
	if !r.dec.More() {
		return io.EOF
	}

	return errors.WrapStatus(kivik.StatusBadResponse, r.dec.Decode(row))
}

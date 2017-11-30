package couchdb

import (
	"context"
	"encoding/json"
	"io"

	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/driver"
	"github.com/flimzy/kivik/driver/couchdb/chttp"
)

type couchUpdates struct {
	body io.ReadCloser
	dec  *json.Decoder
}

var _ driver.DBUpdates = &couchUpdates{}

func (u *couchUpdates) Next(update *driver.DBUpdate) error {
	return u.dec.Decode(update)
}

func (u *couchUpdates) Close() error {
	return u.body.Close()
}

func (c *client) DBUpdates() (updates driver.DBUpdates, err error) {
	resp, err := c.DoReq(context.Background(), kivik.MethodGet, "/_db_updates?feed=continuous&since=now", nil)
	if err != nil {
		return nil, err
	}
	if err := chttp.ResponseError(resp); err != nil {
		return nil, err
	}
	return &couchUpdates{
		body: resp.Body,
		dec:  json.NewDecoder(resp.Body),
	}, nil
}

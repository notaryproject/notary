package couchdb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/driver"
	"github.com/flimzy/kivik/driver/couchdb/chttp"
	"github.com/flimzy/kivik/driver/util"
	"github.com/flimzy/kivik/errors"
)

// deJSONify unmarshals a string, []byte, or json.RawMessage. All other types
// are returned as-is.
func deJSONify(i interface{}) (interface{}, error) {
	var data []byte
	switch t := i.(type) {
	case string:
		data = []byte(t)
	case []byte:
		data = t
	case json.RawMessage:
		data = []byte(t)
	default:
		return i, nil
	}
	var x interface{}
	err := json.Unmarshal(data, &x)
	return x, errors.WrapStatus(kivik.StatusBadRequest, err)
}

var findNotImplemented = errors.Status(kivik.StatusNotImplemented, "kivik: Find interface not implemented prior to CouchDB 2.0.0")

func (d *db) CreateIndex(ctx context.Context, ddoc, name string, index interface{}) error {
	if d.client.Compat == CompatCouch16 {
		return findNotImplemented
	}
	indexObj, err := deJSONify(index)
	if err != nil {
		return err
	}
	parameters := struct {
		Index interface{} `json:"index"`
		Ddoc  string      `json:"ddoc,omitempty"`
		Name  string      `json:"name,omitempty"`
	}{
		Index: indexObj,
		Ddoc:  ddoc,
		Name:  name,
	}
	body := &bytes.Buffer{}
	if err = json.NewEncoder(body).Encode(parameters); err != nil {
		return errors.WrapStatus(kivik.StatusBadRequest, err)
	}
	_, err = d.Client.DoError(ctx, kivik.MethodPost, d.path("_index", nil), &chttp.Options{Body: body})
	return err
}

func (d *db) GetIndexes(ctx context.Context) ([]driver.Index, error) {
	if d.client.Compat == CompatCouch16 {
		return nil, findNotImplemented
	}
	var result struct {
		Indexes []driver.Index `json:"indexes"`
	}
	_, err := d.Client.DoJSON(ctx, kivik.MethodGet, d.path("_index", nil), nil, &result)
	return result.Indexes, err
}

func (d *db) DeleteIndex(ctx context.Context, ddoc, name string) error {
	path := fmt.Sprintf("_index/%s/json/%s", ddoc, name)
	_, err := d.Client.DoError(ctx, kivik.MethodDelete, d.path(path, nil), nil)
	return err
}

func (d *db) Find(ctx context.Context, query interface{}) (driver.Rows, error) {
	if d.client.Compat == CompatCouch16 {
		return nil, findNotImplemented
	}
	body, err := util.ToJSON(query)
	if err != nil {
		return nil, err
	}
	resp, err := d.Client.DoReq(ctx, kivik.MethodPost, d.path("_find", nil), &chttp.Options{Body: body})
	if err != nil {
		return nil, err
	}
	if err = chttp.ResponseError(resp); err != nil {
		return nil, err
	}
	return newRows(resp.Body), nil
}

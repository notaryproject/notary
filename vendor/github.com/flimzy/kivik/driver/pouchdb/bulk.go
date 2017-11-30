package pouchdb

import (
	"context"
	"fmt"
	"io"

	"github.com/flimzy/kivik/driver"
	"github.com/flimzy/kivik/errors"
	"github.com/gopherjs/gopherjs/js"
)

type bulkResult struct {
	*js.Object
	OK         bool   `js:"ok"`
	ID         string `js:"id"`
	Rev        string `js:"rev"`
	Error      string `js:"name"`
	StatusCode int    `js:"status"`
	Reason     string `js:"message"`
	IsError    bool   `js:"error"`
}

type bulkResults struct {
	results *js.Object
}

var _ driver.BulkResults = &bulkResults{}

func (r *bulkResults) Next(update *driver.BulkResult) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("%v", r)
			}
		}
	}()
	if r.results == js.Undefined || r.results.Length() == 0 {
		return io.EOF
	}
	result := &bulkResult{}
	result.Object = r.results.Call("shift")
	update.ID = result.ID
	update.Rev = result.ID
	update.Error = nil
	if result.IsError {
		update.Error = errors.Status(result.StatusCode, result.Reason)
	}
	return nil
}

func (r *bulkResults) Close() error {
	r.results = nil // Free up memory used by any remaining rows
	return nil
}

func (d *db) BulkDocs(ctx context.Context, docs []interface{}) (driver.BulkResults, error) {
	result, err := d.db.BulkDocs(ctx, docs...)
	if err != nil {
		return nil, err
	}
	return &bulkResults{results: result}, nil
}

package couchdb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/driver"
	"github.com/flimzy/kivik/errors"
	"github.com/go-kivik/couchdb/chttp"
)

type bulkResults struct {
	body io.ReadCloser
	dec  *json.Decoder
}

var _ driver.BulkResults = &bulkResults{}

func newBulkResults(body io.ReadCloser) (*bulkResults, error) {
	dec := json.NewDecoder(body)
	// Consume the opening '[' char
	if err := consumeDelim(dec, json.Delim('[')); err != nil {
		return nil, err
	}
	return &bulkResults{
		body: body,
		dec:  dec,
	}, nil
}

func (r *bulkResults) Next(update *driver.BulkResult) error {
	if !r.dec.More() {
		if err := consumeDelim(r.dec, json.Delim(']')); err != nil {
			return err
		}
		return io.EOF
	}
	var updateResult struct {
		ID     string `json:"id"`
		Rev    string `json:"rev"`
		Error  string `json:"error"`
		Reason string `json:"reason"`
	}
	if err := r.dec.Decode(&updateResult); err != nil {
		return errors.WrapStatus(kivik.StatusBadResponse, err)
	}
	update.ID = updateResult.ID
	update.Rev = updateResult.Rev
	update.Error = nil
	if updateResult.Error != "" {
		var status int
		switch updateResult.Error {
		case "conflict":
			status = kivik.StatusConflict
		default:
			status = 600 // Unknown error
		}
		update.Error = errors.Status(status, updateResult.Reason)
	}
	return nil
}

func (r *bulkResults) Close() error {
	return r.body.Close()
}

func (d *db) BulkDocs(ctx context.Context, docs []interface{}, options map[string]interface{}) (driver.BulkResults, error) {
	if options == nil {
		options = make(map[string]interface{})
	}
	fullCommit, err := fullCommit(d.fullCommit, options)
	if err != nil {
		return nil, err
	}
	options["docs"] = docs
	opts := &chttp.Options{
		Body:       chttp.EncodeBody(options),
		FullCommit: fullCommit,
	}
	resp, err := d.Client.DoReq(ctx, kivik.MethodPost, d.path("_bulk_docs", nil), opts)
	if err != nil {
		return nil, err
	}
	switch resp.StatusCode {
	case kivik.StatusCreated:
		// Nothing to do
	case kivik.StatusExpectationFailed:
		err = &chttp.HTTPError{
			Code:   kivik.StatusExpectationFailed,
			Reason: "one or more document was rejected",
		}
	default:
		if resp.StatusCode < 400 {
			fmt.Printf("Unexpected BulkDoc response code: %d\n", resp.StatusCode)
		}
		// All other errors can consume the response body and return immediately
		if e := chttp.ResponseError(resp); e != nil {
			return nil, e
		}
	}
	results, bulkErr := newBulkResults(resp.Body)
	if bulkErr != nil {
		return nil, bulkErr
	}
	return results, err
}

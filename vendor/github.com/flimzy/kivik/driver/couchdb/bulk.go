package couchdb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/driver"
	"github.com/flimzy/kivik/driver/couchdb/chttp"
	"github.com/flimzy/kivik/errors"
)

type bulkResults struct {
	body io.ReadCloser
	dec  *json.Decoder
}

var _ driver.BulkResults = &bulkResults{}

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
	err := r.dec.Decode(&updateResult)
	if err != nil {
		return err
	}
	update.ID = updateResult.ID
	update.Rev = updateResult.Rev
	update.Error = nil
	if updateResult.Error != "" {
		var status int
		switch updateResult.Error {
		case "conflict":
			status = kivik.StatusConflict
		case "not_implemented":
			status = kivik.StatusNotImplemented
		default:
			fmt.Printf("Unknown error %s / %s \n", updateResult.Error, updateResult.Reason)
		}
		update.Error = errors.Status(status, updateResult.Reason)
	}
	return nil
}

func (r *bulkResults) Close() error {
	return r.body.Close()
}

func (d *db) BulkDocs(ctx context.Context, docs []interface{}) (driver.BulkResults, error) {
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	defer cancel()
	body, errFunc := chttp.EncodeBody(map[string]interface{}{"docs": docs}, cancel)
	opts := &chttp.Options{
		Body:        body,
		ForceCommit: d.forceCommit,
	}
	resp, err := d.Client.DoReq(ctx, kivik.MethodPost, d.path("_bulk_docs", nil), opts)
	if jsonErr := errFunc(); jsonErr != nil {
		if resp.Body != nil {
			resp.Body.Close()
		}
		return nil, jsonErr
	}
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
			fmt.Printf("Unexpected BulkDoc response code: %d", resp.StatusCode)
		}
		// All other errors can consume the response body and return immediately
		return nil, chttp.ResponseError(resp)
	}
	dec := json.NewDecoder(resp.Body)
	// Consume the opening '[' char
	if jsonErr := consumeDelim(dec, json.Delim('[')); jsonErr != nil {
		return nil, jsonErr
	}
	return &bulkResults{
		body: resp.Body,
		dec:  dec,
	}, err
}

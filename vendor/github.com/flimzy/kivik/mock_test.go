package kivik

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"strings"

	"github.com/flimzy/kivik/driver"
)

type mockDB struct {
	driver.DB
}

var _ driver.DB = &mockDB{}

type mockExplainer struct {
	driver.DB
	plan *driver.QueryPlan
	err  error
}

var _ driver.Explainer = &mockExplainer{}

func (db *mockExplainer) Explain(_ context.Context, query interface{}) (*driver.QueryPlan, error) {
	return db.plan, db.err
}

type errReader string

var _ io.ReadCloser = errReader("")

func (r errReader) Close() error {
	return nil
}

func (r errReader) Read(_ []byte) (int, error) {
	return 0, errors.New(string(r))
}

type mockBulkResults struct {
	result *driver.BulkResult
	err    error
}

var _ driver.BulkResults = &mockBulkResults{}

func (r *mockBulkResults) Next(i *driver.BulkResult) error {
	if r.result != nil {
		*i = *r.result
	}
	return r.err
}

func (r *mockBulkResults) Close() error { return nil }

func body(s string) io.ReadCloser {
	return ioutil.NopCloser(strings.NewReader(s))
}

package mock

import "github.com/go-kivik/kivik/driver"

// BulkResults mocks driver.BulkResults
type BulkResults struct {
	// ID identifies a specific BulkResults instance
	ID        string
	NextFunc  func(*driver.BulkResult) error
	CloseFunc func() error
}

var _ driver.BulkResults = &BulkResults{}

// Next calls r.NextFunc
func (r *BulkResults) Next(result *driver.BulkResult) error {
	return r.NextFunc(result)
}

// Close calls r.CloseFunc
func (r *BulkResults) Close() error {
	return r.CloseFunc()
}

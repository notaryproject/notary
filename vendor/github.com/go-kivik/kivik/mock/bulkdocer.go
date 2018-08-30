package mock

import (
	"context"

	"github.com/go-kivik/kivik/driver"
)

// BulkDocer mocks a driver.DB and driver.BulkDocer
type BulkDocer struct {
	*DB
	BulkDocsFunc func(ctx context.Context, docs []interface{}, options map[string]interface{}) (driver.BulkResults, error)
}

var _ driver.BulkDocer = &BulkDocer{}

// BulkDocs calls db.BulkDocsFunc
func (db *BulkDocer) BulkDocs(ctx context.Context, docs []interface{}, options map[string]interface{}) (driver.BulkResults, error) {
	return db.BulkDocsFunc(ctx, docs, options)
}

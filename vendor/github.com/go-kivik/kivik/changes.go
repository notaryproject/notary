package kivik

import (
	"context"

	"github.com/go-kivik/kivik/driver"
)

// Changes is an iterator over the database changes feed.
type Changes struct {
	*iter
	changesi driver.Changes
}

// Next prepares the next result value for reading. It returns true on success
// or false if there are no more results, due to an error or the changes feed
// having been closed. Err should be consulted to determine any error.
func (c *Changes) Next() bool {
	return c.iter.Next()
}

// Err returns the error, if any, that was encountered during iteration. Err may
// be called after an explicit or implicit Close.
func (c *Changes) Err() error {
	return c.iter.Err()
}

// Close closes the Changes feed, preventing further enumeration, and freeing
// any resources (such as the http request body) of the underlying query. If
// Next is called and there are no further results, Changes is closed
// automatically and it will suffice to check the result of Err. Close is
// idempotent and does not affect the result of Err.
func (c *Changes) Close() error {
	return c.iter.Close()
}

type changesIterator struct{ driver.Changes }

var _ iterator = &changesIterator{}

func (c *changesIterator) Next(i interface{}) error { return c.Changes.Next(i.(*driver.Change)) }

func newChanges(ctx context.Context, changesi driver.Changes) *Changes {
	return &Changes{
		iter:     newIterator(ctx, &changesIterator{changesi}, &driver.Change{}),
		changesi: changesi,
	}
}

// Changes returns a list of changed revs.
func (c *Changes) Changes() []string {
	return c.curVal.(*driver.Change).Changes
}

// Deleted returns true if the change relates to a deleted document.
func (c *Changes) Deleted() bool {
	return c.curVal.(*driver.Change).Deleted
}

// ID returns the ID of the current result.
func (c *Changes) ID() string {
	return c.curVal.(*driver.Change).ID
}

// ScanDoc works the same as ScanValue, but on the doc field of the result. It
// is only valid for results that include documents.
func (c *Changes) ScanDoc(dest interface{}) error {
	runlock, err := c.rlock()
	if err != nil {
		return err
	}
	defer runlock()
	return scan(dest, c.curVal.(*driver.Change).Doc)
}

// Changes returns an iterator over the real-time changes feed. The feed remains
// open until explicitly closed, or an error is encountered.
// See http://couchdb.readthedocs.io/en/latest/api/database/changes.html#get--db-_changes
func (db *DB) Changes(ctx context.Context, options ...Options) (*Changes, error) {
	opts, err := mergeOptions(options...)
	if err != nil {
		return nil, err
	}
	changesi, err := db.driverDB.Changes(ctx, opts)
	if err != nil {
		return nil, err
	}
	return newChanges(ctx, changesi), nil
}

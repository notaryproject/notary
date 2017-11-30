// Package memory provides a memory-backed Kivik driver, intended for testing.
// This version of the package is deprecated, and no longer receiving updates.
// Please use github.com/go-kivik/memorydb instead.
package memory

import (
	"context"
	"net/http"
	"regexp"
	"sync"

	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/driver"
	"github.com/flimzy/kivik/driver/common"
	"github.com/flimzy/kivik/errors"
)

type memDriver struct{}

var _ driver.Driver = &memDriver{}

func init() {
	kivik.Register("memory", &memDriver{})
}

type client struct {
	*common.Client
	mutex sync.RWMutex
	dbs   map[string]*database
}

var _ driver.Client = &client{}

// Identifying constants
const (
	Version = "0.0.1"
	Vendor  = "Kivik Memory Adaptor"
)

func (d *memDriver) NewClient(_ context.Context, name string) (driver.Client, error) {
	return &client{
		Client: common.NewClient(Version, Vendor),
		dbs:    make(map[string]*database),
	}, nil
}

func (c *client) AllDBs(_ context.Context, _ map[string]interface{}) ([]string, error) {
	dbs := make([]string, 0, len(c.dbs))
	for k := range c.dbs {
		dbs = append(dbs, k)
	}
	return dbs, nil
}

func (c *client) DBExists(_ context.Context, dbName string, _ map[string]interface{}) (bool, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	_, ok := c.dbs[dbName]
	return ok, nil
}

// Copied verbatim from http://docs.couchdb.org/en/2.0.0/api/database/common.html#head--db
var validDBName = regexp.MustCompile("^[a-z][a-z0-9_$()+/-]*$")
var validNames = map[string]struct{}{
	"_users":      struct{}{},
	"_replicator": struct{}{},
}

func (c *client) CreateDB(ctx context.Context, dbName string, options map[string]interface{}) error {
	if exists, _ := c.DBExists(ctx, dbName, options); exists {
		return errors.Status(http.StatusPreconditionFailed, "database exists")
	}
	if _, ok := validNames[dbName]; !ok {
		if !validDBName.MatchString(dbName) {
			return errors.Status(kivik.StatusBadRequest, "invalid database name")
		}
	}
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.dbs[dbName] = &database{
		docs:     make(map[string]*document),
		security: &driver.Security{},
	}
	return nil
}

func (c *client) DestroyDB(ctx context.Context, dbName string, options map[string]interface{}) error {
	if exists, _ := c.DBExists(ctx, dbName, options); !exists {
		return errors.Status(http.StatusNotFound, "database does not exist")
	}
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.dbs[dbName].mu.Lock()
	defer c.dbs[dbName].mu.Unlock()
	c.dbs[dbName].deleted = true // To invalidate any outstanding db handles
	delete(c.dbs, dbName)
	return nil
}

func (c *client) DB(ctx context.Context, dbName string, options map[string]interface{}) (driver.DB, error) {
	if exists, _ := c.DBExists(ctx, dbName, options); !exists {
		return nil, errors.Status(http.StatusNotFound, "database does not exist")
	}
	return &db{
		client: c,
		dbName: dbName,
		db:     c.dbs[dbName],
	}, nil
}

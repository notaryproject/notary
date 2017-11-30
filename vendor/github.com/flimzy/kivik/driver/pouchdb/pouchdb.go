// Package pouchdb provides a PouchDB driver for Kivik. This version of the
// package is deprecated, and no longer receiving updates. Please use
// github.com/go-kivik/pouchdb instead.
package pouchdb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/driver"
	"github.com/flimzy/kivik/driver/pouchdb/bindings"
	"github.com/flimzy/kivik/errors"
	"github.com/imdario/mergo"
)

type Driver struct{}

var _ driver.Driver = &Driver{}

func init() {
	kivik.Register("pouch", &Driver{})
}

// NewClient returns a PouchDB client handle. Provide a dsn only for remote
// databases. Otherwise specify ""
func (d *Driver) NewClient(_ context.Context, dsn string) (driver.Client, error) {
	var u *url.URL
	var auth authenticator
	var user *url.Userinfo
	if dsn != "" {
		var err error
		u, err = url.Parse(dsn)
		if err != nil {
			return nil, fmt.Errorf("Invalid DSN URL '%s' provided: %s", dsn, err)
		}
		user = u.User
		u.User = nil
	}
	pouch := bindings.GlobalPouchDB()
	client := &client{
		dsn:   u,
		pouch: pouch,
		opts:  make(map[string]Options),
	}
	if user != nil {
		pass, _ := user.Password()
		auth = &BasicAuth{
			Name:     user.Username(),
			Password: pass,
		}
		if err := auth.authenticate(client); err != nil {
			return nil, err
		}
	}
	return client, nil
}

type client struct {
	dsn   *url.URL
	opts  map[string]Options
	pouch *bindings.PouchDB

	// This mantains a list of running replications
	replications   []*replication
	replicationsMU sync.RWMutex
}

var _ driver.Client = &client{}

const optionsDefaultKey = "defaults"

// AllDBs returns the list of all existing databases. This function depends on
// the pouchdb-all-dbs plugin being loaded.
func (c *client) AllDBs(ctx context.Context, _ map[string]interface{}) ([]string, error) {
	if c.dsn == nil {
		return c.pouch.AllDBs(ctx)
	}
	return nil, errors.New("AllDBs() not implemented for remote PouchDB databases")
}

func (c *client) Version(_ context.Context) (*driver.Version, error) {
	ver := c.pouch.Version()
	return &driver.Version{
		Version:     ver,
		Vendor:      "PouchDB",
		RawResponse: json.RawMessage(fmt.Sprintf(`{"version":"%s","vendor":{"name":"PouchDB"}}`, ver)),
	}, nil
}

func (c *client) dbURL(db string) string {
	if c.dsn == nil {
		// No transformation for local databases
		return db
	}
	myURL := *c.dsn // Make a copy
	myURL.Path = myURL.Path + strings.TrimLeft(db, "/")
	return myURL.String()
}

// Options is a struct of options, as documented in the PouchDB API.
type Options map[string]interface{}

func (c *client) options(options ...Options) (Options, error) {
	o := Options{}
	for _, defOpts := range c.opts {
		if err := mergo.MergeWithOverwrite(&o, defOpts); err != nil {
			return nil, err
		}
	}
	for _, opts := range options {
		if err := mergo.MergeWithOverwrite(&o, opts); err != nil {
			return nil, err
		}
	}
	return o, nil
}

func (c *client) isRemote() bool {
	return c.dsn != nil
}

// DBExists returns true if the requested DB exists. This function only works
// for remote databases. For local databases, it creates the database.
// Silly PouchDB.
func (c *client) DBExists(ctx context.Context, dbName string, options map[string]interface{}) (bool, error) {
	opts, err := c.options(options, Options{"skip_setup": true})
	if err != nil {
		return false, err
	}
	_, err = c.pouch.New(c.dbURL(dbName), opts).Info(ctx)
	if err == nil {
		return true, nil
	}
	if kivik.StatusCode(err) == http.StatusNotFound {
		return false, nil
	}
	return false, err
}

func (c *client) CreateDB(ctx context.Context, dbName string, options map[string]interface{}) error {
	if c.isRemote() {
		if exists, _ := c.DBExists(ctx, dbName, options); exists {
			return errors.Status(http.StatusPreconditionFailed, "database exists")
		}
	}
	opts, err := c.options(options)
	if err != nil {
		return err
	}
	_, err = c.pouch.New(c.dbURL(dbName), opts).Info(ctx)
	return err
}

func (c *client) DestroyDB(ctx context.Context, dbName string, options map[string]interface{}) error {
	opts, err := c.options(options)
	if err != nil {
		return err
	}
	exists, err := c.DBExists(ctx, dbName, opts)
	if err != nil {
		return err
	}
	if !exists {
		// This will only ever do anything for a remote database
		return errors.Status(http.StatusNotFound, "database does not exist")
	}
	return c.pouch.New(c.dbURL(dbName), opts).Destroy(ctx, nil)
}

func (c *client) DB(ctx context.Context, dbName string, options map[string]interface{}) (driver.DB, error) {
	opts, err := c.options(options)
	if err != nil {
		return nil, err
	}
	exists, err := c.DBExists(ctx, dbName, opts)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.Status(kivik.StatusNotFound, "database does not exist")
	}
	return &db{
		db:     c.pouch.New(c.dbURL(dbName), opts),
		client: c,
	}, nil
}

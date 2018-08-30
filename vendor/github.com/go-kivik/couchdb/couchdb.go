// Package couchdb is a driver for connecting with a CouchDB server over HTTP.
package couchdb

import (
	"context"
	"strings"
	"sync"

	"github.com/go-kivik/couchdb/chttp"
	"github.com/go-kivik/kivik"
	"github.com/go-kivik/kivik/driver"
	"github.com/go-kivik/kivik/errors"
)

// Couch represents the parent driver instance.
type Couch struct{}

var _ driver.Driver = &Couch{}

func init() {
	kivik.Register("couch", &Couch{})
}

// CompatMode is a flag indicating the compatibility mode of the driver.
type CompatMode int

// Compatibility modes
const (
	CompatUnknown = iota
	CompatCouch16
	CompatCouch20
)

// Known vendor strings
const (
	VendorCouchDB  = "The Apache Software Foundation"
	VendorCloudant = "IBM Cloudant"
)

type client struct {
	*chttp.Client
	Compat CompatMode

	// schedulerDetected will be set once the scheduler has been detected.
	// It should only be accessed through the schedulerSupported() method.
	schedulerDetected *bool
	sdMU              sync.Mutex

	// noFind will be set to true if the Mango _find support is found not to be
	// supported.
	noFind bool
}

var _ driver.Client = &client{}

// NewClient establishes a new connection to a CouchDB server instance. If
// auth credentials are included in the URL, they are used to authenticate using
// CookieAuth (or BasicAuth if compiled with GopherJS). If you wish to use a
// different auth mechanism, do not specify credentials here, and instead call
// Authenticate() later.
func (d *Couch) NewClient(ctx context.Context, dsn string) (driver.Client, error) {
	chttpClient, err := chttp.New(ctx, dsn)
	if err != nil {
		return nil, errors.WrapStatus(kivik.StatusBadRequest, err)
	}
	c := &client{
		Client: chttpClient,
	}
	c.setCompatMode(ctx)
	return c, nil
}

func (c *client) setCompatMode(ctx context.Context) {
	info, err := c.Version(ctx)
	if err != nil {
		// We don't want to error here, in case the / endpoint is just blocked
		// for security reasons or something; but then we also can't infer the
		// compat mode, so just return, defaulting to CompatUnknown.
		return
	}
	schedulerSupported := false
	c.sdMU.Lock()
	defer c.sdMU.Unlock()
	switch info.Vendor {
	case VendorCouchDB, VendorCloudant:
		switch {
		case strings.HasPrefix(info.Version, "2.1"):
			c.Compat = CompatCouch20
		case strings.HasPrefix(info.Version, "2.0"):
			c.Compat = CompatCouch20
			c.schedulerDetected = &schedulerSupported
		case strings.HasPrefix(info.Version, "1.6"):
			c.Compat = CompatCouch16
			c.schedulerDetected = &schedulerSupported
		case strings.HasPrefix(info.Version, "1.7"):
			c.Compat = CompatCouch16
			c.noFind = true
			c.schedulerDetected = &schedulerSupported
		}
	}
}

func (c *client) DB(_ context.Context, dbName string, _ map[string]interface{}) (driver.DB, error) {
	if dbName == "" {
		return nil, missingArg("dbName")
	}
	return &db{
		client: c,
		dbName: dbName,
	}, nil
}

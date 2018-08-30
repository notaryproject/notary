package kivik

import (
	"context"
	"encoding/json"

	"github.com/imdario/mergo"

	"github.com/go-kivik/kivik/driver"
	"github.com/go-kivik/kivik/errors"
)

// Client is a client connection handle to a CouchDB-like server.
type Client struct {
	dsn          string
	driverName   string
	driverClient driver.Client
}

// Options is a collection of options. The keys and values are backend specific.
type Options map[string]interface{}

func mergeOptions(otherOpts ...Options) (Options, error) {
	var options Options
	for _, opts := range otherOpts {
		if err := mergo.MergeWithOverwrite(&options, opts); err != nil {
			return nil, err
		}
	}
	return options, nil
}

// New creates a new client object specified by its database driver name
// and a driver-specific data source name.
func New(ctx context.Context, driverName, dataSourceName string) (*Client, error) {
	driversMu.RLock()
	driveri, ok := drivers[driverName]
	driversMu.RUnlock()
	if !ok {
		return nil, errors.Statusf(StatusBadRequest, "kivik: unknown driver %q (forgotten import?)", driverName)
	}
	client, err := driveri.NewClient(ctx, dataSourceName)
	if err != nil {
		return nil, err
	}
	return &Client{
		dsn:          dataSourceName,
		driverName:   driverName,
		driverClient: client,
	}, nil
}

// Driver returns the name of the driver string used to connect this client.
func (c *Client) Driver() string {
	return c.driverName
}

// DSN returns the data source name used to connect this client.
func (c *Client) DSN() string {
	return c.dsn
}

// Version represents a server version response.
type Version struct {
	// Version is the version number reported by the server or backend.
	Version string
	// Vendor is the vendor string reported by the server or backend.
	Vendor string
	// RawResponse is the raw response body returned by the server, useful if
	// you need additional backend-specific information.
	//
	// For the format of this document, see
	// http://docs.couchdb.org/en/2.0.0/api/server/common.html#get
	RawResponse json.RawMessage
}

// Version returns version and vendor info about the backend.
func (c *Client) Version(ctx context.Context) (*Version, error) {
	ver, err := c.driverClient.Version(ctx)
	if err != nil {
		return nil, err
	}
	return &Version{
		Version:     ver.Version,
		Vendor:      ver.Vendor,
		RawResponse: ver.RawResponse,
	}, nil
}

// DB returns a handle to the requested database. Any options parameters
// passed are merged, with later values taking precidence.
func (c *Client) DB(ctx context.Context, dbName string, options ...Options) (*DB, error) {
	opts, err := mergeOptions(options...)
	if err != nil {
		return nil, err
	}
	db, err := c.driverClient.DB(ctx, dbName, opts)
	return &DB{
		client:   c,
		name:     dbName,
		driverDB: db,
	}, err
}

// AllDBs returns a list of all databases.
func (c *Client) AllDBs(ctx context.Context, options ...Options) ([]string, error) {
	opts, err := mergeOptions(options...)
	if err != nil {
		return nil, err
	}
	return c.driverClient.AllDBs(ctx, opts)
}

// DBExists returns true if the specified database exists.
func (c *Client) DBExists(ctx context.Context, dbName string, options ...Options) (bool, error) {
	opts, err := mergeOptions(options...)
	if err != nil {
		return false, err
	}
	return c.driverClient.DBExists(ctx, dbName, opts)
}

// CreateDB creates a DB of the requested name.
func (c *Client) CreateDB(ctx context.Context, dbName string, options ...Options) (*DB, error) {
	opts, err := mergeOptions(options...)
	if err != nil {
		return nil, err
	}
	if e := c.driverClient.CreateDB(ctx, dbName, opts); e != nil {
		return nil, e
	}
	return c.DB(ctx, dbName, nil)
}

// DestroyDB deletes the requested DB.
func (c *Client) DestroyDB(ctx context.Context, dbName string, options ...Options) error {
	opts, err := mergeOptions(options...)
	if err != nil {
		return err
	}
	return c.driverClient.DestroyDB(ctx, dbName, opts)
}

// Authenticate authenticates the client with the passed authenticator, which
// is driver-specific. If the driver does not understand the authenticator, an
// error will be returned.
func (c *Client) Authenticate(ctx context.Context, a interface{}) error {
	if auth, ok := c.driverClient.(driver.Authenticator); ok {
		return auth.Authenticate(ctx, a)
	}
	return errors.Status(StatusNotImplemented, "kivik: driver does not support authentication")
}

func missingArg(arg string) error {
	return errors.Statusf(StatusBadRequest, "kivik: %s required", arg)
}

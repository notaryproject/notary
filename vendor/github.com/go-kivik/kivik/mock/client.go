package mock

import (
	"context"

	"github.com/go-kivik/kivik/driver"
)

// Client mocks driver.Client
type Client struct {
	// ID identifies a specific Client instance
	ID            string
	AllDBsFunc    func(context.Context, map[string]interface{}) ([]string, error)
	CreateDBFunc  func(context.Context, string, map[string]interface{}) error
	DBFunc        func(context.Context, string, map[string]interface{}) (driver.DB, error)
	DBExistsFunc  func(context.Context, string, map[string]interface{}) (bool, error)
	DestroyDBFunc func(context.Context, string, map[string]interface{}) error
	VersionFunc   func(context.Context) (*driver.Version, error)
}

var _ driver.Client = &Client{}

// AllDBs calls c.AllDBsFunc
func (c *Client) AllDBs(ctx context.Context, opts map[string]interface{}) ([]string, error) {
	return c.AllDBsFunc(ctx, opts)
}

// CreateDB calls c.CreateDBFunc
func (c *Client) CreateDB(ctx context.Context, dbname string, opts map[string]interface{}) error {
	return c.CreateDBFunc(ctx, dbname, opts)
}

// DB calls c.DBFunc
func (c *Client) DB(ctx context.Context, dbname string, opts map[string]interface{}) (driver.DB, error) {
	return c.DBFunc(ctx, dbname, opts)
}

// DBExists calls c.DBExistsFunc
func (c *Client) DBExists(ctx context.Context, dbname string, opts map[string]interface{}) (bool, error) {
	return c.DBExistsFunc(ctx, dbname, opts)
}

// DestroyDB calls c.DestroyDBFunc
func (c *Client) DestroyDB(ctx context.Context, dbname string, opts map[string]interface{}) error {
	return c.DestroyDBFunc(ctx, dbname, opts)
}

// Version calls c.VersionFunc
func (c *Client) Version(ctx context.Context) (*driver.Version, error) {
	return c.VersionFunc(ctx)
}

// ClientReplicator mocks driver.Client and driver.ClientReplicator
type ClientReplicator struct {
	*Client
	GetReplicationsFunc func(context.Context, map[string]interface{}) ([]driver.Replication, error)
	ReplicateFunc       func(context.Context, string, string, map[string]interface{}) (driver.Replication, error)
}

var _ driver.ClientReplicator = &ClientReplicator{}

// GetReplications calls c.GetReplicationsFunc
func (c *ClientReplicator) GetReplications(ctx context.Context, opts map[string]interface{}) ([]driver.Replication, error) {
	return c.GetReplicationsFunc(ctx, opts)
}

// Replicate calls c.ReplicateFunc
func (c *ClientReplicator) Replicate(ctx context.Context, target, source string, opts map[string]interface{}) (driver.Replication, error) {
	return c.ReplicateFunc(ctx, target, source, opts)
}

// Authenticator mocks driver.Client and driver.Authenticator
type Authenticator struct {
	*Client
	AuthenticateFunc func(context.Context, interface{}) error
}

var _ driver.Authenticator = &Authenticator{}

// Authenticate calls c.AuthenticateFunc
func (c *Authenticator) Authenticate(ctx context.Context, a interface{}) error {
	return c.AuthenticateFunc(ctx, a)
}

// DBUpdater mocks driver.Client and driver.DBUpdater
type DBUpdater struct {
	*Client
	DBUpdatesFunc func() (driver.DBUpdates, error)
}

var _ driver.DBUpdater = &DBUpdater{}

// DBUpdates calls c.DBUpdatesFunc
func (c *DBUpdater) DBUpdates() (driver.DBUpdates, error) {
	return c.DBUpdatesFunc()
}

package couchdb

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/docker/go-connections/tlsconfig"
	"github.com/go-kivik/kivik"
	couch "github.com/go-kivik/couchdb"
	"github.com/go-kivik/couchdb/chttp"
	"github.com/sirupsen/logrus"
)

// ID of a document in CouchDB; includes Rev for updating an existing
// document
type ID struct {
	ID  string `json:"_id,omitempty"`
	Rev string `json:"_rev,omitempty"`
}

// Timing can be embedded into other models to add time tracking fields
type Timing struct {
	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

func createConnection(tlsOpts tlsconfig.Options, host string, username, password string) (*kivik.Client, error) {
	// remove username and password from URL to avoid authentication step
	// before we can set the TLS transport
	_host, err := url.Parse(host)
	if err != nil {
		return nil, err
	}
	_host.User = nil

	logrus.Debugf("attempting to connect %s to host %s", username, _host.String())

	client, err := kivik.New(context.TODO(), "couch", _host.String())
	if err != nil {
		return nil, err
	}

	t, err := tlsconfig.Client(tlsOpts)
	if err != nil {
		return nil, err
	}
	// We have to use the original CipherSuites to be able to create a
	// connection with CouchDB; this may be due to a CouchDB/Erlang
	// regression that requires us to offer more CipherSuites.
	t.CipherSuites = tls.Config{}.CipherSuites

	setXport := couch.SetTransport(&http.Transport{
		TLSClientConfig: t,
	})

	// The following sets the transport without actually connecting
	// to the server
	if err = client.Authenticate(context.TODO(), setXport); err != nil {
		return nil, err
	}

	if username != "" {
		var basicAuth chttp.Authenticator = &chttp.BasicAuth{
			Username: username,
			Password: password,
		}
		if err = client.Authenticate(context.TODO(), basicAuth); err != nil {
			return nil, err
		}
		// Transport is set now and we have a valid connection with
		// a valid username and password
	}
	return client, nil
}

// AdminConnection sets up an admin CouchDB connection to the host (`host:port` format)
// using the CA .pem file provided at path `caFile`
func AdminConnection(tlsOpts tlsconfig.Options, host string) (*kivik.Client, error) {
	var username, password string

	_host, err := url.Parse(host)
	if err != nil {
		logrus.Debugf("failed to parse host URL: %s", err)
		return nil, err
	}
	userinfo := _host.User
	if userinfo != nil {
		username = userinfo.Username()
		password, _ = userinfo.Password()
		_host.User = nil
	}
	return createConnection(tlsOpts, host, username, password)
}

// UserConnection sets up a user CouchDB connection to the host (`host:port` format)
// using the CA .pem file provided at path `caFile`, using the provided username.
func UserConnection(tlsOpts tlsconfig.Options, host, username, password string) (*kivik.Client, error) {

	return createConnection(tlsOpts, host, username, password)
}

// GetAllDocs gets all documents from a given database and table; this function returns
// the Rows to iterate over.
func GetAllDocs(client *kivik.Client, dbName, tableName string) (*kivik.DB, *kivik.Rows, error) {
	db, err := DB(client, dbName, tableName)
	if err != nil {
		return nil, nil, err
	}

	rows, err := db.AllDocs(context.TODO())
	if err == nil {
		err = rows.Err()
	}
	if err != nil {
		return nil, nil, fmt.Errorf("GetAllDocs: Could not get all document for %s$%s: %s", dbName, tableName, err)
	}

	return db, rows, nil
}

// CreateDBName creates the Couch database name given the name of the database and the
// name of a table
func CreateDBName(dbName, tableName string) string {
	if tableName == "" {
		return dbName
	}
	return dbName + "$" + tableName
}

// DBDrop drops a Couch database given the the name of the database and the name of the table
func DBDrop(client *kivik.Client, dbName, tableName string) error {
	return client.DestroyDB(context.TODO(), CreateDBName(dbName, tableName))
}

// DB gets access to a Couch database given the name of the database and the name of the
// table
func DB(client *kivik.Client, dbName, tableName string) (*kivik.DB, error) {
	name := CreateDBName(dbName, tableName)
	db, err := client.DB(context.TODO(), name)
	if err != nil {
		return nil, fmt.Errorf("Could not connect to DB %s: %s", name, err)
	}
	return db, nil
}

// DBExists checks whether a Couch database exists given the name of the database and the
// name of the table
func DBExists(client *kivik.Client, dbName, tableName string) (bool, error) {
	name := CreateDBName(dbName, tableName)
	exists, err := client.DBExists(context.TODO(), name)
	if err != nil {
		return false, fmt.Errorf("Could not determine whether DB %s exists: %s", name, err)
	}
	return exists, nil
}

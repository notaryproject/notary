package driver

import (
	"context"
	"encoding/json"
	"io"
	"time"
)

// Driver is the interface that must be implemented by a database driver.
type Driver interface {
	// NewClient returns a connection handle to the database. The name is in a
	// driver-specific format.
	NewClient(ctx context.Context, name string) (Client, error)
}

// Version represents a server version response.
type Version struct {
	// Version is the version number reported by the server or backend.
	Version string
	// Vendor is the vendor string reported by the server or backend.
	Vendor string
	// Features is a list of enabled, optional features.  This was added in
	// CouchDB 2.1.0, and can be expected to be empty for older versions.
	Features []string
	// RawResponse is the raw response body as returned by the server.
	RawResponse json.RawMessage
}

// Client is a connection to a database server.
type Client interface {
	// Version returns the server implementation's details.
	Version(ctx context.Context) (*Version, error)
	// AllDBs returns a list of all existing database names.
	AllDBs(ctx context.Context, options map[string]interface{}) ([]string, error)
	// DBExists returns true if the database exists.
	DBExists(ctx context.Context, dbName string, options map[string]interface{}) (bool, error)
	// CreateDB creates the requested DB. The dbName is validated as a valid
	// CouchDB database name prior to calling this function, so the driver can
	// assume a valid name.
	CreateDB(ctx context.Context, dbName string, options map[string]interface{}) error
	// DestroyDB deletes the requested DB.
	DestroyDB(ctx context.Context, dbName string, options map[string]interface{}) error
	// DB returns a handleto the requested database
	DB(ctx context.Context, dbName string, options map[string]interface{}) (DB, error)
}

// Replication represents a _replicator document.
type Replication interface {
	// The following methods are called just once, when the Replication is first
	// returned from Replicate() or GetReplications().
	ReplicationID() string
	Source() string
	Target() string
	StartTime() time.Time
	EndTime() time.Time
	State() string
	Err() error

	// These methods may be triggered by user actions.

	// Delete deletes a replication, which cancels it if it is running.
	Delete(context.Context) error
	// Update fetches the latest replication state from the server.
	Update(context.Context, *ReplicationInfo) error
}

// ReplicationInfo represents a snap-shot state of a replication, as provided
// by the _active_tasks endpoint.
type ReplicationInfo struct {
	DocWriteFailures int64
	DocsRead         int64
	DocsWritten      int64
	Progress         float64
}

// ClientReplicator is an optional interface that may be implemented by a Client
// that supports replication between two database.
type ClientReplicator interface {
	// Replicate initiates a replication.
	Replicate(ctx context.Context, targetDSN, sourceDSN string, options map[string]interface{}) (Replication, error)
	// GetReplications returns a list of replicatoins (i.e. all docs in the
	// _replicator database)
	GetReplications(ctx context.Context, options map[string]interface{}) ([]Replication, error)
}

// Authenticator is an optional interface that may be implemented by a Client
// that supports authenitcated connections.
type Authenticator interface {
	// Authenticate attempts to authenticate the client using an authenticator.
	// If the authenticator is not known to the client, an error should be
	// returned.
	Authenticate(ctx context.Context, authenticator interface{}) error
}

// DBStats contains database statistics..
type DBStats struct {
	Name           string `json:"db_name"`
	CompactRunning bool   `json:"compact_running"`
	DocCount       int64  `json:"doc_count"`
	DeletedCount   int64  `json:"doc_del_count"`
	UpdateSeq      string `json:"update_seq"`
	DiskSize       int64  `json:"disk_size"`
	ActiveSize     int64  `json:"data_size"`
	ExternalSize   int64  `json:"-"`
}

// Members represents the members of a database security document.
type Members struct {
	Names []string `json:"names,omitempty"`
	Roles []string `json:"roles,omitempty"`
}

// Security represents a database security document.
type Security struct {
	Admins  Members `json:"admins"`
	Members Members `json:"members"`
}

// DB is a database handle.
type DB interface {
	// AllDocs returns all of the documents in the database, subject to the
	// options provided.
	AllDocs(ctx context.Context, options map[string]interface{}) (Rows, error)
	// Get fetches the requested document from the database, and unmarshals it
	// into doc.
	Get(ctx context.Context, docID string, options map[string]interface{}) (json.RawMessage, error)
	// CreateDoc creates a new doc, with a server-generated ID.
	CreateDoc(ctx context.Context, doc interface{}) (docID, rev string, err error)
	// Put writes the document in the database.
	Put(ctx context.Context, docID string, doc interface{}) (rev string, err error)
	// Delete marks the specified document as deleted.
	Delete(ctx context.Context, docID, rev string) (newRev string, err error)
	// Stats returns database statistics.
	Stats(ctx context.Context) (*DBStats, error)
	// Compact initiates compaction of the database.
	Compact(ctx context.Context) error
	// CompactView initiates compaction of the view.
	CompactView(ctx context.Context, ddocID string) error
	// ViewCleanup cleans up stale view files.
	ViewCleanup(ctx context.Context) error
	// Security returns the database's security document.
	Security(ctx context.Context) (*Security, error)
	// SetSecurity sets the database's security document.
	SetSecurity(ctx context.Context, security *Security) error
	// Changes returns a Rows iterator for the changes feed. In continuous mode,
	// the iterator will continue indefinitely, until Close is called.
	Changes(ctx context.Context, options map[string]interface{}) (Changes, error)
	// PutAttachment uploads an attachment to the specified document, returning
	// the new revision.
	PutAttachment(ctx context.Context, docID, rev, filename, contentType string, body io.Reader) (newRev string, err error)
	// GetAttachment fetches an attachment for the associated document ID. rev
	// may be an empty string to fetch the most recent document version.
	GetAttachment(ctx context.Context, docID, rev, filename string) (contentType string, md5sum MD5sum, body io.ReadCloser, err error)
	// DeleteAttachment deletes an attachment from a document, returning the
	// document's new revision.
	DeleteAttachment(ctx context.Context, docID, rev, filename string) (newRev string, err error)
	// Query performs a query against a view, subject to the options provided.
	// ddoc will be the design doc name without the '_design/' previx.
	// view will be the view name without the '_view/' prefix.
	Query(ctx context.Context, ddoc, view string, options map[string]interface{}) (Rows, error)
}

// DBOpts will be merged with DB in Kivik 2.0. It wraps functions that take
// additional options arguments.
type DBOpts interface {
	CreateDocOpts(ctx context.Context, doc interface{}, options map[string]interface{}) (docID, rev string, err error)
	// PutOpts writes the document in the database.
	PutOpts(ctx context.Context, docID string, doc interface{}, options map[string]interface{}) (rev string, err error)
	// DeleteOpts marks the specified document as deleted.
	DeleteOpts(ctx context.Context, docID, rev string, options map[string]interface{}) (newRev string, err error)
	// StatsOpts returns database statistics.
	PutAttachmentOpts(ctx context.Context, docID, rev, filename, contentType string, body io.Reader, options map[string]interface{}) (newRev string, err error)
	// GetAttachmentOpts fetches an attachment for the associated document ID. rev
	// may be an empty string to fetch the most recent document version.
	GetAttachmentOpts(ctx context.Context, docID, rev, filename string, options map[string]interface{}) (contentType string, md5sum MD5sum, body io.ReadCloser, err error)
	// DeleteAttachmentOpts deletes an attachment from a document, returning the
	// document's new revision.
	DeleteAttachmentOpts(ctx context.Context, docID, rev, filename string, options map[string]interface{}) (newRev string, err error)
}

// OldBulkDocer is deprecated and will be removed in Kivik 2.0. Use BulkDocer instead.
type OldBulkDocer interface {
	// BulkDocs alls bulk create, update and/or delete operations. It returns an
	// iterator over the results.
	BulkDocs(ctx context.Context, docs []interface{}) (BulkResults, error)
}

// BulkDocer is an optional interface which may be implemented by a driver to
// support bulk insert/update operations. For any driver that does not support
// the BulkDocer interface, the Put or CreateDoc methods will be called for each
// document to emulate the same functionality.
type BulkDocer interface {
	// BulkDocs alls bulk create, update and/or delete operations. It returns an
	// iterator over the results.
	BulkDocs(ctx context.Context, docs []interface{}, options map[string]interface{}) (BulkResults, error)
}

// The Finder is an optional interface which may be implemented by a database. The
// Finder interface provides access to the new (in CouchDB 2.0) MongoDB-style
// query interface.
type Finder interface {
	// Find executes a query using the new /_find interface. If query is a
	// string, []byte, or json.RawMessage, it should be treated as a raw JSON
	// payload. Any other type should be marshaled to JSON.
	Find(ctx context.Context, query interface{}) (Rows, error)
	// CreateIndex creates an index if it doesn't already exist. If the index
	// already exists, it should do nothing. ddoc and name may be empty, in
	// which case they should be provided by the backend. If index is a string,
	// []byte, or json.RawMessage, it should be treated as a raw JSON payload.
	// Any other type should be marshaled to JSON.
	CreateIndex(ctx context.Context, ddoc, name string, index interface{}) error
	// GetIndexes returns a list of all indexes in the database.
	GetIndexes(ctx context.Context) ([]Index, error)
	// Delete deletes the requested index.
	DeleteIndex(ctx context.Context, ddoc, name string) error
}

// QueryPlan is the response of an Explain query.
type QueryPlan struct {
	DBName   string                 `json:"dbname"`
	Index    map[string]interface{} `json:"index"`
	Selector map[string]interface{} `json:"selector"`
	Options  map[string]interface{} `json:"opts"`
	Limit    int64                  `json:"limit"`
	Skip     int64                  `json:"skip"`

	// Fields is the list of fields to be returned in the result set, or
	// an empty list if all fields are to be returned.
	Fields []interface{}          `json:"fields"`
	Range  map[string]interface{} `json:"range"`
}

// The Explainer is an optional interface which provides access to the query
// explanation API supported by CouchDB 2.0 and newer, and PouchDB 6.3.4 and
// newer.
type Explainer interface {
	Explain(ctx context.Context, query interface{}) (*QueryPlan, error)
}

// Index is a MonboDB-style index definition.
type Index struct {
	DesignDoc  string      `json:"ddoc,omitempty"`
	Name       string      `json:"name"`
	Type       string      `json:"type"`
	Definition interface{} `json:"def"`
}

// MD5sum is a 128-bit MD5 checksum.
type MD5sum [16]byte

// OldAttachmentMetaer is deprected. Use AttachmentMetaer instead.
type OldAttachmentMetaer interface {
	// GetAttachmentMeta returns meta information about an attachment.
	GetAttachmentMeta(ctx context.Context, docID, rev, filename string) (contentType string, md5sum MD5sum, err error)
}

// AttachmentMetaer is an optional interface which may be satisfied by a
// DB. If satisfied, it may be used to fetch meta data about an attachment. If
// not satisfied, GetAttachment will be used instead.
type AttachmentMetaer interface {
	// GetAttachmentMetaOpts returns meta information about an attachment.
	GetAttachmentMeta(ctx context.Context, docID, rev, filename string, options map[string]interface{}) (contentType string, md5sum MD5sum, err error)
}

// BulkResult is the result of a single doc update in a BulkDocs request.
type BulkResult struct {
	ID    string `json:"id"`
	Rev   string `json:"rev"`
	Error error
}

// BulkResults is an iterator over the results for a BulkDocs call.
type BulkResults interface {
	// Next is called to populate *BulkResult with the values of the next bulk
	// result in the set.
	//
	// Next should return io.EOF when there are no more results.
	Next(*BulkResult) error
	// Close closes the bulk results iterator.
	Close() error
}

// Rever is an optional interface that may be implemented by a database. If not
// implemented by the driver, the Get method will be used to emulate the
// functionality.
type Rever interface {
	// Rev returns the most current revision of the requested document.
	Rev(ctx context.Context, docID string) (rev string, err error)
}

// DBFlusher is an optional interface that may be implemented by a database
// that can force a flush of the database backend file(s) to disk or other
// permanent storage.
type DBFlusher interface {
	// Flush requests a flush of disk cache to disk or other permanent storage.
	//
	// See http://docs.couchdb.org/en/2.0.0/api/database/compact.html#db-ensure-full-commit
	Flush(ctx context.Context) error
}

// Copier is an optional interface that may be implemented by a DB.
//
// If a DB does implement Copier, Copy() functions will use it.  If a DB does
// not implement the Copier interface, or if a call to Copy() returns an
// http.StatusUnimplemented, the driver will emulate a copy by doing
// a GET followed by PUT.
type Copier interface {
	Copy(ctx context.Context, targetID, sourceID string, options map[string]interface{}) (targetRev string, err error)
}

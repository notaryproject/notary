package kivik

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"reflect"
	"strings"

	"github.com/go-kivik/kivik/driver"
	"github.com/go-kivik/kivik/errors"
)

// DB is a handle to a specific database.
type DB struct {
	client   *Client
	name     string
	driverDB driver.DB
}

// Client returns the Client used to connect to the database.
func (db *DB) Client() *Client {
	return db.client
}

// Name returns the database name as passed when creating the DB connection.
func (db *DB) Name() string {
	return db.name
}

// AllDocs returns a list of all documents in the database.
func (db *DB) AllDocs(ctx context.Context, options ...Options) (*Rows, error) {
	opts, err := mergeOptions(options...)
	if err != nil {
		return nil, err
	}
	rowsi, err := db.driverDB.AllDocs(ctx, opts)
	if err != nil {
		return nil, err
	}
	return newRows(ctx, rowsi), nil
}

// Query executes the specified view function from the specified design
// document. ddoc and view may or may not be be prefixed with '_design/'
// and '_view/' respectively. No other
func (db *DB) Query(ctx context.Context, ddoc, view string, options ...Options) (*Rows, error) {
	opts, err := mergeOptions(options...)
	if err != nil {
		return nil, err
	}
	ddoc = strings.TrimPrefix(ddoc, "_design/")
	view = strings.TrimPrefix(view, "_view/")
	rowsi, err := db.driverDB.Query(ctx, ddoc, view, opts)
	if err != nil {
		return nil, err
	}
	return newRows(ctx, rowsi), nil
}

// Row contains the result of calling Get for a single document. For most uses,
// it is sufficient just to call the ScanDoc method. For more advanced uses, the
// fields may be accessed directly.
type Row struct {
	// ContentLength records the size of the JSON representation of the document
	// as requestd. The value -1 indicates that the length is unknown. Values
	// >= 0 indicate that the given number of bytes may be read from Body.
	ContentLength int64

	// Rev is the revision ID of the returned document.
	Rev string

	// Body represents the document's content.
	//
	// Kivik will always return a non-nil Body, except when Err is non-nil. The
	// ScanDoc method will close Body. When not using ScanDoc, it is the
	// caller's responsibility to close Body
	Body io.ReadCloser

	// Err contains any error that occurred while fetching the document. It is
	// typically returned by ScanDoc.
	Err error

	// Attachments is experimental
	Attachments *AttachmentsIterator
}

// ScanDoc unmarshals the data from the fetched row into dest. It is an
// intelligent wrapper around json.Unmarshal which also handles
// multipart/related responses. When done, the underlying reader is closed.
func (r *Row) ScanDoc(dest interface{}) error {
	if r.Err != nil {
		return r.Err
	}
	if reflect.TypeOf(dest).Kind() != reflect.Ptr {
		return errNonPtr
	}
	defer r.Body.Close() // nolint: errcheck
	return errors.WrapStatus(StatusBadResponse, json.NewDecoder(r.Body).Decode(dest))
}

// Get fetches the requested document. Any errors are deferred until the
// row.ScanDoc call.
func (db *DB) Get(ctx context.Context, docID string, options ...Options) *Row {
	opts, err := mergeOptions(options...)
	if err != nil {
		return &Row{Err: err}
	}
	doc, err := db.driverDB.Get(ctx, docID, opts)
	if err != nil {
		return &Row{Err: err}
	}
	row := &Row{
		ContentLength: doc.ContentLength,
		Rev:           doc.Rev,
		Body:          doc.Body,
	}
	if doc.Attachments != nil {
		row.Attachments = &AttachmentsIterator{atti: doc.Attachments}
	}
	return row
}

// GetMeta returns the size and rev of the specified document. GetMeta accepts
// the same options as the Get method.
func (db *DB) GetMeta(ctx context.Context, docID string, options ...Options) (size int64, rev string, err error) {
	opts, err := mergeOptions(options...)
	if err != nil {
		return 0, "", err
	}
	if r, ok := db.driverDB.(driver.MetaGetter); ok {
		return r.GetMeta(ctx, docID, opts)
	}
	row := db.Get(ctx, docID, nil)
	if row.Err != nil {
		return 0, "", row.Err
	}
	if row.Rev != "" {
		_ = row.Body.Close()
		return row.ContentLength, row.Rev, nil
	}
	var doc struct {
		Rev string `json:"_rev"`
	}
	// These last two lines cannot be combined for GopherJS due to a bug.
	// See https://github.com/gopherjs/gopherjs/issues/608
	err = row.ScanDoc(&doc)
	return row.ContentLength, doc.Rev, err
}

// CreateDoc creates a new doc with an auto-generated unique ID. The generated
// docID and new rev are returned.
func (db *DB) CreateDoc(ctx context.Context, doc interface{}, options ...Options) (docID, rev string, err error) {
	opts, err := mergeOptions(options...)
	if err != nil {
		return "", "", err
	}
	return db.driverDB.CreateDoc(ctx, doc, opts)
}

// normalizeFromJSON unmarshals a []byte, json.RawMessage or io.Reader to a
// map[string]interface{}, or passed through any other types.
func normalizeFromJSON(i interface{}) (interface{}, error) {
	var body []byte
	switch t := i.(type) {
	case []byte:
		body = t
	case json.RawMessage:
		body = t
	default:
		r, ok := i.(io.Reader)
		if !ok {
			return i, nil
		}
		var err error
		body, err = ioutil.ReadAll(r)
		if err != nil {
			return nil, errors.WrapStatus(StatusUnknownError, err)
		}
	}
	var x map[string]interface{}
	if err := json.Unmarshal(body, &x); err != nil {
		return nil, errors.WrapStatus(StatusBadRequest, err)
	}
	return x, nil
}

func extractDocID(i interface{}) (string, bool) {
	if i == nil {
		return "", false
	}
	var id string
	var ok bool
	switch t := i.(type) {
	case map[string]interface{}:
		id, ok = t["_id"].(string)
	case map[string]string:
		id, ok = t["_id"]
	default:
		data, err := json.Marshal(i)
		if err != nil {
			return "", false
		}
		var result struct {
			ID string `json:"_id"`
		}
		if err := json.Unmarshal(data, &result); err != nil {
			return "", false
		}
		id = result.ID
		ok = result.ID != ""
	}
	if !ok {
		return "", false
	}
	return id, true
}

// Put creates a new doc or updates an existing one, with the specified docID.
// If the document already exists, the current revision must be included in doc,
// with JSON key '_rev', otherwise a conflict will occur. The new rev is
// returned.
//
// doc may be one of:
//
//  - An object to be marshaled to JSON. The resulting JSON structure must
//    conform to CouchDB standards.
//  - A []byte value, containing a valid JSON document
//  - A json.RawMessage value containing a valid JSON document
//  - An io.Reader, from which a valid JSON document may be read.
func (db *DB) Put(ctx context.Context, docID string, doc interface{}, options ...Options) (rev string, err error) {
	if docID == "" {
		return "", missingArg("docID")
	}
	i, err := normalizeFromJSON(doc)
	if err != nil {
		return "", err
	}
	opts, err := mergeOptions(options...)
	if err != nil {
		return "", err
	}
	return db.driverDB.Put(ctx, docID, i, opts)
}

// Delete marks the specified document as deleted.
func (db *DB) Delete(ctx context.Context, docID, rev string, options ...Options) (newRev string, err error) {
	if docID == "" {
		return "", missingArg("docID")
	}
	opts, err := mergeOptions(options...)
	if err != nil {
		return "", err
	}
	return db.driverDB.Delete(ctx, docID, rev, opts)
}

// Flush requests a flush of disk cache to disk or other permanent storage.
//
// See http://docs.couchdb.org/en/2.0.0/api/database/compact.html#db-ensure-full-commit
func (db *DB) Flush(ctx context.Context) error {
	if flusher, ok := db.driverDB.(driver.Flusher); ok {
		return flusher.Flush(ctx)
	}
	return errors.Status(StatusNotImplemented, "kivik: flush not supported by driver")
}

// DBStats contains database statistics..
type DBStats struct {
	// Name is the name of the database.
	Name string `json:"db_name"`
	// CompactRunning is true if the database is currently being compacted.
	CompactRunning bool `json:"compact_running"`
	// DocCount is the number of documents are currently stored in the database.
	DocCount int64 `json:"doc_count"`
	// DeletedCount is a count of documents which have been deleted from the
	// database.
	DeletedCount int64 `json:"doc_del_count"`
	// UpdateSeq is the current update sequence for the database.
	UpdateSeq string `json:"update_seq"`
	// DiskSize is the number of bytes used on-disk to store the database.
	DiskSize int64 `json:"disk_size"`
	// ActiveSize is the number of bytes used on-disk to store active documents.
	// If this number is lower than DiskSize, then compaction would free disk
	// space.
	ActiveSize int64 `json:"data_size"`
	// ExternalSize is the size of the documents in the database, as represented
	// as JSON, before compression.
	ExternalSize int64 `json:"-"`
	// Cluster reports the cluster replication configuration variables.
	Cluster *ClusterConfig `json:"cluster,omitempty"`
	// RawResponse is the raw response body returned by the server, useful if
	// you need additional backend-specific information.
	//
	// For the format of this document, see
	// http://docs.couchdb.org/en/2.1.1/api/database/common.html#get--db
	RawResponse json.RawMessage `json:"-"`
}

// ClusterConfig contains the cluster configuration for the database.
type ClusterConfig struct {
	Replicas    int `json:"n"`
	Shards      int `json:"q"`
	ReadQuorum  int `json:"r"`
	WriteQuorum int `json:"w"`
}

// Stats returns database statistics.
func (db *DB) Stats(ctx context.Context) (*DBStats, error) {
	i, err := db.driverDB.Stats(ctx)
	if err != nil {
		return nil, err
	}
	var cluster *ClusterConfig
	if i.Cluster != nil {
		c := ClusterConfig(*i.Cluster)
		cluster = &c
	}
	return &DBStats{
		Name:           i.Name,
		CompactRunning: i.CompactRunning,
		DocCount:       i.DocCount,
		DeletedCount:   i.DeletedCount,
		UpdateSeq:      i.UpdateSeq,
		DiskSize:       i.DiskSize,
		ActiveSize:     i.ActiveSize,
		ExternalSize:   i.ExternalSize,
		Cluster:        cluster,
		RawResponse:    i.RawResponse,
	}, nil
}

// Compact begins compaction of the database. Check the CompactRunning field
// returned by Info() to see if the compaction has completed.
// See http://docs.couchdb.org/en/2.0.0/api/database/compact.html#db-compact
func (db *DB) Compact(ctx context.Context) error {
	return db.driverDB.Compact(ctx)
}

// CompactView compats the view indexes associated with the specified design
// document.
// See http://docs.couchdb.org/en/2.0.0/api/database/compact.html#db-compact-design-doc
func (db *DB) CompactView(ctx context.Context, ddocID string) error {
	return db.driverDB.CompactView(ctx, ddocID)
}

// ViewCleanup removes view index files that are no longer required as a result
// of changed views within design documents.
// See http://docs.couchdb.org/en/2.0.0/api/database/compact.html#db-view-cleanup
func (db *DB) ViewCleanup(ctx context.Context) error {
	return db.driverDB.ViewCleanup(ctx)
}

// Security returns the database's security document.
// See http://couchdb.readthedocs.io/en/latest/api/database/security.html#get--db-_security
func (db *DB) Security(ctx context.Context) (*Security, error) {
	s, err := db.driverDB.Security(ctx)
	if err != nil {
		return nil, err
	}
	return &Security{
		Admins:  Members(s.Admins),
		Members: Members(s.Members),
	}, err
}

// SetSecurity sets the database's security document.
// See http://couchdb.readthedocs.io/en/latest/api/database/security.html#put--db-_security
func (db *DB) SetSecurity(ctx context.Context, security *Security) error {
	if security == nil {
		return missingArg("security")
	}
	sec := &driver.Security{
		Admins:  driver.Members(security.Admins),
		Members: driver.Members(security.Members),
	}
	return db.driverDB.SetSecurity(ctx, sec)
}

// Copy copies the source document to a new document with an ID of targetID. If
// the database backend does not support COPY directly, the operation will be
// emulated with a Get followed by Put. The target will be an exact copy of the
// source, with only the ID and revision changed.
//
// See http://docs.couchdb.org/en/2.0.0/api/document/common.html#copy--db-docid
func (db *DB) Copy(ctx context.Context, targetID, sourceID string, options ...Options) (targetRev string, err error) {
	if targetID == "" {
		return "", missingArg("targetID")
	}
	if sourceID == "" {
		return "", missingArg("sourceID")
	}
	opts, err := mergeOptions(options...)
	if err != nil {
		return "", err
	}
	if copier, ok := db.driverDB.(driver.Copier); ok {
		return copier.Copy(ctx, targetID, sourceID, opts)
	}
	var doc map[string]interface{}
	if err = db.Get(ctx, sourceID, opts).ScanDoc(&doc); err != nil {
		return "", err
	}
	delete(doc, "_rev")
	doc["_id"] = targetID
	delete(opts, "rev") // rev has a completely different meaning for Copy and Put
	return db.Put(ctx, targetID, doc, opts)
}

// PutAttachment uploads the supplied content as an attachment to the specified
// document.
func (db *DB) PutAttachment(ctx context.Context, docID, rev string, att *Attachment, options ...Options) (newRev string, err error) {
	if docID == "" {
		return "", missingArg("docID")
	}
	if e := att.validate(); e != nil {
		return "", e
	}
	opts, err := mergeOptions(options...)
	if err != nil {
		return "", err
	}
	a := driver.Attachment(*att)
	return db.driverDB.PutAttachment(ctx, docID, rev, &a, opts)
}

// GetAttachment returns a file attachment associated with the document.
func (db *DB) GetAttachment(ctx context.Context, docID, rev, filename string, options ...Options) (*Attachment, error) {
	if docID == "" {
		return nil, missingArg("docID")
	}
	if filename == "" {
		return nil, missingArg("filename")
	}
	opts, e := mergeOptions(options...)
	if e != nil {
		return nil, e
	}
	att, err := db.driverDB.GetAttachment(ctx, docID, rev, filename, opts)
	if err != nil {
		return nil, err
	}
	a := Attachment(*att)
	return &a, nil
}

type nilContentReader struct{}

var _ io.ReadCloser = &nilContentReader{}

func (c nilContentReader) Read(_ []byte) (int, error) { return 0, io.EOF }
func (c nilContentReader) Close() error               { return nil }

var nilContent = nilContentReader{}

// GetAttachmentMeta returns meta data about an attachment. The attachment
// content returned will be empty.
func (db *DB) GetAttachmentMeta(ctx context.Context, docID, rev, filename string, options ...Options) (*Attachment, error) {
	if docID == "" {
		return nil, missingArg("docID")
	}
	if filename == "" {
		return nil, missingArg("filename")
	}
	var att *Attachment
	if metaer, ok := db.driverDB.(driver.AttachmentMetaGetter); ok {
		opts, err := mergeOptions(options...)
		if err != nil {
			return nil, err
		}
		a, err := metaer.GetAttachmentMeta(ctx, docID, rev, filename, opts)
		if err != nil {
			return nil, err
		}
		att = new(Attachment)
		*att = Attachment(*a)
	} else {
		var err error
		att, err = db.GetAttachment(ctx, docID, rev, filename, options...)
		if err != nil {
			return nil, err
		}
	}
	if att.Content != nil {
		_ = att.Content.Close() // Ensure this is closed
	}
	att.Content = nilContent
	return att, nil
}

// DeleteAttachment delets an attachment from a document, returning the
// document's new revision.
func (db *DB) DeleteAttachment(ctx context.Context, docID, rev, filename string, options ...Options) (newRev string, err error) {
	if docID == "" {
		return "", missingArg("docID")
	}
	if filename == "" {
		return "", missingArg("filename")
	}
	opts, err := mergeOptions(options...)
	if err != nil {
		return "", err
	}
	return db.driverDB.DeleteAttachment(ctx, docID, rev, filename, opts)
}

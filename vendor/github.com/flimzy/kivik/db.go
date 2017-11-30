package kivik

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"strings"

	"github.com/flimzy/kivik/driver"
	"github.com/flimzy/kivik/errors"
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

// Row is the result of calling Get for a single document.
type Row struct {
	doc json.RawMessage
}

// ScanDoc unmarshals the data from the fetched row into dest. See documentation
// on Rows.ScanDoc for details.
func (r *Row) ScanDoc(dest interface{}) error {
	return scan(dest, r.doc)
}

// Get fetches the requested document.
func (db *DB) Get(ctx context.Context, docID string, options ...Options) (*Row, error) {
	opts, err := mergeOptions(options...)
	if err != nil {
		return nil, err
	}
	row, err := db.driverDB.Get(ctx, docID, opts)
	if err != nil {
		return nil, err
	}
	return &Row{doc: row}, nil
}

// CreateDoc creates a new doc with an auto-generated unique ID. The generated
// docID and new rev are returned.
func (db *DB) CreateDoc(ctx context.Context, doc interface{}, options ...Options) (docID, rev string, err error) {
	if dbopt, ok := db.driverDB.(driver.DBOpts); ok {
		opts, err := mergeOptions(options...)
		if err != nil {
			return "", "", err
		}
		return dbopt.CreateDocOpts(ctx, doc, opts)
	}
	return db.driverDB.CreateDoc(ctx, doc)
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
	if dbopt, ok := db.driverDB.(driver.DBOpts); ok {
		opts, err := mergeOptions(options...)
		if err != nil {
			return "", err
		}
		return dbopt.PutOpts(ctx, docID, i, opts)
	}
	return db.driverDB.Put(ctx, docID, i)
}

// Delete marks the specified document as deleted.
func (db *DB) Delete(ctx context.Context, docID, rev string, options ...Options) (newRev string, err error) {
	if docID == "" {
		return "", missingArg("docID")
	}
	if dbopt, ok := db.driverDB.(driver.DBOpts); ok {
		opts, err := mergeOptions(options...)
		if err != nil {
			return "", err
		}
		return dbopt.DeleteOpts(ctx, docID, rev, opts)
	}
	return db.driverDB.Delete(ctx, docID, rev)
}

// Flush requests a flush of disk cache to disk or other permanent storage.
//
// See http://docs.couchdb.org/en/2.0.0/api/database/compact.html#db-ensure-full-commit
func (db *DB) Flush(ctx context.Context) error {
	if flusher, ok := db.driverDB.(driver.DBFlusher); ok {
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
}

// Stats returns database statistics.
func (db *DB) Stats(ctx context.Context) (*DBStats, error) {
	i, err := db.driverDB.Stats(ctx)
	stats := DBStats(*i)
	return &stats, err
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
	sec := &driver.Security{
		Admins:  driver.Members(security.Admins),
		Members: driver.Members(security.Members),
	}
	return db.driverDB.SetSecurity(ctx, sec)
}

// Rev returns the most current rev of the requested document. This can
// be more efficient than a full document fetch, because only the rev is
// fetched from the server.
func (db *DB) Rev(ctx context.Context, docID string) (rev string, err error) {
	if r, ok := db.driverDB.(driver.Rever); ok {
		return r.Rev(ctx, docID)
	}
	// These last two lines cannot be combined for GopherJS due to a bug.
	// See https://github.com/gopherjs/gopherjs/issues/608
	row, err := db.Get(ctx, docID, nil)
	if err != nil {
		return "", err
	}
	var doc struct {
		Rev string `json:"_rev"`
	}
	if err = row.ScanDoc(&doc); err != nil {
		return "", err
	}
	return doc.Rev, nil
}

// Copy copies the source document to a new document with an ID of targetID. If
// the database backend does not support COPY directly, the operation will be
// emulated with a Get followed by Put. The target will be an exact copy of the
// source, with only the ID and revision changed.
//
// See http://docs.couchdb.org/en/2.0.0/api/document/common.html#copy--db-docid
func (db *DB) Copy(ctx context.Context, targetID, sourceID string, options ...Options) (targetRev string, err error) {
	opts, err := mergeOptions(options...)
	if err != nil {
		return "", err
	}
	if copier, ok := db.driverDB.(driver.Copier); ok {
		targetRev, err = copier.Copy(ctx, targetID, sourceID, opts)
		if StatusCode(err) != StatusNotImplemented {
			return targetRev, err
		}
	}
	row, err := db.Get(ctx, sourceID, opts)
	if err != nil {
		return "", err
	}
	var doc map[string]interface{}
	if err = row.ScanDoc(&doc); err != nil {
		return "", err
	}
	delete(doc, "_rev")
	doc["_id"] = targetID
	return db.Put(ctx, targetID, doc)
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
	if dbopt, ok := db.driverDB.(driver.DBOpts); ok {
		opts, err := mergeOptions(options...)
		if err != nil {
			return "", err
		}
		return dbopt.PutAttachmentOpts(ctx, docID, rev, att.Filename, att.ContentType, att, opts)
	}
	return db.driverDB.PutAttachment(ctx, docID, rev, att.Filename, att.ContentType, att)
}

// GetAttachment returns a file attachment associated with the document.
func (db *DB) GetAttachment(ctx context.Context, docID, rev, filename string, options ...Options) (*Attachment, error) {
	if docID == "" {
		return nil, missingArg("docID")
	}
	if filename == "" {
		return nil, missingArg("filename")
	}
	var cType string
	var md5sum driver.MD5sum
	var body io.ReadCloser
	var err error
	if dbopt, ok := db.driverDB.(driver.DBOpts); ok {
		opts, e := mergeOptions(options...)
		if e != nil {
			return nil, e
		}
		cType, md5sum, body, err = dbopt.GetAttachmentOpts(ctx, docID, rev, filename, opts)
	} else {
		cType, md5sum, body, err = db.driverDB.GetAttachment(ctx, docID, rev, filename)
	}
	if err != nil {
		return nil, err
	}
	return &Attachment{
		ReadCloser:  body,
		Filename:    filename,
		ContentType: cType,
		MD5:         MD5sum(md5sum),
	}, nil
}

// GetAttachmentMeta returns meta data about an attachment. The attachment
// content returned will be empty.
func (db *DB) GetAttachmentMeta(ctx context.Context, docID, rev, filename string, options ...Options) (*Attachment, error) {
	if docID == "" {
		return nil, missingArg("docID")
	}
	if filename == "" {
		return nil, missingArg("filename")
	}
	if metaer, ok := db.driverDB.(driver.AttachmentMetaer); ok {
		opts, err := mergeOptions(options...)
		if err != nil {
			return nil, err
		}
		cType, md5sum, err := metaer.GetAttachmentMeta(ctx, docID, rev, filename, opts)
		if err != nil {
			return nil, err
		}
		return &Attachment{
			Filename:    filename,
			ContentType: cType,
			MD5:         MD5sum(md5sum),
		}, nil
	}
	if metaer, ok := db.driverDB.(driver.OldAttachmentMetaer); ok {
		cType, md5sum, err := metaer.GetAttachmentMeta(ctx, docID, rev, filename)
		if err != nil {
			return nil, err
		}
		return &Attachment{
			Filename:    filename,
			ContentType: cType,
			MD5:         MD5sum(md5sum),
		}, nil
	}
	att, err := db.GetAttachment(ctx, docID, rev, filename, options...)
	if err != nil {
		return nil, err
	}
	_ = att.Close()
	return &Attachment{
		Filename:    att.Filename,
		ContentType: att.ContentType,
		MD5:         att.MD5,
	}, nil
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
	if dbopt, ok := db.driverDB.(driver.DBOpts); ok {
		opts, err := mergeOptions(options...)
		if err != nil {
			return "", err
		}
		return dbopt.DeleteAttachmentOpts(ctx, docID, rev, filename, opts)
	}
	return db.driverDB.DeleteAttachment(ctx, docID, rev, filename)
}

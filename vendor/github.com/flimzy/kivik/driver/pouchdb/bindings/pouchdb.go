// Package bindings provides minimal GopherJS bindings around the PouchDB
// library. (https://pouchdb.com/api.html)
package bindings

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/gopherjs/gopherjs/js"
	"github.com/gopherjs/jsbuiltin"

	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/errors"
)

// DB is a PouchDB database object.
type DB struct {
	*js.Object
}

// PouchDB represents a PouchDB constructor.
type PouchDB struct {
	*js.Object
}

// GlobalPouchDB returns the global PouchDB object.
func GlobalPouchDB() *PouchDB {
	return &PouchDB{Object: js.Global.Get("PouchDB")}
}

// Defaults returns a new PouchDB constructor with the specified default options.
// See https://pouchdb.com/api.html#defaults
func Defaults(options map[string]interface{}) *PouchDB {
	return &PouchDB{Object: js.Global.Get("PouchDB").Call("defaults", options)}
}

// New creates a database or opens an existing one.
//
// See https://pouchdb.com/api.html#create_database
func (p *PouchDB) New(dbName string, options map[string]interface{}) *DB {
	return &DB{Object: p.Object.New(dbName, options)}
}

// Version returns the version of the currently running PouchDB library.
func (p *PouchDB) Version() string {
	return p.Get("version").String()
}

func setTimeout(ctx context.Context, options map[string]interface{}) map[string]interface{} {
	if ctx == nil { // Just to be safe
		return options
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		return options
	}
	if options == nil {
		options = make(map[string]interface{})
	}
	if _, ok := options["ajax"]; !ok {
		options["ajax"] = make(map[string]interface{})
	}
	ajax := options["ajax"].(map[string]interface{})
	ajax["timeout"] = int(deadline.Sub(time.Now()) * 1000)
	return options
}

type caller interface {
	Call(string, ...interface{}) *js.Object
}

// callBack executes the 'method' of 'o' as a callback, setting result to the
// callback's return value. An error is returned if either the callback returns
// an error, or if the context is cancelled. No attempt is made to abort the
// callback in the case that the context is cancelled.
func callBack(ctx context.Context, o caller, method string, args ...interface{}) (r *js.Object, e error) {
	defer RecoverError(&e)
	resultCh := make(chan *js.Object)
	var err error
	o.Call(method, args...).Call("then", func(r *js.Object) {
		resultCh <- r
	}).Call("catch", func(e *js.Object) {
		err = NewPouchError(e)
		close(resultCh)
	})
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-resultCh:
		return result, err
	}
}

// AllDBs returns the list of all existing (undeleted) databases.
func (p *PouchDB) AllDBs(ctx context.Context) ([]string, error) {
	if jsbuiltin.TypeOf(p.Get("allDbs")) != "function" {
		return nil, errors.New("pouchdb-all-dbs plugin not loaded")
	}
	result, err := callBack(ctx, p, "allDbs")
	if err != nil {
		return nil, err
	}
	if result == js.Undefined {
		return nil, nil
	}
	allDBs := make([]string, result.Length())
	for i := range allDBs {
		allDBs[i] = result.Index(i).String()
	}
	return allDBs, nil
}

// DBInfo is a struct respresenting information about a specific database.
type DBInfo struct {
	*js.Object
	Name      string `js:"db_name"`
	DocCount  int64  `js:"doc_count"`
	UpdateSeq string `js:"update_seq"`
}

// Info returns info about the database.
func (db *DB) Info(ctx context.Context) (*DBInfo, error) {
	result, err := callBack(ctx, db, "info")
	return &DBInfo{Object: result}, err
}

// Put creates a new document or update an existing document.
// See https://pouchdb.com/api.html#create_document
func (db *DB) Put(ctx context.Context, doc interface{}) (rev string, err error) {
	result, err := callBack(ctx, db, "put", doc, setTimeout(ctx, nil))
	if err != nil {
		return "", err
	}
	return result.Get("rev").String(), nil
}

// Post creates a new document and lets PouchDB auto-generate the ID.
// See https://pouchdb.com/api.html#using-dbpost
func (db *DB) Post(ctx context.Context, doc interface{}) (docID, rev string, err error) {
	result, err := callBack(ctx, db, "post", doc, setTimeout(ctx, nil))
	if err != nil {
		return "", "", err
	}
	return result.Get("id").String(), result.Get("rev").String(), nil
}

// Get fetches the requested document from the database.
// See https://pouchdb.com/api.html#fetch_document
func (db *DB) Get(ctx context.Context, docID string, opts map[string]interface{}) (doc []byte, err error) {
	result, err := callBack(ctx, db, "get", docID, setTimeout(ctx, opts))
	if err != nil {
		return nil, err
	}
	resultJSON := js.Global.Get("JSON").Call("stringify", result).String()
	return []byte(resultJSON), err
}

// Delete marks a document as deleted.
// See https://pouchdb.com/api.html#delete_document
func (db *DB) Delete(ctx context.Context, doc interface{}) (rev string, err error) {
	result, err := callBack(ctx, db, "remove", doc, setTimeout(ctx, nil))
	if err != nil {
		return "", err
	}
	return result.Get("rev").String(), nil
}

// Destroy destroys the database.
func (db *DB) Destroy(ctx context.Context, options map[string]interface{}) error {
	_, err := callBack(ctx, db, "destroy", setTimeout(ctx, options))
	return err
}

// AllDocs returns a list of all documents in the database.
func (db *DB) AllDocs(ctx context.Context, options map[string]interface{}) (*js.Object, error) {
	return callBack(ctx, db, "allDocs", setTimeout(ctx, options))
}

// Query queries a map/reduce function.
func (db *DB) Query(ctx context.Context, ddoc, view string, options map[string]interface{}) (*js.Object, error) {
	return callBack(ctx, db, "query", ddoc+"/"+view, setTimeout(ctx, options))
}

var findPluginNotLoaded = errors.Status(kivik.StatusNotImplemented, "kivik: pouchdb-find plugin not loaded")

// Find executes a MongoDB-style find query with the pouchdb-find plugin, if it
// is installed. If the plugin is not installed, a NotImplemented error will be
// returned.
//
// See https://github.com/pouchdb/pouchdb/tree/master/packages/node_modules/pouchdb-find#dbfindrequest--callback
func (db *DB) Find(ctx context.Context, query interface{}) (*js.Object, error) {
	if jsbuiltin.TypeOf(db.Object.Get("find")) != jsbuiltin.TypeFunction {
		return nil, findPluginNotLoaded
	}
	queryObj, err := Objectify(query)
	if err != nil {
		return nil, err
	}
	return callBack(ctx, db, "find", queryObj)
}

// Objectify unmarshals a string, []byte, or json.RawMessage into an interface{}.
// All other types are just passed through.
func Objectify(i interface{}) (interface{}, error) {
	var buf []byte
	switch t := i.(type) {
	case string:
		buf = []byte(t)
	case []byte:
		buf = t
	case json.RawMessage:
		buf = t
	default:
		return i, nil
	}
	var x interface{}
	err := json.Unmarshal(buf, &x)
	return x, errors.WrapStatus(kivik.StatusBadRequest, err)
}

// Compact compacts the database, and waits for it to complete. This may take
// a long time! Please wrap this call in a goroutine.
func (db *DB) Compact() error {
	_, err := callBack(context.Background(), db, "compact")
	return err
}

// ViewCleanup cleans up views, and waits for it to complete. This may take a
// long time! Please wrap this call in a goroutine.
func (db *DB) ViewCleanup() error {
	_, err := callBack(context.Background(), db, "viewCleanup")
	return err
}

var jsJSON = js.Global.Get("JSON")

// BulkDocs creates, updates, or deletes docs in bulk.
// See https://pouchdb.com/api.html#batch_create
func (db *DB) BulkDocs(ctx context.Context, docs ...interface{}) (result *js.Object, err error) {
	defer RecoverError(&err)
	jsDocs := make([]*js.Object, len(docs))
	for i, doc := range docs {
		jsonDoc, err := json.Marshal(doc)
		if err != nil {
			return nil, err
		}
		jsDocs[i] = jsJSON.Call("parse", string(jsonDoc))
	}
	return callBack(ctx, db, "bulkDocs", jsDocs, setTimeout(ctx, nil))
}

// Changes returns an event emitter object.
//
// See https://pouchdb.com/api.html#changes
func (db *DB) Changes(ctx context.Context, options map[string]interface{}) (changes *js.Object, e error) {
	defer RecoverError(&e)
	return db.Call("changes", setTimeout(ctx, options)), nil
}

// PutAttachment attaches a binary object to a document.
//
// See https://pouchdb.com/api.html#save_attachment
func (db *DB) PutAttachment(ctx context.Context, docID, filename, rev string, body io.Reader, ctype string) (*js.Object, error) {
	att, err := attachmentObject(ctype, body)
	if err != nil {
		return nil, err
	}
	if rev == "" {
		return callBack(ctx, db, "putAttachment", docID, filename, att, ctype)
	}
	return callBack(ctx, db, "putAttachment", docID, filename, rev, att, ctype)
}

// attachmentObject converts an io.Reader to a JavaScript Buffer in node, or
// a Blob in the browser
func attachmentObject(contentType string, content io.Reader) (att *js.Object, err error) {
	RecoverError(&err)
	buf := new(bytes.Buffer)
	buf.ReadFrom(content)
	if buffer := js.Global.Get("Buffer"); jsbuiltin.TypeOf(buffer) == "function" {
		// The Buffer type is supported, so we'll use that
		return buffer.New(buf.String()), nil
	}
	if js.Global.Get("Blob") != js.Undefined {
		// We have Blob support, must be in a browser
		return js.Global.Get("Blob").New([]interface{}{buf.Bytes()}, map[string]string{"type": contentType}), nil
	}
	// Not sure what to do
	return nil, errors.New("No Blob or Buffer support?!?")
}

// GetAttachment returns attachment data.
//
// See https://pouchdb.com/api.html#get_attachment
func (db *DB) GetAttachment(ctx context.Context, docID, filename string, options map[string]interface{}) (*js.Object, error) {
	return callBack(ctx, db, "getAttachment", docID, filename, setTimeout(ctx, options))
}

// RemoveAttachment deletes an attachment from a document.
//
// See https://pouchdb.com/api.html#delete_attachment
func (db *DB) RemoveAttachment(ctx context.Context, docID, filename, rev string) (*js.Object, error) {
	return callBack(ctx, db, "removeAttachment", docID, filename, rev)
}

// CreateIndex creates an index to be used by MongoDB-style queries with the
// pouchdb-find plugin, if it is installed. If the plugin is not installed, a
// NotImplemented error will be returned.
//
// See https://github.com/pouchdb/pouchdb/tree/master/packages/node_modules/pouchdb-find#dbcreateindexindex--callback
func (db *DB) CreateIndex(ctx context.Context, index interface{}) (*js.Object, error) {
	if jsbuiltin.TypeOf(db.Object.Get("find")) != jsbuiltin.TypeFunction {
		return nil, findPluginNotLoaded
	}
	return callBack(ctx, db, "createIndex", index)
}

// GetIndexes returns the list of currently defined indexes on the database.
//
// See https://github.com/pouchdb/pouchdb/tree/master/packages/node_modules/pouchdb-find#dbgetindexescallback
func (db *DB) GetIndexes(ctx context.Context) (*js.Object, error) {
	if jsbuiltin.TypeOf(db.Object.Get("find")) != jsbuiltin.TypeFunction {
		return nil, findPluginNotLoaded
	}
	return callBack(ctx, db, "getIndexes")
}

// DeleteIndex deletes an index used by the MongoDB-style queries with the
// pouchdb-find plugin, if it is installed. If the plugin is not installed, a
// NotImplemeneted error will be returned.
//
// See: https://github.com/pouchdb/pouchdb/tree/master/packages/node_modules/pouchdb-find#dbdeleteindexindex--callback
func (db *DB) DeleteIndex(ctx context.Context, index interface{}) (*js.Object, error) {
	if jsbuiltin.TypeOf(db.Object.Get("find")) != jsbuiltin.TypeFunction {
		return nil, findPluginNotLoaded
	}
	return callBack(ctx, db, "deleteIndex", index)
}

// Replication events
const (
	ReplicationEventChange   = "change"
	ReplicationEventComplete = "complete"
	ReplicationEventPaused   = "paused"
	ReplicationEventActive   = "active"
	ReplicationEventDenied   = "denied"
	ReplicationEventError    = "error"
)

// Replicate initiates a replication.
// See https://pouchdb.com/api.html#replication
func (p *PouchDB) Replicate(source, target interface{}, options map[string]interface{}) (result *js.Object, err error) {
	defer RecoverError(&err)
	return p.Call("replicate", source, target, options), nil
}

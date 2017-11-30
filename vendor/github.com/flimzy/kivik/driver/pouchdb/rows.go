package pouchdb

import (
	"encoding/json"
	"io"

	"github.com/gopherjs/gopherjs/js"

	"github.com/flimzy/kivik/driver"
	"github.com/flimzy/kivik/driver/pouchdb/bindings"
)

var jsJSON *js.Object

func init() {
	jsJSON = js.Global.Get("JSON")
}

type rows struct {
	*js.Object
	Off   int64  `js:"offset"`
	TRows int64  `js:"total_rows"`
	USeq  string `js:"update_seq"`
}

var _ driver.Rows = &rows{}

func (r *rows) Close() error {
	r.Delete("rows") // Free up memory used by any remaining rows
	return nil
}

func (r *rows) Next(row *driver.Row) (err error) {
	defer bindings.RecoverError(&err)
	if r.Get("rows") == js.Undefined || r.Get("rows").Length() == 0 {
		return io.EOF
	}
	next := r.Get("rows").Call("shift")
	row.ID = next.Get("id").String()
	row.Key = json.RawMessage(jsJSON.Call("stringify", next.Get("key")).String())
	row.Value = json.RawMessage(jsJSON.Call("stringify", next.Get("value")).String())
	if doc := next.Get("doc"); doc != js.Undefined {
		row.Doc = json.RawMessage(jsJSON.Call("stringify", doc).String())
	}
	return nil
}

func (r *rows) Offset() int64 {
	return r.Off
}

func (r *rows) TotalRows() int64 {
	return r.TRows
}

func (r *rows) UpdateSeq() string {
	return "" // PouchDB doesn't support this option
}

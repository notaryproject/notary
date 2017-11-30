package pouchdb

import (
	"context"
	"io"
	"io/ioutil"
	"strings"
	"sync"

	"github.com/flimzy/kivik/driver"
	"github.com/flimzy/kivik/driver/pouchdb/bindings"
	"github.com/gopherjs/gopherjs/js"
	"github.com/gopherjs/jsbuiltin"
)

func (d *db) PutAttachment(ctx context.Context, docID, rev, filename, contentType string, body io.Reader) (newRev string, err error) {
	result, err := d.db.PutAttachment(ctx, docID, filename, rev, body, contentType)
	if err != nil {
		return "", err
	}
	return result.Get("rev").String(), nil
}

func (d *db) GetAttachment(ctx context.Context, docID, rev, filename string) (cType string, md5sum driver.MD5sum, body io.ReadCloser, err error) {
	result, err := d.fetchAttachment(ctx, docID, rev, filename)
	if err != nil {
		return "", driver.MD5sum{}, nil, err
	}
	cType, body, err = parseAttachment(result)
	return
}

func (d *db) fetchAttachment(ctx context.Context, docID, rev, filename string) (*js.Object, error) {
	var opts map[string]interface{}
	if rev != "" {
		opts["rev"] = rev
	}
	return d.db.GetAttachment(ctx, docID, filename, opts)
}

func parseAttachment(att *js.Object) (cType string, content io.ReadCloser, err error) {
	defer bindings.RecoverError(&err)
	if jsbuiltin.TypeOf(att.Get("write")) == "function" {
		// This looks like a Buffer object; we're in Node.js
		body := att.Call("toString", "binary").String()
		// It might make sense to wrap the Buffer itself in an io.Reader interface,
		// but since this is only for testing, I'm taking the lazy way out, even
		// though it means slurping an extra copy into memory.
		return "", ioutil.NopCloser(strings.NewReader(body)), nil
	}
	// We're in the browser
	return att.Get("type").String(), &blobReader{Object: att}, nil
}

type blobReader struct {
	*js.Object
	offset int
	Size   int `js:"size"`
}

var _ io.ReadCloser = &blobReader{}

func (b *blobReader) Read(p []byte) (n int, err error) {
	defer bindings.RecoverError(&err)
	if b.offset >= b.Size {
		return 0, io.EOF
	}
	end := b.offset + len(p) + 1 // end is the first byte not included, not the last byte included, so add 1
	if end > b.Size {
		end = b.Size
	}
	slice := b.Call("slice", b.offset, end)
	fileReader := js.Global.Get("FileReader").New()
	var wg sync.WaitGroup
	wg.Add(1)
	fileReader.Set("onload", js.MakeFunc(func(this *js.Object, _ []*js.Object) interface{} {
		defer wg.Done()
		n = copy(p, js.Global.Get("Uint8Array").New(this.Get("result")).Interface().([]uint8))
		return nil
	}))
	fileReader.Call("readAsArrayBuffer", slice)
	wg.Wait()
	b.offset += n
	return
}

func (b *blobReader) Close() (err error) {
	defer bindings.RecoverError(&err)
	b.Call("close")
	return nil
}

func (d *db) DeleteAttachment(ctx context.Context, docID, rev, filename string) (newRev string, err error) {
	result, err := d.db.RemoveAttachment(ctx, docID, filename, rev)
	if err != nil {
		return "", err
	}
	return result.Get("rev").String(), nil
}

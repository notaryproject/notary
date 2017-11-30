package kivik

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/flimzy/diff"
	"github.com/flimzy/testy"

	"github.com/flimzy/kivik/driver"
	"github.com/flimzy/kivik/errors"
)

func TestClient(t *testing.T) {
	client := &Client{}
	db := &DB{client: client}
	result := db.Client()
	if result != client {
		t.Errorf("Unexpected result. Expected %p, got %p", client, result)
	}
}

func TestName(t *testing.T) {
	dbName := "foo"
	db := &DB{name: dbName}
	result := db.Name()
	if result != dbName {
		t.Errorf("Unexpected result. Expected %s, got %s", dbName, result)
	}
}

type dummyDB struct {
	driver.DB
}

var _ driver.DB = &dummyDB{}

func TestFlushNotSupported(t *testing.T) {
	db := &DB{
		driverDB: &dummyDB{},
	}
	err := db.Flush(context.Background())
	if StatusCode(err) != StatusNotImplemented {
		t.Errorf("Expected NotImplemented, got %s", err)
	}
}

type errorReader struct{}

var _ io.Reader = &errorReader{}

func (r *errorReader) Read(_ []byte) (int, error) {
	return 0, errors.New("errorReader")
}

func TestNormalizeFromJSON(t *testing.T) {
	type njTest struct {
		Name     string
		Input    interface{}
		Expected interface{}
		Status   int
		Error    string
	}
	tests := []njTest{
		{
			Name:     "Interface",
			Input:    int(5),
			Expected: int(5),
		},
		{
			Name:   "InvalidJSON",
			Input:  []byte(`invalid`),
			Status: StatusBadRequest,
			Error:  "invalid character 'i' looking for beginning of value",
		},
		{
			Name:     "Bytes",
			Input:    []byte(`{"foo":"bar"}`),
			Expected: map[string]interface{}{"foo": "bar"},
		},
		{
			Name:     "RawMessage",
			Input:    json.RawMessage(`{"foo":"bar"}`),
			Expected: map[string]interface{}{"foo": "bar"},
		},
		{
			Name:     "ioReader",
			Input:    strings.NewReader(`{"foo":"bar"}`),
			Expected: map[string]interface{}{"foo": "bar"},
		},
		{
			Name:   "ErrorReader",
			Input:  &errorReader{},
			Status: StatusUnknownError,
			Error:  "errorReader",
		},
	}
	for _, test := range tests {
		func(test njTest) {
			t.Run(test.Name, func(t *testing.T) {
				result, err := normalizeFromJSON(test.Input)
				var msg string
				var status int
				if err != nil {
					msg = err.Error()
					status = StatusCode(err)
				}
				if msg != test.Error || status != test.Status {
					t.Errorf("Unexpected error: %d %s", status, msg)
				}
				if err != nil {
					return
				}
				if d := diff.Interface(test.Expected, result); d != nil {
					t.Error(d)
				}
			})
		}(test)
	}
}

type legacyPutGrabber struct {
	driver.DB
	docID string
	doc   interface{}

	newRev string
	err    error
}

func (db *legacyPutGrabber) Put(_ context.Context, docID string, i interface{}) (string, error) {
	if db.docID != docID {
		return "", errors.Errorf("Unexpected docID: %s", docID)
	}
	if d := diff.Interface(db.doc, i); d != nil {
		return "", errors.Errorf("Unexpected doc: %s", d)
	}
	return db.newRev, db.err
}

type putGrabber struct {
	driver.DB
	driver.DBOpts
	docID string
	doc   interface{}
	opts  map[string]interface{}

	newRev string
	err    error
}

func (db *putGrabber) Put(_ context.Context, _ string, _ interface{}) (string, error) {
	panic("Put called")
}

func (db *putGrabber) PutOpts(_ context.Context, docID string, i interface{}, opts map[string]interface{}) (string, error) {
	if db.docID != docID {
		return "", errors.Errorf("Unexpected docID: %s", docID)
	}
	if d := diff.Interface(db.doc, i); d != nil {
		return "", errors.Errorf("Unexpected doc: %s", d)
	}
	if d := diff.Interface(db.opts, opts); d != nil {
		return "", errors.Errorf("Unexpected opts: %s", d)
	}
	return db.newRev, db.err
}

func TestPut(t *testing.T) {
	type putTest struct {
		name    string
		db      *DB
		docID   string
		input   interface{}
		options Options
		status  int
		err     string
		newRev  string
	}
	tests := []putTest{
		{
			name:   "no docID",
			status: StatusBadRequest,
			err:    "kivik: docID required",
		},
		{
			name: "db error",
			db: &DB{driverDB: &putGrabber{
				docID: "foo",
				err:   errors.Status(StatusBadRequest, "db error"),
			}},
			docID:  "foo",
			status: StatusBadRequest,
			err:    "db error",
		},
		{
			name: "Interface",
			db: &DB{driverDB: &putGrabber{
				docID:  "foo",
				newRev: "1-xxx",
				doc:    map[string]string{"foo": "bar"},
			}},
			docID:  "foo",
			input:  map[string]string{"foo": "bar"},
			newRev: "1-xxx",
		},
		{
			name:   "InvalidJSON",
			docID:  "foo",
			input:  []byte("Something bogus"),
			status: StatusBadRequest,
			err:    "invalid character 'S' looking for beginning of value",
		},
		{
			name: "Bytes",
			db: &DB{driverDB: &putGrabber{
				docID:  "foo",
				newRev: "1-xxx",
				doc:    map[string]interface{}{"foo": "bar"},
			}},
			docID:  "foo",
			input:  []byte(`{"foo":"bar"}`),
			newRev: "1-xxx",
		},
		{
			name: "RawMessage",
			db: &DB{driverDB: &putGrabber{
				docID:  "foo",
				newRev: "1-xxx",
				doc:    map[string]interface{}{"foo": "bar"},
			}},
			docID:  "foo",
			input:  json.RawMessage(`{"foo":"bar"}`),
			newRev: "1-xxx",
		},
		{
			name: "Reader",
			db: &DB{driverDB: &putGrabber{
				docID:  "foo",
				newRev: "1-xxx",
				doc:    map[string]interface{}{"foo": "bar"},
			}},
			docID:  "foo",
			input:  strings.NewReader(`{"foo":"bar"}`),
			newRev: "1-xxx",
		},
		{
			name:   "ErrorReader",
			docID:  "foo",
			input:  &errorReader{},
			status: StatusUnknownError,
			err:    "errorReader",
		},
		{
			name: "valid",
			db: &DB{driverDB: &putGrabber{
				docID:  "foo",
				newRev: "1-xxx",
				doc:    map[string]string{"foo": "bar"},
				opts:   map[string]interface{}{"opt": "opt"},
			}},
			docID:   "foo",
			input:   map[string]string{"foo": "bar"},
			options: Options{"opt": "opt"},
			newRev:  "1-xxx",
		},
		{
			name: "legacy",
			db: &DB{driverDB: &legacyPutGrabber{
				docID:  "foo",
				newRev: "1-xxx",
				doc:    map[string]string{"foo": "bar"},
			}},
			docID:  "foo",
			input:  map[string]string{"foo": "bar"},
			newRev: "1-xxx",
		},
	}
	for _, test := range tests {
		func(test putTest) {
			t.Run(test.name, func(t *testing.T) {
				newRev, err := test.db.Put(context.Background(), test.docID, test.input, test.options)
				testy.StatusError(t, test.err, test.status, err)
				if newRev != test.newRev {
					t.Errorf("Unexpected new rev: %s", newRev)
				}
			})
		}(test)
	}
}

func TestExtractDocID(t *testing.T) {
	type ediTest struct {
		name     string
		i        interface{}
		id       string
		expected bool
	}
	tests := []ediTest{
		{
			name: "nil",
		},
		{
			name: "string/interface map, no id",
			i: map[string]interface{}{
				"value": "foo",
			},
		},
		{
			name: "string/interface map, with id",
			i: map[string]interface{}{
				"_id": "foo",
			},
			id:       "foo",
			expected: true,
		},
		{
			name: "string/string map, with id",
			i: map[string]string{
				"_id": "foo",
			},
			id:       "foo",
			expected: true,
		},
		{
			name: "invalid JSON",
			i:    make(chan int),
		},
		{
			name: "valid JSON",
			i: struct {
				ID string `json:"_id"`
			}{ID: "oink"},
			id:       "oink",
			expected: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			id, ok := extractDocID(test.i)
			if ok != test.expected || test.id != id {
				t.Errorf("Expected %t/%s, got %t/%s", test.expected, test.id, ok, id)
			}
		})
	}
}

type legacyCreateDocGrabber struct {
	driver.DB
	doc interface{}

	id, rev string
	err     error
}

func (db *legacyCreateDocGrabber) CreateDoc(_ context.Context, doc interface{}) (string, string, error) {
	if d := diff.Interface(db.doc, doc); d != nil {
		return "", "", errors.Errorf("Unexpected doc: %s", d)
	}
	return db.id, db.rev, db.err
}

type createDocGrabber struct {
	driver.DB
	driver.DBOpts
	doc  interface{}
	opts map[string]interface{}

	id, rev string
	err     error
}

func (db *createDocGrabber) CreateDoc(_ context.Context, _ interface{}) (string, string, error) {
	panic("CreateDoc called")
}

func (db *createDocGrabber) CreateDocOpts(_ context.Context, doc interface{}, opts map[string]interface{}) (string, string, error) {
	if d := diff.Interface(db.doc, doc); d != nil {
		return "", "", errors.Errorf("Unexpected doc: %s", d)
	}
	if d := diff.Interface(db.opts, opts); d != nil {
		return "", "", errors.Errorf("Unexpected opts: %s", d)
	}
	return db.id, db.rev, db.err
}

func TestCreateDoc(t *testing.T) {
	tests := []struct {
		name       string
		db         *DB
		doc        interface{}
		options    Options
		docID, rev string
		status     int
		err        string
	}{
		{
			name:   "error",
			db:     &DB{driverDB: &createDocGrabber{err: errors.Status(StatusBadRequest, "create error")}},
			status: StatusBadRequest,
			err:    "create error",
		},
		{
			name: "success",
			db: &DB{driverDB: &createDocGrabber{
				id:   "foo",
				rev:  "1-xxx",
				doc:  map[string]string{"type": "test"},
				opts: map[string]interface{}{"opt": 1},
			}},
			doc:     map[string]string{"type": "test"},
			options: Options{"opt": 1},
			docID:   "foo",
			rev:     "1-xxx",
		},
		{
			name: "legacy",
			db: &DB{driverDB: &legacyCreateDocGrabber{
				id:  "foo",
				rev: "1-xxx",
				doc: map[string]string{"type": "test"},
			}},
			doc:   map[string]string{"type": "test"},
			docID: "foo",
			rev:   "1-xxx",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			docID, rev, err := test.db.CreateDoc(context.Background(), test.doc, test.options)
			testy.StatusError(t, test.err, test.status, err)
			if docID != test.docID || test.rev != test.rev {
				t.Errorf("Unexpected result: %s / %s", docID, rev)
			}
		})
	}
}

type legacyDeleteRecorder struct {
	driver.DB
	docID, rev string

	newRev string
	err    error
}

func (db *legacyDeleteRecorder) Delete(_ context.Context, docID, rev string) (string, error) {
	if db.docID != docID || db.rev != rev {
		return "", errors.Errorf("Unexpected docID/rev: %s/%s", docID, rev)
	}
	return db.newRev, db.err
}

type deleteRecorder struct {
	driver.DB
	driver.DBOpts
	docID, rev string
	opts       map[string]interface{}

	newRev string
	err    error
}

func (db *deleteRecorder) Delete(_ context.Context, _, _ string) (string, error) {
	panic("Delete called")
}

func (db *deleteRecorder) DeleteOpts(_ context.Context, docID, rev string, opts map[string]interface{}) (string, error) {
	if db.docID != docID || db.rev != rev {
		return "", errors.Errorf("Unexpected docID/rev: %s/%s", docID, rev)
	}
	if d := diff.Interface(db.opts, opts); d != nil {
		return "", errors.Errorf("Unexpected opts: %s", d)
	}
	return db.newRev, db.err
}

func TestDelete(t *testing.T) {
	tests := []struct {
		name       string
		db         *DB
		docID, rev string
		options    Options
		newRev     string
		status     int
		err        string
	}{
		{
			name:   "no doc ID",
			status: StatusBadRequest,
			err:    "kivik: docID required",
		},
		{
			name: "error",
			db: &DB{driverDB: &deleteRecorder{
				docID: "foo",
				err:   errors.Status(StatusBadRequest, "delete error"),
			}},
			docID:  "foo",
			status: StatusBadRequest,
			err:    "delete error",
		},
		{
			name: "success",
			db: &DB{driverDB: &deleteRecorder{
				docID:  "foo",
				rev:    "1-xxx",
				opts:   map[string]interface{}{"opt": 1},
				newRev: "2-xxx",
			}},
			docID:   "foo",
			rev:     "1-xxx",
			options: Options{"opt": 1},
			newRev:  "2-xxx",
		},
		{
			name: "legacy",
			db: &DB{driverDB: &legacyDeleteRecorder{
				docID:  "foo",
				rev:    "1-xxx",
				newRev: "2-xxx",
			}},
			docID:  "foo",
			rev:    "1-xxx",
			newRev: "2-xxx",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			newRev, err := test.db.Delete(context.Background(), test.docID, test.rev, test.options)
			testy.StatusError(t, test.err, test.status, err)
			if newRev != test.newRev {
				t.Errorf("Unexpected newRev: %s", newRev)
			}
		})
	}
}

type legacyPutAttRecorder struct {
	driver.DB
	docID, rev, filename, cType string
	body                        string

	newRev string
	err    error
}

func (db *legacyPutAttRecorder) PutAttachment(_ context.Context, docID, rev, filename, contentType string, body io.Reader) (string, error) {
	if db.docID != docID || db.rev != rev {
		return "", errors.Errorf("Unexpected id/rev: %s/%s", docID, rev)
	}
	if db.filename != filename || db.cType != contentType {
		return "", errors.Errorf("Unexpected file data: %s / %s", filename, contentType)
	}
	content, err := ioutil.ReadAll(body)
	if err != nil {
		panic(err)
	}
	if d := diff.Text(db.body, string(content)); d != nil {
		return "", errors.Errorf("Unexpected content: %s", d)
	}
	return db.newRev, db.err
}

type putAttRecorder struct {
	driver.DB
	driver.DBOpts
	docID, rev, filename, cType string
	body                        string
	opts                        map[string]interface{}

	newRev string
	err    error
}

func (db *putAttRecorder) PutAttachment(_ context.Context, _, _, _, _ string, _ io.Reader) (string, error) {
	panic("PutAttachment called")
}

func (db *putAttRecorder) PutAttachmentOpts(_ context.Context, docID, rev, filename, contentType string, body io.Reader, opts map[string]interface{}) (string, error) {
	if db.docID != docID || db.rev != rev {
		return "", errors.Errorf("Unexpected id/rev: %s/%s", docID, rev)
	}
	if db.filename != filename || db.cType != contentType {
		return "", errors.Errorf("Unexpected file data: %s / %s", filename, contentType)
	}
	content, err := ioutil.ReadAll(body)
	if err != nil {
		panic(err)
	}
	if d := diff.Text(db.body, string(content)); d != nil {
		return "", errors.Errorf("Unexpected content: %s", d)
	}
	if d := diff.Interface(db.opts, opts); d != nil {
		return "", errors.Errorf("Unexpected options: %s", d)
	}

	return db.newRev, db.err
}

func TestPutAttachment(t *testing.T) {
	tests := []struct {
		name       string
		db         *DB
		docID, rev string
		att        *Attachment
		options    Options
		newRev     string
		status     int
		err        string

		body string
	}{
		{
			name:  "db error",
			docID: "foo",
			db: &DB{driverDB: &putAttRecorder{
				docID:    "foo",
				filename: "foo.txt",
				err:      errors.Status(StatusBadRequest, "db error"),
			}},
			att: &Attachment{
				Filename:   "foo.txt",
				ReadCloser: ioutil.NopCloser(strings.NewReader("")),
			},
			status: StatusBadRequest,
			err:    "db error",
		},
		{
			name:   "no doc id",
			status: StatusBadRequest,
			err:    "kivik: docID required",
		},
		{
			name:   "no filename",
			docID:  "foo",
			att:    &Attachment{},
			status: StatusBadRequest,
			err:    "kivik: filename required",
		},
		{
			name:  "success",
			docID: "foo",
			rev:   "1-xxx",
			db: &DB{driverDB: &putAttRecorder{
				docID:    "foo",
				rev:      "1-xxx",
				filename: "foo.txt",
				cType:    "text/plain",
				body:     "Test file",
				opts:     map[string]interface{}{"opt": 1},
				newRev:   "2-xxx",
			}},
			att: &Attachment{
				Filename:    "foo.txt",
				ContentType: "text/plain",
				ReadCloser:  ioutil.NopCloser(strings.NewReader("Test file")),
			},
			options: Options{"opt": 1},
			newRev:  "2-xxx",
			body:    "Test file",
		},
		{
			name:  "legacy",
			docID: "foo",
			rev:   "1-xxx",
			db: &DB{driverDB: &legacyPutAttRecorder{
				docID:    "foo",
				rev:      "1-xxx",
				filename: "foo.txt",
				cType:    "text/plain",
				body:     "Test file",
				newRev:   "2-xxx",
			}},
			att: &Attachment{
				Filename:    "foo.txt",
				ContentType: "text/plain",
				ReadCloser:  ioutil.NopCloser(strings.NewReader("Test file")),
			},
			newRev: "2-xxx",
			body:   "Test file",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			newRev, err := test.db.PutAttachment(context.Background(), test.docID, test.rev, test.att, test.options)
			testy.StatusError(t, test.err, test.status, err)
			if newRev != test.newRev {
				t.Errorf("Unexpected newRev: %s", newRev)
			}
		})
	}
}

type legacyMockDelAtt struct {
	driver.DB
	docID, rev, filename string

	newRev string
	err    error
}

func (db *legacyMockDelAtt) DeleteAttachment(ctx context.Context, docID, rev, filename string) (string, error) {
	if db.docID != docID || db.rev != rev {
		return "", errors.Errorf("Unexpected id/rev: %s/%s", docID, rev)
	}
	if db.filename != filename {
		return "", errors.Errorf("Unexpected filename: %s", filename)
	}
	return db.newRev, db.err
}

type mockDelAtt struct {
	driver.DB
	driver.DBOpts
	docID, rev, filename string
	opts                 map[string]interface{}

	newRev string
	err    error
}

func (db *mockDelAtt) DeleteAttachment(_ context.Context, _, _, _ string) (string, error) {
	panic("DeleteAttachment called")
}

func (db *mockDelAtt) DeleteAttachmentOpts(_ context.Context, docID, rev, filename string, opts map[string]interface{}) (string, error) {
	if db.docID != docID || db.rev != rev {
		return "", errors.Errorf("Unexpected id/rev: %s/%s", docID, rev)
	}
	if db.filename != filename {
		return "", errors.Errorf("Unexpected filename: %s", filename)
	}
	if d := diff.Interface(db.opts, opts); d != nil {
		return "", errors.Errorf("Unexpected options: %s", d)
	}
	return db.newRev, db.err
}

func TestDeleteAttachment(t *testing.T) {
	tests := []struct {
		name                 string
		db                   *DB
		docID, rev, filename string
		options              Options

		newRev string
		status int
		err    string
	}{
		{
			name:   "missing doc id",
			status: StatusBadRequest,
			err:    "kivik: docID required",
		},
		{
			name:   "missing filename",
			docID:  "foo",
			status: StatusBadRequest,
			err:    "kivik: filename required",
		},
		{
			name:     "db error",
			docID:    "foo",
			filename: "foo.txt",
			db: &DB{driverDB: &mockDelAtt{
				docID:    "foo",
				filename: "foo.txt",
				err:      errors.Status(StatusBadRequest, "db error"),
			}},
			status: StatusBadRequest,
			err:    "db error",
		},
		{
			name: "success",
			db: &DB{driverDB: &mockDelAtt{
				docID:    "foo",
				rev:      "1-xxx",
				filename: "foo.txt",
				opts:     map[string]interface{}{"opt": 1},

				newRev: "2-xxx",
			}},
			docID:    "foo",
			rev:      "1-xxx",
			filename: "foo.txt",
			options:  Options{"opt": 1},
			newRev:   "2-xxx",
		},
		{
			name: "legacy",
			db: &DB{driverDB: &legacyMockDelAtt{
				docID:    "foo",
				rev:      "1-xxx",
				filename: "foo.txt",

				newRev: "2-xxx",
			}},
			docID:    "foo",
			rev:      "1-xxx",
			filename: "foo.txt",
			newRev:   "2-xxx",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			newRev, err := test.db.DeleteAttachment(context.Background(), test.docID, test.rev, test.filename, test.options)
			testy.StatusError(t, test.err, test.status, err)
			if newRev != test.newRev {
				t.Errorf("Unexpected new rev: %s", newRev)
			}
		})
	}
}

type mockOldAttGetter struct {
	driver.DB
	docID, rev, filename string

	cType   string
	md5     driver.MD5sum
	content io.ReadCloser
	err     error
}

func (db *mockOldAttGetter) GetAttachment(_ context.Context, docID, rev, filename string) (string, driver.MD5sum, io.ReadCloser, error) {
	if docID != db.docID {
		return "", driver.MD5sum{}, nil, errors.Errorf("Unexpected docID: %s", docID)
	}
	if rev != db.rev {
		return "", driver.MD5sum{}, nil, errors.Errorf("Unexpected rev: %s", rev)
	}
	if filename != db.filename {
		return "", driver.MD5sum{}, nil, errors.Errorf("Unexpected filename: %s", filename)
	}
	return db.cType, db.md5, db.content, db.err
}

type mockAttGetter struct {
	driver.DB
	driver.DBOpts
	docID, rev, filename string
	opts                 map[string]interface{}

	cType   string
	md5     driver.MD5sum
	content io.ReadCloser
	err     error
}

func (db *mockAttGetter) GetAttachmentOpts(_ context.Context, docID, rev, filename string, opts map[string]interface{}) (string, driver.MD5sum, io.ReadCloser, error) {
	if docID != db.docID {
		return "", driver.MD5sum{}, nil, errors.Errorf("Unexpected docID: %s", docID)
	}
	if rev != db.rev {
		return "", driver.MD5sum{}, nil, errors.Errorf("Unexpected rev: %s", rev)
	}
	if filename != db.filename {
		return "", driver.MD5sum{}, nil, errors.Errorf("Unexpected filename: %s", filename)
	}
	if d := diff.Interface(db.opts, opts); d != nil {
		return "", driver.MD5sum{}, nil, errors.Errorf("Unexpected options: %s", d)
	}
	return db.cType, db.md5, db.content, db.err
}

func TestGetAttachment(t *testing.T) {
	tests := []struct {
		name                 string
		db                   *DB
		docID, rev, filename string
		options              Options

		content  string
		expected *Attachment
		status   int
		err      string
	}{
		{
			name: "legacy, error",
			db: &DB{driverDB: &mockOldAttGetter{
				docID:    "foo",
				filename: "foo.txt",
				err:      errors.New("fail"),
			}},
			docID:    "foo",
			filename: "foo.txt",
			status:   500,
			err:      "fail",
		},
		{
			name: "legacy, success",
			db: &DB{driverDB: &mockOldAttGetter{
				docID:    "foo",
				rev:      "1-xxx",
				filename: "foo.txt",
				cType:    "text/plain",
				md5:      driver.MD5sum{0x01},
				content:  body("Test"),
			}},
			docID:    "foo",
			rev:      "1-xxx",
			filename: "foo.txt",
			content:  "Test",
			expected: &Attachment{
				Filename:    "foo.txt",
				ContentType: "text/plain",
				MD5:         driver.MD5sum{0x01},
			},
		},
		{
			name: "new, error",
			db: &DB{driverDB: &mockAttGetter{
				docID:    "foo",
				filename: "foo.txt",
				err:      errors.New("fail"),
			}},
			docID:    "foo",
			filename: "foo.txt",
			status:   500,
			err:      "fail",
		},
		{
			name: "new, success",
			db: &DB{driverDB: &mockAttGetter{
				docID:    "foo",
				rev:      "1-xxx",
				filename: "foo.txt",
				opts:     map[string]interface{}{"foo": "bar"},
				cType:    "text/plain",
				md5:      driver.MD5sum{0x01},
				content:  body("Test"),
			}},
			docID:    "foo",
			rev:      "1-xxx",
			filename: "foo.txt",
			options:  Options{"foo": "bar"},
			content:  "Test",
			expected: &Attachment{
				Filename:    "foo.txt",
				ContentType: "text/plain",
				MD5:         driver.MD5sum{0x01},
			},
		},
		{
			name:   "no docID",
			status: StatusBadRequest,
			err:    "kivik: docID required",
		},
		{
			name:   "no filename",
			docID:  "foo",
			status: StatusBadRequest,
			err:    "kivik: filename required",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.db.GetAttachment(context.Background(), test.docID, test.rev, test.filename, test.options)
			testy.StatusError(t, test.err, test.status, err)
			content, err := ioutil.ReadAll(result)
			if err != nil {
				t.Fatal(err)
			}
			if d := diff.Text(test.content, string(content)); d != nil {
				t.Errorf("Unexpected content:\n%s", d)
			}
			result.ReadCloser = nil
			if d := diff.Interface(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}

type mockOldAttMetaer struct {
	driver.DB
	docID, rev, filename string

	cType string
	md5   driver.MD5sum
	err   error
}

var _ driver.OldAttachmentMetaer = &mockOldAttMetaer{}

func (db *mockOldAttMetaer) GetAttachmentMeta(_ context.Context, docID, rev, filename string) (string, driver.MD5sum, error) {
	if docID != db.docID {
		return "", driver.MD5sum{}, errors.Errorf("Unexpected docID: %s", docID)
	}
	if rev != db.rev {
		return "", driver.MD5sum{}, errors.Errorf("Unexpected rev: %s", rev)
	}
	if filename != db.filename {
		return "", driver.MD5sum{}, errors.Errorf("Unexpected filename: %s", filename)
	}
	return db.cType, db.md5, db.err
}

type mockAttMetaer struct {
	driver.DB
	docID, rev, filename string
	opts                 map[string]interface{}

	cType string
	md5   driver.MD5sum
	err   error
}

var _ driver.AttachmentMetaer = &mockAttMetaer{}

func (db *mockAttMetaer) GetAttachmentMeta(_ context.Context, docID, rev, filename string, opts map[string]interface{}) (string, driver.MD5sum, error) {
	if docID != db.docID {
		return "", driver.MD5sum{}, errors.Errorf("Unexpected docID: %s", docID)
	}
	if rev != db.rev {
		return "", driver.MD5sum{}, errors.Errorf("Unexpected rev: %s", rev)
	}
	if filename != db.filename {
		return "", driver.MD5sum{}, errors.Errorf("Unexpected filename: %s", filename)
	}
	if d := diff.Interface(db.opts, opts); d != nil {
		return "", driver.MD5sum{}, errors.Errorf("Unexpected options: %s", d)
	}
	return db.cType, db.md5, db.err
}

func TestGetAttachmentMeta(t *testing.T) {
	tests := []struct {
		name                 string
		db                   *DB
		docID, rev, filename string
		options              Options

		expected *Attachment
		status   int
		err      string
	}{
		{
			name: "plain db, error",
			db: &DB{driverDB: &mockAttGetter{
				docID:    "foo",
				filename: "foo.txt",
				err:      errors.New("fail"),
			}},
			docID:    "foo",
			filename: "foo.txt",
			status:   500,
			err:      "fail",
		},
		{
			name: "plain db, success",
			db: &DB{driverDB: &mockAttGetter{
				docID:    "foo",
				rev:      "1-xxx",
				filename: "foo.txt",
				cType:    "text/plain",
				md5:      driver.MD5sum{0x01},
				content:  body("Test"),
			}},
			docID:    "foo",
			rev:      "1-xxx",
			filename: "foo.txt",
			expected: &Attachment{
				Filename:    "foo.txt",
				ContentType: "text/plain",
				MD5:         driver.MD5sum{0x01},
			},
		},
		{
			name: "legacy metaer, error",
			db: &DB{driverDB: &mockOldAttMetaer{
				docID:    "foo",
				filename: "foo.txt",
				err:      errors.New("fail"),
			}},
			docID:    "foo",
			filename: "foo.txt",
			status:   500,
			err:      "fail",
		},
		{
			name: "legacy metaer, success",
			db: &DB{driverDB: &mockOldAttMetaer{
				docID:    "foo",
				rev:      "1-xxx",
				filename: "foo.txt",
				cType:    "text/plain",
				md5:      driver.MD5sum{0x01},
			}},
			docID:    "foo",
			rev:      "1-xxx",
			filename: "foo.txt",
			expected: &Attachment{
				Filename:    "foo.txt",
				ContentType: "text/plain",
				MD5:         driver.MD5sum{0x01},
			},
		},
		{
			name: "new metaer, error",
			db: &DB{driverDB: &mockAttMetaer{
				docID:    "foo",
				filename: "foo.txt",
				err:      errors.New("fail"),
			}},
			docID:    "foo",
			filename: "foo.txt",
			status:   500,
			err:      "fail",
		},
		{
			name: "new metaer, success",
			db: &DB{driverDB: &mockAttMetaer{
				docID:    "foo",
				rev:      "1-xxx",
				filename: "foo.txt",
				cType:    "text/plain",
				md5:      driver.MD5sum{0x01},
				opts:     map[string]interface{}{"foo": "bar"},
			}},
			docID:    "foo",
			rev:      "1-xxx",
			filename: "foo.txt",
			options:  Options{"foo": "bar"},
			expected: &Attachment{
				Filename:    "foo.txt",
				ContentType: "text/plain",
				MD5:         driver.MD5sum{0x01},
			},
		},
		{
			name:   "no doc id",
			status: StatusBadRequest,
			err:    "kivik: docID required",
		},
		{
			name:   "no filename",
			docID:  "foo",
			status: StatusBadRequest,
			err:    "kivik: filename required",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.db.GetAttachmentMeta(context.Background(), test.docID, test.rev, test.filename, test.options)
			testy.StatusError(t, test.err, test.status, err)
			if d := diff.Interface(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}

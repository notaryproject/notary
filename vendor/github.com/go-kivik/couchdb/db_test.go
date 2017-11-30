package couchdb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/flimzy/diff"
	"github.com/flimzy/testy"
	"github.com/go-kivik/couchdb/chttp"

	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/driver"
	"github.com/flimzy/kivik/errors"
)

func TestAllDocs(t *testing.T) {
	db := newTestDB(nil, errors.New("test error"))
	_, err := db.AllDocs(context.Background(), nil)
	testy.Error(t, "Get http://example.com/testdb/_all_docs: test error", err)
}

func TestQuery(t *testing.T) {
	db := newTestDB(nil, errors.New("test error"))
	_, err := db.Query(context.Background(), "ddoc", "view", nil)
	testy.Error(t, "Get http://example.com/testdb/_design/ddoc/_view/view: test error", err)
}

func TestGet(t *testing.T) {
	tests := []struct {
		name     string
		db       *db
		id       string
		options  map[string]interface{}
		expected string
		status   int
		err      string
	}{
		{
			name:   "missing doc ID",
			status: kivik.StatusBadRequest,
			err:    "kivik: docID required",
		},
		{
			name:    "invalid options",
			id:      "foo",
			options: map[string]interface{}{"foo": make(chan int)},
			status:  kivik.StatusBadRequest,
			err:     "kivik: invalid type chan int for options",
		},
		{
			name:   "network failure",
			id:     "foo",
			db:     newTestDB(nil, errors.New("net error")),
			status: kivik.StatusNetworkError,
			err:    "Get http://example.com/testdb/foo: net error",
		},
		{
			name: "error response",
			id:   "foo",
			db: newTestDB(&http.Response{
				StatusCode: kivik.StatusBadRequest,
				Body:       ioutil.NopCloser(strings.NewReader("")),
			}, nil),
			status: kivik.StatusBadRequest,
			err:    "Bad Request",
		},
		{
			name: "status OK",
			id:   "foo",
			db: newTestDB(&http.Response{
				StatusCode: kivik.StatusOK,
				Body:       ioutil.NopCloser(strings.NewReader("some response")),
			}, nil),
			expected: "some response",
		},
		{
			name: "body read failure",
			id:   "foo",
			db: newTestDB(&http.Response{
				StatusCode: kivik.StatusOK,
				Body:       errorReadCloser{errors.New("read error")},
			}, nil),
			status: kivik.StatusUnknownError,
			err:    "read error",
		},
		{
			name: "If-None-Match",
			db: newCustomDB(func(req *http.Request) (*http.Response, error) {
				if err := consume(req.Body); err != nil {
					return nil, err
				}
				if inm := req.Header.Get("If-None-Match"); inm != `"foo"` {
					return nil, errors.Errorf(`If-None-Match: %s != "foo"`, inm)
				}
				return nil, errors.New("success")
			}),
			id:      "foo",
			options: map[string]interface{}{OptionIfNoneMatch: "foo"},
			status:  kivik.StatusNetworkError,
			err:     "Get http://example.com/testdb/foo: success",
		},
		{
			name:    "invalid If-None-Match value",
			id:      "foo",
			options: map[string]interface{}{OptionIfNoneMatch: 123},
			status:  kivik.StatusBadRequest,
			err:     "kivik: option 'If-None-Match' must be string, not int",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.db.Get(context.Background(), test.id, test.options)
			testy.StatusError(t, test.err, test.status, err)
			if string(result) != test.expected {
				t.Errorf("Unexpected result: %s", string(result))
			}
		})
	}
}

func TestCreateDoc(t *testing.T) {
	db := newTestDB(nil, errors.New("error"))
	_, _, err := db.CreateDoc(context.Background(), map[string]string{"foo": "bar"})
	testy.Error(t, "Post http://example.com/testdb: error", err)
}

func TestCreateDocOpts(t *testing.T) {
	tests := []struct {
		name    string
		db      *db
		doc     interface{}
		options map[string]interface{}
		id, rev string
		status  int
		err     string
	}{
		{
			name:   "network error",
			db:     newTestDB(nil, errors.New("foo error")),
			status: kivik.StatusNetworkError,
			err:    "Post http://example.com/testdb: foo error",
		},
		{
			name:   "invalid doc",
			doc:    make(chan int),
			db:     newTestDB(nil, errors.New("")),
			status: kivik.StatusBadRequest,
			err:    "Post http://example.com/testdb: json: unsupported type: chan int",
		},
		{
			name: "error response",
			doc:  map[string]interface{}{"foo": "bar"},
			db: newTestDB(&http.Response{
				StatusCode: kivik.StatusBadRequest,
				Body:       ioutil.NopCloser(strings.NewReader("")),
			}, nil),
			status: kivik.StatusBadRequest,
			err:    "Bad Request",
		},
		{
			name: "invalid JSON response",
			doc:  map[string]interface{}{"foo": "bar"},
			db: newTestDB(&http.Response{
				StatusCode: kivik.StatusOK,
				Body:       ioutil.NopCloser(strings.NewReader("invalid json")),
			}, nil),
			status: kivik.StatusBadResponse,
			err:    "invalid character 'i' looking for beginning of value",
		},
		{
			name: "success, 1.6.1",
			doc:  map[string]interface{}{"foo": "bar"},
			db: newTestDB(&http.Response{
				StatusCode: kivik.StatusOK,
				Header: map[string][]string{
					"Server":         {"CouchDB/1.6.1 (Erlang OTP/17)"},
					"Location":       {"http://localhost:5984/foo/43734cf3ce6d5a37050c050bb600006b"},
					"ETag":           {`"1-4c6114c65e295552ab1019e2b046b10e"`},
					"Date":           {"Wed, 25 Oct 2017 10:38:38 GMT"},
					"Content-Type":   {"text/plain; charset=utf-8"},
					"Content-Length": {"95"},
					"Cache-Control":  {"must-revalidate"},
				},
				Body: ioutil.NopCloser(strings.NewReader(`{"ok":true,"id":"43734cf3ce6d5a37050c050bb600006b","rev":"1-4c6114c65e295552ab1019e2b046b10e"}
`)),
			}, nil),
			id:  "43734cf3ce6d5a37050c050bb600006b",
			rev: "1-4c6114c65e295552ab1019e2b046b10e",
		},
		{
			name:    "batch mode",
			db:      newTestDB(nil, errors.New("success")),
			doc:     map[string]string{"foo": "bar"},
			options: map[string]interface{}{"batch": "ok"},
			status:  kivik.StatusNetworkError,
			err:     "Post http://example.com/testdb?batch=ok: success",
		},
		{
			name: "full commit",
			db: newCustomDB(func(req *http.Request) (*http.Response, error) {
				if err := consume(req.Body); err != nil {
					return nil, err
				}
				if fullCommit := req.Header.Get("X-Couch-Full-Commit"); fullCommit != "true" {
					return nil, errors.New("X-Couch-Full-Commit not true")
				}
				return nil, errors.New("success")
			}),
			options: map[string]interface{}{OptionFullCommit: true},
			status:  kivik.StatusNetworkError,
			err:     "Post http://example.com/testdb: success",
		},
		{
			name:    "invalid options",
			db:      &db{},
			options: map[string]interface{}{"foo": make(chan int)},
			status:  kivik.StatusBadRequest,
			err:     "kivik: invalid type chan int for options",
		},
		{
			name:    "invalid full commit type",
			db:      &db{},
			options: map[string]interface{}{OptionFullCommit: 123},
			status:  kivik.StatusBadRequest,
			err:     "kivik: option 'X-Couch-Full-Commit' must be bool, not int",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			id, rev, err := test.db.CreateDocOpts(context.Background(), test.doc, test.options)
			testy.StatusError(t, test.err, test.status, err)
			if test.id != id || test.rev != rev {
				t.Errorf("Unexpected results: ID=%s rev=%s", id, rev)
			}
		})
	}
}

func TestStats(t *testing.T) {
	tests := []struct {
		name     string
		db       *db
		expected *driver.DBStats
		status   int
		err      string
	}{
		{
			name:   "network error",
			db:     newTestDB(nil, errors.New("net error")),
			status: kivik.StatusNetworkError,
			err:    "Get http://example.com/testdb: net error",
		},
		{
			name: "1.6.1",
			db: newTestDB(&http.Response{
				StatusCode: kivik.StatusOK,
				Header: http.Header{
					"Server":         {"CouchDB/1.6.1 (Erlang OTP/17)"},
					"Date":           {"Thu, 26 Oct 2017 12:58:14 GMT"},
					"Content-Type":   {"text/plain; charset=utf-8"},
					"Content-Length": {"235"},
					"Cache-Control":  {"must-revalidate"},
				},
				Body: ioutil.NopCloser(strings.NewReader(`{"db_name":"_users","doc_count":3,"doc_del_count":14,"update_seq":31,"purge_seq":0,"compact_running":false,"disk_size":127080,"data_size":6028,"instance_start_time":"1509022681259533","disk_format_version":6,"committed_update_seq":31}`)),
			}, nil),
			expected: &driver.DBStats{
				Name:         "_users",
				DocCount:     3,
				DeletedCount: 14,
				UpdateSeq:    "31",
				DiskSize:     127080,
				ActiveSize:   6028,
			},
		},
		{
			name: "2.0.0",
			db: newTestDB(&http.Response{
				StatusCode: kivik.StatusOK,
				Header: http.Header{
					"Server":              {"CouchDB/2.0.0 (Erlang OTP/17)"},
					"Date":                {"Thu, 26 Oct 2017 13:01:13 GMT"},
					"Content-Type":        {"application/json"},
					"Content-Length":      {"429"},
					"Cache-Control":       {"must-revalidate"},
					"X-Couch-Request-ID":  {"2486f27546"},
					"X-CouchDB-Body-Time": {"0"},
				},
				Body: ioutil.NopCloser(strings.NewReader(`{"db_name":"_users","update_seq":"13-g1AAAAEzeJzLYWBg4MhgTmHgzcvPy09JdcjLz8gvLskBCjMlMiTJ____PyuRAYeCJAUgmWQPVsOCS40DSE08WA0rLjUJIDX1eO3KYwGSDA1ACqhsPiF1CyDq9mclMuFVdwCi7j4hdQ8g6kDuywIAkRBjAw","sizes":{"file":87323,"external":2495,"active":6082},"purge_seq":0,"other":{"data_size":2495},"doc_del_count":6,"doc_count":1,"disk_size":87323,"disk_format_version":6,"data_size":6082,"compact_running":false,"instance_start_time":"0"}`)),
			}, nil),
			expected: &driver.DBStats{
				Name:         "_users",
				DocCount:     1,
				DeletedCount: 6,
				UpdateSeq:    "13-g1AAAAEzeJzLYWBg4MhgTmHgzcvPy09JdcjLz8gvLskBCjMlMiTJ____PyuRAYeCJAUgmWQPVsOCS40DSE08WA0rLjUJIDX1eO3KYwGSDA1ACqhsPiF1CyDq9mclMuFVdwCi7j4hdQ8g6kDuywIAkRBjAw",
				DiskSize:     87323,
				ActiveSize:   6082,
				ExternalSize: 2495,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.db.Stats(context.Background())
			testy.StatusError(t, test.err, test.status, err)
			if d := diff.Interface(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestOptionsToParams(t *testing.T) {
	type otpTest struct {
		Name     string
		Input    map[string]interface{}
		Expected url.Values
		Error    string
	}
	tests := []otpTest{
		{
			Name:     "String",
			Input:    map[string]interface{}{"foo": "bar"},
			Expected: map[string][]string{"foo": {"bar"}},
		},
		{
			Name:     "StringSlice",
			Input:    map[string]interface{}{"foo": []string{"bar", "baz"}},
			Expected: map[string][]string{"foo": {"bar", "baz"}},
		},
		{
			Name:     "Bool",
			Input:    map[string]interface{}{"foo": true},
			Expected: map[string][]string{"foo": {"true"}},
		},
		{
			Name:     "Int",
			Input:    map[string]interface{}{"foo": 123},
			Expected: map[string][]string{"foo": {"123"}},
		},
		{
			Name:  "Error",
			Input: map[string]interface{}{"foo": []byte("foo")},
			Error: "kivik: invalid type []uint8 for options",
		},
	}
	for _, test := range tests {
		func(test otpTest) {
			t.Run(test.Name, func(t *testing.T) {
				params, err := optionsToParams(test.Input)
				var msg string
				if err != nil {
					msg = err.Error()
				}
				if msg != test.Error {
					t.Errorf("Error\n\tExpected: %s\n\t  Actual: %s\n", test.Error, msg)
				}
				if d := diff.Interface(test.Expected, params); d != nil {
					t.Errorf("Params not as expected:\n%s\n", d)
				}
			})
		}(test)
	}
}

func TestCompact(t *testing.T) {
	tests := []struct {
		name   string
		db     *db
		status int
		err    string
	}{
		{
			name:   "net error",
			db:     newTestDB(nil, errors.New("net error")),
			status: kivik.StatusNetworkError,
			err:    "Post http://example.com/testdb/_compact: net error",
		},
		{
			name: "1.6.1",
			db: newCustomDB(func(req *http.Request) (*http.Response, error) {
				if ct, _, _ := mime.ParseMediaType(req.Header.Get("Content-Type")); ct != "application/json" {
					return nil, fmt.Errorf("Expected Content-Type: application/json, got %s", ct)
				}
				return &http.Response{
					StatusCode: kivik.StatusOK,
					Header: http.Header{
						"Server":         {"CouchDB/1.6.1 (Erlang OTP/17)"},
						"Date":           {"Thu, 26 Oct 2017 13:07:52 GMT"},
						"Content-Type":   {"text/plain; charset=utf-8"},
						"Content-Length": {"12"},
						"Cache-Control":  {"must-revalidate"},
					},
					Body: ioutil.NopCloser(strings.NewReader(`{"ok":true}`)),
				}, nil
			}),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.db.Compact(context.Background())
			testy.StatusError(t, test.err, test.status, err)
		})
	}
}

func TestCompactView(t *testing.T) {
	tests := []struct {
		name   string
		db     *db
		id     string
		status int
		err    string
	}{
		{
			name:   "no ddoc",
			status: kivik.StatusBadRequest,
			err:    "kivik: ddocID required",
		},
		{
			name:   "network error",
			db:     newTestDB(nil, errors.New("net error")),
			id:     "foo",
			status: kivik.StatusNetworkError,
			err:    "Post http://example.com/testdb/_compact/foo: net error",
		},
		{
			name: "1.6.1",
			id:   "foo",
			db: newCustomDB(func(req *http.Request) (*http.Response, error) {
				if ct, _, _ := mime.ParseMediaType(req.Header.Get("Content-Type")); ct != "application/json" {
					return nil, fmt.Errorf("Expected Content-Type: application/json, got %s", ct)
				}
				return &http.Response{
					StatusCode: kivik.StatusAccepted,
					Header: http.Header{
						"Server":         {"CouchDB/1.6.1 (Erlang OTP/17)"},
						"Date":           {"Thu, 26 Oct 2017 13:07:52 GMT"},
						"Content-Type":   {"text/plain; charset=utf-8"},
						"Content-Length": {"12"},
						"Cache-Control":  {"must-revalidate"},
					},
					Body: ioutil.NopCloser(strings.NewReader(`{"ok":true}`)),
				}, nil
			}),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.db.CompactView(context.Background(), test.id)
			testy.StatusError(t, test.err, test.status, err)
		})
	}
}

func TestViewCleanup(t *testing.T) {
	tests := []struct {
		name   string
		db     *db
		status int
		err    string
	}{
		{
			name:   "net error",
			db:     newTestDB(nil, errors.New("net error")),
			status: kivik.StatusNetworkError,
			err:    "Post http://example.com/testdb/_view_cleanup: net error",
		},
		{
			name: "1.6.1",
			db: newCustomDB(func(req *http.Request) (*http.Response, error) {
				if ct, _, _ := mime.ParseMediaType(req.Header.Get("Content-Type")); ct != "application/json" {
					return nil, fmt.Errorf("Expected Content-Type: application/json, got %s", ct)
				}
				return &http.Response{
					StatusCode: kivik.StatusOK,
					Header: http.Header{
						"Server":         {"CouchDB/1.6.1 (Erlang OTP/17)"},
						"Date":           {"Thu, 26 Oct 2017 13:07:52 GMT"},
						"Content-Type":   {"text/plain; charset=utf-8"},
						"Content-Length": {"12"},
						"Cache-Control":  {"must-revalidate"},
					},
					Body: ioutil.NopCloser(strings.NewReader(`{"ok":true}`)),
				}, nil
			}),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.db.ViewCleanup(context.Background())
			testy.StatusError(t, test.err, test.status, err)
		})
	}
}

func TestPut(t *testing.T) {
	db := &db{}
	_, err := db.Put(context.Background(), "", nil)
	testy.Error(t, "kivik: docID required", err)
}

func TestPutOpts(t *testing.T) {
	tests := []struct {
		name    string
		db      *db
		id      string
		doc     interface{}
		options map[string]interface{}
		rev     string
		status  int
		err     string
	}{
		{
			name:   "missing docID",
			status: kivik.StatusBadRequest,
			err:    "kivik: docID required",
		},
		{
			name:   "network error",
			id:     "foo",
			db:     newTestDB(nil, errors.New("net error")),
			status: kivik.StatusNetworkError,
			err:    "Put http://example.com/testdb/foo: net error",
		},
		{
			name: "error response",
			id:   "foo",
			db: newTestDB(&http.Response{
				StatusCode: kivik.StatusBadRequest,
				Body:       ioutil.NopCloser(strings.NewReader("")),
			}, nil),
			status: kivik.StatusBadRequest,
			err:    "Bad Request",
		},
		{
			name: "invalid JSON response",
			id:   "foo",
			db: newTestDB(&http.Response{
				StatusCode: kivik.StatusOK,
				Body:       ioutil.NopCloser(strings.NewReader("invalid json")),
			}, nil),
			status: kivik.StatusBadResponse,
			err:    "invalid character 'i' looking for beginning of value",
		},
		{
			name: "invalid document",
			id:   "foo",
			doc:  make(chan int),
			db: newTestDB(&http.Response{
				StatusCode: kivik.StatusOK,
				Body:       ioutil.NopCloser(strings.NewReader("")),
			}, nil),
			status: kivik.StatusBadRequest,
			err:    "Put http://example.com/testdb/foo: json: unsupported type: chan int",
		},
		{
			name: "doc created, 1.6.1",
			id:   "foo",
			doc:  map[string]string{"foo": "bar"},
			db: newTestDB(&http.Response{
				StatusCode: kivik.StatusCreated,
				Header: http.Header{
					"Server":         {"CouchDB/1.6.1 (Erlang OTP/17)"},
					"Location":       {"http://localhost:5984/foo/foo"},
					"ETag":           {`"1-4c6114c65e295552ab1019e2b046b10e"`},
					"Date":           {"Wed, 25 Oct 2017 12:33:09 GMT"},
					"Content-Type":   {"text/plain; charset=utf-8"},
					"Content-Length": {"66"},
					"Cache-Control":  {"must-revalidate"},
				},
				Body: ioutil.NopCloser(strings.NewReader(`{"ok":true,"id":"foo","rev":"1-4c6114c65e295552ab1019e2b046b10e"}`)),
			}, nil),
			rev: "1-4c6114c65e295552ab1019e2b046b10e",
		},
		{
			name: "unexpected id in response",
			id:   "foo",
			doc:  map[string]string{"foo": "bar"},
			db: newTestDB(&http.Response{
				StatusCode: kivik.StatusCreated,
				Body:       ioutil.NopCloser(strings.NewReader(`{"ok":true,"id":"unexpected","rev":"1-4c6114c65e295552ab1019e2b046b10e"}`)),
			}, nil),
			status: kivik.StatusBadResponse,
			err:    "modified document ID \\(unexpected\\) does not match that requested \\(foo\\)",
		},
		{
			name: "full commit",
			db: newCustomDB(func(req *http.Request) (*http.Response, error) {
				if err := consume(req.Body); err != nil {
					return nil, err
				}
				if fullCommit := req.Header.Get("X-Couch-Full-Commit"); fullCommit != "true" {
					return nil, errors.New("X-Couch-Full-Commit not true")
				}
				return nil, errors.New("success")
			}),
			id:      "foo",
			doc:     map[string]string{"foo": "bar"},
			options: map[string]interface{}{OptionFullCommit: true},
			status:  kivik.StatusNetworkError,
			err:     "Put http://example.com/testdb/foo: success",
		},
		{
			name:    "invalid full commit",
			db:      &db{},
			id:      "foo",
			doc:     map[string]string{"foo": "bar"},
			options: map[string]interface{}{OptionFullCommit: 123},
			status:  kivik.StatusBadRequest,
			err:     "kivik: option 'X-Couch-Full-Commit' must be bool, not int",
		},
		{
			name: "connection refused",
			db: func() *db {
				c, err := chttp.New(context.Background(), "http://127.0.0.1:1/")
				if err != nil {
					t.Fatal(err)
				}
				return &db{
					client: &client{Client: c},
					dbName: "animals",
				}
			}(),
			id:     "cow",
			doc:    map[string]interface{}{"feet": 4},
			status: kivik.StatusNetworkError,
			err:    "Put http://127.0.0.1:1/animals/cow: dial tcp ([::1]|127.0.0.1):1: getsockopt: connection refused",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rev, err := test.db.PutOpts(context.Background(), test.id, test.doc, test.options)
			testy.StatusErrorRE(t, test.err, test.status, err)
			if rev != test.rev {
				t.Errorf("Unexpected rev: %s", rev)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	db := &db{}
	_, err := db.Delete(context.Background(), "", "")
	testy.Error(t, "kivik: docID required", err)
}

func TestDeleteOpts(t *testing.T) {
	tests := []struct {
		name    string
		db      *db
		id, rev string
		options map[string]interface{}
		newrev  string
		status  int
		err     string
	}{
		{
			name:   "no doc id",
			status: kivik.StatusBadRequest,
			err:    "kivik: docID required",
		},
		{
			name:   "no rev",
			id:     "foo",
			status: kivik.StatusBadRequest,
			err:    "kivik: rev required",
		},
		{
			name:   "network error",
			id:     "foo",
			rev:    "1-xxx",
			db:     newTestDB(nil, errors.New("net error")),
			status: kivik.StatusNetworkError,
			err:    "(Delete http://example.com/testdb/foo?rev=: )?net error",
		},
		{
			name: "1.6.1 conflict",
			id:   "43734cf3ce6d5a37050c050bb600006b",
			rev:  "1-xxx",
			db: newTestDB(&http.Response{
				StatusCode: 409,
				Header: http.Header{
					"Server":         {"CouchDB/1.6.1 (Erlang OTP/17)"},
					"Date":           {"Thu, 26 Oct 2017 13:29:06 GMT"},
					"Content-Type":   {"text/plain; charset=utf-8"},
					"Content-Length": {"58"},
					"Cache-Control":  {"must-revalidate"},
				},
				Body: ioutil.NopCloser(strings.NewReader(`{"error":"conflict","reason":"Document update conflict."}`)),
			}, nil),
			status: kivik.StatusConflict,
			err:    "Conflict",
		},
		{
			name: "1.6.1 success",
			id:   "43734cf3ce6d5a37050c050bb600006b",
			rev:  "1-4c6114c65e295552ab1019e2b046b10e",
			db: newTestDB(&http.Response{
				StatusCode: 200,
				Header: http.Header{
					"Server":         {"CouchDB/1.6.1 (Erlang OTP/17)"},
					"Date":           {"Thu, 26 Oct 2017 13:29:06 GMT"},
					"Content-Type":   {"text/plain; charset=utf-8"},
					"ETag":           {`"2-185ccf92154a9f24a4f4fd12233bf463"`},
					"Content-Length": {"95"},
					"Cache-Control":  {"must-revalidate"},
				},
				Body: ioutil.NopCloser(strings.NewReader(`{"ok":true,"id":"43734cf3ce6d5a37050c050bb600006b","rev":"2-185ccf92154a9f24a4f4fd12233bf463"}`)),
			}, nil),
			newrev: "2-185ccf92154a9f24a4f4fd12233bf463",
		},
		{
			name: "batch mode",
			db: newCustomDB(func(req *http.Request) (*http.Response, error) {
				if err := consume(req.Body); err != nil {
					return nil, err
				}
				if batch := req.URL.Query().Get("batch"); batch != "ok" {
					return nil, errors.Errorf("Unexpected query batch=%s", batch)
				}
				return nil, errors.New("success")
			}),
			id:      "foo",
			rev:     "1-xxx",
			options: map[string]interface{}{"batch": "ok"},
			status:  kivik.StatusNetworkError,
			err:     "success",
		},
		{
			name:    "invalid options",
			db:      &db{},
			id:      "foo",
			rev:     "1-xxx",
			options: map[string]interface{}{"foo": make(chan int)},
			status:  kivik.StatusBadRequest,
			err:     "kivik: invalid type chan int for options",
		},
		{
			name: "full commit",
			db: newCustomDB(func(req *http.Request) (*http.Response, error) {
				if err := consume(req.Body); err != nil {
					return nil, err
				}
				if fullCommit := req.Header.Get("X-Couch-Full-Commit"); fullCommit != "true" {
					return nil, errors.New("X-Couch-Full-Commit not true")
				}
				return nil, errors.New("success")
			}),
			id:      "foo",
			rev:     "1-xxx",
			options: map[string]interface{}{OptionFullCommit: true},
			status:  kivik.StatusNetworkError,
			err:     "success",
		},
		{
			name:    "invalid full commit type",
			db:      &db{},
			id:      "foo",
			rev:     "1-xxx",
			options: map[string]interface{}{OptionFullCommit: 123},
			status:  kivik.StatusBadRequest,
			err:     "kivik: option 'X-Couch-Full-Commit' must be bool, not int",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			newrev, err := test.db.DeleteOpts(context.Background(), test.id, test.rev, test.options)
			testy.StatusErrorRE(t, test.err, test.status, err)
			if newrev != test.newrev {
				t.Errorf("Unexpected new rev: %s", newrev)
			}
		})
	}
}

func TestFlush(t *testing.T) {
	tests := []struct {
		name   string
		db     *db
		status int
		err    string
	}{
		{
			name:   "network error",
			db:     newTestDB(nil, errors.New("net error")),
			status: kivik.StatusNetworkError,
			err:    "Post http://example.com/testdb/_ensure_full_commit: net error",
		},
		{
			name: "1.6.1",
			db: newCustomDB(func(req *http.Request) (*http.Response, error) {
				if ct, _, _ := mime.ParseMediaType(req.Header.Get("Content-Type")); ct != "application/json" {
					return nil, fmt.Errorf("Expected Content-Type: application/json, got %s", ct)
				}
				return &http.Response{
					StatusCode: 201,
					Header: http.Header{
						"Server":         {"CouchDB/1.6.1 (Erlang OTP/17)"},
						"Date":           {"Thu, 26 Oct 2017 13:07:52 GMT"},
						"Content-Type":   {"text/plain; charset=utf-8"},
						"Content-Length": {"53"},
						"Cache-Control":  {"must-revalidate"},
					},
					Body: ioutil.NopCloser(strings.NewReader(`{"ok":true,"instance_start_time":"1509022681259533"}`)),
				}, nil
			}),
		},
		{
			name: "2.0.0",
			db: newCustomDB(func(req *http.Request) (*http.Response, error) {
				if ct, _, _ := mime.ParseMediaType(req.Header.Get("Content-Type")); ct != "application/json" {
					return nil, fmt.Errorf("Expected Content-Type: application/json, got %s", ct)
				}
				return &http.Response{
					StatusCode: 201,
					Header: http.Header{
						"Server":              {"CouchDB/2.0.0 (Erlang OTP/17)"},
						"Date":                {"Thu, 26 Oct 2017 13:07:52 GMT"},
						"Content-Type":        {"application/json"},
						"Content-Length":      {"38"},
						"Cache-Control":       {"must-revalidate"},
						"X-Couch-Request-ID":  {"e454023cb8"},
						"X-CouchDB-Body-Time": {"0"},
					},
					Body: ioutil.NopCloser(strings.NewReader(`{"ok":true,"instance_start_time":"0"}`)),
				}, nil
			}),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.db.Flush(context.Background())
			testy.StatusError(t, test.err, test.status, err)
		})
	}
}

func TestRowsQuery(t *testing.T) {
	type queryResult struct {
		Offset    int64
		TotalRows int64
		Warning   string
		UpdateSeq string
		Err       string
		Rows      []driver.Row
	}
	tests := []struct {
		name     string
		db       *db
		path     string
		options  map[string]interface{}
		expected queryResult
		status   int
		err      string
	}{
		{
			name:    "invalid options",
			path:    "_all_docs",
			options: map[string]interface{}{"foo": make(chan int)},
			status:  kivik.StatusBadRequest,
			err:     "kivik: invalid type chan int for options",
		},
		{
			name:   "network error",
			path:   "_all_docs",
			db:     newTestDB(nil, errors.New("go away")),
			status: kivik.StatusNetworkError,
			err:    "Get http://example.com/testdb/_all_docs: go away",
		},
		{
			name: "error response",
			path: "_all_docs",
			db: newTestDB(&http.Response{
				StatusCode: kivik.StatusBadRequest,
				Body:       ioutil.NopCloser(strings.NewReader("")),
			}, nil),
			status: kivik.StatusBadRequest,
			err:    "Bad Request",
		},
		{
			name: "all docs default 1.6.1",
			path: "_all_docs",
			db: newTestDB(&http.Response{
				StatusCode: kivik.StatusOK,
				Header: map[string][]string{
					"Transfer-Encoding": {"chunked"},
					"Date":              {"Tue, 24 Oct 2017 21:17:12 GMT"},
					"Server":            {"CouchDB/1.6.1 (Erlang OTP/17)"},
					"ETag":              {`"2MVNDK3T2PN4JUK89TKD10QDA"`},
					"Content-Type":      {"text/plain; charset=utf-8"},
					"Cache-Control":     {"must-revalidate"},
				},
				Body: ioutil.NopCloser(strings.NewReader(`{"total_rows":3,"offset":0,"rows":[
{"id":"_design/_auth","key":"_design/_auth","value":{"rev":"1-75efcce1f083316d622d389f3f9813f7"}},
{"id":"org.couchdb.user:5wmxzru3b4i6pdmvhslq5egiye","key":"org.couchdb.user:5wmxzru3b4i6pdmvhslq5egiye","value":{"rev":"1-747e6766038164010fd0efcabd1a31dd"}},
{"id":"org.couchdb.user:zqfdn6u3cqi6pol3hslq5egiye","key":"org.couchdb.user:zqfdn6u3cqi6pol3hslq5egiye","value":{"rev":"1-4645438e6e1aa2230a1b06b5c1f5c63f"}}
]}
`)),
			}, nil),
			expected: queryResult{
				TotalRows: 3,
				Rows: []driver.Row{
					{
						ID:    "_design/_auth",
						Key:   []byte(`"_design/_auth"`),
						Value: []byte(`{"rev":"1-75efcce1f083316d622d389f3f9813f7"}`),
					},
					{
						ID:    "org.couchdb.user:5wmxzru3b4i6pdmvhslq5egiye",
						Key:   []byte(`"org.couchdb.user:5wmxzru3b4i6pdmvhslq5egiye"`),
						Value: []byte(`{"rev":"1-747e6766038164010fd0efcabd1a31dd"}`),
					},
					{
						ID:    "org.couchdb.user:zqfdn6u3cqi6pol3hslq5egiye",
						Key:   []byte(`"org.couchdb.user:zqfdn6u3cqi6pol3hslq5egiye"`),
						Value: []byte(`{"rev":"1-4645438e6e1aa2230a1b06b5c1f5c63f"}`),
					},
				},
			},
		},
		{
			name: "all docs options 1.6.1",
			path: "/_all_docs?update_seq=true&limit=1&skip=1",
			db: newTestDB(&http.Response{
				StatusCode: kivik.StatusOK,
				Header: map[string][]string{
					"Transfer-Encoding": {"chunked"},
					"Date":              {"Tue, 24 Oct 2017 21:17:12 GMT"},
					"Server":            {"CouchDB/1.6.1 (Erlang OTP/17)"},
					"ETag":              {`"2MVNDK3T2PN4JUK89TKD10QDA"`},
					"Content-Type":      {"text/plain; charset=utf-8"},
					"Cache-Control":     {"must-revalidate"},
				},
				Body: ioutil.NopCloser(strings.NewReader(`{"total_rows":3,"offset":1,"update_seq":31,"rows":[
{"id":"org.couchdb.user:5wmxzru3b4i6pdmvhslq5egiye","key":"org.couchdb.user:5wmxzru3b4i6pdmvhslq5egiye","value":{"rev":"1-747e6766038164010fd0efcabd1a31dd"}}
]}
`)),
			}, nil),
			expected: queryResult{
				TotalRows: 3,
				Offset:    1,
				UpdateSeq: "31",
				Rows: []driver.Row{
					{
						ID:    "org.couchdb.user:5wmxzru3b4i6pdmvhslq5egiye",
						Key:   []byte(`"org.couchdb.user:5wmxzru3b4i6pdmvhslq5egiye"`),
						Value: []byte(`{"rev":"1-747e6766038164010fd0efcabd1a31dd"}`),
					},
				},
			},
		},
		{
			name: "all docs options 2.0.0, no results",
			path: "/_all_docs?update_seq=true&limit=1",
			db: newTestDB(&http.Response{
				StatusCode: kivik.StatusOK,
				Header: map[string][]string{
					"Transfer-Encoding":  {"chunked"},
					"Date":               {"Tue, 24 Oct 2017 21:21:30 GMT"},
					"Server":             {"CouchDB/2.0.0 (Erlang OTP/17)"},
					"Content-Type":       {"application/json"},
					"Cache-Control":      {"must-revalidate"},
					"X-Couch-Request-ID": {"a9688d9335"},
					"X-Couch-Body-Time":  {"0"},
				},
				Body: ioutil.NopCloser(strings.NewReader(`{"total_rows":1,"offset":0,"update_seq":"13-g1AAAAEzeJzLYWBg4MhgTmHgzcvPy09JdcjLz8gvLskBCjPlsQBJhgdA6j8QZCUy4FV3AKLuflYiE151DRB18wmZtwCibj9u85ISgGRSPV63JSmA1NiD1bDgUJPIkCSP3xAHkCHxYDWsWQDg12MD","rows":[
{"id":"_design/_auth","key":"_design/_auth","value":{"rev":"1-75efcce1f083316d622d389f3f9813f7"}}
]}
`)),
			}, nil),
			expected: queryResult{
				TotalRows: 1,
				UpdateSeq: "13-g1AAAAEzeJzLYWBg4MhgTmHgzcvPy09JdcjLz8gvLskBCjPlsQBJhgdA6j8QZCUy4FV3AKLuflYiE151DRB18wmZtwCibj9u85ISgGRSPV63JSmA1NiD1bDgUJPIkCSP3xAHkCHxYDWsWQDg12MD",
				Rows: []driver.Row{
					{
						ID:    "_design/_auth",
						Key:   []byte(`"_design/_auth"`),
						Value: []byte(`{"rev":"1-75efcce1f083316d622d389f3f9813f7"}`),
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rows, err := test.db.rowsQuery(context.Background(), test.path, test.options)
			testy.StatusError(t, test.err, test.status, err)
			result := queryResult{
				Rows: []driver.Row{},
			}
			for {
				var row driver.Row
				if e := rows.Next(&row); e != nil {
					if e != io.EOF {
						result.Err = e.Error()
					}
					break
				}
				result.Rows = append(result.Rows, row)
			}
			result.Offset = rows.Offset()
			result.TotalRows = rows.TotalRows()
			result.UpdateSeq = rows.UpdateSeq()
			if warner, ok := rows.(driver.RowsWarner); ok {
				result.Warning = warner.Warning()
			} else {
				t.Errorf("RowsWarner interface not satisified!!?")
			}

			if d := diff.Interface(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestSecurity(t *testing.T) {
	tests := []struct {
		name     string
		db       *db
		expected *driver.Security
		status   int
		err      string
	}{
		{
			name:   "network error",
			db:     newTestDB(nil, errors.New("net error")),
			status: kivik.StatusNetworkError,
			err:    "Get http://example.com/testdb/_security: net error",
		},
		{
			name: "1.6.1 empty",
			db: newTestDB(&http.Response{
				StatusCode: 200,
				Header: http.Header{
					"Server":         {"CouchDB/1.6.1 (Erlang OTP/17)"},
					"Date":           {"Thu, 26 Oct 2017 14:28:14 GMT"},
					"Content-Type":   {"text/plain; charset=utf-8"},
					"Content-Length": {"3"},
					"Cache-Control":  {"must-revalidate"},
				},
				Body: ioutil.NopCloser(strings.NewReader("{}")),
			}, nil),
			expected: &driver.Security{},
		},
		{
			name: "1.6.1 non-empty",
			db: newTestDB(&http.Response{
				StatusCode: 200,
				Header: http.Header{
					"Server":         {"CouchDB/1.6.1 (Erlang OTP/17)"},
					"Date":           {"Thu, 26 Oct 2017 14:28:14 GMT"},
					"Content-Type":   {"text/plain; charset=utf-8"},
					"Content-Length": {"65"},
					"Cache-Control":  {"must-revalidate"},
				},
				Body: ioutil.NopCloser(strings.NewReader(`{"admins":{},"members":{"names":["32dgsme3cmi6pddghslq5egiye"]}}`)),
			}, nil),
			expected: &driver.Security{
				Members: driver.Members{
					Names: []string{"32dgsme3cmi6pddghslq5egiye"},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.db.Security(context.Background())
			testy.StatusError(t, test.err, test.status, err)
			if d := diff.Interface(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestSetSecurity(t *testing.T) {
	tests := []struct {
		name     string
		db       *db
		security *driver.Security
		status   int
		err      string
	}{
		{
			name:   "network error",
			db:     newTestDB(nil, errors.New("net error")),
			status: kivik.StatusNetworkError,
			err:    "Put http://example.com/testdb/_security: net error",
		},
		{
			name: "1.6.1",
			security: &driver.Security{
				Admins: driver.Members{
					Names: []string{"bob"},
				},
				Members: driver.Members{
					Roles: []string{"users"},
				},
			},
			db: newCustomDB(func(req *http.Request) (*http.Response, error) {
				defer req.Body.Close() // nolint: errcheck
				if ct, _, _ := mime.ParseMediaType(req.Header.Get("Content-Type")); ct != "application/json" {
					return nil, fmt.Errorf("Expected Content-Type: application/json, got %s", ct)
				}
				var body interface{}
				if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
					return nil, err
				}
				expected := map[string]interface{}{
					"admins": map[string]interface{}{
						"names": []string{"bob"},
					},
					"members": map[string]interface{}{
						"roles": []string{"users"},
					},
				}
				if d := diff.AsJSON(expected, body); d != nil {
					t.Error(d)
				}
				return &http.Response{
					StatusCode: 200,
					Header: http.Header{
						"Server":         {"CouchDB/1.6.1 (Erlang OTP/17)"},
						"Date":           {"Thu, 26 Oct 2017 15:06:21 GMT"},
						"Content-Type":   {"text/plain; charset=utf-8"},
						"Content-Length": {"12"},
						"Cache-Control":  {"must-revalidate"},
					},
					Body: ioutil.NopCloser(strings.NewReader(`{"ok":true}`)),
				}, nil
			}),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.db.SetSecurity(context.Background(), test.security)
			testy.StatusError(t, test.err, test.status, err)
		})
	}
}

func TestRev(t *testing.T) {
	tests := []struct {
		name   string
		db     *db
		id     string
		rev    string
		status int
		err    string
	}{
		{
			name:   "no doc id",
			status: kivik.StatusBadRequest,
			err:    "kivik: docID required",
		},
		{
			name:   "network error",
			id:     "foo",
			db:     newTestDB(nil, errors.New("net error")),
			status: kivik.StatusNetworkError,
			err:    "Head http://example.com/testdb/foo: net error",
		},
		{
			name: "1.6.1",
			id:   "foo",
			db: newTestDB(&http.Response{
				StatusCode: 200,
				Request: &http.Request{
					Method: "HEAD",
				},
				Header: http.Header{
					"Server":         {"CouchDB/1.6.1 (Erlang OTP/17)"},
					"ETag":           {`"1-4c6114c65e295552ab1019e2b046b10e"`},
					"Date":           {"Thu, 26 Oct 2017 15:21:15 GMT"},
					"Content-Type":   {"text/plain; charset=utf-8"},
					"Content-Length": {"70"},
					"Cache-Control":  {"must-revalidate"},
				},
				Body: ioutil.NopCloser(strings.NewReader("")),
			}, nil),
			rev: "1-4c6114c65e295552ab1019e2b046b10e",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rev, err := test.db.Rev(context.Background(), test.id)
			testy.StatusError(t, test.err, test.status, err)
			if rev != test.rev {
				t.Errorf("Got %s, expected %s", rev, test.rev)
			}
		})
	}
}

func TestCopy(t *testing.T) {
	tests := []struct {
		name           string
		target, source string
		options        map[string]interface{}
		db             *db
		rev            string
		status         int
		err            string
	}{
		{
			name:   "missing source",
			status: kivik.StatusBadRequest,
			err:    "kivik: sourceID required",
		},
		{
			name:   "missing target",
			source: "foo",
			status: kivik.StatusBadRequest,
			err:    "kivik: targetID required",
		},
		{
			name:   "network error",
			source: "foo",
			target: "bar",
			db:     newTestDB(nil, errors.New("net error")),
			status: kivik.StatusNetworkError,
			err:    "(Copy http://example.com/testdb/foo: )?net error",
		},
		{
			name:    "invalid options",
			db:      &db{},
			source:  "foo",
			target:  "bar",
			options: map[string]interface{}{"foo": make(chan int)},
			status:  kivik.StatusBadRequest,
			err:     "kivik: invalid type chan int for options",
		},
		{
			name:    "invalid full commit type",
			db:      &db{},
			source:  "foo",
			target:  "bar",
			options: map[string]interface{}{OptionFullCommit: 123},
			status:  kivik.StatusBadRequest,
			err:     "kivik: option 'X-Couch-Full-Commit' must be bool, not int",
		},
		{
			name:   "create 1.6.1",
			source: "foo",
			target: "bar",
			db: newCustomDB(func(req *http.Request) (*http.Response, error) {
				if req.Header.Get("Destination") != "bar" {
					return nil, errors.New("Unexpected destination")
				}
				return &http.Response{
					StatusCode: 201,
					Header: http.Header{
						"Server":         {"CouchDB/1.6.1 (Erlang OTP/17)"},
						"Location":       {"http://example.com/foo/bar"},
						"ETag":           {`"1-f81c8a795b0c6f9e9f699f64c6b82256"`},
						"Date":           {"Thu, 26 Oct 2017 15:45:57 GMT"},
						"Content-Type":   {"text/plain; charset=utf-8"},
						"Content-Length": {"66"},
						"Cache-Control":  {"must-revalidate"},
					},
					Body: Body(`{"ok":true,"id":"bar","rev":"1-f81c8a795b0c6f9e9f699f64c6b82256"}`),
				}, nil
			}),
			rev: "1-f81c8a795b0c6f9e9f699f64c6b82256",
		},
		{
			name:   "full commit 1.6.1",
			source: "foo",
			target: "bar",
			options: map[string]interface{}{
				OptionFullCommit: true,
			},
			db: newCustomDB(func(req *http.Request) (*http.Response, error) {
				if dest := req.Header.Get("Destination"); dest != "bar" {
					return nil, fmt.Errorf("Unexpected destination: %s", dest)
				}
				if fc := req.Header.Get("X-Couch-Full-Commit"); fc != "true" {
					return nil, fmt.Errorf("X-Couch-Full-Commit: %s", fc)
				}
				return &http.Response{
					StatusCode: 201,
					Header: http.Header{
						"Server":         {"CouchDB/1.6.1 (Erlang OTP/17)"},
						"Location":       {"http://example.com/foo/bar"},
						"ETag":           {`"1-f81c8a795b0c6f9e9f699f64c6b82256"`},
						"Date":           {"Thu, 26 Oct 2017 15:45:57 GMT"},
						"Content-Type":   {"text/plain; charset=utf-8"},
						"Content-Length": {"66"},
						"Cache-Control":  {"must-revalidate"},
					},
					Body: Body(`{"ok":true,"id":"bar","rev":"1-f81c8a795b0c6f9e9f699f64c6b82256"}`),
				}, nil
			}),
			rev: "1-f81c8a795b0c6f9e9f699f64c6b82256",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rev, err := test.db.Copy(context.Background(), test.target, test.source, test.options)
			testy.StatusErrorRE(t, test.err, test.status, err)
			if rev != test.rev {
				t.Errorf("Got %s, expected %s", rev, test.rev)
			}
		})
	}
}

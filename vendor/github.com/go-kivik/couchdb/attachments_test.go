package couchdb

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"strings"
	"testing"

	"github.com/flimzy/diff"
	"github.com/flimzy/testy"

	"github.com/go-kivik/kivik"
	"github.com/go-kivik/kivik/driver"
	"github.com/go-kivik/kivik/errors"
)

type closer struct {
	io.Reader
	closed bool
}

var _ io.ReadCloser = &closer{}

func (c *closer) Close() error {
	c.closed = true
	return nil
}

func TestPutAttachment(t *testing.T) {
	type paoTest struct {
		name    string
		db      *db
		id, rev string
		att     *driver.Attachment
		options map[string]interface{}

		newRev string
		status int
		err    string
		final  func(*testing.T)
	}
	tests := []paoTest{
		{
			name:   "missing docID",
			status: kivik.StatusBadRequest,
			err:    "kivik: docID required",
		},
		{
			name:   "nil attachment",
			id:     "foo",
			rev:    "1-xxx",
			status: kivik.StatusBadRequest,
			err:    "kivik: att required",
		},
		{
			name:   "missing filename",
			id:     "foo",
			rev:    "1-xxx",
			att:    &driver.Attachment{},
			status: kivik.StatusBadRequest,
			err:    "kivik: att.Filename required",
		},
		{
			name: "missing content type",
			id:   "foo",
			rev:  "1-xxx",
			att: &driver.Attachment{
				Filename: "x.jpg",
			},
			status: kivik.StatusBadRequest,
			err:    "kivik: att.ContentType required",
		},
		{
			name: "no body",
			id:   "foo",
			rev:  "1-xxx",
			att: &driver.Attachment{
				Filename:    "x.jpg",
				ContentType: "image/jpeg",
			},
			status: kivik.StatusBadRequest,
			err:    "kivik: att.Content required",
		},
		{
			name: "network error",
			db:   newTestDB(nil, errors.New("net error")),
			id:   "foo",
			rev:  "1-xxx",
			att: &driver.Attachment{
				Filename:    "x.jpg",
				ContentType: "image/jpeg",
				Content:     Body("x"),
			},
			status: kivik.StatusNetworkError,
			err:    "Put http://example.com/testdb/foo/x.jpg\\?rev=1-xxx: net error",
		},
		{
			name: "1.6.1",
			id:   "foo",
			rev:  "1-4c6114c65e295552ab1019e2b046b10e",
			att: &driver.Attachment{
				Filename:    "foo.txt",
				ContentType: "text/plain",
				Content:     Body("Hello, World!"),
			},
			db: newCustomDB(func(req *http.Request) (*http.Response, error) {
				defer req.Body.Close() // nolint: errcheck
				if ct, _, _ := mime.ParseMediaType(req.Header.Get("Content-Type")); ct != "text/plain" {
					return nil, fmt.Errorf("Unexpected Content-Type: %s", ct)
				}
				expectedRev := "1-4c6114c65e295552ab1019e2b046b10e"
				if rev := req.URL.Query().Get("rev"); rev != expectedRev {
					return nil, fmt.Errorf("Unexpected rev: %s", rev)
				}
				body, err := ioutil.ReadAll(req.Body)
				if err != nil {
					return nil, err
				}
				expected := "Hello, World!"
				if d := diff.Text(expected, string(body)); d != nil {
					t.Errorf("Unexpected body:\n%s", d)
				}
				return &http.Response{
					StatusCode: 201,
					Header: http.Header{
						"Server":         {"CouchDB/1.6.1 (Erlang OTP/17)"},
						"Location":       {"http://localhost:5984/foo/foo/foo.txt"},
						"ETag":           {`"2-8ee3381d24ee4ac3e9f8c1f6c7395641"`},
						"Date":           {"Thu, 26 Oct 2017 20:51:35 GMT"},
						"Content-Type":   {"text/plain; charset=utf-8"},
						"Content-Length": {"66"},
						"Cache-Control":  {"must-revalidate"},
					},
					Body: Body(`{"ok":true,"id":"foo","rev":"2-8ee3381d24ee4ac3e9f8c1f6c7395641"}`),
				}, nil
			}),
			newRev: "2-8ee3381d24ee4ac3e9f8c1f6c7395641",
		},
		{
			name: "no rev",
			db: newCustomDB(func(req *http.Request) (*http.Response, error) {
				if _, ok := req.URL.Query()["rev"]; ok {
					t.Errorf("'rev' should not be present in the query")
				}
				return nil, errors.New("ignore this error")
			}),
			id: "foo",
			att: &driver.Attachment{
				Filename:    "foo.txt",
				ContentType: "text/plain",
				Content:     Body("x"),
			},
			status: 601,
			err:    "Put http://example.com/testdb/foo/foo.txt: ignore this error",
		},
		{
			name: "with options",
			db:   newTestDB(nil, errors.New("success")),
			id:   "foo",
			rev:  "1-xxx",
			att: &driver.Attachment{
				Filename:    "foo.txt",
				ContentType: "text/plain",
				Content:     Body("x"),
			},
			options: map[string]interface{}{"foo": "oink"},
			status:  kivik.StatusNetworkError,
			err:     "foo=oink",
		},
		{
			name: "invalid options",
			db:   &db{},
			id:   "foo",
			rev:  "1-xxx",
			att: &driver.Attachment{
				Filename:    "foo.txt",
				ContentType: "text/plain",
				Content:     Body("x"),
			},
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
			id:  "foo",
			rev: "1-xxx",
			att: &driver.Attachment{
				Filename:    "foo.txt",
				ContentType: "text/plain",
				Content:     Body("x"),
			},
			options: map[string]interface{}{OptionFullCommit: true},
			status:  kivik.StatusNetworkError,
			err:     "success",
		},
		{
			name: "invalid full commit type",
			db:   &db{},
			id:   "foo",
			rev:  "1-xxx",
			att: &driver.Attachment{
				Filename:    "foo.txt",
				ContentType: "text/plain",
				Content:     Body("x"),
			},
			options: map[string]interface{}{OptionFullCommit: 123},
			status:  kivik.StatusBadRequest,
			err:     "kivik: option 'X-Couch-Full-Commit' must be bool, not int",
		},
		func() paoTest {
			body := &closer{Reader: strings.NewReader("x")}
			return paoTest{
				name: "ReadCloser",
				db: newCustomDB(func(req *http.Request) (*http.Response, error) {
					if err := consume(req.Body); err != nil {
						return nil, err
					}
					if fullCommit := req.Header.Get("X-Couch-Full-Commit"); fullCommit != "true" {
						return nil, errors.New("X-Couch-Full-Commit not true")
					}
					return nil, errors.New("success")
				}),
				id:  "foo",
				rev: "1-xxx",
				att: &driver.Attachment{
					Filename:    "foo.txt",
					ContentType: "text/plain",
					Content:     Body("x"),
				},
				options: map[string]interface{}{OptionFullCommit: true},
				status:  kivik.StatusNetworkError,
				err:     "success",
				final: func(t *testing.T) {
					if !body.closed {
						t.Fatal("body wasn't closed")
					}
				},
			}
		}(),
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			newRev, err := test.db.PutAttachment(context.Background(), test.id, test.rev, test.att, test.options)
			testy.StatusErrorRE(t, test.err, test.status, err)
			if newRev != test.newRev {
				t.Errorf("Expected %s, got %s\n", test.newRev, newRev)
			}
			if test.final != nil {
				test.final(t)
			}
		})
	}
}

func TestGetAttachmentMeta(t *testing.T) {
	tests := []struct {
		name              string
		db                *db
		id, rev, filename string
		options           map[string]interface{}

		expected *driver.Attachment
		status   int
		err      string
	}{
		{
			name:     "network error",
			id:       "foo",
			filename: "foo.txt",
			db:       newTestDB(nil, errors.New("net error")),
			status:   kivik.StatusNetworkError,
			err:      "Head http://example.com/testdb/foo/foo.txt: net error",
		},
		{
			name:     "1.6.1",
			id:       "foo",
			filename: "foo.txt",
			db: newTestDB(&http.Response{
				StatusCode: 200,
				Header: http.Header{
					"Server":         {"CouchDB/1.6.1 (Erlang OTP/17)"},
					"ETag":           {`"gSr8dSmynwAoomH7V6RVYw=="`},
					"Date":           {"Thu, 26 Oct 2017 21:15:13 GMT"},
					"Content-Type":   {"text/plain"},
					"Content-Length": {"13"},
					"Cache-Control":  {"must-revalidate"},
					"Accept-Ranges":  {"none"},
				},
				Body: Body(""),
			}, nil),
			expected: &driver.Attachment{
				ContentType: "text/plain",
				Digest:      "gSr8dSmynwAoomH7V6RVYw==",
				Content:     Body(""),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			att, err := test.db.GetAttachmentMeta(context.Background(), test.id, test.rev, test.filename, test.options)
			testy.StatusError(t, test.err, test.status, err)
			if d := diff.Interface(test.expected, att); d != nil {
				t.Errorf("Unexpected attachment:\n%s", d)
			}
		})
	}
}

func TestGetDigest(t *testing.T) {
	tests := []struct {
		name     string
		resp     *http.Response
		expected string
		status   int
		err      string
	}{
		{
			name:   "no etag header",
			resp:   &http.Response{},
			status: kivik.StatusBadResponse,
			err:    "ETag header not found",
		},
		{
			name: "Standard ETag header",
			resp: &http.Response{
				Header: http.Header{"ETag": []string{`"ENGoH7oK8V9R3BMnfDHZmw=="`}},
			},
			expected: "ENGoH7oK8V9R3BMnfDHZmw==",
		},
		{
			name: "normalized Etag header",
			resp: &http.Response{
				Header: http.Header{"Etag": []string{`"ENGoH7oK8V9R3BMnfDHZmw=="`}},
			},
			expected: "ENGoH7oK8V9R3BMnfDHZmw==",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			digest, err := getDigest(test.resp)
			testy.Error(t, test.err, err)
			if digest != test.expected {
				t.Errorf("Unexpected result: %0x", digest)
			}
		})
	}
}

func TestGetAttachment(t *testing.T) {
	tests := []struct {
		name              string
		db                *db
		id, rev, filename string
		options           map[string]interface{}

		expected *driver.Attachment
		content  string
		status   int
		err      string
	}{
		{
			name:     "network error",
			id:       "foo",
			filename: "foo.txt",
			db:       newTestDB(nil, errors.New("net error")),
			status:   kivik.StatusNetworkError,
			err:      "Get http://example.com/testdb/foo/foo.txt: net error",
		},
		{
			name:     "1.6.1",
			id:       "foo",
			filename: "foo.txt",
			db: newCustomDB(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: 200,
					Header: http.Header{
						"Server":         {"CouchDB/1.6.1 (Erlang OTP/17)"},
						"ETag":           {`"gSr8dSmynwAoomH7V6RVYw=="`},
						"Date":           {"Fri, 27 Oct 2017 11:24:50 GMT"},
						"Content-Type":   {"text/plain"},
						"Content-Length": {"13"},
						"Cache-Control":  {"must-revalidate"},
						"Accept-Ranges":  {"none"},
					},
					Body: Body(`Hello, world!`),
				}, nil
			}),
			expected: &driver.Attachment{
				ContentType: "text/plain",
				Digest:      "gSr8dSmynwAoomH7V6RVYw==",
			},
			content: "Hello, world!",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			att, err := test.db.GetAttachment(context.Background(), test.id, test.rev, test.filename, test.options)
			testy.StatusError(t, test.err, test.status, err)
			fileContent, err := ioutil.ReadAll(att.Content)
			if err != nil {
				t.Fatal(err)
			}
			if d := diff.Text(test.content, string(fileContent)); d != nil {
				t.Errorf("Unexpected content:\n%s", d)
			}
			_ = att.Content.Close()
			att.Content = nil // Determinism
			if d := diff.Interface(test.expected, att); d != nil {
				t.Errorf("Unexpected attachment:\n%s", d)
			}
		})
	}
}

func TestFetchAttachment(t *testing.T) {
	tests := []struct {
		name                      string
		db                        *db
		method, id, rev, filename string
		options                   map[string]interface{}

		resp   *http.Response
		status int
		err    string
	}{
		{
			name:   "no method",
			status: kivik.StatusInternalServerError,
			err:    "method required",
		},
		{
			name:   "no docID",
			method: "GET",
			status: kivik.StatusBadRequest,
			err:    "kivik: docID required",
		},
		{
			name:   "no filename",
			method: "GET",
			id:     "foo",
			status: kivik.StatusBadRequest,
			err:    "kivik: filename required",
		},
		{
			name:     "no rev",
			method:   "GET",
			id:       "foo",
			filename: "foo.txt",
			db:       newTestDB(nil, errors.New("ignore this error")),
			status:   601,
			err:      "http://example.com/testdb/foo/foo.txt:",
		},
		{
			name:     "with rev",
			method:   "GET",
			id:       "foo",
			filename: "foo.txt",
			rev:      "1-xxx",
			db:       newTestDB(nil, errors.New("ignore this error")),
			status:   601,
			err:      "http://example.com/testdb/foo/foo.txt\\?rev=1-xxx:",
		},
		{
			name:     "success",
			method:   "GET",
			id:       "foo",
			filename: "foo.txt",
			rev:      "1-xxx",
			db: newTestDB(&http.Response{
				StatusCode: 200,
			}, nil),
			resp: &http.Response{
				StatusCode: 200,
			},
		},
		{
			name:     "options",
			db:       newTestDB(nil, errors.New("success")),
			method:   "GET",
			id:       "foo",
			filename: "foo.txt",
			options:  map[string]interface{}{"foo": "bar"},
			status:   kivik.StatusNetworkError,
			err:      "foo=bar",
		},
		{
			name:     "invalid option",
			db:       &db{},
			method:   "GET",
			id:       "foo",
			filename: "foo.txt",
			options:  map[string]interface{}{"foo": make(chan int)},
			status:   kivik.StatusBadRequest,
			err:      "kivik: invalid type chan int for options",
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
			method:   "GET",
			id:       "foo",
			filename: "foo.txt",
			options:  map[string]interface{}{OptionIfNoneMatch: "foo"},
			status:   kivik.StatusNetworkError,
			err:      "success",
		},
		{
			name:     "invalid if-none-match type",
			db:       &db{},
			method:   "GET",
			id:       "foo",
			filename: "foo.txt",
			options:  map[string]interface{}{OptionIfNoneMatch: 123},
			status:   kivik.StatusBadRequest,
			err:      "kivik: option 'If-None-Match' must be string, not int",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resp, err := test.db.fetchAttachment(context.Background(), test.method, test.id, test.rev, test.filename, test.options)
			testy.StatusErrorRE(t, test.err, test.status, err)
			resp.Request = nil
			if d := diff.Interface(test.resp, resp); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestDecodeAttachment(t *testing.T) {
	tests := []struct {
		name     string
		resp     *http.Response
		expected *driver.Attachment
		content  string
		status   int
		err      string
	}{
		{
			name:   "no content type",
			resp:   &http.Response{},
			status: kivik.StatusBadResponse,
			err:    "no Content-Type in response",
		},
		{
			name: "no etag header",
			resp: &http.Response{
				Header: http.Header{"Content-Type": {"text/plain"}},
			},
			status: kivik.StatusBadResponse,
			err:    "ETag header not found",
		},
		{
			name: "success",
			resp: &http.Response{
				Header: http.Header{
					"Content-Type": {"text/plain"},
					"ETag":         {`"gSr8dSmynwAoomH7V6RVYw=="`},
				},
				Body: Body("Hello, World!"),
			},
			expected: &driver.Attachment{
				ContentType: "text/plain",
				Digest:      "gSr8dSmynwAoomH7V6RVYw==",
			},
			content: "Hello, World!",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			att, err := decodeAttachment(test.resp)
			testy.StatusError(t, test.err, test.status, err)
			fileContent, err := ioutil.ReadAll(att.Content)
			if err != nil {
				t.Fatal(err)
			}
			if d := diff.Text(test.content, string(fileContent)); d != nil {
				t.Errorf("Unexpected content:\n%s", d)
			}
			_ = att.Content.Close()
			att.Content = nil // Determinism
			if d := diff.Interface(test.expected, att); d != nil {
				t.Errorf("Unexpected attachment:\n%s", d)
			}
		})
	}
}

func TestDeleteAttachment(t *testing.T) {
	tests := []struct {
		name              string
		db                *db
		id, rev, filename string
		options           map[string]interface{}

		newRev string
		status int
		err    string
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
			name:   "no filename",
			id:     "foo",
			rev:    "1-xxx",
			status: kivik.StatusBadRequest,
			err:    "kivik: filename required",
		},
		{
			name:     "network error",
			id:       "foo",
			rev:      "1-xxx",
			filename: "foo.txt",
			db:       newTestDB(nil, errors.New("net error")),
			status:   kivik.StatusNetworkError,
			err:      "(Delete http://example.com/testdb/foo/foo.txt\\?rev=1-xxx: )?net error",
		},
		{
			name:     "success 1.6.1",
			id:       "foo",
			rev:      "2-8ee3381d24ee4ac3e9f8c1f6c7395641",
			filename: "foo.txt",
			db: newTestDB(&http.Response{
				StatusCode: 200,
				Header: http.Header{
					"Server":         {"CouchDB/1.6.1 (Erlang OTP/17)"},
					"ETag":           {`"3-231a932924f61816915289fecd35b14a"`},
					"Date":           {"Fri, 27 Oct 2017 13:30:40 GMT"},
					"Content-Type":   {"text/plain; charset=utf-8"},
					"Content-Length": {"66"},
					"Cache-Control":  {"must-revalidate"},
				},
				Body: Body(`{"ok":true,"id":"foo","rev":"3-231a932924f61816915289fecd35b14a"}`),
			}, nil),
			newRev: "3-231a932924f61816915289fecd35b14a",
		},
		{
			name: "with options",
			db: newCustomDB(func(req *http.Request) (*http.Response, error) {
				if err := consume(req.Body); err != nil {
					return nil, err
				}
				if foo := req.URL.Query().Get("foo"); foo != "oink" {
					return nil, errors.Errorf("Unexpected query foo=%s", foo)
				}
				return nil, errors.New("success")
			}),
			id:       "foo",
			rev:      "1-xxx",
			filename: "foo.txt",
			options:  map[string]interface{}{"foo": "oink"},
			status:   kivik.StatusNetworkError,
			err:      "success",
		},
		{
			name:     "invalid option",
			db:       &db{},
			id:       "foo",
			rev:      "1-xxx",
			filename: "foo.txt",
			options:  map[string]interface{}{"foo": make(chan int)},
			status:   kivik.StatusBadRequest,
			err:      "kivik: invalid type chan int for options",
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
			id:       "foo",
			rev:      "1-xxx",
			filename: "foo.txt",
			options:  map[string]interface{}{OptionFullCommit: true},
			status:   kivik.StatusNetworkError,
			err:      "success",
		},
		{
			name:     "invalid full commit type",
			db:       &db{},
			id:       "foo",
			rev:      "1-xxx",
			filename: "foo.txt",
			options:  map[string]interface{}{OptionFullCommit: 123},
			status:   kivik.StatusBadRequest,
			err:      "kivik: option 'X-Couch-Full-Commit' must be bool, not int",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			newRev, err := test.db.DeleteAttachment(context.Background(), test.id, test.rev, test.filename, test.options)
			testy.StatusErrorRE(t, test.err, test.status, err)
			if newRev != test.newRev {
				t.Errorf("Unexpected new rev: %s", newRev)
			}
		})
	}
}

package couchdb

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/flimzy/diff"
	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/driver"
	"github.com/flimzy/testy"
)

func TestAllDBs(t *testing.T) {
	tests := []struct {
		name     string
		client   *client
		expected []string
		status   int
		err      string
	}{
		{
			name:   "network error",
			client: newTestClient(nil, errors.New("net error")),
			status: kivik.StatusNetworkError,
			err:    "Get http://example.com/_all_dbs: net error",
		},
		{
			name: "2.0.0",
			client: newTestClient(&http.Response{
				StatusCode: 200,
				Header: http.Header{
					"Server":              {"CouchDB/2.0.0 (Erlang OTP/17)"},
					"Date":                {"Fri, 27 Oct 2017 15:15:07 GMT"},
					"Content-Type":        {"application/json"},
					"ETag":                {`"33UVNAZU752CYNGBBTMWQFP7U"`},
					"Transfer-Encoding":   {"chunked"},
					"X-Couch-Request-ID":  {"ab5cd97c3e"},
					"X-CouchDB-Body-Time": {"0"},
				},
				Body: Body(`["_global_changes","_replicator","_users"]`),
			}, nil),
			expected: []string{"_global_changes", "_replicator", "_users"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.client.AllDBs(context.Background(), nil)
			testy.StatusError(t, test.err, test.status, err)
			if d := diff.Interface(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestDBExists(t *testing.T) {
	tests := []struct {
		name   string
		client *client
		dbName string
		exists bool
		status int
		err    string
	}{
		{
			name:   "no db specified",
			status: kivik.StatusBadRequest,
			err:    "kivik: dbName required",
		},
		{
			name:   "network error",
			dbName: "foo",
			client: newTestClient(nil, errors.New("net error")),
			status: kivik.StatusNetworkError,
			err:    "Head http://example.com/foo: net error",
		},
		{
			name:   "not found, 1.6.1",
			dbName: "foox",
			client: newTestClient(&http.Response{
				StatusCode: 404,
				Header: http.Header{
					"Server":         {"CouchDB/1.6.1 (Erlang OTP/17)"},
					"Date":           {"Fri, 27 Oct 2017 15:09:19 GMT"},
					"Content-Type":   {"text/plain; charset=utf-8"},
					"Content-Length": {"44"},
					"Cache-Control":  {"must-revalidate"},
				},
				Body: Body(""),
			}, nil),
			exists: false,
		},
		{
			name:   "exists, 1.6.1",
			dbName: "foo",
			client: newTestClient(&http.Response{
				StatusCode: 200,
				Header: http.Header{
					"Server":         {"CouchDB/1.6.1 (Erlang OTP/17)"},
					"Date":           {"Fri, 27 Oct 2017 15:09:19 GMT"},
					"Content-Type":   {"text/plain; charset=utf-8"},
					"Content-Length": {"229"},
					"Cache-Control":  {"must-revalidate"},
				},
				Body: Body(""),
			}, nil),
			exists: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			exists, err := test.client.DBExists(context.Background(), test.dbName, nil)
			testy.StatusError(t, test.err, test.status, err)
			if exists != test.exists {
				t.Errorf("Unexpected result: %t", exists)
			}
		})
	}
}

func TestCreateDB(t *testing.T) {
	tests := []struct {
		name   string
		dbName string
		client *client
		status int
		err    string
	}{
		{
			name:   "missing dbname",
			status: kivik.StatusBadRequest,
			err:    "kivik: dbName required",
		},
		{
			name:   "network error",
			dbName: "foo",
			client: newTestClient(nil, errors.New("net error")),
			status: kivik.StatusNetworkError,
			err:    "Put http://example.com/foo: net error",
		},
		{
			name:   "conflict 1.6.1",
			dbName: "foo",
			client: newTestClient(&http.Response{
				StatusCode: 412,
				Header: http.Header{
					"Server":         {"CouchDB/1.6.1 (Erlang OTP/17)"},
					"Date":           {"Fri, 27 Oct 2017 15:23:57 GMT"},
					"Content-Type":   {"application/json"},
					"Content-Length": {"94"},
					"Cache-Control":  {"must-revalidate"},
				},
				ContentLength: 94,
				Body:          Body(`{"error":"file_exists","reason":"The database could not be created, the file already exists."}`),
			}, nil),
			status: kivik.StatusPreconditionFailed,
			err:    "Precondition Failed: The database could not be created, the file already exists.",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.client.CreateDB(context.Background(), test.dbName, nil)
			testy.StatusError(t, test.err, test.status, err)
		})
	}
}

func TestDestroyDB(t *testing.T) {
	tests := []struct {
		name   string
		client *client
		dbName string
		status int
		err    string
	}{
		{
			name:   "no db name",
			status: kivik.StatusBadRequest,
			err:    "kivik: dbName required",
		},
		{
			name:   "network error",
			dbName: "foo",
			client: newTestClient(nil, errors.New("net error")),
			status: kivik.StatusNetworkError,
			err:    "(Delete http://example.com/foo: )?net error",
		},
		{
			name:   "1.6.1",
			dbName: "foo",
			client: newTestClient(&http.Response{
				StatusCode: 200,
				Header: http.Header{
					"Server":         {"CouchDB/1.6.1 (Erlang OTP/17)"},
					"Date":           {"Fri, 27 Oct 2017 17:12:45 GMT"},
					"Content-Type":   {"application/json"},
					"Content-Length": {"12"},
					"Cache-Control":  {"must-revalidate"},
				},
				Body: Body(`{"ok":true}`),
			}, nil),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.client.DestroyDB(context.Background(), test.dbName, nil)
			testy.StatusErrorRE(t, test.err, test.status, err)
		})
	}
}

func TestDBUpdates(t *testing.T) {
	tests := []struct {
		name   string
		client *client
		status int
		err    string
	}{
		{
			name:   "network error",
			client: newTestClient(nil, errors.New("net error")),
			status: kivik.StatusNetworkError,
			err:    "Get http://example.com/_db_updates?feed=continuous&since=now: net error",
		},
		{
			name: "error response",
			client: newTestClient(&http.Response{
				StatusCode: 400,
				Body:       Body(""),
			}, nil),
			status: kivik.StatusBadRequest,
			err:    "Bad Request",
		},
		{
			name: "Success 1.6.1",
			client: newTestClient(&http.Response{
				StatusCode: 200,
				Header: http.Header{
					"Transfer-Encoding": {"chunked"},
					"Server":            {"CouchDB/1.6.1 (Erlang OTP/17)"},
					"Date":              {"Fri, 27 Oct 2017 19:55:43 GMT"},
					"Content-Type":      {"application/json"},
					"Cache-Control":     {"must-revalidate"},
				},
				Body: Body(""),
			}, nil),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.client.DBUpdates()
			testy.StatusError(t, test.err, test.status, err)
			if _, ok := result.(*couchUpdates); !ok {
				t.Errorf("Unexpected type returned: %t", result)
			}
		})
	}
}

func TestUpdatesNext(t *testing.T) {
	tests := []struct {
		name     string
		updates  *couchUpdates
		status   int
		err      string
		expected *driver.DBUpdate
	}{
		{
			name:    "consumed feed",
			updates: newUpdates(Body("")),
			status:  500,
			err:     "EOF",
		},
		{
			name:    "read feed",
			updates: newUpdates(Body(`{"db_name":"mailbox","type":"created","seq":"1-g1AAAAFReJzLYWBg4MhgTmHgzcvPy09JdcjLz8gvLskBCjMlMiTJ____PyuDOZExFyjAnmJhkWaeaIquGIf2JAUgmWQPMiGRAZcaB5CaePxqEkBq6vGqyWMBkgwNQAqobD4h"},`)),
			expected: &driver.DBUpdate{
				DBName: "mailbox",
				Type:   "created",
				Seq:    "1-g1AAAAFReJzLYWBg4MhgTmHgzcvPy09JdcjLz8gvLskBCjMlMiTJ____PyuDOZExFyjAnmJhkWaeaIquGIf2JAUgmWQPMiGRAZcaB5CaePxqEkBq6vGqyWMBkgwNQAqobD4h",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := new(driver.DBUpdate)
			err := test.updates.Next(result)
			testy.StatusError(t, test.err, test.status, err)
			if d := diff.Interface(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestUpdatesClose(t *testing.T) {
	body := &closeTracker{ReadCloser: Body("")}
	u := newUpdates(body)
	if err := u.Close(); err != nil {
		t.Fatal(err)
	}
	if !body.closed {
		t.Errorf("Failed to close")
	}
}

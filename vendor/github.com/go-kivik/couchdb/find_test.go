package couchdb

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/flimzy/diff"
	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/driver"
	"github.com/flimzy/kivik/errors"
	"github.com/flimzy/testy"
)

func TestExplain(t *testing.T) {
	tests := []struct {
		name     string
		db       *db
		query    interface{}
		expected *driver.QueryPlan
		status   int
		err      string
	}{
		{
			name: "CouchDB 1.6",
			db: &db{
				client: &client{Compat: CompatCouch16},
			},
			status: kivik.StatusNotImplemented,
			err:    "kivik: Find interface not implemented prior to CouchDB 2.0.0",
		},
		{
			name:   "invalid query",
			db:     newTestDB(nil, nil),
			query:  make(chan int),
			status: kivik.StatusBadRequest,
			err:    "Post http://example.com/testdb/_explain: json: unsupported type: chan int",
		},
		{
			name:   "network error",
			db:     newTestDB(nil, errors.New("net error")),
			status: kivik.StatusNetworkError,
			err:    "Post http://example.com/testdb/_explain: net error",
		},
		{
			name: "error response",
			db: newTestDB(&http.Response{
				StatusCode: kivik.StatusNotFound,
				Body:       ioutil.NopCloser(strings.NewReader("")),
			}, nil),
			status: kivik.StatusNotFound,
			err:    "Not Found",
		},
		{
			name: "success",
			db: newTestDB(&http.Response{
				StatusCode: kivik.StatusOK,
				Body:       ioutil.NopCloser(strings.NewReader(`{"dbname":"foo"}`)),
			}, nil),
			expected: &driver.QueryPlan{DBName: "foo"},
		},
		{
			name: "raw query",
			db: newCustomDB(func(req *http.Request) (*http.Response, error) {
				defer req.Body.Close() // nolint: errcheck
				var result interface{}
				if err := json.NewDecoder(req.Body).Decode(&result); err != nil {
					return nil, errors.Errorf("decode error: %s", err)
				}
				expected := map[string]interface{}{"_id": "foo"}
				if d := diff.Interface(expected, result); d != nil {
					return nil, errors.Errorf("Unexpected result:\n%s\n", d)
				}
				return nil, errors.New("success")
			}),
			query:  []byte(`{"_id":"foo"}`),
			status: kivik.StatusNetworkError,
			err:    "Post http://example.com/testdb/_explain: success",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.db.Explain(context.Background(), test.query)
			testy.StatusError(t, test.err, test.status, err)
			if d := diff.Interface(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestUnmarshalQueryPlan(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *queryPlan
		err      string
	}{
		{
			name:  "non-array",
			input: `{"fields":{}}`,
			err:   "json: cannot unmarshal object into Go",
		},
		{
			name:     "all_fields",
			input:    `{"fields":"all_fields","dbname":"foo"}`,
			expected: &queryPlan{DBName: "foo"},
		},
		{
			name:     "simple field list",
			input:    `{"fields":["foo","bar"],"dbname":"foo"}`,
			expected: &queryPlan{Fields: []interface{}{"foo", "bar"}, DBName: "foo"},
		},
		{
			name:  "complex field list",
			input: `{"dbname":"foo", "fields":[{"foo":"asc"},{"bar":"desc"}]}`,
			expected: &queryPlan{DBName: "foo",
				Fields: []interface{}{map[string]interface{}{"foo": "asc"},
					map[string]interface{}{"bar": "desc"}}},
		},
		{
			name:  "invalid bare string",
			input: `{"fields":"not_all_fields"}`,
			err:   "json: cannot unmarshal string into Go",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := new(queryPlan)
			err := json.Unmarshal([]byte(test.input), &result)
			testy.ErrorRE(t, test.err, err)
			if d := diff.Interface(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestCreateIndex(t *testing.T) {
	tests := []struct {
		name            string
		ddoc, indexName string
		index           interface{}
		db              *db
		status          int
		err             string
	}{
		{
			name:   "Couch 1.6",
			db:     &db{client: &client{Compat: CompatCouch16}},
			status: kivik.StatusNotImplemented,
			err:    "kivik: Find interface not implemented prior to CouchDB 2.0.0",
		},
		{
			name:   "invalid JSON index",
			db:     newTestDB(nil, nil),
			index:  `invalid json`,
			status: kivik.StatusBadRequest,
			err:    "invalid character 'i' looking for beginning of value",
		},
		{
			name:   "invalid raw index",
			db:     newTestDB(nil, nil),
			index:  map[string]interface{}{"foo": make(chan int)},
			status: kivik.StatusBadRequest,
			err:    "Post http://example.com/testdb/_index: json: unsupported type: chan int",
		},
		{
			name:   "network error",
			db:     newTestDB(nil, errors.New("net error")),
			status: kivik.StatusNetworkError,
			err:    "Post http://example.com/testdb/_index: net error",
		},
		{
			name: "success 2.1.0",
			db: newTestDB(&http.Response{
				StatusCode: 200,
				Header: http.Header{
					"X-CouchDB-Body-Time": {"0"},
					"X-Couch-Request-ID":  {"8e4aef0c2f"},
					"Server":              {"CouchDB/2.1.0 (Erlang OTP/17)"},
					"Date":                {"Fri, 27 Oct 2017 18:14:38 GMT"},
					"Content-Type":        {"application/json"},
					"Content-Length":      {"126"},
					"Cache-Control":       {"must-revalidate"},
				},
				Body: Body(`{"result":"created","id":"_design/a7ee061f1a2c0c6882258b2f1e148b714e79ccea","name":"a7ee061f1a2c0c6882258b2f1e148b714e79ccea"}`),
			}, nil),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.db.CreateIndex(context.Background(), test.ddoc, test.indexName, test.index)
			testy.StatusError(t, test.err, test.status, err)
		})
	}
}

func TestGetIndexes(t *testing.T) {
	tests := []struct {
		name     string
		db       *db
		expected []driver.Index
		status   int
		err      string
	}{
		{
			name:   "Couch 1.6",
			db:     &db{client: &client{Compat: CompatCouch16}},
			status: kivik.StatusNotImplemented,
			err:    "kivik: Find interface not implemented prior to CouchDB 2.0.0",
		},
		{
			name:   "network error",
			db:     newTestDB(nil, errors.New("net error")),
			status: kivik.StatusNetworkError,
			err:    "Get http://example.com/testdb/_index: net error",
		},
		{
			name: "2.1.0",
			db: newTestDB(&http.Response{
				StatusCode: 200,
				Header: http.Header{
					"X-CouchDB-Body-Time": {"0"},
					"X-Couch-Request-ID":  {"f44881735c"},
					"Server":              {"CouchDB/2.1.0 (Erlang OTP/17)"},
					"Date":                {"Fri, 27 Oct 2017 18:23:29 GMT"},
					"Content-Type":        {"application/json"},
					"Content-Length":      {"269"},
					"Cache-Control":       {"must-revalidate"},
				},
				Body: Body(`{"total_rows":2,"indexes":[{"ddoc":null,"name":"_all_docs","type":"special","def":{"fields":[{"_id":"asc"}]}},{"ddoc":"_design/a7ee061f1a2c0c6882258b2f1e148b714e79ccea","name":"a7ee061f1a2c0c6882258b2f1e148b714e79ccea","type":"json","def":{"fields":[{"foo":"asc"}]}}]}`),
			}, nil),
			expected: []driver.Index{
				{
					Name: "_all_docs",
					Type: "special",
					Definition: map[string]interface{}{
						"fields": []interface{}{
							map[string]interface{}{"_id": "asc"},
						},
					},
				},
				{
					DesignDoc: "_design/a7ee061f1a2c0c6882258b2f1e148b714e79ccea",
					Name:      "a7ee061f1a2c0c6882258b2f1e148b714e79ccea",
					Type:      "json",
					Definition: map[string]interface{}{
						"fields": []interface{}{
							map[string]interface{}{"foo": "asc"}},
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.db.GetIndexes(context.Background())
			testy.StatusError(t, test.err, test.status, err)
			if d := diff.Interface(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestDeleteIndex(t *testing.T) {
	tests := []struct {
		name            string
		ddoc, indexName string
		db              *db
		status          int
		err             string
	}{
		{
			name:   "Couch 1.6",
			db:     &db{client: &client{Compat: CompatCouch16}},
			status: kivik.StatusNotImplemented,
			err:    "kivik: Find interface not implemented prior to CouchDB 2.0.0",
		},
		{
			name:   "no ddoc",
			status: kivik.StatusBadRequest,
			db:     newTestDB(nil, nil),
			err:    "kivik: ddoc required",
		},
		{
			name:   "no index name",
			ddoc:   "foo",
			status: kivik.StatusBadRequest,
			db:     newTestDB(nil, nil),
			err:    "kivik: name required",
		},
		{
			name:      "network error",
			ddoc:      "foo",
			indexName: "bar",
			db:        newTestDB(nil, errors.New("net error")),
			status:    kivik.StatusNetworkError,
			err:       "^(Delete http://example.com/testdb/_index/foo/json/bar: )?net error",
		},
		{
			name:      "2.1.0 success",
			ddoc:      "_design/a7ee061f1a2c0c6882258b2f1e148b714e79ccea",
			indexName: "a7ee061f1a2c0c6882258b2f1e148b714e79ccea",
			db: newTestDB(&http.Response{
				StatusCode: 200,
				Header: http.Header{
					"X-CouchDB-Body-Time": {"0"},
					"X-Couch-Request-ID":  {"6018a0a693"},
					"Server":              {"CouchDB/2.1.0 (Erlang OTP/17)"},
					"Date":                {"Fri, 27 Oct 2017 19:06:28 GMT"},
					"Content-Type":        {"application/json"},
					"Content-Length":      {"11"},
					"Cache-Control":       {"must-revalidate"},
				},
				Body: Body(`{"ok":true}`),
			}, nil),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.db.DeleteIndex(context.Background(), test.ddoc, test.indexName)
			testy.StatusErrorRE(t, test.err, test.status, err)
		})
	}
}

func TestFind(t *testing.T) {
	tests := []struct {
		name   string
		db     *db
		query  interface{}
		status int
		err    string
	}{
		{
			name:   "Couch 1.6",
			db:     &db{client: &client{Compat: CompatCouch16}},
			status: kivik.StatusNotImplemented,
			err:    "kivik: Find interface not implemented prior to CouchDB 2.0.0",
		},
		{
			name:   "invalid query json",
			db:     newTestDB(nil, nil),
			query:  make(chan int),
			status: kivik.StatusBadRequest,
			err:    "Post http://example.com/testdb/_find: json: unsupported type: chan int",
		},
		{
			name:   "network error",
			db:     newTestDB(nil, errors.New("net error")),
			status: kivik.StatusNetworkError,
			err:    "Post http://example.com/testdb/_find: net error",
		},
		{
			name: "error response",
			db: newTestDB(&http.Response{
				StatusCode: 415,
				Header: http.Header{
					"Content-Type":        {"application/json"},
					"X-CouchDB-Body-Time": {"0"},
					"X-Couch-Request-ID":  {"aa1f852b27"},
					"Server":              {"CouchDB/2.1.0 (Erlang OTP/17)"},
					"Date":                {"Fri, 27 Oct 2017 19:20:04 GMT"},
					"Content-Length":      {"77"},
					"Cache-Control":       {"must-revalidate"},
				},
				ContentLength: 77,
				Body:          Body(`{"error":"bad_content_type","reason":"Content-Type must be application/json"}`),
			}, nil),
			status: kivik.StatusBadContentType,
			err:    "Unsupported Media Type: Content-Type must be application/json",
		},
		{
			name: "success 2.1.0",
			query: map[string]interface{}{
				"selector": map[string]string{"_id": "foo"},
			},
			db: newTestDB(&http.Response{
				StatusCode: 200,
				Header: http.Header{
					"Content-Type":        {"application/json"},
					"X-CouchDB-Body-Time": {"0"},
					"X-Couch-Request-ID":  {"a0884508d8"},
					"Server":              {"CouchDB/2.1.0 (Erlang OTP/17)"},
					"Date":                {"Fri, 27 Oct 2017 19:20:04 GMT"},
					"Transfer-Encoding":   {"chunked"},
					"Cache-Control":       {"must-revalidate"},
				},
				Body: Body(`{"docs":[
{"_id":"foo","_rev":"2-f5d2de1376388f1b54d93654df9dc9c7","_attachments":{"foo.txt":{"content_type":"text/plain","revpos":2,"digest":"md5-ENGoH7oK8V9R3BMnfDHZmw==","length":13,"stub":true}}}
]}`),
			}, nil),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.db.Find(context.Background(), test.query)
			testy.StatusError(t, test.err, test.status, err)
			if _, ok := result.(*rows); !ok {
				t.Errorf("Unexpected type returned: %t", result)
			}
		})
	}
}

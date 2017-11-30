package couchdb

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/flimzy/kivik"
	"github.com/flimzy/testy"
)

func TestNewClient(t *testing.T) {
	type ncTest struct {
		name    string
		dsn     string
		status  int
		err     string
		cleanup func()
	}
	tests := []ncTest{
		{
			name:   "invalid url",
			dsn:    "foo.com/%xxx",
			status: kivik.StatusBadRequest,
			err:    `parse foo.com/%xxx: invalid URL escape "%xx"`,
		},
		func() ncTest {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
			s := httptest.NewServer(handler)
			return ncTest{
				name: "success",
				dsn:  s.URL,
				cleanup: func() {
					s.Close()
				},
			}
		}(),
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.cleanup != nil {
				defer test.cleanup()
			}
			driver := &Couch{}
			result, err := driver.NewClient(context.Background(), test.dsn)
			testy.StatusError(t, test.err, test.status, err)
			if _, ok := result.(*client); !ok {
				t.Errorf("Unexpected type returned: %t", result)
			}
		})
	}
}

func TestSetCompatMode(t *testing.T) {
	tests := []struct {
		name     string
		client   *client
		expected CompatMode
	}{
		{
			name:   "error response",
			client: newTestClient(nil, errors.New("some error")),
		},
		{
			name: "1.6.1",
			client: newTestClient(&http.Response{
				StatusCode: 200,
				Header: http.Header{
					"Server":         {"CouchDB/1.6.1 (Erlang OTP/17)"},
					"Date":           {"Fri, 27 Oct 2017 17:32:13 GMT"},
					"Content-Type":   {"application/json"},
					"Content-Length": {"151"},
					"Cache-Control":  {"must-revalidate"},
				},
				Body: Body(`{"couchdb":"Welcome","uuid":"ad577ffa1c26fae018f5fd8980a81854","version":"1.6.1","vendor":{"version":"1.6.1","name":"The Apache Software Foundation"}}`),
			}, nil),
			expected: CompatCouch16,
		},
		{
			name: "1.7.0",
			client: newTestClient(&http.Response{
				StatusCode: 200,
				Header: http.Header{
					"Server":         {"CouchDB/1.7.0 (Erlang OTP/17)"},
					"Date":           {"Fri, 27 Oct 2017 17:32:13 GMT"},
					"Content-Type":   {"application/json"},
					"Content-Length": {"151"},
					"Cache-Control":  {"must-revalidate"},
				},
				Body: Body(`{"couchdb":"Welcome","uuid":"7962695b9f542ce8693fa209044d051d","version":"1.7.0","vendor":{"version":"1.7.0","name":"The Apache Software Foundation"}}`),
			}, nil),
			expected: CompatCouch16,
		},
		{
			name: "1.7.1",
			client: newTestClient(&http.Response{
				StatusCode: 200,
				Header: http.Header{
					"Server":         {"CouchDB/1.7.1 (Erlang OTP/17)"},
					"Date":           {"Fri, 27 Oct 2017 17:32:13 GMT"},
					"Content-Type":   {"application/json"},
					"Content-Length": {"151"},
					"Cache-Control":  {"must-revalidate"},
				},
				Body: Body(`{"couchdb":"Welcome","uuid":"7962695b9f542ce8693fa209044d051d","version":"1.7.1","vendor":{"version":"1.7.1","name":"The Apache Software Foundation"}}`),
			}, nil),
			expected: CompatCouch16,
		},
		{
			name: "2.0.0",
			client: newTestClient(&http.Response{
				StatusCode: 200,
				Header: http.Header{
					"Server":              {"CouchDB/2.0.0 (Erlang OTP/17)"},
					"Date":                {"Fri, 27 Oct 2017 17:32:13 GMT"},
					"Content-Type":        {"application/json"},
					"Content-Length":      {"90"},
					"Cache-Control":       {"must-revalidate"},
					"X-Couch-Request-ID":  {"44e64ecb76"},
					"X-CouchDB-Body-Time": {"0"},
				},
				Body: Body(`{"couchdb":"Welcome","version":"2.0.0","vendor":{"name":"The Apache Software Foundation"}}`),
			}, nil),
			expected: CompatCouch20,
		},
		{
			name: "2.1.0",
			client: newTestClient(&http.Response{
				StatusCode: 200,
				Header: http.Header{
					"Server":              {"CouchDB/2.1.0 (Erlang OTP/17)"},
					"Date":                {"Fri, 27 Oct 2017 17:32:13 GMT"},
					"Content-Type":        {"application/json"},
					"Content-Length":      {"115"},
					"Cache-Control":       {"must-revalidate"},
					"X-Couch-Request-ID":  {"9d387d5370"},
					"X-CouchDB-Body-Time": {"0"},
				},
				Body: Body(`{"couchdb":"Welcome","version":"2.1.0","features":["scheduler"],"vendor":{"name":"The Apache Software Foundation"}}`),
			}, nil),
			expected: CompatCouch20,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.client.setCompatMode(context.Background())
			if test.client.Compat != test.expected {
				t.Errorf("Unexpected compat mode: %d", test.client.Compat)
			}
		})
	}
}

func TestDB(t *testing.T) {
	tests := []struct {
		name     string
		client   *client
		dbName   string
		options  map[string]interface{}
		expected *db
		status   int
		err      string
	}{
		{
			name:   "no dbname",
			status: kivik.StatusBadRequest,
			err:    "kivik: dbName required",
		},
		{
			name:    "invalid full commit type",
			dbName:  "foo",
			options: map[string]interface{}{OptionFullCommit: 123},
			status:  kivik.StatusBadRequest,
			err:     "kivik: option 'X-Couch-Full-Commit' must be bool, not int",
		},
		{
			name:    "full commit",
			dbName:  "foo",
			options: map[string]interface{}{OptionFullCommit: true},
			expected: &db{
				dbName:     "foo",
				fullCommit: true,
			},
		},
		{
			name:   "no full commit",
			dbName: "foo",
			expected: &db{
				dbName: "foo",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.client.DB(context.Background(), test.dbName, test.options)
			testy.StatusError(t, test.err, test.status, err)
			if _, ok := result.(*db); !ok {
				t.Errorf("Unexpected result type: %T", result)
			}
		})
	}
}

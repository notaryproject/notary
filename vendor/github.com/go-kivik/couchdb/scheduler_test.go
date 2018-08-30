package couchdb

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/flimzy/diff"
	"github.com/flimzy/testy"

	"github.com/go-kivik/kivik"
	"github.com/go-kivik/kivik/driver"
)

func TestSRUpdate(t *testing.T) {
	tests := []struct {
		name     string
		rep      *schedulerReplication
		status   int
		err      string
		expected *driver.ReplicationInfo
	}{
		{
			name: "network error",
			rep: &schedulerReplication{
				database: "_replicator",
				docID:    "foo",
				db:       newTestDB(nil, errors.New("net error")),
			},
			status: kivik.StatusNetworkError,
			err:    "Get http://example.com/_scheduler/docs/_replicator/foo: net error",
		},
		{
			name: "real example",
			rep: &schedulerReplication{
				database: "_replicator",
				docID:    "foo2",
				db: newTestDB(&http.Response{
					StatusCode: 200,
					Header: http.Header{
						"Server":         {"CouchDB/2.1.0 (Erlang OTP/17)"},
						"Date":           {"Thu, 09 Nov 2017 15:23:20 GMT"},
						"Content-Type":   {"application/json"},
						"Content-Length": {"687"},
						"Cache-Control":  {"must-revalidate"},
					},
					Body: Body(`{"database":"_replicator","doc_id":"foo2","id":null,"source":"http://localhost:5984/foo/","target":"http://localhost:5984/bar/","state":"completed","error_count":0,"info":{"revisions_checked":23,"missing_revisions_found":23,"docs_read":23,"docs_written":23,"changes_pending":null,"doc_write_failures":0,"checkpointed_source_seq":"27-g1AAAAIbeJyV0EsOgjAQBuAGMOLCM-gRSoUKK7mJ9kWQYLtQ13oTvYneRG-CfZAYSUjqZppM5v_SmRYAENchB3OppOKilKpWx1Or2wEBdNF1XVOHJD7oxnTFKMOcDYdH4nSpK930wsQKAmYIVdBXKI2w_RGQyFJYFb7CzgiXXgDuDywXKUk4mJ0lF9VeCj6SlpGu4KofDdyMEFoBk3QtMt87OOXulIdRAqvABHPO0F_K0ymv7zYU5UVe-W_zdoK9R2QFxhjBUAwzzQch86VT"},"start_time":"2017-11-01T21:05:03Z","last_updated":"2017-11-01T21:05:06Z"}`),
				}, nil),
			},
			expected: &driver.ReplicationInfo{
				DocsRead:    23,
				DocsWritten: 23,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var result driver.ReplicationInfo
			err := test.rep.Update(context.Background(), &result)
			testy.StatusError(t, test.err, test.status, err)
			if d := diff.Interface(test.expected, &result); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestRepInfoUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *repInfo
		err      string
	}{
		{
			name:     "null",
			input:    "null",
			expected: &repInfo{},
		},
		{
			name:  "error string",
			input: `"db_not_found: could not open foo"`,
			expected: &repInfo{
				Error: &replicationError{
					status: 404,
					reason: "db_not_found: could not open foo",
				},
			},
		},
		{
			name:  "stats",
			input: `{"revisions_checked":23,"missing_revisions_found":23,"docs_read":23,"docs_written":23,"changes_pending":null,"doc_write_failures":0,"checkpointed_source_seq":"27-g1AAAAIbeJyV0EsOgjAQBuAGMOLCM-gRSoUKK7mJ9kWQYLtQ13oTvYneRG-CfZAYSUjqZppM5v_SmRYAENchB3OppOKilKpWx1Or2wEBdNF1XVOHJD7oxnTFKMOcDYdH4nSpK930wsQKAmYIVdBXKI2w_RGQyFJYFb7CzgiXXgDuDywXKUk4mJ0lF9VeCj6SlpGu4KofDdyMEFoBk3QtMt87OOXulIdRAqvABHPO0F_K0ymv7zYU5UVe-W_zdoK9R2QFxhjBUAwzzQch86VT"}`,
			expected: &repInfo{
				DocsRead:         23,
				DocsWritten:      23,
				DocWriteFailures: 0,
			},
		},
		{
			name:  "invalid stats object",
			input: `{"docs_written":"chicken"}`,
			err:   "^json: cannot unmarshal string into Go ",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := new(repInfo)
			err := json.Unmarshal([]byte(test.input), result)
			testy.ErrorRE(t, test.err, err)
			if d := diff.Interface(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestGetReplicationsFromScheduler(t *testing.T) {
	tests := []struct {
		name     string
		options  map[string]interface{}
		client   *client
		expected []*schedulerReplication
		status   int
		err      string
	}{
		{
			name:   "network error",
			client: newTestClient(nil, errors.New("net error")),
			status: kivik.StatusNetworkError,
			err:    "Get http://example.com/_scheduler/docs: net error",
		},
		{
			name:    "invalid options",
			options: map[string]interface{}{"foo": make(chan int)},
			status:  kivik.StatusBadRequest,
			err:     "kivik: invalid type chan int for options",
		},
		{
			name: "valid response, 2.1.0",
			client: newTestClient(&http.Response{
				StatusCode: 200,
				Header: http.Header{
					"Server":              {"CouchDB/2.1.0 (Erlang OTP/17)"},
					"Date":                {"Wed, 08 Nov 2017 18:04:11 GMT"},
					"Content-Type":        {"application/json"},
					"Transfer-Encoding":   {"chunked"},
					"Cache-Control":       {"must-revalidate"},
					"X-CouchDB-Body-Time": {"0"},
					"X-Couch-Request-ID":  {"6d47891c37"},
				},
				Body: Body(`{"total_rows":2,"offset":0,"docs":[
{"database":"_replicator","doc_id":"foo","id":"81cc3633ee8de1332e412ef9052c7b6f","node":"nonode@nohost","source":"foo","target":"bar","state":"crashing","info":"db_not_found: could not open foo","error_count":6,"last_updated":"2017-11-08T18:07:38Z","start_time":"2017-11-08T17:51:52Z","proxy":null},
{"database":"_replicator","doc_id":"foo2","id":null,"source":"http://admin:*****@localhost:5984/foo/","target":"http://admin:*****@localhost:5984/bar/","state":"completed","error_count":0,"info":{"revisions_checked":23,"missing_revisions_found":23,"docs_read":23,"docs_written":23,"changes_pending":null,"doc_write_failures":0,"checkpointed_source_seq":"27-g1AAAAIbeJyV0EsOgjAQBuAGMOLCM-gRSoUKK7mJ9kWQYLtQ13oTvYneRG-CfZAYSUjqZppM5v_SmRYAENchB3OppOKilKpWx1Or2wEBdNF1XVOHJD7oxnTFKMOcDYdH4nSpK930wsQKAmYIVdBXKI2w_RGQyFJYFb7CzgiXXgDuDywXKUk4mJ0lF9VeCj6SlpGu4KofDdyMEFoBk3QtMt87OOXulIdRAqvABHPO0F_K0ymv7zYU5UVe-W_zdoK9R2QFxhjBUAwzzQch86VT"},"start_time":"2017-11-01T21:05:03Z","last_updated":"2017-11-01T21:05:06Z"}
]}`),
			}, nil),
			expected: []*schedulerReplication{
				{
					database:      "_replicator",
					docID:         "foo",
					replicationID: "81cc3633ee8de1332e412ef9052c7b6f",
					state:         "crashing",
					source:        "foo",
					target:        "bar",
					startTime:     parseTime(t, "2017-11-08T17:51:52Z"),
					lastUpdated:   parseTime(t, "2017-11-08T18:07:38Z"),
					info: repInfo{
						Error: &replicationError{
							status: 404,
							reason: "db_not_found: could not open foo",
						},
					},
				},
				{
					database:    "_replicator",
					docID:       "foo2",
					source:      "http://admin:*****@localhost:5984/foo/",
					target:      "http://admin:*****@localhost:5984/bar/",
					state:       "completed",
					startTime:   parseTime(t, "2017-11-01T21:05:03Z"),
					lastUpdated: parseTime(t, "2017-11-01T21:05:06Z"),
					info: repInfo{
						DocsRead:    23,
						DocsWritten: 23,
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reps, err := test.client.getReplicationsFromScheduler(context.Background(), test.options)
			testy.StatusError(t, test.err, test.status, err)
			result := make([]*schedulerReplication, len(reps))
			for i, rep := range reps {
				result[i] = rep.(*schedulerReplication)
				result[i].db = nil
			}
			if d := diff.Interface(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestSchedulerReplicationDelete(t *testing.T) {
	tests := []struct {
		name   string
		rep    *schedulerReplication
		status int
		err    string
	}{
		{
			name: "HEAD network error",
			rep: &schedulerReplication{
				docID: "foo",
				db:    newTestDB(nil, errors.New("net error")),
			},
			status: kivik.StatusNetworkError,
			err:    "Head http://example.com/testdb/foo: net error",
		},
		{
			name: "DELETE network error",
			rep: &schedulerReplication{
				docID: "foo",
				db: newCustomDB(func(r *http.Request) (*http.Response, error) {
					if r.Method == http.MethodHead {
						return &http.Response{
							StatusCode: 200,
							Header: http.Header{
								"ETag": {`"9-b38287cbde7623a328843f830f418c92"`},
							},
							Body: Body(""),
						}, nil
					}
					return nil, errors.New("net error")
				}),
			},
			status: kivik.StatusNetworkError,
			err:    "(Delete http://example.com/testdb/foo?rev=9-b38287cbde7623a328843f830f418c92: )?net error",
		},
		{
			name: "success",
			rep: &schedulerReplication{
				docID: "foo",
				db: newCustomDB(func(r *http.Request) (*http.Response, error) {
					if r.Method == http.MethodHead {
						return &http.Response{
							StatusCode: 200,
							Header: http.Header{
								"ETag": {`"9-b38287cbde7623a328843f830f418c92"`},
							},
							Body: Body(""),
						}, nil
					}
					expected := "http://example.com/testdb/foo?rev=9-b38287cbde7623a328843f830f418c92"
					if r.URL.String() != expected {
						panic("Unexpected url: " + r.URL.String())
					}
					return &http.Response{
						StatusCode: 200,
						Header: http.Header{
							"X-CouchDB-Body-Time": {"0"},
							"X-Couch-Request-ID":  {"03b7ff8976"},
							"Server":              {"CouchDB/2.1.0 (Erlang OTP/17)"},
							"ETag":                {`"10-a4f1941d02a2bcc6b4fe8a463dbea746"`},
							"Date":                {"Sat, 11 Nov 2017 16:28:26 GMT"},
							"Content-Type":        {"application/json"},
							"Content-Length":      {"67"},
							"Cache-Control":       {"must-revalidate"},
						},
						Body: Body(`{"ok":true,"id":"foo","rev":"10-a4f1941d02a2bcc6b4fe8a463dbea746"}`),
					}, nil
				}),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.rep.Delete(context.Background())
			testy.StatusErrorRE(t, test.err, test.status, err)
		})
	}
}

func TestSchedulerReplicationGetters(t *testing.T) {
	repID := "a"
	source := "b"
	target := "c"
	state := "completed"
	err := "e"
	start := parseTime(t, "2017-01-01T01:01:01Z")
	end := parseTime(t, "2017-01-01T01:01:02Z")
	rep := &schedulerReplication{
		replicationID: repID,
		source:        source,
		target:        target,
		startTime:     start,
		lastUpdated:   end,
		state:         state,
		info:          repInfo{Error: errors.New(err)},
	}
	if result := rep.ReplicationID(); result != repID {
		t.Errorf("Unexpected replication ID: %s", result)
	}
	if result := rep.Source(); result != source {
		t.Errorf("Unexpected source: %s", result)
	}
	if result := rep.Target(); result != target {
		t.Errorf("Unexpected target: %s", result)
	}
	if result := rep.StartTime(); !result.Equal(start) {
		t.Errorf("Unexpected start time: %v", result)
	}
	if result := rep.EndTime(); !result.Equal(end) {
		t.Errorf("Unexpected end time: %v", result)
	}
	if result := rep.State(); result != state {
		t.Errorf("Unexpected state: %s", result)
	}
	testy.Error(t, err, rep.Err())
}

func TestSchedulerSupported(t *testing.T) {
	supported := true
	unsupported := false
	tests := []struct {
		name          string
		client        *client
		expected      bool
		expectedState *bool
		status        int
		err           string
	}{
		{
			name:          "already set true",
			client:        &client{schedulerDetected: func() *bool { b := true; return &b }()},
			expected:      true,
			expectedState: &supported,
		},
		{
			name: "1.6.1, not supported",
			client: newTestClient(&http.Response{
				StatusCode: 400,
				Header: http.Header{
					"Server":         {"CouchDB/1.6.1 (Erlang OTP/17)"},
					"Date":           {"Thu, 16 Nov 2017 17:37:32 GMT"},
					"Content-Type":   {"application/json"},
					"Content-Length": {"201"},
					"Cache-Control":  {"must-revalidate"},
				},
				Request: &http.Request{Method: "HEAD"},
				Body:    Body(""),
			}, nil),
			expected:      false,
			expectedState: &unsupported,
		},
		{
			name: "1.7.1, not supported",
			client: newTestClient(&http.Response{
				StatusCode: 400,
				Header: http.Header{
					"Server":         {"CouchDB/1.7.1 (Erlang OTP/17)"},
					"Date":           {"Thu, 16 Nov 2017 17:37:32 GMT"},
					"Content-Type":   {"application/json"},
					"Content-Length": {"201"},
					"Cache-Control":  {"must-revalidate"},
				},
				Request: &http.Request{Method: "HEAD"},
				Body:    Body(""),
			}, nil),
			expected:      false,
			expectedState: &unsupported,
		},
		{
			name: "2.0.0, not supported",
			client: newTestClient(&http.Response{
				StatusCode: 404,
				Header: http.Header{
					"Cache-Control":       {"must-revalidate"},
					"Content-Length":      {"58"},
					"Content-Type":        {"application/json"},
					"Date":                {"Thu, 16 Nov 2017 17:45:34 GMT"},
					"Server":              {"CouchDB/2.0.0 (Erlang OTP/17)"},
					"X-Couch-Request-ID":  {"027c1e7ffe"},
					"X-CouchDB-Body-Time": {"0"},
				},
				Request: &http.Request{Method: "HEAD"},
				Body:    Body(""),
			}, nil),
			expected:      false,
			expectedState: &unsupported,
		},
		{
			name: "2.1.1, supported",
			client: newTestClient(&http.Response{
				StatusCode: 200,
				Header: http.Header{
					"Server":         {"CouchDB/2.1.0 (Erlang OTP/17)"},
					"Date":           {"Thu, 16 Nov 2017 17:47:58 GMT"},
					"Content-Type":   {"application/json"},
					"Content-Length": {"38"},
					"Cache-Control":  {"must-revalidate"},
				},
				Request: &http.Request{Method: "HEAD"},
				Body:    Body(""),
			}, nil),
			expected:      true,
			expectedState: &supported,
		},
		{
			name:          "network error",
			client:        newTestClient(nil, errors.New("net error")),
			expectedState: nil,
			status:        kivik.StatusNetworkError,
			err:           "Head http://example.com/_scheduler/jobs: net error",
		},
		{
			name: "Unexpected response code",
			client: newTestClient(&http.Response{
				StatusCode: 500,
				Request:    &http.Request{Method: "HEAD"},
				Body:       Body(""),
			}, nil),
			expectedState: nil,
			status:        kivik.StatusBadResponse,
			err:           "Unknown response code 500",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.client.schedulerSupported(context.Background())
			if d := diff.Interface(test.expectedState, test.client.schedulerDetected); d != nil {
				t.Error(d)
			}
			testy.StatusError(t, test.err, test.status, err)
			if result != test.expected {
				t.Errorf("Unexpected result: %v", result)
			}
		})
	}
}

func TestSRinnerUpdate(t *testing.T) {
	tests := []struct {
		name     string
		r        *schedulerReplication
		status   int
		err      string
		expected *schedulerReplication
	}{
		{
			name: "network error",
			r: &schedulerReplication{
				database: "_replicator",
				docID:    "foo",
				db:       newTestDB(nil, errors.New("net error")),
			},
			status: kivik.StatusNetworkError,
			err:    "Get http://example.com/_scheduler/docs/_replicator/foo: net error",
		},
		{
			name: "2.1.1 500 bug",
			r: &schedulerReplication{
				database: "_replicator",
				docID:    "foo",
				db: func() *db {
					var count int
					db := newCustomDB(func(req *http.Request) (*http.Response, error) {
						if count == 0 {
							count++
							return &http.Response{
								StatusCode: 500,
								Header: http.Header{
									"Content-Length":      {"70"},
									"Cache-Control":       {"must-revalidate"},
									"Content-Type":        {"application/json"},
									"Date":                {"Thu, 16 Nov 2017 20:14:25 GMT"},
									"Server":              {"CouchDB/2.1.0 (Erlang OTP/17)"},
									"X-Couch-Request-Id":  {"65913f4727"},
									"X-Couch-Stack-Hash":  {"3194022798"},
									"X-Couchdb-Body-Time": {"0"},
								},
								Request:       &http.Request{Method: "GET"},
								ContentLength: 70,
								Body:          Body(`{"error":"unknown_error","reason":"function_clause","ref":3194022798}`),
							}, nil
						}
						return &http.Response{
							StatusCode: 200,
							Header: http.Header{
								"Server":         {"CouchDB/2.1.0 (Erlang OTP/17)"},
								"Date":           {"Thu, 09 Nov 2017 15:23:20 GMT"},
								"Content-Type":   {"application/json"},
								"Content-Length": {"687"},
								"Cache-Control":  {"must-revalidate"},
							},
							Body: Body(`{"database":"_replicator","doc_id":"foo2","id":null,"source":"http://localhost:5984/foo/","target":"http://localhost:5984/bar/","state":"completed","error_count":0,"info":{"revisions_checked":23,"missing_revisions_found":23,"docs_read":23,"docs_written":23,"changes_pending":null,"doc_write_failures":0,"checkpointed_source_seq":"27-g1AAAAIbeJyV0EsOgjAQBuAGMOLCM-gRSoUKK7mJ9kWQYLtQ13oTvYneRG-CfZAYSUjqZppM5v_SmRYAENchB3OppOKilKpWx1Or2wEBdNF1XVOHJD7oxnTFKMOcDYdH4nSpK930wsQKAmYIVdBXKI2w_RGQyFJYFb7CzgiXXgDuDywXKUk4mJ0lF9VeCj6SlpGu4KofDdyMEFoBk3QtMt87OOXulIdRAqvABHPO0F_K0ymv7zYU5UVe-W_zdoK9R2QFxhjBUAwzzQch86VT"},"start_time":"2017-11-01T21:05:03Z","last_updated":"2017-11-01T21:05:06Z"}`),
						}, nil
					})
					return db
				}(),
			},
			expected: &schedulerReplication{
				docID:       "foo2",
				database:    "_replicator",
				source:      "http://localhost:5984/foo/",
				target:      "http://localhost:5984/bar/",
				startTime:   parseTime(t, "2017-11-01T21:05:03Z"),
				lastUpdated: parseTime(t, "2017-11-01T21:05:06Z"),
				state:       "completed",
				info: repInfo{
					DocsRead:    23,
					DocsWritten: 23,
				},
			},
		},
		{
			name: "db not found",
			r: &schedulerReplication{
				database: "_replicator",
				docID:    "56d257bd2125c8f15870b3ddd202c4ca",
				db: newTestDB(&http.Response{
					StatusCode: 200,
					Header: http.Header{
						"Server":         {"CouchDB/2.1.0 (Erlang OTP/17)"},
						"Date":           {"Fri, 17 Nov 2017 13:05:52 GMT"},
						"Content-Type":   {"application/json"},
						"Content-Length": {"328"},
						"Cache-Control":  {"must-revalidate"},
					},
					Body: Body(`{"database":"_replicator","doc_id":"56d257bd2125c8f15870b3ddd202c4ca","id":"c636d089fbdc3a9a937a466acf8f42c3","node":"nonode@nohost","source":"foo","target":"bar","state":"crashing","info":"db_not_found: could not open foo","error_count":7,"last_updated":"2017-11-17T12:59:35Z","start_time":"2017-11-17T12:22:25Z","proxy":null}`),
				}, nil),
			},
			expected: &schedulerReplication{
				docID:         "56d257bd2125c8f15870b3ddd202c4ca",
				database:      "_replicator",
				replicationID: "c636d089fbdc3a9a937a466acf8f42c3",
				source:        "foo",
				target:        "bar",
				startTime:     parseTime(t, "2017-11-17T12:22:25Z"),
				lastUpdated:   parseTime(t, "2017-11-17T12:59:35Z"),
				state:         "crashing",
				info: repInfo{
					Error: &replicationError{
						status: 404,
						reason: "db_not_found: could not open foo",
					},
				},
			},
		},
		{
			name: "null time",
			r: &schedulerReplication{
				database: "_replicator",
				docID:    "56d257bd2125c8f15870b3ddd202c4ca",
				db: newTestDB(&http.Response{
					StatusCode: 200,
					Header: http.Header{
						"Server":         {"CouchDB/2.1.0 (Erlang OTP/17)"},
						"Date":           {"Fri, 17 Nov 2017 13:05:52 GMT"},
						"Content-Type":   {"application/json"},
						"Content-Length": {"275"},
						"Cache-Control":  {"must-revalidate"},
					},
					Body: Body(`{"database":"_replicator","doc_id":"733c70a35768b7a8fc2e178bd9003f1b","id":null,"source":"http://localhost:5984/kivik$replicate_rw_admin$5fbcf68d8d9aaee0/","target":"http://localhost:5984/foo/","state":null,"error_count":0,"info":null,"start_time":null,"last_updated":null}`),
				}, nil),
			},
			expected: &schedulerReplication{
				docID:         "733c70a35768b7a8fc2e178bd9003f1b",
				database:      "_replicator",
				replicationID: "",
				source:        "http://localhost:5984/kivik$replicate_rw_admin$5fbcf68d8d9aaee0/",
				target:        "http://localhost:5984/foo/",
				startTime:     time.Time{},
				lastUpdated:   time.Time{},
				state:         "",
				info:          repInfo{},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.r.update(context.Background())
			testy.StatusError(t, test.err, test.status, err)
			test.r.db = nil
			if d := diff.Interface(test.expected, test.r); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestFetchSchedulerReplication(t *testing.T) {
	tests := []struct {
		name     string
		client   *client
		docID    string
		expected *schedulerReplication
		status   int
		err      string
	}{
		{
			name:   "network error",
			client: newTestClient(nil, errors.New("net error")),
			status: kivik.StatusNetworkError,
			err:    "Get http://example.com/_scheduler/docs/_replicator/: net error",
		},
		{
			name: "loop wait",
			client: func() *client {
				var count int
				return newCustomClient(func(_ *http.Request) (*http.Response, error) {
					if count < 2 {
						count++
						return &http.Response{
							StatusCode: 200,
							Body:       Body(`{"database":"_replicator","doc_id":"56d257bd2125c8f15870b3ddd2074759","id":null,"state":"initializing","info":null,"error_count":0,"node":"nonode@nohost","last_updated":"2017-11-17T19:56:09Z","start_time":"2017-11-17T19:56:09Z"}`),
						}, nil
					}
					return &http.Response{
						StatusCode: 200,
						Body:       Body(`{"database":"_replicator","doc_id":"56d257bd2125c8f15870b3ddd2074759","id":"c636d089fbdc3a9a937a466acf8f42c3","node":"nonode@nohost","source":"foo","target":"bar","state":"crashing","info":"db_not_found: could not open foo","error_count":1,"last_updated":"2017-11-17T19:57:09Z","start_time":"2017-11-17T19:56:09Z","proxy":null}`),
					}, nil
				})
			}(),
			expected: &schedulerReplication{
				docID:         "56d257bd2125c8f15870b3ddd2074759",
				database:      "_replicator",
				replicationID: "c636d089fbdc3a9a937a466acf8f42c3",
				source:        "foo",
				target:        "bar",
				startTime:     parseTime(t, "2017-11-17T19:56:09Z"),
				lastUpdated:   parseTime(t, "2017-11-17T19:57:09Z"),
				state:         "crashing",
				info: repInfo{
					Error: &replicationError{
						status: 404,
						reason: "db_not_found: could not open foo",
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.client.fetchSchedulerReplication(context.Background(), test.docID)
			testy.StatusError(t, test.err, test.status, err)
			result.db = nil
			if d := diff.Interface(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}

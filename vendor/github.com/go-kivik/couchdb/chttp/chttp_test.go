package chttp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/flimzy/diff"
	"github.com/flimzy/testy"
	"golang.org/x/net/publicsuffix"

	"github.com/go-kivik/kivik"
	"github.com/go-kivik/kivik/errors"
)

func TestNew(t *testing.T) {
	type newTest struct {
		name     string
		dsn      string
		expected *Client
		status   int
		err      string
	}
	tests := []newTest{
		{
			name:   "invalid url",
			dsn:    "http://foo.com/%xx",
			status: kivik.StatusBadRequest,
			err:    `parse http://foo.com/%xx: invalid URL escape "%xx"`,
		},
		{
			name: "no auth",
			dsn:  "http://foo.com/",
			expected: &Client{
				Client: &http.Client{},
				rawDSN: "http://foo.com/",
				dsn: &url.URL{
					Scheme: "http",
					Host:   "foo.com",
					Path:   "/",
				},
			},
		},
		func() newTest {
			h := func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(kivik.StatusUnauthorized)
			}
			s := httptest.NewServer(http.HandlerFunc(h))
			dsn, _ := url.Parse(s.URL)
			dsn.User = url.UserPassword("user", "password")
			return newTest{
				name:   "auth failed",
				dsn:    dsn.String(),
				status: kivik.StatusUnauthorized,
				err:    "Unauthorized",
			}
		}(),
		func() newTest {
			h := func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(kivik.StatusOK)
				fmt.Fprintf(w, `{"userCtx":{"name":"user"}}`)
			}
			s := httptest.NewServer(http.HandlerFunc(h))
			authDSN, _ := url.Parse(s.URL)
			dsn, _ := url.Parse(s.URL)
			authDSN.User = url.UserPassword("user", "password")
			jar, _ := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
			return newTest{
				name: "auth success",
				dsn:  authDSN.String(),
				expected: &Client{
					Client: &http.Client{Jar: jar},
					rawDSN: authDSN.String(),
					dsn:    dsn,
					auth: &CookieAuth{
						Username: "user",
						Password: "password",
						dsn:      dsn,
						setJar:   true,
						jar:      jar,
					},
				},
			}
		}(),
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := New(context.Background(), test.dsn)
			testy.StatusError(t, test.err, test.status, err)
			if d := diff.Interface(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestDSN(t *testing.T) {
	expected := "foo"
	client := &Client{rawDSN: expected}
	result := client.DSN()
	if result != expected {
		t.Errorf("Unexpected result: %s", result)
	}
}

func TestFixPath(t *testing.T) {
	tests := []struct {
		Input    string
		Expected string
	}{
		{Input: "foo", Expected: "/foo"},
		{Input: "foo?oink=yes", Expected: "/foo"},
		{Input: "foo/bar", Expected: "/foo/bar"},
		{Input: "foo%2Fbar", Expected: "/foo%2Fbar"},
	}
	for _, test := range tests {
		req, _ := http.NewRequest("GET", "http://localhost/"+test.Input, nil)
		fixPath(req, test.Input)
		if req.URL.EscapedPath() != test.Expected {
			t.Errorf("Path for '%s' not fixed.\n\tExpected: %s\n\t  Actual: %s\n", test.Input, test.Expected, req.URL.EscapedPath())
		}
	}
}

func TestEncodeBody(t *testing.T) {
	type encodeTest struct {
		name  string
		input interface{}

		expected string
		status   int
		err      string
	}
	tests := []encodeTest{
		{
			name:     "Null",
			input:    nil,
			expected: "null",
		},
		{
			name: "Struct",
			input: struct {
				Foo string `json:"foo"`
			}{Foo: "bar"},
			expected: `{"foo":"bar"}`,
		},
		{
			name:   "JSONError",
			input:  func() {}, // Functions cannot be marshaled to JSON
			status: kivik.StatusBadRequest,
			err:    "json: unsupported type: func()",
		},
		{
			name:     "raw json input",
			input:    json.RawMessage(`{"foo":"bar"}`),
			expected: `{"foo":"bar"}`,
		},
		{
			name:     "byte slice input",
			input:    []byte(`{"foo":"bar"}`),
			expected: `{"foo":"bar"}`,
		},
		{
			name:     "string input",
			input:    `{"foo":"bar"}`,
			expected: `{"foo":"bar"}`,
		},
	}
	for _, test := range tests {
		func(test encodeTest) {
			t.Run(test.name, func(t *testing.T) {
				t.Parallel()
				r := EncodeBody(test.input)
				defer r.Close() // nolint: errcheck
				body, err := ioutil.ReadAll(r)
				testy.StatusError(t, test.err, test.status, err)
				result := strings.TrimSpace(string(body))
				if result != test.expected {
					t.Errorf("Result\nExpected: %s\n  Actual: %s\n", test.expected, result)
				}
			})
		}(test)
	}
}

func TestSetHeaders(t *testing.T) {
	type shTest struct {
		Name     string
		Options  *Options
		Expected http.Header
	}
	tests := []shTest{
		{
			Name: "NoOpts",
			Expected: http.Header{
				"Accept":       {"application/json"},
				"Content-Type": {"application/json"},
			},
		},
		{
			Name:    "Content-Type",
			Options: &Options{ContentType: "image/gif"},
			Expected: http.Header{
				"Accept":       {"application/json"},
				"Content-Type": {"image/gif"},
			},
		},
		{
			Name:    "Accept",
			Options: &Options{Accept: "image/gif"},
			Expected: http.Header{
				"Accept":       {"image/gif"},
				"Content-Type": {"application/json"},
			},
		},
		{
			Name:    "FullCommit",
			Options: &Options{FullCommit: true},
			Expected: http.Header{
				"Accept":              {"application/json"},
				"Content-Type":        {"application/json"},
				"X-Couch-Full-Commit": {"true"},
			},
		},
		{
			Name:    "Destination",
			Options: &Options{Destination: "somewhere nice"},
			Expected: http.Header{
				"Accept":       {"application/json"},
				"Content-Type": {"application/json"},
				"Destination":  {"somewhere nice"},
			},
		},
		{
			Name:    "If-None-Match",
			Options: &Options{IfNoneMatch: `"foo"`},
			Expected: http.Header{
				"Accept":        {"application/json"},
				"Content-Type":  {"application/json"},
				"If-None-Match": {`"foo"`},
			},
		},
	}
	for _, test := range tests {
		func(test shTest) {
			t.Run(test.Name, func(t *testing.T) {
				t.Parallel()
				req, err := http.NewRequest("GET", "/", nil)
				if err != nil {
					panic(err)
				}
				setHeaders(req, test.Options)
				if d := diff.Interface(test.Expected, req.Header); d != nil {
					t.Errorf("Headers:\n%s\n", d)
				}
			})
		}(test)
	}
}

func TestETag(t *testing.T) {
	tests := []struct {
		name     string
		input    *http.Response
		expected string
		found    bool
	}{
		{
			name:     "nil response",
			input:    nil,
			expected: "",
			found:    false,
		},
		{
			name:     "No etag",
			input:    &http.Response{},
			expected: "",
			found:    false,
		},
		{
			name: "ETag",
			input: &http.Response{
				Header: http.Header{
					"ETag": {`"foo"`},
				},
			},
			expected: "foo",
			found:    true,
		},
		{
			name: "Etag",
			input: &http.Response{
				Header: http.Header{
					"Etag": {`"bar"`},
				},
			},
			expected: "bar",
			found:    true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, found := ETag(test.input)
			if result != test.expected {
				t.Errorf("Unexpected result: %s", result)
			}
			if found != test.found {
				t.Errorf("Unexpected found: %v", found)
			}
		})
	}
}

func TestGetRev(t *testing.T) {
	tests := []struct {
		name          string
		resp          *http.Response
		expected, err string
	}{
		{
			name: "error response",
			resp: &http.Response{
				StatusCode: 400,
				Request:    &http.Request{Method: "POST"},
				Body:       ioutil.NopCloser(strings.NewReader("")),
			},
			err: "Bad Request",
		},
		{
			name: "no ETag header",
			resp: &http.Response{
				StatusCode: 200,
				Request:    &http.Request{Method: "POST"},
				Body:       ioutil.NopCloser(strings.NewReader("")),
			},
			err: "no ETag header found",
		},
		{
			name: "normalized Etag header",
			resp: &http.Response{
				StatusCode: 200,
				Request:    &http.Request{Method: "POST"},
				Header:     http.Header{"Etag": {`"12345"`}},
				Body:       ioutil.NopCloser(strings.NewReader("")),
			},
			expected: `12345`,
		},
		{
			name: "satndard ETag header",
			resp: &http.Response{
				StatusCode: 200,
				Request:    &http.Request{Method: "POST"},
				Header:     http.Header{"ETag": {`"12345"`}},
				Body:       Body(""),
			},
			expected: `12345`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := GetRev(test.resp)
			testy.Error(t, test.err, err)
			if result != test.expected {
				t.Errorf("Got %s, expected %s", result, test.expected)
			}
		})
	}
}

func TestDoJSON(t *testing.T) {
	tests := []struct {
		name         string
		method, path string
		opts         *Options
		client       *Client
		expected     interface{}
		response     *http.Response
		status       int
		err          string
	}{
		{
			name:   "network error",
			method: "GET",
			client: newTestClient(nil, errors.New("net error")),
			status: kivik.StatusNetworkError,
			err:    "Get http://example.com: net error",
		},
		{
			name:   "error response",
			method: "GET",
			client: newTestClient(&http.Response{
				StatusCode: 401,
				Header: http.Header{
					"Content-Type":   {"application/json"},
					"Content-Length": {"67"},
				},
				ContentLength: 67,
				Body:          Body(`{"error":"unauthorized","reason":"Name or password is incorrect."}`),
				Request:       &http.Request{Method: "GET"},
			}, nil),
			status: kivik.StatusUnauthorized,
			err:    "Unauthorized: Name or password is incorrect.",
		},
		{
			name:   "invalid JSON in response",
			method: "GET",
			client: newTestClient(&http.Response{
				StatusCode: 200,
				Header: http.Header{
					"Content-Type":   {"application/json"},
					"Content-Length": {"67"},
				},
				ContentLength: 67,
				Body:          Body(`invalid response`),
				Request:       &http.Request{Method: "GET"},
			}, nil),
			status: kivik.StatusBadResponse,
			err:    "invalid character 'i' looking for beginning of value",
		},
		{
			name:   "success",
			method: "GET",
			client: newTestClient(&http.Response{
				StatusCode: 200,
				Header: http.Header{
					"Content-Type":   {"application/json"},
					"Content-Length": {"15"},
				},
				ContentLength: 15,
				Body:          Body(`{"foo":"bar"}`),
				Request:       &http.Request{Method: "GET"},
			}, nil),
			expected: map[string]interface{}{"foo": "bar"},
			response: &http.Response{
				StatusCode: 200,
				Header: http.Header{
					"Content-Type":   {"application/json"},
					"Content-Length": {"15"},
				},
				ContentLength: 15,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var i interface{}
			response, err := test.client.DoJSON(context.Background(), test.method, test.path, test.opts, &i)
			testy.StatusError(t, test.err, test.status, err)
			if d := diff.Interface(test.expected, i); d != nil {
				t.Errorf("JSON result differs:\n%s\n", d)
			}
			response.Request = nil
			response.Body = nil
			if d := diff.Interface(test.response, response); d != nil {
				t.Errorf("Response differs:\n%s\n", d)
			}
		})
	}
}

func TestNewRequest(t *testing.T) {
	tests := []struct {
		name         string
		method, path string
		body         io.Reader
		expected     *http.Request
		client       *Client
		status       int
		err          string
	}{
		{
			name:   "invalid URL",
			method: "GET",
			path:   "%xx",
			status: kivik.StatusBadRequest,
			err:    `parse %xx: invalid URL escape "%xx"`,
		},
		{
			name:   "invlaid method",
			method: "FOO BAR",
			client: newTestClient(nil, nil),
			status: kivik.StatusBadRequest,
			err:    `net/http: invalid method "FOO BAR"`,
		},
		{
			name:   "success",
			method: "GET",
			path:   "foo",
			client: newTestClient(nil, nil),
			expected: &http.Request{
				Method: "GET",
				URL: func() *url.URL {
					url := newTestClient(nil, nil).dsn
					url.Path = "/foo"
					return url
				}(),
				Proto:      "HTTP/1.1",
				ProtoMajor: 1,
				ProtoMinor: 1,
				Header:     http.Header{},
				Host:       "example.com",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req, err := test.client.NewRequest(context.Background(), test.method, test.path, test.body)
			testy.StatusError(t, test.err, test.status, err)
			test.expected = test.expected.WithContext(req.Context()) // determinism
			if d := diff.Interface(test.expected, req); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestDoReq(t *testing.T) {
	tests := []struct {
		name         string
		method, path string
		opts         *Options
		client       *Client
		status       int
		err          string
	}{
		{
			name:   "no method",
			status: kivik.StatusBadRequest,
			err:    "chttp: method required",
		},
		{
			name:   "invalid url",
			method: "GET",
			path:   "%xx",
			client: newTestClient(nil, nil),
			status: kivik.StatusBadRequest,
			err:    `parse %xx: invalid URL escape "%xx"`,
		},
		{
			name:   "network error",
			method: "GET",
			path:   "foo",
			client: newTestClient(nil, errors.New("net error")),
			status: kivik.StatusNetworkError,
			err:    "Get http://example.com/foo: net error",
		},
		{
			name:   "error response",
			method: "GET",
			path:   "foo",
			client: newTestClient(&http.Response{
				StatusCode: 400,
				Body:       Body(""),
			}, nil),
			// No error here
		},
		{
			name:   "success",
			method: "GET",
			path:   "foo",
			client: newTestClient(&http.Response{
				StatusCode: 200,
				Body:       Body(""),
			}, nil),
			// success!
		},
		{
			name:   "body error",
			method: "PUT",
			path:   "foo",
			client: newTestClient(nil, errors.Status(kivik.StatusBadRequest, "bad request")),
			status: kivik.StatusBadRequest,
			err:    "Put http://example.com/foo: bad request",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := test.client.DoReq(context.Background(), test.method, test.path, test.opts)
			testy.StatusError(t, test.err, test.status, err)
		})
	}
}

func TestDoError(t *testing.T) {
	tests := []struct {
		name         string
		method, path string
		opts         *Options
		client       *Client
		status       int
		err          string
	}{
		{
			name:   "no method",
			status: kivik.StatusBadRequest,
			err:    "chttp: method required",
		},
		{
			name:   "error response",
			method: "GET",
			path:   "foo",
			client: newTestClient(&http.Response{
				StatusCode: kivik.StatusBadRequest,
				Body:       Body(""),
				Request:    &http.Request{Method: "GET"},
			}, nil),
			status: kivik.StatusBadRequest,
			err:    "Bad Request",
		},
		{
			name:   "success",
			method: "GET",
			path:   "foo",
			client: newTestClient(&http.Response{
				StatusCode: kivik.StatusOK,
				Body:       Body(""),
				Request:    &http.Request{Method: "GET"},
			}, nil),
			// No error
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := test.client.DoError(context.Background(), test.method, test.path, test.opts)
			testy.StatusError(t, test.err, test.status, err)
		})
	}
}

func TestNetError(t *testing.T) {
	tests := []struct {
		name  string
		input error

		status int
		err    string
	}{
		{
			name:  "nil",
			input: nil,
			err:   "",
		},
		{
			name: "url error",
			input: &url.Error{
				Op:  "Get",
				URL: "http://foo.com/",
				Err: errors.New("some error"),
			},
			status: kivik.StatusNetworkError,
			err:    "Get http://foo.com/: some error",
		},
		{
			name: "url error with embedded status",
			input: &url.Error{
				Op:  "Get",
				URL: "http://foo.com/",
				Err: errors.Status(kivik.StatusBadRequest, "some error"),
			},
			status: kivik.StatusBadRequest,
			err:    "Get http://foo.com/: some error",
		},
		{
			name:   "other error",
			input:  errors.New("other error"),
			status: kivik.StatusNetworkError,
			err:    "other error",
		},
		{
			name:   "other error with embedded status",
			input:  errors.Status(kivik.StatusBadRequest, "bad req"),
			status: kivik.StatusBadRequest,
			err:    "bad req",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := netError(test.input)
			testy.StatusError(t, test.err, test.status, err)
		})
	}
}

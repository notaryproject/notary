package chttp

import (
	"context"
	"errors"
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
)

func TestBasicAuthRoundTrip(t *testing.T) {
	type rtTest struct {
		name     string
		auth     *BasicAuth
		req      *http.Request
		expected *http.Response
		cleanup  func()
	}
	tests := []rtTest{
		{
			name: "Provided transport",
			req:  httptest.NewRequest("GET", "/", nil),
			auth: &BasicAuth{
				Username: "foo",
				Password: "bar",
				transport: customTransport(func(req *http.Request) (*http.Response, error) {
					u, p, ok := req.BasicAuth()
					if !ok {
						t.Error("BasicAuth not set in request")
					}
					if u != "foo" || p != "bar" {
						t.Errorf("Unexpected user/password: %s/%s", u, p)
					}
					return &http.Response{StatusCode: 200}, nil
				}),
			},
			expected: &http.Response{StatusCode: 200},
		},
		func() rtTest {
			h := func(w http.ResponseWriter, r *http.Request) {
				u, p, ok := r.BasicAuth()
				if !ok {
					t.Error("BasicAuth not set in request")
				}
				if u != "foo" || p != "bar" {
					t.Errorf("Unexpected user/password: %s/%s", u, p)
				}
				w.Header().Set("Date", "Wed, 01 Nov 2017 19:32:41 GMT")
				w.Header().Set("Content-Type", "application/json")
			}
			s := httptest.NewServer(http.HandlerFunc(h))
			return rtTest{
				name: "default transport",
				auth: &BasicAuth{Username: "foo", Password: "bar"},
				req:  httptest.NewRequest("GET", s.URL, nil),
				expected: &http.Response{
					Status:     "200 OK",
					StatusCode: 200,
					Proto:      "HTTP/1.1",
					ProtoMajor: 1,
					ProtoMinor: 1,
					Header: http.Header{
						"Content-Length": {"0"},
						"Content-Type":   {"application/json"},
						"Date":           {"Wed, 01 Nov 2017 19:32:41 GMT"},
					},
				},
				cleanup: func() { s.Close() },
			}
		}(),
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res, err := test.auth.RoundTrip(test.req)
			if err != nil {
				t.Fatal(err)
			}
			res.Body = nil
			res.Request = nil
			if d := diff.Interface(test.expected, res); d != nil {
				t.Error(d)
			}
		})
	}
}

type mockRT struct {
	resp *http.Response
	err  error
}

var _ http.RoundTripper = &mockRT{}

func (rt *mockRT) RoundTrip(_ *http.Request) (*http.Response, error) {
	return rt.resp, rt.err
}

func TestCookieAuthAuthenticate(t *testing.T) {
	tests := []struct {
		name           string
		auth           *CookieAuth
		client         *Client
		status         int
		err            string
		expectedCookie *http.Cookie
	}{
		{
			name: "standard request",
			auth: &CookieAuth{
				Username: "foo",
				Password: "bar",
			},
			client: &Client{
				Client: &http.Client{
					Transport: &mockRT{
						resp: &http.Response{
							Header: http.Header{
								"Set-Cookie": []string{
									"AuthSession=cm9vdDo1MEJCRkYwMjq0LO0ylOIwShrgt8y-UkhI-c6BGw; Version=1; Path=/; HttpOnly",
								},
							},
							Body: ioutil.NopCloser(strings.NewReader(`{"userCtx":{"name":"foo"}}`)),
						},
					},
				},
				dsn: &url.URL{Scheme: "http", Host: "foo.com"},
			},
			expectedCookie: &http.Cookie{
				Name:  kivik.SessionCookieName,
				Value: "cm9vdDo1MEJCRkYwMjq0LO0ylOIwShrgt8y-UkhI-c6BGw",
			},
		},
		{
			name: "Invalid JSON response",
			auth: &CookieAuth{
				Username: "foo",
				Password: "bar",
			},
			client: &Client{
				Client: &http.Client{
					Jar: &cookiejar.Jar{},
					Transport: &mockRT{
						resp: &http.Response{
							Body: ioutil.NopCloser(strings.NewReader(`{"asdf"}`)),
						},
					},
				},
				dsn: &url.URL{Scheme: "http", Host: "foo.com"},
			},
			status: kivik.StatusBadResponse,
			err:    "invalid character '}' after object key",
		},
		{
			name: "names don't match",
			auth: &CookieAuth{
				Username: "foo",
				Password: "bar",
			},
			client: &Client{
				Client: &http.Client{
					Transport: &mockRT{
						resp: &http.Response{
							Header: http.Header{
								"Set-Cookie": []string{
									"AuthSession=cm9vdDo1MEJCRkYwMjq0LO0ylOIwShrgt8y-UkhI-c6BGw; Version=1; Path=/; HttpOnly",
								},
							},
							Body: ioutil.NopCloser(strings.NewReader(`{"userCtx":{"name":"notfoo"}}`)),
						},
					},
				},
				dsn: &url.URL{Scheme: "http", Host: "foo.com"},
			},
			status: kivik.StatusBadResponse,
			err:    "auth response for unexpected user",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.auth.Authenticate(context.Background(), test.client)
			testy.StatusError(t, test.err, test.status, err)
			cookie, ok := test.auth.Cookie()
			if !ok {
				t.Errorf("Expected cookie")
				return
			}
			if d := diff.Interface(test.expectedCookie, cookie); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestBasicAuthAuthenticate(t *testing.T) {
	tests := []struct {
		name   string
		auth   *BasicAuth
		client *Client
		status int
		err    string
	}{
		{
			name:   "network error",
			auth:   &BasicAuth{},
			client: newTestClient(nil, errors.New("net error")),
			status: kivik.StatusInternalServerError,
			err:    "Get http://example.com/_session: net error",
		},
		{
			name: "error response 1.6.1",
			auth: &BasicAuth{Username: "invalid", Password: "invalid"},
			client: newTestClient(&http.Response{
				StatusCode: 401,
				Header: http.Header{
					"Server":         {"CouchDB/1.6.1 (Erlang OTP/17)"},
					"Date":           {"Tue, 31 Oct 2017 11:34:32 GMT"},
					"Content-Type":   {"application/json"},
					"Content-Length": {"67"},
					"Cache-Control":  {"must-revalidate"},
				},
				ContentLength: 67,
				Request:       &http.Request{Method: "GET"},
				Body:          Body(`{"error":"unauthorized","reason":"Name or password is incorrect."}`),
			}, nil),
			status: kivik.StatusUnauthorized,
			err:    "Unauthorized: Name or password is incorrect.",
		},
		{
			name: "invalid JSON response",
			auth: &BasicAuth{Username: "invalid", Password: "invalid"},
			client: newTestClient(&http.Response{
				StatusCode: 200,
				Header: http.Header{
					"Server":         {"CouchDB/1.6.1 (Erlang OTP/17)"},
					"Date":           {"Tue, 31 Oct 2017 11:34:32 GMT"},
					"Content-Type":   {"application/json"},
					"Content-Length": {"13"},
					"Cache-Control":  {"must-revalidate"},
				},
				ContentLength: 13,
				Request:       &http.Request{Method: "GET"},
				Body:          Body(`invalid json`),
			}, nil),
			status: kivik.StatusBadResponse,
			err:    "invalid character 'i' looking for beginning of value",
		},
		{
			name: "wrong user name in response",
			auth: &BasicAuth{Username: "admin", Password: "password"},
			client: newTestClient(&http.Response{
				StatusCode: 200,
				Header: http.Header{
					"Server":         {"CouchDB/1.6.1 (Erlang OTP/17)"},
					"Date":           {"Tue, 31 Oct 2017 11:34:32 GMT"},
					"Content-Type":   {"application/json"},
					"Content-Length": {"177"},
					"Cache-Control":  {"must-revalidate"},
				},
				ContentLength: 177,
				Request:       &http.Request{Method: "GET"},
				Body:          Body(`{"ok":true,"userCtx":{"name":"other","roles":["_admin"]},"info":{"authentication_db":"_users","authentication_handlers":["oauth","cookie","default"],"authenticated":"default"}}`),
			}, nil),
			status: kivik.StatusBadResponse,
			err:    "authentication failed",
		},
		{
			name: "Success 1.6.1",
			auth: &BasicAuth{Username: "admin", Password: "password"},
			client: newTestClient(&http.Response{
				StatusCode: 200,
				Header: http.Header{
					"Server":         {"CouchDB/1.6.1 (Erlang OTP/17)"},
					"Date":           {"Tue, 31 Oct 2017 11:34:32 GMT"},
					"Content-Type":   {"application/json"},
					"Content-Length": {"177"},
					"Cache-Control":  {"must-revalidate"},
				},
				ContentLength: 177,
				Request:       &http.Request{Method: "GET"},
				Body:          Body(`{"ok":true,"userCtx":{"name":"admin","roles":["_admin"]},"info":{"authentication_db":"_users","authentication_handlers":["oauth","cookie","default"],"authenticated":"default"}}`),
			}, nil),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.auth.Authenticate(context.Background(), test.client)
			testy.StatusError(t, test.err, test.status, err)
			if test.client.Client.Transport != test.auth {
				t.Errorf("transport not set as expected")
			}
		})
	}
}

func TestCookie(t *testing.T) {
	tests := []struct {
		name     string
		auth     *CookieAuth
		expected *http.Cookie
		found    bool
	}{
		{
			name:     "No cookie jar",
			auth:     &CookieAuth{},
			expected: nil,
			found:    false,
		},
		{
			name:     "No dsn",
			auth:     &CookieAuth{jar: &cookiejar.Jar{}},
			expected: nil,
			found:    false,
		},
		{
			name:     "no cookies",
			auth:     &CookieAuth{jar: &cookiejar.Jar{}, dsn: &url.URL{}},
			expected: nil,
			found:    false,
		},
		{
			name: "cookie found",
			auth: func() *CookieAuth {
				dsn, err := url.Parse("http://example.com/")
				if err != nil {
					t.Fatal(err)
				}
				jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
				if err != nil {
					t.Fatal(err)
				}
				jar.SetCookies(dsn, []*http.Cookie{
					{Name: kivik.SessionCookieName, Value: "foo"},
					{Name: "other", Value: "bar"},
				})
				return &CookieAuth{
					jar: jar,
					dsn: dsn,
				}
			}(),
			expected: &http.Cookie{Name: kivik.SessionCookieName, Value: "foo"},
			found:    true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, found := test.auth.Cookie()
			if found != test.found {
				t.Errorf("Unexpected found: %T", found)
			}
			if d := diff.Interface(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}

package proxy

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
)

func mustNew(t *testing.T, url string) *Proxy {
	p, err := New(url)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestNew(t *testing.T) {
	type newTest struct {
		Name  string
		URL   string
		Error string
	}
	tests := []newTest{
		{
			Name:  "NoURL",
			Error: "no URL specified",
		},
		{
			Name: "ValidURL",
			URL:  "http://foo.com/",
		},
		{
			Name:  "InvalidURL",
			URL:   "http://foo.com:port with spaces/",
			Error: `parse http://foo.com:port with spaces/: invalid character " " in host name`,
		},
		{
			Name:  "Auth",
			URL:   "http://foo:bar@foo.com/",
			Error: "proxy URL must not contain auth credentials",
		},
		{
			Name:  "Query",
			URL:   "http://foo.com?yes=no",
			Error: "proxy URL must not contain query parameters",
		},
	}
	for _, test := range tests {
		func(test newTest) {
			t.Run(test.Name, func(t *testing.T) {
				t.Parallel()
				_, err := New(test.URL)
				var msg string
				if err != nil {
					msg = err.Error()
				}
				if msg != test.Error {
					t.Errorf("Expected error: %s\n  Actual error: %s", test.Error, msg)
				}
			})
		}(test)
	}
}

func serverDSN(t *testing.T) string {
	for _, env := range []string{"KIVIK_TEST_DSN_COUCH21", "KIVIK_TEST_DSN_COUCH20", "KIVIK_TEST_DSN_COUCH16"} {
		if dsn := os.Getenv(env); dsn != "" {
			parsed, err := url.Parse(dsn)
			if err != nil {
				panic(err)
			}
			parsed.User = nil
			return parsed.String()
		}
	}
	t.Skip("No server specified in environment")
	return ""
}

func TestProxy(t *testing.T) {
	type getTest struct {
		Name        string
		Method      string
		URL         string
		Body        io.Reader
		Status      int
		BodyMatch   string
		HeaderMatch map[string]string
	}
	tests := []getTest{
		{
			Name:   "InvalidMethod",
			Method: "CHICKEN",
			URL:    "http://foo.com/_utils",
			Status: http.StatusMethodNotAllowed,
		},
		{
			Name:        "UtilsRedirect",
			Method:      http.MethodGet,
			URL:         "http://foo.com/_utils",
			Status:      http.StatusMovedPermanently,
			HeaderMatch: map[string]string{"Location": "http://foo.com/_utils/"},
		},
		// {
		// 	Name:        "UtilsRedirectQuery",
		// 	Method:      http.MethodGet,
		// 	URL:         "http://foo.com/_utils?today=tomorrow",
		// 	Status:      http.StatusMovedPermanently,
		// 	HeaderMatch: map[string]string{"Location": "http://foo.com/_utils/?today=tomorrow"},
		// },
		{
			Name:      "Utils",
			Method:    http.MethodGet,
			URL:       "http://foo.com/_utils/",
			Status:    http.StatusOK,
			BodyMatch: "Licensed under the Apache License",
		},
	}
	p, err := New(serverDSN(t))
	if err != nil {
		t.Fatalf("Failed to initialize proxy: %s", err)
	}
	p.StrictMethods = true
	for _, test := range tests {
		func(test getTest) {
			t.Run(test.Name, func(t *testing.T) {
				req := httptest.NewRequest(test.Method, test.URL, test.Body)
				w := httptest.NewRecorder()
				p.ServeHTTP(w, req)
				resp := w.Result()
				if resp.StatusCode != test.Status {
					t.Errorf("Expected status: %d/%s\n  Actual status: %d/%s", test.Status, http.StatusText(test.Status), resp.StatusCode, resp.Status)
				}
				body, _ := ioutil.ReadAll(resp.Body)
				if test.BodyMatch != "" && !bytes.Contains(body, []byte(test.BodyMatch)) {
					t.Errorf("Body does not contain '%s':\n%s", test.BodyMatch, body)
				}
				for header, value := range test.HeaderMatch {
					hv := resp.Header.Get(header)
					if hv != value {
						t.Errorf("Header %s: %s, expected %s", header, hv, value)
					}
				}
			})
		}(test)
	}
}

func TestMethodAllowed(t *testing.T) {
	type maTest struct {
		Method   string
		Strict   bool
		Expected bool
	}
	tests := []maTest{
		{Method: "GET", Strict: true, Expected: true},
		{Method: "GET", Strict: false, Expected: true},
		{Method: "COPY", Strict: true, Expected: true},
		{Method: "COPY", Strict: false, Expected: true},
		{Method: "CHICKEN", Strict: true, Expected: false},
		{Method: "CHICKEN", Strict: false, Expected: true},
	}
	for _, test := range tests {
		func(test maTest) {
			t.Run(fmt.Sprintf("%s:%t", test.Method, test.Strict), func(t *testing.T) {
				t.Parallel()
				p := mustNew(t, "http://foo.com/")
				p.StrictMethods = test.Strict
				result := p.methodAllowed(test.Method)
				if result != test.Expected {
					t.Errorf("Expected: %t, Actual: %t", test.Expected, result)
				}
			})
		}(test)
	}
}

func TestSingleJoiningSlash(t *testing.T) {
	type joinTest struct {
		Name     string
		PathA    string
		PathB    string
		Expected string
	}
	tests := []joinTest{
		{
			Name:     "Empty",
			PathA:    "",
			PathB:    "",
			Expected: "/",
		},
		{
			Name:     "NoSlashes",
			PathA:    "foo",
			PathB:    "bar",
			Expected: "foo/bar",
		},
		{
			Name:     "Trailing",
			PathA:    "foo/",
			PathB:    "bar",
			Expected: "foo/bar",
		},
		{
			Name:     "Leading",
			PathA:    "foo",
			PathB:    "/bar",
			Expected: "foo/bar",
		},
		{
			Name:     "Both",
			PathA:    "foo/",
			PathB:    "/bar",
			Expected: "foo/bar",
		},
	}
	for _, test := range tests {
		func(test joinTest) {
			t.Run(test.Name, func(t *testing.T) {
				result := singleJoiningSlash(test.PathA, test.PathB)
				if result != test.Expected {
					t.Errorf("Expected: %s, Actual: %s", test.Expected, result)
				}
			})
		}(test)
	}
}

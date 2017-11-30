package chttp

import (
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/flimzy/diff"
	"github.com/flimzy/kivik"
)

func TestResponseError(t *testing.T) {
	tests := []struct {
		name     string
		resp     *http.Response
		expected interface{}
	}{
		{
			name:     "non error",
			resp:     &http.Response{StatusCode: 200},
			expected: nil,
		},
		{
			name: "HEAD error",
			resp: &http.Response{
				StatusCode: 404,
				Request:    &http.Request{Method: "HEAD"},
				Body:       Body(""),
			},
			expected: &HTTPError{Code: 404},
		},
		{
			name: "2.0.0 error",
			resp: &http.Response{
				StatusCode: 400,
				Header: http.Header{
					"Cache-Control":       {"must-revalidate"},
					"Content-Length":      {"194"},
					"Content-Type":        {"application/json"},
					"Date":                {"Fri, 27 Oct 2017 15:34:07 GMT"},
					"Server":              {"CouchDB/2.0.0 (Erlang OTP/17)"},
					"X-Couch-Request-ID":  {"92d05bd015"},
					"X-CouchDB-Body-Time": {"0"},
				},
				ContentLength: 194,
				Body:          Body(`{"error":"illegal_database_name","reason":"Name: '_foo'. Only lowercase characters (a-z), digits (0-9), and any of the characters _, $, (, ), +, -, and / are allowed. Must begin with a letter."}`),
				Request:       &http.Request{Method: "PUT"},
			},
			expected: &HTTPError{
				Code:   400,
				Reason: "Name: '_foo'. Only lowercase characters (a-z), digits (0-9), and any of the characters _, $, (, ), +, -, and / are allowed. Must begin with a letter.",
			},
		},
		{
			name: "invalid json error",
			resp: &http.Response{
				StatusCode: 400,
				Header: http.Header{
					"Server":         {"CouchDB/1.6.1 (Erlang OTP/17)"},
					"Date":           {"Fri, 27 Oct 2017 15:42:34 GMT"},
					"Content-Type":   {"application/json"},
					"Content-Length": {"194"},
					"Cache-Control":  {"must-revalidate"},
				},
				ContentLength: 194,
				Body:          Body("invalid json"),
				Request:       &http.Request{Method: "PUT"},
			},
			expected: &HTTPError{Code: 400},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ResponseError(test.resp)
			if d := diff.Interface(test.expected, err); d != nil {
				t.Error(d)
			}
		})
	}
}

func xTestErrors(t *testing.T) {
	type errTest struct {
		Name           string
		Func           func() error
		ExpectedStatus int
		ExpectedMsg    string
	}
	tests := []errTest{
		{
			Name: "200Response",
			Func: func() error { return ResponseError(&http.Response{StatusCode: 200}) },
		},
		{
			Name: "HEADError",
			Func: func() error {
				return ResponseError(&http.Response{
					StatusCode: 400,
					Request: &http.Request{
						Method: "HEAD",
					},
					Body: ioutil.NopCloser(strings.NewReader("")),
				})
			},
			ExpectedStatus: 400,
			ExpectedMsg:    "Bad Request",
		},
		{
			Name: "WithReason",
			Func: func() error {
				return ResponseError(&http.Response{
					StatusCode: 404,
					Request: &http.Request{
						Method: "GET",
					},
					Header:        map[string][]string{"Content-Type": {"application/json"}},
					ContentLength: 1, // Just non-zero for this test
					Body:          ioutil.NopCloser(strings.NewReader(`{"code":404,"reason":"db_not_found"}`)),
				})
			},
			ExpectedStatus: 404,
			ExpectedMsg:    "Not Found: db_not_found",
		},
		{
			Name: "WithoutReason",
			Func: func() error {
				return ResponseError(&http.Response{
					StatusCode: 404,
					Request: &http.Request{
						Method: "GET",
					},
					Header:        map[string][]string{"Content-Type": {"application/json"}},
					ContentLength: 1, // Just non-zero for this test
					Body:          ioutil.NopCloser(strings.NewReader(`{"code":404}`)),
				})
			},
			ExpectedStatus: 404,
			ExpectedMsg:    "Not Found",
		},
		{
			Name: "BadJSON",
			Func: func() error {
				return ResponseError(&http.Response{
					StatusCode: 404,
					Request: &http.Request{
						Method: "GET",
					},
					Header:        map[string][]string{"Content-Type": {"application/json"}},
					ContentLength: 1, // Just non-zero for this test
					Body:          ioutil.NopCloser(strings.NewReader(`asdf`)),
				})
			},
			ExpectedStatus: 404,
			ExpectedMsg:    "Not Found: unknown (failed to decode error response: invalid character 'a' looking for beginning of value)",
		},
	}
	for _, test := range tests {
		func(test errTest) {
			t.Run(test.Name, func(t *testing.T) {
				err := test.Func()
				if err == nil {
					if test.ExpectedStatus == 0 {
						return
					}
					t.Errorf("Got an error when none expected: %s", err)
				}
				if status := kivik.StatusCode(err); status != test.ExpectedStatus {
					t.Errorf("Status. Expected %d, Actual %d", test.ExpectedStatus, status)
				}
				if msg := err.Error(); msg != test.ExpectedMsg {
					t.Errorf("Error. Expected '%s', Actual '%s'", test.ExpectedMsg, msg)
				}
			})
		}(test)
	}
}

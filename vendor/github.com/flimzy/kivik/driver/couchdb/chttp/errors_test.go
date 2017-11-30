package chttp

import (
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/flimzy/kivik"
)

func TestErrors(t *testing.T) {
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

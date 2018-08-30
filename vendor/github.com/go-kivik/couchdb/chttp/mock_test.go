package chttp

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

type customTransport func(*http.Request) (*http.Response, error)

var _ http.RoundTripper = customTransport(nil)

func (c customTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return c(req)
}

func newCustomClient(fn func(*http.Request) (*http.Response, error)) *Client {
	dsn, err := url.Parse("http://example.com/")
	if err != nil {
		panic(err)
	}
	return &Client{
		dsn: dsn,
		Client: &http.Client{
			Transport: customTransport(fn),
		},
	}
}

func newTestClient(resp *http.Response, err error) *Client {
	return newCustomClient(func(_ *http.Request) (*http.Response, error) {
		return resp, err
	})
}

func Body(str string) io.ReadCloser {
	return ioutil.NopCloser(strings.NewReader(str))
}

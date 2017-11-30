// Package proxy provides a simple proxy for a CouchDB server
package proxy

import (
	"errors"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// Proxy is an http.Handler which proxies connections to a CouchDB server.
type Proxy struct {
	*httputil.ReverseProxy
	// StrictMethods will reject any non-standard CouchDB methods immediately,
	// rather than relaying to the CouchDB server.
	StrictMethods bool
}

var _ http.Handler = &Proxy{}

// New returns a new Proxy instance, which redirects all requests to the
// specified URL.
func New(serverURL string) (*Proxy, error) {
	if serverURL == "" {
		return nil, errors.New("no URL specified")
	}
	target, err := url.Parse(serverURL)
	if err != nil {
		return nil, err
	}
	if target.User != nil {
		return nil, errors.New("proxy URL must not contain auth credentials")
	}
	if target.RawQuery != "" {
		return nil, errors.New("proxy URL must not contain query parameters")
	}
	director := func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = singleJoiningSlash(target.Path, req.URL.Path)
		if _, ok := req.Header["User-Agent"]; !ok {
			// explicitly disable User-Agent so it's not set to default value
			req.Header.Set("User-Agent", "")
		}
	}
	return &Proxy{
		ReverseProxy: &httputil.ReverseProxy{
			Director: director,
		},
	}, nil
}

// Any other methods are rejected immediately, if StrictMethods is true.
var supportedMethods = []string{"GET", "HEAD", "POST", "PUT", "DELETE", "COPY"}

func (p *Proxy) methodAllowed(method string) bool {
	if !p.StrictMethods {
		return true
	}
	for _, m := range supportedMethods {
		if m == method {
			return true
		}
	}
	return false
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !p.methodAllowed(r.Method) {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	p.ReverseProxy.ServeHTTP(w, r)
}

// singleJoiningSlash is copied from net/http/httputil/reverseproxy.go in the
// standard library.
func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}

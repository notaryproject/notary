package couchdb

import (
	"context"
	"net/http"

	"github.com/go-kivik/couchdb/chttp"
	"github.com/go-kivik/kivik"
	"github.com/go-kivik/kivik/errors"
)

func (c *client) Authenticate(ctx context.Context, a interface{}) error {
	if auth, ok := a.(chttp.Authenticator); ok {
		return auth.Authenticate(ctx, c.Client)
	}
	if auth, ok := a.(Authenticator); ok {
		return auth.auth(ctx, c)
	}
	return errors.Status(kivik.StatusUnknownError, "kivik: invalid authenticator")
}

// Authenticator is a CouchDB authenticator.
type Authenticator interface {
	auth(context.Context, *client) error
}

type xportAuth struct {
	http.RoundTripper
}

var _ Authenticator = &xportAuth{}

func (a *xportAuth) auth(_ context.Context, c *client) error {
	if c.Client.Client.Transport != nil {
		return errors.New("kivik: HTTP client transport already set")
	}
	c.Client.Client.Transport = a.RoundTripper
	return nil
}

// SetTransport returns an authenticator that can be used to set a client
// connection's HTTP Transport. This can be used to control proxies, TLS
// configuration, keep-alives, compression, etc.
//
// Example:
//
//     setXport := couchdb.SetTransport(&http.Transport{
//         // .. custom config
//     })
//     client, _ := kivik.New( ... )
//     client.Authenticate(setXport)
func SetTransport(t http.RoundTripper) Authenticator {
	return &xportAuth{t}
}

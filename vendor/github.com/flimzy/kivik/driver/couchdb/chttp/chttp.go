// Package chttp provides a minimal HTTP driver backend for communicating with
// CouchDB servers.
package chttp

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"
)

const (
	typeJSON = "application/json"
)

// Client represents a client connection. It embeds an *http.Client
type Client struct {
	*http.Client

	rawDSN string
	dsn    *url.URL
	auth   Authenticator
}

// New returns a connection to a remote CouchDB server. If credentials are
// included in the URL, CookieAuth is attempted first, with BasicAuth used as
// a fallback. If both fail, an error is returned. If you wish to use some other
// authentication mechanism, do not specify credentials in the URL, and instead
// call the Auth() method later.
func New(ctx context.Context, dsn string) (*Client, error) {
	dsnURL, err := url.Parse(dsn)
	if err != nil {
		return nil, err
	}
	user := dsnURL.User
	dsnURL.User = nil
	c := &Client{
		Client: &http.Client{},
		dsn:    dsnURL,
		rawDSN: dsn,
	}
	if user != nil {
		password, _ := user.Password()
		if err := c.defaultAuth(ctx, user.Username(), password); err != nil {
			return nil, err
		}
	}
	return c, nil
}

// DSN returns the unparsed DSN used to connect.
func (c *Client) DSN() string {
	return c.rawDSN
}

func (c *Client) defaultAuth(ctx context.Context, username, password string) error {
	err := c.Auth(ctx, &CookieAuth{
		Username: username,
		Password: password,
	})
	if err == nil {
		return nil
	}
	return c.Auth(ctx, &BasicAuth{
		Username: username,
		Password: password,
	})
}

// Auth authenticates using the provided Authenticator.
func (c *Client) Auth(ctx context.Context, a Authenticator) error {
	if c.auth != nil {
		return errors.New("auth already set; log out first")
	}
	if err := a.Authenticate(ctx, c); err != nil {
		return err
	}
	c.auth = a
	return nil
}

// Logout logs out after authentication.
func (c *Client) Logout(ctx context.Context) error {
	if c.auth == nil {
		return errors.New("not authenticated")
	}
	err := c.auth.Logout(ctx, c)
	c.auth = nil
	return err
}

// Options are optional parameters which may be sent with a request.
type Options struct {
	// Accept sets the request's Accept header. Defaults to "application/json".
	// To specify any, use "*/*".
	Accept string
	// ContentType sets the requests's Content-Type header. Defaults to "application/json".
	ContentType string
	// Body sets the body of the request.
	Body io.Reader
	// JSON is an arbitrary data type which is marshaled to the request's body.
	// It an error to set both Body and JSON on the same request. When this is
	// set, ContentType is unconditionally set to 'application/json'. Note that
	// for large JSON payloads, it can be beneficial to do your own JSON stream
	// encoding, so that the request can be live on the wire during JSON
	// encoding.
	JSON interface{}
	// ForceCommit adds the X-Couch-Full-Commit: true header to requests
	ForceCommit bool
	// Destination is the target ID for COPY
	Destination string
}

// Response represents a response from a CouchDB server.
type Response struct {
	*http.Response

	// ContentType is the base content type, parsed from the response headers.
	ContentType string
}

// DecodeJSON unmarshals the response body into i. This method consumes and
// closes the response body.
func DecodeJSON(r *http.Response, i interface{}) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(i)
}

// DoJSON combines DoReq() and, ResponseError(), and (*Response).DecodeJSON(), and
// closes the response body.
func (c *Client) DoJSON(ctx context.Context, method, path string, opts *Options, i interface{}) (*http.Response, error) {
	res, err := c.DoReq(ctx, method, path, opts)
	if err != nil {
		return res, err
	}
	if err = ResponseError(res); err != nil {
		return res, err
	}
	err = DecodeJSON(res, i)
	return res, err
}

// NewRequest returns a new *http.Request to the CouchDB server, and the
// specified path. The host, schema, etc, of the specified path are ignored.
func (c *Client) NewRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	reqPath, err := url.Parse(path)
	if err != nil {
		return nil, err
	}
	url := *c.dsn // Make a copy
	url.Path = reqPath.Path
	url.RawQuery = reqPath.RawQuery
	req, err := http.NewRequest(method, url.String(), body)
	if err != nil {
		return nil, err
	}
	return req.WithContext(ctx), nil
}

// DoReq does an HTTP request. An error is returned only if there was an error
// processing the request. In particular, an error status code, such as 400
// or 500, does _not_ cause an error to be returned.
func (c *Client) DoReq(ctx context.Context, method, path string, opts *Options) (*http.Response, error) {
	var body io.Reader
	if opts != nil {
		if opts.Body != nil {
			body = opts.Body
		}
	}
	req, err := c.NewRequest(ctx, method, path, body)
	if err != nil {
		return nil, err
	}
	fixPath(req, path)
	setHeaders(req, opts)

	return c.Do(req)
}

// fixPath sets the request's URL.RawPath to work with escaped characters in
// paths.
func fixPath(req *http.Request, path string) {
	// Remove any query parameters
	parts := strings.SplitN(path, "?", 2)
	req.URL.RawPath = "/" + strings.TrimPrefix(parts[0], "/")
}

// EncodeBody JSON encodes i to r. A call to errFunc will block until encoding
// has completed, then return the errur status of the encoding job. If an
// encoding error occurs, cancel() called.
func EncodeBody(i interface{}, cancel context.CancelFunc) (r io.Reader, errFunc func() error) {
	r, w := io.Pipe()
	errChan := make(chan error, 1)
	go func() {
		if err := json.NewEncoder(w).Encode(i); err != nil {
			cancel()
			errChan <- err
		}
		close(errChan)
		w.Close()
	}()
	return r, func() error {
		return <-errChan
	}
}

func setHeaders(req *http.Request, opts *Options) {
	accept := typeJSON
	contentType := typeJSON
	if opts != nil {
		if opts.Accept != "" {
			accept = opts.Accept
		}
		if opts.ContentType != "" {
			contentType = opts.ContentType
		}
		if opts.ForceCommit {
			req.Header.Add("X-Couch-Full-Commit", "true")
		}
		if opts.Destination != "" {
			req.Header.Add("Destination", opts.Destination)
		}
	}
	req.Header.Add("Accept", accept)
	req.Header.Add("Content-Type", contentType)
}

// DoError is the same as DoReq(), followed by checking the response error. This
// method is meant for cases where the only information you need from the
// response is the status code. It unconditionally closes the response body.
func (c *Client) DoError(ctx context.Context, method, path string, opts *Options) (*http.Response, error) {
	res, err := c.DoReq(ctx, method, path, opts)
	if err != nil {
		return res, err
	}
	defer func() { _ = res.Body.Close() }()
	err = ResponseError(res)
	return res, err
}

// GetRev extracts the revision from the response's Etag header
func GetRev(resp *http.Response) (rev string, err error) {
	if err = ResponseError(resp); err != nil {
		return "", err
	}
	if _, ok := resp.Header["Etag"]; !ok {
		return "", errors.New("no Etag header found")
	}
	rev = resp.Header.Get("Etag")
	// trim quote marks (")
	return rev[1 : len(rev)-1], nil
}

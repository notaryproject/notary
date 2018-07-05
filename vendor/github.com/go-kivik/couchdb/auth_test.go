package couchdb

import (
	"context"
	"fmt"
	"testing"

	"github.com/flimzy/testy"

	"github.com/go-kivik/couchdb/chttp"
	"github.com/go-kivik/kivik"
	"github.com/go-kivik/kivik/errors"
)

type mockAuther struct {
	authCalls int
	authErr   error
}

var _ chttp.Authenticator = &mockAuther{}

func (a *mockAuther) Authenticate(ctx context.Context, c *chttp.Client) error {
	a.authCalls++
	return a.authErr
}

func (a *mockAuther) Logout(ctx context.Context, c *chttp.Client) error {
	return nil
}

func (a *mockAuther) Check() error {
	if a.authCalls == 1 {
		return nil
	}
	return fmt.Errorf("auth called %d times", a.authCalls)
}

type checker interface {
	Check() error
}

func TestAuthenticate(t *testing.T) {
	tests := []struct {
		name          string
		client        *client
		authenticator interface{}
		status        int
		err           string
	}{
		{
			name:          "invalid authenticator",
			authenticator: 1,
			status:        kivik.StatusUnknownError,
			err:           "kivik: invalid authenticator",
		},
		{
			name:          "valid authenticator",
			client:        &client{Client: &chttp.Client{}},
			authenticator: &mockAuther{},
		},
		{
			name:          "auth failure",
			client:        &client{Client: &chttp.Client{}},
			authenticator: &mockAuther{authErr: errors.Status(kivik.StatusUnauthorized, "auth failed")},
			status:        kivik.StatusUnauthorized,
			err:           "auth failed",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.client.Authenticate(context.Background(), test.authenticator)
			testy.StatusError(t, test.err, test.status, err)
			if c, ok := test.authenticator.(checker); ok {
				if e := c.Check(); e != nil {
					t.Error(e)
				}
			}
		})
	}
}

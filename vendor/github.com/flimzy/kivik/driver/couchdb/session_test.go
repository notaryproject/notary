// +build !js

// GopherJS can't run a test server

package couchdb

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/flimzy/diff"
	"github.com/flimzy/kivik"
)

func TestSession(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		body      string
		expected  interface{}
		errStatus int
	}{
		{
			name:   "valid",
			status: http.StatusOK,
			body:   `{"ok":true,"userCtx":{"name":"admin","roles":["_admin"]},"info":{"authentication_db":"_users","authentication_handlers":["oauth","cookie","default"],"authenticated":"cookie"}}`,
			expected: &kivik.Session{
				Name:                   "admin",
				Roles:                  []string{"_admin"},
				AuthenticationMethod:   "cookie",
				AuthenticationHandlers: []string{"oauth", "cookie", "default"},
				RawResponse:            []byte(`{"ok":true,"userCtx":{"name":"admin","roles":["_admin"]},"info":{"authentication_db":"_users","authentication_handlers":["oauth","cookie","default"],"authenticated":"cookie"}}`),
			},
		},
		{
			name:      "invalid response",
			body:      `{"userCtx":"asdf"}`,
			errStatus: kivik.StatusInternalServerError,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(test.status)
				_, _ = w.Write([]byte(test.body))
			}))
			client, err := kivik.New(context.Background(), "couch", s.URL)
			session, err := client.Session(context.Background())
			if status := kivik.StatusCode(err); status != test.errStatus {
				t.Errorf("Unexpected error: %s", err)
			}
			if err != nil {
				return
			}
			if d := diff.Interface(test.expected, session); d != nil {
				t.Error(d)
			}
		})
	}
}

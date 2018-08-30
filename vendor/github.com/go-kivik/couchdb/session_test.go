// +build !js

// GopherJS can't run a test server

package couchdb

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/flimzy/diff"
	"github.com/flimzy/testy"

	"github.com/go-kivik/kivik"
)

func TestSession(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		body      string
		expected  interface{}
		errStatus int
		err       string
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
			status:    http.StatusOK,
			body:      `{"userCtx":"asdf"}`,
			errStatus: kivik.StatusBadResponse,
			err:       "json: cannot unmarshal string into Go ",
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
			if err != nil {
				t.Fatal(err)
			}
			session, err := client.Session(context.Background())
			testy.StatusErrorRE(t, test.err, test.errStatus, err)
			if d := diff.Interface(test.expected, session); d != nil {
				t.Error(d)
			}
		})
	}
}

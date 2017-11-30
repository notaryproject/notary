package auth

import (
	"encoding/json"
	"net/http"

	"github.com/flimzy/kivik/authdb"
)

// Handler is an auth handler.
type Handler interface {
	// MethodName identifies the handler. It is only called once on server
	// start up.
	MethodName() string
	// Authenticate authenticates the HTTP request. On success, a user context
	// must be returned. Any error will immediately terminate the authentication
	// process, returning an error to the client. In particular, this means that
	// an "unauthorized" error must not be returned if fallthrough is intended.
	// If a response is sent, execution does not continue. This allows handlers
	// to expose their own API endpoints (for example, the default cookie auth
	// handler adds POST /_session and DELETE /_session handlers).
	Authenticate(http.ResponseWriter, *http.Request) (*authdb.UserContext, error)
}

// Session represents an authenticated session.
type Session struct {
	AuthMethod string
	AuthDB     string
	Handlers   []string
	User       *authdb.UserContext
}

// MarshalJSON satisfies the json.Marshaler interface.
func (s *Session) MarshalJSON() ([]byte, error) {
	user := s.User
	if user == nil {
		user = &authdb.UserContext{}
	}
	result := map[string]interface{}{
		"info": map[string]interface{}{
			"authenticated":           s.AuthMethod,
			"authentication_db":       s.AuthDB,
			"authentication_handlers": s.Handlers,
		},
		"ok":      true,
		"userCtx": user,
	}
	return json.Marshal(result)
}

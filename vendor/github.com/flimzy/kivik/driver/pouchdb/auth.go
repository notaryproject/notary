package pouchdb

import "errors"

// Authenticator is an authentication interface, which may be implemented by
// any PouchDB-centric authentication type.
type authenticator interface {
	authenticate(*client) error
}

func (c *client) Authenticate(a interface{}) error {
	if auth, ok := a.(authenticator); ok {
		return auth.authenticate(c)
	}
	return errors.New("invalid authenticator")
}

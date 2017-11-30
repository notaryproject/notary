package pouchdb

// BasicAuth handles HTTP Basic Auth for remote PouchDB connections. This
// is the only auth support built directly into PouchDB, so this is a very
// thin wrapper.
type BasicAuth struct {
	Name     string
	Password string
}

// authenticate sets the HTTP Basic Auth parameters.
func (a *BasicAuth) authenticate(c *client) error {
	c.opts["authenticator"] = Options{
		"auth": map[string]interface{}{
			"username": a.Name,
			"password": a.Password,
		},
	}
	return nil
}

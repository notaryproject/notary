// Package kivik provides a generic interface to CouchDB or CouchDB-like databases.
//
// The kivik package must be used in conjunction with a database driver. See
// https://github.com/flimzy/kivik/wiki/Kivik-database-drivers for a list.
//
// The kivik driver system is modeled after the standard library's sql and
// sql/driver packages, although the client API is completely different due to
// the different  database models implemented by SQL and NoSQL databases such as
// CouchDB.
//
// Contexts in Kivik
//
// Most functions that may block require a Context as their first argument. The
// context permits cancelling a blocked function in case of a timeout, or some
// other event (such as a cancelled HTTP request if a web server). Be sure to
// read the context package GoDocs at https://golang.org/pkg/context/ and see
// https://blog.golang.org/context for example code for a server that uses
// Contexts.
//
// If in doubt, you can pass context.TODO() as the context variable. Think of
// the TODO context as a place-holder for cases when it is unclear which Context
// to use, or when surrounding functions have not yet been extended to support
// a Context parameter.
//
// For example:
//  client, err := kivik.New(context.TODO(), "couch", "http://localhost:5984/")
package kivik // import "github.com/flimzy/kivik"

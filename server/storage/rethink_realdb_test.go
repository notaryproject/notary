// +build rethinkdb

// Uses a real RethinkDB connection testing purposes

package storage

import (
	"os"
	"testing"

	"github.com/docker/go-connections/tlsconfig"
	"github.com/docker/notary/storage/rethinkdb"
	"github.com/docker/notary/tuf/data"
	"github.com/stretchr/testify/require"
	"gopkg.in/dancannon/gorethink.v2"
)

var tlsOpts = tlsconfig.Options{InsecureSkipVerify: true}

func rethinkSessionSetup(t *testing.T) (*gorethink.Session, string) {
	// Get the MYSQL connection string from an environment variable
	rethinkSource := os.Getenv("RETHINK")
	require.NotEqual(t, "", rethinkSource)

	sess, err := rethinkdb.AdminConnection(tlsOpts, rethinkSource)
	require.NoError(t, err)

	return sess, rethinkSource
}

func TestBootstrapSetsUsernamePassword(t *testing.T) {
	adminSession, source := rethinkSessionSetup(t)
	dbname, username, password := "testdb", "testuser", "testpassword"
	otherDB, otherUser, otherPass := "otherdb", "otheruser", "otherpassword"

	// create a separate user with access to a different DB
	require.NoError(t, rethinkdb.SetupDB(adminSession, otherDB, nil))
	require.NoError(t, rethinkdb.CreateAndGrantDBUser(adminSession, otherDB, otherUser, otherPass))

	// Bootstrap
	s := NewRethinkDBStorage(dbname, username, password, adminSession)
	require.NoError(t, s.Bootstrap())

	// A user with an invalid password cannot connect to rethink DB at all
	_, err := rethinkdb.UserConnection(tlsOpts, source, username, "wrongpass")
	require.Error(t, err)

	// the other user cannot access rethink
	userSession, err := rethinkdb.UserConnection(tlsOpts, source, otherUser, otherPass)
	s = NewRethinkDBStorage(dbname, otherUser, otherPass, userSession)
	_, _, err = s.GetCurrent("gun", data.CanonicalRootRole)
	require.Error(t, err)
	require.IsType(t, gorethink.RQLRuntimeError{}, err)

	// our user can access the DB though
	userSession, err = rethinkdb.UserConnection(tlsOpts, source, username, password)
	s = NewRethinkDBStorage(dbname, username, password, userSession)
	_, _, err = s.GetCurrent("gun", data.CanonicalRootRole)
	require.Error(t, err)
	require.IsType(t, ErrNotFound{}, err)
}

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
	// Get the Rethink connection string from an environment variable
	rethinkSource := os.Getenv("DBURL")
	require.NotEqual(t, "", rethinkSource)

	sess, err := rethinkdb.AdminConnection(tlsOpts, rethinkSource)
	require.NoError(t, err)

	return sess, rethinkSource
}

func rethinkDBSetup(t *testing.T) (RethinkDB, func()) {
	session, _ := rethinkSessionSetup(t)
	dbName := "servertestdb"
	var cleanup = func() { gorethink.DBDrop(dbName).Exec(session) }

	cleanup()
	require.NoError(t, rethinkdb.SetupDB(session, dbName, []rethinkdb.Table{
		TUFFilesRethinkTable,
		PubKeysRethinkTable,
	}))
	return NewRethinkDBStorage(dbName, "", "", session), cleanup
}

func TestRethinkBootstrapSetsUsernamePassword(t *testing.T) {
	adminSession, source := rethinkSessionSetup(t)
	dbname, username, password := "servertestdb", "testuser", "testpassword"
	otherDB, otherUser, otherPass := "otherservertestdb", "otheruser", "otherpassword"

	// create a separate user with access to a different DB
	require.NoError(t, rethinkdb.SetupDB(adminSession, otherDB, nil))
	defer gorethink.DBDrop(otherDB).Exec(adminSession)
	require.NoError(t, rethinkdb.CreateAndGrantDBUser(adminSession, otherDB, otherUser, otherPass))

	// Bootstrap
	s := NewRethinkDBStorage(dbname, username, password, adminSession)
	require.NoError(t, s.Bootstrap())
	defer gorethink.DBDrop(dbname).Exec(adminSession)

	// A user with an invalid password cannot connect to rethink DB at all
	_, err := rethinkdb.UserConnection(tlsOpts, source, username, "wrongpass")
	require.Error(t, err)

	// the other user cannot access rethink
	userSession, err := rethinkdb.UserConnection(tlsOpts, source, otherUser, otherPass)
	require.NoError(t, err)
	s = NewRethinkDBStorage(dbname, otherUser, otherPass, userSession)
	_, _, err = s.GetCurrent("gun", data.CanonicalRootRole)
	require.Error(t, err)
	require.IsType(t, gorethink.RQLRuntimeError{}, err)

	// our user can access the DB though
	userSession, err = rethinkdb.UserConnection(tlsOpts, source, username, password)
	require.NoError(t, err)
	s = NewRethinkDBStorage(dbname, username, password, userSession)
	_, _, err = s.GetCurrent("gun", data.CanonicalRootRole)
	require.Error(t, err)
	require.IsType(t, ErrNotFound{}, err)
}

// UpdateCurrent will add a new TUF file if no previous version of that gun and role existed.
func TestRethinkUpdateCurrentEmpty(t *testing.T) {
	dbStore, cleanup := rethinkDBSetup(t)
	defer cleanup()

	testUpdateCurrentEmptyStore(t, dbStore)
}

// UpdateCurrent will add a new TUF file if the version is higher than previous, but fail
// if the version is equal to or less than the current, whether or not that previous
// version exists
func TestRethinkUpdateCurrentVersionCheck(t *testing.T) {
	t.Skip("Currently rethink only errors if the previous version exists - it doesn't check for strictly increasing")
	dbStore, cleanup := rethinkDBSetup(t)
	defer cleanup()

	testUpdateCurrentVersionCheck(t, dbStore)
}

// UpdateMany succeeds if the updates do not conflict with each other or with what's
// already in the DB
func TestRethinkUpdateManyNoConflicts(t *testing.T) {
	dbStore, cleanup := rethinkDBSetup(t)
	defer cleanup()

	testUpdateManyNoConflicts(t, dbStore)
}

// UpdateMany does not insert any rows (or at least rolls them back) if there
// are any conflicts.
func TestRethinkUpdateManyConflictRollback(t *testing.T) {
	dbStore, cleanup := rethinkDBSetup(t)
	defer cleanup()

	testUpdateManyConflictRollback(t, dbStore)
}

// Delete will remove all TUF metadata, all versions, associated with a gun
func TestRethinkDeleteSuccess(t *testing.T) {
	dbStore, cleanup := rethinkDBSetup(t)
	defer cleanup()

	testDeleteSuccess(t, dbStore)
}

func TestRethinkTUFMetaStoreGetCurrent(t *testing.T) {
	dbStore, cleanup := rethinkDBSetup(t)
	defer cleanup()

	testTUFMetaStoreGetCurrent(t, dbStore)
}

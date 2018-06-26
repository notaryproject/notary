// +build couchdb

// Uses a real CouchDB connection testing purposes

package storage

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/docker/go-connections/tlsconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/theupdateframework/notary/storage/couchdb"
	"github.com/theupdateframework/notary/tuf/data"

	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/driver/couchdb/chttp"
	_ "github.com/go-kivik/couchdb"
)

var tlsOpts = tlsconfig.Options{InsecureSkipVerify: true, ExclusiveRootPools: true}

func couchSessionSetup(t *testing.T) (*kivik.Client, string) {
	// Get the CouchDB connection string from an environment variable
	couchSource := os.Getenv("DBURL")
	require.NotEqual(t, "", couchSource)

	sess, err := couchdb.AdminConnection(tlsOpts, couchSource)
	require.NoError(t, err)

	return sess, couchSource
}

func couchDBSetup(t *testing.T) (CouchDB, func()) {
	session, _ := couchSessionSetup(t)
	dbName := "servertestdb"
	var cleanup = func() {
		couchdb.DBDrop(session, dbName, TUFFilesCouchTable.Name)
		couchdb.DBDrop(session, dbName, ChangeCouchTable.Name)
	}

	cleanup()
	require.NoError(t, couchdb.SetupDB(session, dbName, []couchdb.Table{
		TUFFilesCouchTable,
		ChangeCouchTable,
	}))
	return NewCouchDBStorage(dbName, "", "", session), cleanup
}

func TestCouchBootstrapSetsUsernamePassword(t *testing.T) {
	adminSession, source := couchSessionSetup(t)
	dbname, username, password := "servertestdb", "testuser", "testpassword"
	otherDB, otherUser, otherPass := "otherservertestdb", "otheruser", "otherpassword"

	// create a separate user with access to a different DB
	require.NoError(t, couchdb.SetupDB(adminSession, otherDB, []couchdb.Table{
		{Name: "othertable"},
	}))
	defer couchdb.DBDrop(adminSession, otherDB, "othertable")
	require.NoError(t, couchdb.CreateAndGrantDBUser(adminSession, otherDB, otherUser, otherPass))

	// Bootstrap
	s := NewCouchDBStorage(dbname, username, password, adminSession)
	require.NoError(t, s.Bootstrap())
	// Bootstrap creates two databases we need to clean up
	defer couchdb.DBDrop(adminSession, dbname, TUFFilesCouchTable.Name)
	defer couchdb.DBDrop(adminSession, dbname, ChangeCouchTable.Name)

	// A user with an invalid password cannot connect to couch DB at all
	_, err := couchdb.UserConnection(tlsOpts, source, username, "wrongpass")
	require.Error(t, err)

	// the other user cannot access couch, causing health checks to fail
	userSession, err := couchdb.UserConnection(tlsOpts, source, otherUser, otherPass)
	require.NoError(t, err)
	s = NewCouchDBStorage(dbname, otherUser, otherPass, userSession)
	_, _, err = s.GetCurrent("gun", data.CanonicalRootRole)
	require.Error(t, err)

	// our user can access the DB though
	userSession, err = couchdb.UserConnection(tlsOpts, source, username, password)
	require.NoError(t, err)
	s = NewCouchDBStorage(dbname, username, password, userSession)
	_, _, err = s.GetCurrent("gun", data.CanonicalRootRole)
	require.Error(t, err)

	// CouchDB returns ErrNotFound
	expType1 := reflect.TypeOf(ErrNotFound{})
	// Cloudant doesn't allow other users to execute queries and returns
	//	*chttp.HTTPError: Forbidden: one of _design, _reader is required for this request
	expType2 := reflect.TypeOf(&chttp.HTTPError{})

	errType := reflect.TypeOf(err)
	if !assert.ObjectsAreEqual(expType1, errType) && !assert.ObjectsAreEqual(expType2, errType) {
		assert.Fail(t, fmt.Sprintf("err must be either of type %v or %v but is of type %v", expType1, expType2, errType))
	}
	if assert.ObjectsAreEqual(expType1, err) {
		require.NoError(t, s.CheckHealth())
	}
}

// CreateAgainPass will create an existing database again; this will not fail
func TestCouchCreateAgainPass(t *testing.T) {
	_, cleanup := couchDBSetup(t)
	defer cleanup()

	session, _ := couchSessionSetup(t)
	require.NoError(t, couchdb.SetupDB(session, "servertestdb", []couchdb.Table{
		TUFFilesCouchTable,
	}))
}

// UpdateCurrent will add a new TUF file if no previous version of that gun and role existed.
func TestCouchUpdateCurrentEmpty(t *testing.T) {
	dbStore, cleanup := couchDBSetup(t)
	defer cleanup()

	testUpdateCurrentEmptyStore(t, dbStore)
}

// UpdateCurrent will add a new TUF file if the version is higher than previous, but fail
// if the version already exists in the DB
func TestCouchUpdateCurrentVersionCheckOldVersionExists(t *testing.T) {
	dbStore, cleanup := couchDBSetup(t)
	defer cleanup()

	testUpdateCurrentVersionCheck(t, dbStore, true)
}

// UpdateCurrent will successfully add a new (higher) version of an existing TUF file,
// but will return an error if the to-be-added version does not exist in the DB, but
// is older than an existing version in the DB.
func TestCouchUpdateCurrentVersionCheckOldVersionNotExist(t *testing.T) {
	t.Skip("Currently couch only errors if the previous version exists - it doesn't check for strictly increasing")
	dbStore, cleanup := couchDBSetup(t)
	defer cleanup()

	testUpdateCurrentVersionCheck(t, dbStore, false)
}

func TestCouchGetVersion(t *testing.T) {
	dbStore, cleanup := couchDBSetup(t)
	defer cleanup()

	testGetVersion(t, dbStore)
}

// UpdateMany succeeds if the updates do not conflict with each other or with what's
// already in the DB
func TestCouchUpdateManyNoConflicts(t *testing.T) {
	dbStore, cleanup := couchDBSetup(t)
	defer cleanup()

	testUpdateManyNoConflicts(t, dbStore)
}

// UpdateMany does not insert any rows (or at least rolls them back) if there
// are any conflicts.
func TestCouchUpdateManyConflictRollback(t *testing.T) {
	dbStore, cleanup := couchDBSetup(t)
	defer cleanup()

	testUpdateManyConflictRollback(t, dbStore)
}

// Delete will remove all TUF metadata, all versions, associated with a gun
func TestCouchDeleteSuccess(t *testing.T) {
	dbStore, cleanup := couchDBSetup(t)
	defer cleanup()

	testDeleteSuccess(t, dbStore)
}

func TestCouchTUFMetaStoreGetCurrent(t *testing.T) {
	dbStore, cleanup := couchDBSetup(t)
	defer cleanup()

	testTUFMetaStoreGetCurrent(t, dbStore)
}

func TestCouchDBGetChanges(t *testing.T) {
	dbStore, cleanup := couchDBSetup(t)
	defer cleanup()

	testGetChanges(t, dbStore)
}

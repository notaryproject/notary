// +build rethinkdb

// Uses a real RethinkDB connection testing purposes

package keydbstore

import (
	"os"
	"testing"

	"github.com/docker/go-connections/tlsconfig"
	"github.com/docker/notary/storage/rethinkdb"
	"github.com/docker/notary/trustmanager"
	"github.com/dvsekhvalnov/jose2go"
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

func rethinkDBSetup(t *testing.T, dbName string) (*RethinkDBKeyStore, func()) {
	session, _ := rethinkSessionSetup(t)
	var cleanup = func() { gorethink.DBDrop(dbName).Exec(session) }

	cleanup()

	err := rethinkdb.SetupDB(session, dbName, []rethinkdb.Table{PrivateKeysRethinkTable})
	require.NoError(t, err)

	return NewRethinkDBKeyStore(dbName, "", "", multiAliasRetriever, validAliases[0], session), cleanup
}

func TestRethinkBootstrapSetsUsernamePassword(t *testing.T) {
	adminSession, source := rethinkSessionSetup(t)
	dbname, username, password := "signertestdb", "testuser", "testpassword"
	otherDB, otherUser, otherPass := "othersignertestdb", "otheruser", "otherpassword"

	// create a separate user with access to a different DB
	require.NoError(t, rethinkdb.SetupDB(adminSession, otherDB, nil))
	defer gorethink.DBDrop(otherDB).Exec(adminSession)
	require.NoError(t, rethinkdb.CreateAndGrantDBUser(adminSession, otherDB, otherUser, otherPass))

	// Bootstrap
	s := NewRethinkDBKeyStore(dbname, username, password, constRetriever, "ignored", adminSession)
	require.NoError(t, s.Bootstrap())
	defer gorethink.DBDrop(dbname).Exec(adminSession)

	// A user with an invalid password cannot connect to rethink DB at all
	_, err := rethinkdb.UserConnection(tlsOpts, source, username, "wrongpass")
	require.Error(t, err)

	// the other user cannot access rethink, causing health checks to fail
	userSession, err := rethinkdb.UserConnection(tlsOpts, source, otherUser, otherPass)
	require.NoError(t, err)
	s = NewRethinkDBKeyStore(dbname, otherUser, otherPass, constRetriever, "ignored", userSession)
	_, _, err = s.GetKey("nonexistent")
	require.Error(t, err)
	require.IsType(t, gorethink.RQLRuntimeError{}, err)
	require.Error(t, s.CheckHealth())

	// our user can access the DB though
	userSession, err = rethinkdb.UserConnection(tlsOpts, source, username, password)
	require.NoError(t, err)
	s = NewRethinkDBKeyStore(dbname, username, password, constRetriever, "ignored", userSession)
	_, _, err = s.GetKey("nonexistent")
	require.Error(t, err)
	require.IsType(t, trustmanager.ErrKeyNotFound{}, err)
	require.NoError(t, s.CheckHealth())
}

func getRethinkDBRows(t *testing.T, dbStore *RethinkDBKeyStore) []RDBPrivateKey {
	res, err := gorethink.DB(dbStore.dbName).Table(PrivateKeysRethinkTable.Name).Run(dbStore.sess)
	require.NoError(t, err)

	var rows []RDBPrivateKey
	require.NoError(t, res.All(&rows))

	return rows
}

func TestRethinkKeyCanOnlyBeAddedOnce(t *testing.T) {
	dbStore, _ := rethinkDBSetup(t, "signerAddTests")
	// defer cleanup()
	expectedKeys := testKeyCanOnlyBeAddedOnce(t, dbStore)

	rows := getRethinkDBRows(t, dbStore)
	require.Len(t, rows, len(expectedKeys))
}

func TestRethinkCreateDelete(t *testing.T) {
	dbStore, cleanup := rethinkDBSetup(t, "signerDeleteTests")
	defer cleanup()
	testCreateDelete(t, dbStore)

	rows := getRethinkDBRows(t, dbStore)
	require.Len(t, rows, 0)
}

func TestRethinkKeyRotation(t *testing.T) {
	dbStore, cleanup := rethinkDBSetup(t, "signerRotationTests")
	defer cleanup()
	privKey := testKeyRotation(t, dbStore, validAliases[1])

	rows := getRethinkDBRows(t, dbStore)
	require.Len(t, rows, 1)

	// require that the key is encrypted with the new passphrase
	require.Equal(t, validAliases[1], rows[0].PassphraseAlias)
	decryptedKey, _, err := jose.Decode(string(rows[0].Private), validAliasesAndPasswds[validAliases[1]])
	require.NoError(t, err)
	require.Equal(t, string(privKey.Private()), decryptedKey)
}

func TestRethinkCheckHealth(t *testing.T) {
	dbStore, cleanup := rethinkDBSetup(t, "signerHealthcheckTests")
	defer cleanup()

	// sanity check - all tables present - health check passes
	require.NoError(t, dbStore.CheckHealth())

	// if the DB is unreachable, health check fails
	require.NoError(t, dbStore.sess.Close())
	require.Error(t, dbStore.CheckHealth())

	// if the connection is reopened, health check succeeds
	require.NoError(t, dbStore.sess.Reconnect())
	require.NoError(t, dbStore.CheckHealth())

	// No tables, health check fails
	require.NoError(t, gorethink.DB(dbStore.dbName).TableDrop(PrivateKeysRethinkTable.Name).Exec(dbStore.sess))
	require.Error(t, dbStore.CheckHealth())

	// No DB, health check fails
	cleanup()
	require.Error(t, dbStore.CheckHealth())
}

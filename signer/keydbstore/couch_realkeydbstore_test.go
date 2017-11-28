// +build couchdb

// Uses a real Couch DB connection testing purposes

package keydbstore

import (
	"context"
	"crypto/rand"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/docker/go-connections/tlsconfig"
	"github.com/dvsekhvalnov/jose2go"
	"github.com/stretchr/testify/require"
	"github.com/theupdateframework/notary/storage/couchdb"
	"github.com/theupdateframework/notary/trustmanager"
	"github.com/theupdateframework/notary/tuf/data"
	"github.com/theupdateframework/notary/tuf/signed"

	"github.com/flimzy/kivik"
	_ "github.com/go-kivik/couchdb" //
)

var tlsOpts = tlsconfig.Options{InsecureSkipVerify: true, ExclusiveRootPools: true}
var cdbNow = time.Date(2016, 12, 31, 1, 1, 1, 0, time.UTC)

func couchSessionSetup(t *testing.T) (*kivik.Client, string) {
	// Get the CouchDB connection string from an environment variable
	couchSource := os.Getenv("DBURL")
	require.NotEqual(t, "", couchSource)

	sess, err := couchdb.AdminConnection(tlsOpts, couchSource)
	require.NoError(t, err)

	return sess, couchSource
}

func couchDBSetup(t *testing.T, dbName string) (*CouchDBKeyStore, func()) {
	session, _ := couchSessionSetup(t)
	var cleanup = func() { couchdb.DBDrop(session, dbName, PrivateKeysCouchTable.Name) }

	cleanup()

	err := couchdb.SetupDB(session, dbName, []couchdb.Table{PrivateKeysCouchTable})
	require.NoError(t, err)

	dbStore := NewCouchDBKeyStore(dbName, "", "", multiAliasRetriever, validAliases[0], session)
	require.Equal(t, "CouchDB", dbStore.Name())

	dbStore.nowFunc = func() time.Time { return cdbNow }

	return dbStore, cleanup
}

func TestCouchBootstrapSetsUsernamePassword(t *testing.T) {
	adminSession, source := couchSessionSetup(t)
	dbname, username, password := "signertestdb", "testuser", "testpassword"
	otherDB, otherUser, otherPass := "othersignertestdb", "otheruser", "otherpassword"

	// create a separate user with access to a different DB
	require.NoError(t, couchdb.SetupDB(adminSession, otherDB, []couchdb.Table{
		{Name: "othertable"},
	}))
	defer couchdb.DBDrop(adminSession, otherDB, "othertable")
	require.NoError(t, couchdb.CreateAndGrantDBUser(adminSession, otherDB, otherUser, otherPass))

	// Bootstrap
	s := NewCouchDBKeyStore(dbname, username, password, constRetriever, "ignored", adminSession)
	require.NoError(t, s.Bootstrap())
	defer couchdb.DBDrop(adminSession, dbname, PrivateKeysCouchTable.Name)

	// A user with an invalid password cannot connect to couch DB at all
	_, err := couchdb.UserConnection(tlsOpts, source, username, "wrongpass")
	require.Error(t, err)

	// the other user cannot access couch, causing health checks to fail
	userSession, err := couchdb.UserConnection(tlsOpts, source, otherUser, otherPass)
	require.NoError(t, err)
	s = NewCouchDBKeyStore(dbname, otherUser, otherPass, constRetriever, "ignored", userSession)
	_, _, err = s.GetPrivateKey("nonexistent")
	require.Error(t, err)
	key := s.GetKey("nonexistent")
	require.Nil(t, key)

	// our user can access the DB though
	userSession, err = couchdb.UserConnection(tlsOpts, source, username, password)
	require.NoError(t, err)
	s = NewCouchDBKeyStore(dbname, username, password, constRetriever, "ignored", userSession)
	_, _, err = s.GetPrivateKey("nonexistent")
	require.Error(t, err)
	require.IsType(t, trustmanager.ErrKeyNotFound{}, err)
	require.NoError(t, s.CheckHealth())
}

func getAllKeys(client *kivik.Client, dbName, tableName string) ([]CDBPrivateKey, error) {
	db, res, err := couchdb.GetAllDocs(client, dbName, tableName)
	if err != nil {
		return nil, err
	}

	var rows []CDBPrivateKey
	for res.Next() {
		var row CDBPrivateKey
		arow, err := db.Get(context.TODO(), res.Key())
		if err != nil {
			return nil, err
		}
		if err = arow.ScanDoc(&row); err != nil {
			return nil, err
		}
		// having an index in the DB we may encounter design documents
		// that we need to skip here
		if strings.HasPrefix(row.ID.ID, "_design/") {
			continue
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// Checks that the DB contains the expected keys, and returns a map of the GormPrivateKey object by key ID
func requireExpectedCDBKeys(t *testing.T, dbStore *CouchDBKeyStore, expectedKeys []data.PrivateKey) map[string]CDBPrivateKey {
	rows, err := getAllKeys(dbStore.client, dbStore.dbName, PrivateKeysCouchTable.Name)
	require.NoError(t, err)

	require.Len(t, rows, len(expectedKeys))
	result := make(map[string]CDBPrivateKey)

	for _, rdbKey := range rows {
		result[rdbKey.KeyID] = rdbKey
	}

	for _, key := range expectedKeys {
		cdbKey, ok := result[key.ID()]
		require.True(t, ok)
		require.NotNil(t, cdbKey)
		require.Equal(t, key.Public(), cdbKey.Public)
		require.Equal(t, key.Algorithm(), cdbKey.Algorithm)

		// because we have to manually set the created and modified times
		require.True(t, cdbKey.CreatedAt.Equal(cdbNow))
		require.True(t, cdbKey.UpdatedAt.Equal(cdbNow))
		require.True(t, cdbKey.DeletedAt.Equal(time.Time{}))
	}

	return result
}

func TestCouchKeyCanOnlyBeAddedOnce(t *testing.T) {
	dbStore, cleanup := couchDBSetup(t, "signer_add_tests")
	defer cleanup()

	expectedKeys := testKeyCanOnlyBeAddedOnce(t, dbStore)

	rdbKeys := requireExpectedCDBKeys(t, dbStore, expectedKeys)

	// none of these keys are active, since they have not been activated
	for _, rdbKey := range rdbKeys {
		require.True(t, rdbKey.LastUsed.Equal(time.Time{}))
	}
}

func TestCouchCreateDelete(t *testing.T) {
	dbStore, cleanup := couchDBSetup(t, "signer_delete_tests")
	defer cleanup()
	expectedKeys := testCreateDelete(t, dbStore)

	rdbKeys := requireExpectedCDBKeys(t, dbStore, expectedKeys)

	// none of these keys are active, since they have not been activated
	for _, rdbKey := range rdbKeys {
		require.True(t, rdbKey.LastUsed.Equal(time.Time{}))
	}
}

func TestCouchKeyRotation(t *testing.T) {
	dbStore, cleanup := couchDBSetup(t, "signer_rotation_tests")
	defer cleanup()

	rotatedKey, nonRotatedKey := testKeyRotation(t, dbStore, validAliases[1])

	rdbKeys := requireExpectedCDBKeys(t, dbStore, []data.PrivateKey{rotatedKey, nonRotatedKey})

	// none of these keys are active, since they have not been activated
	for _, rdbKey := range rdbKeys {
		require.True(t, rdbKey.LastUsed.Equal(time.Time{}))
	}

	// require that the rotated key is encrypted with the new passphrase
	rotatedCDBKey := rdbKeys[rotatedKey.ID()]
	require.Equal(t, validAliases[1], rotatedCDBKey.PassphraseAlias)
	decryptedKey, _, err := jose.Decode(string(rotatedCDBKey.Private), validAliasesAndPasswds[validAliases[1]])
	require.NoError(t, err)
	require.Equal(t, string(rotatedKey.Private()), decryptedKey)

	// require that the nonrotated key is encrypted with the old passphrase
	nonRotatedCDBKey := rdbKeys[nonRotatedKey.ID()]
	require.Equal(t, validAliases[0], nonRotatedCDBKey.PassphraseAlias)
	decryptedKey, _, err = jose.Decode(string(nonRotatedCDBKey.Private), validAliasesAndPasswds[validAliases[0]])
	require.NoError(t, err)
	require.Equal(t, string(nonRotatedKey.Private()), decryptedKey)
}

func TestCouchSigningMarksKeyActive(t *testing.T) {
	dbStore, cleanup := couchDBSetup(t, "signer_activation_tests")
	defer cleanup()

	activeKey, nonActiveKey := testSigningWithKeyMarksAsActive(t, dbStore)

	rdbKeys := requireExpectedCDBKeys(t, dbStore, []data.PrivateKey{activeKey, nonActiveKey})

	// check that activation updates the activated key but not the unactivated key
	require.True(t, rdbKeys[activeKey.ID()].LastUsed.Equal(cdbNow))
	require.True(t, rdbKeys[nonActiveKey.ID()].LastUsed.Equal(time.Time{}))

	// check that signing succeeds
	msg := []byte("successful, db closed")
	sig, err := nonActiveKey.Sign(rand.Reader, msg, nil)
	require.NoError(t, err)
	require.NoError(t, signed.Verifiers[data.ECDSASignature].Verify(
		data.PublicKeyFromPrivate(nonActiveKey), sig, msg))
}

func TestCouchCreateKey(t *testing.T) {
	dbStore, cleanup := couchDBSetup(t, "signer_creation_tests")
	defer cleanup()

	activeED25519Key, pendingED25519Key, pendingECDSAKey := testCreateKey(t, dbStore)

	rdbKeys := requireExpectedCDBKeys(t, dbStore, []data.PrivateKey{activeED25519Key, pendingED25519Key, pendingECDSAKey})

	// check that activation updates the activated key but not the unactivated keys
	require.True(t, rdbKeys[activeED25519Key.ID()].LastUsed.Equal(cdbNow))
	require.True(t, rdbKeys[pendingED25519Key.ID()].LastUsed.Equal(time.Time{}))
	require.True(t, rdbKeys[pendingECDSAKey.ID()].LastUsed.Equal(time.Time{}))
}

func TestCouchUnimplementedInterfaceBehavior(t *testing.T) {
	dbStore, cleanup := couchDBSetup(t, "signer_interface_tests")
	defer cleanup()
	testUnimplementedInterfaceMethods(t, dbStore)
}

package keydbstore

import (
	"testing"

	"github.com/dvsekhvalnov/jose2go"
	"github.com/stretchr/testify/require"
)

func SetupSQLDB(t *testing.T, dbtype, dburl string) *SQLKeyDBStore {
	dbStore, err := NewSQLKeyDBStore(multiAliasRetriever, validAliases[0], dbtype, dburl)
	require.NoError(t, err)

	// Create the DB tables if they don't exist
	dbStore.db.CreateTable(&GormPrivateKey{})

	// verify that the table is empty
	var count int
	query := dbStore.db.Model(&GormPrivateKey{}).Count(&count)
	require.NoError(t, query.Error)
	require.Equal(t, 0, count)

	return dbStore
}

type sqldbSetupFunc func(*testing.T) (*SQLKeyDBStore, func())

var sqldbSetup sqldbSetupFunc

// Creating a new KeyDBStore propagates any db opening error
func TestNewSQLKeyDBStorePropagatesDBError(t *testing.T) {
	dbStore, err := NewSQLKeyDBStore(constRetriever, "ignoredalias", "nodb", "somestring")
	require.Error(t, err)
	require.Nil(t, dbStore)
}

func TestSQLDBHealthCheckMissingTable(t *testing.T) {
	dbStore, cleanup := sqldbSetup(t)
	defer cleanup()

	// health check passes because the table exists
	require.NoError(t, dbStore.HealthCheck())

	// delete the table - health check fails
	require.NoError(t, dbStore.db.DropTableIfExists(&GormPrivateKey{}).Error)
	require.Error(t, dbStore.HealthCheck())
}

func TestSQLDBHealthCheckNoConnection(t *testing.T) {
	dbStore, cleanup := sqldbSetup(t)
	defer cleanup()

	// health check passes because the table exists and connection is open
	require.NoError(t, dbStore.HealthCheck())

	// Close the connection - health check fails
	require.NoError(t, dbStore.db.Close())
	require.Error(t, dbStore.HealthCheck())
}

func getSQLDBRows(t *testing.T, dbStore *SQLKeyDBStore) []GormPrivateKey {
	var rows []GormPrivateKey
	query := dbStore.db.Find(&rows)
	require.NoError(t, query.Error)
	return rows
}

func TestSQLKeyCanOnlyBeAddedOnce(t *testing.T) {
	dbStore, cleanup := sqldbSetup(t)
	defer cleanup()
	expectedKeys := testKeyCanOnlyBeAddedOnce(t, dbStore)

	rows := getSQLDBRows(t, dbStore)
	require.Len(t, rows, len(expectedKeys))
}

func TestSQLCreateDelete(t *testing.T) {
	dbStore, cleanup := sqldbSetup(t)
	defer cleanup()
	testCreateDelete(t, dbStore)

	rows := getSQLDBRows(t, dbStore)
	require.Len(t, rows, 0)
}

func TestSQLKeyRotation(t *testing.T) {
	dbStore, cleanup := sqldbSetup(t)
	defer cleanup()
	privKey := testKeyRotation(t, dbStore, validAliases[1])

	rows := getSQLDBRows(t, dbStore)
	require.Len(t, rows, 1)

	// require that the key is encrypted with the new passphrase
	require.Equal(t, validAliases[1], rows[0].PassphraseAlias)
	decryptedKey, _, err := jose.Decode(string(rows[0].Private), validAliasesAndPasswds[validAliases[1]])
	require.NoError(t, err)
	require.Equal(t, string(privKey.Private()), decryptedKey)
}

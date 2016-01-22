package storage

import (
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/require"
)

// SetUpSQLite creates a sqlite database for testing
func SetUpSQLite(t *testing.T, dbDir string) (*gorm.DB, *SQLStorage) {
	dbStore, err := NewSQLStorage("sqlite3", dbDir+"test_db")
	require.NoError(t, err)

	// Create the DB tables
	err = CreateTUFTable(dbStore.DB)
	require.NoError(t, err)

	err = CreateKeyTable(dbStore.DB)
	require.NoError(t, err)

	// verify that the tables are empty
	var count int
	for _, model := range [2]interface{}{&TUFFile{}, &Key{}} {
		query := dbStore.DB.Model(model).Count(&count)
		require.NoError(t, query.Error)
		require.Equal(t, 0, count)
	}
	return &dbStore.DB, dbStore
}

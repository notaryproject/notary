// +build !mysqldb,!rethinkdb

// Initializes an SQLlite DBs for testing purposes

package storage

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/jinzhu/gorm"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func sqlite3Setup(t *testing.T) (*gorm.DB, *SQLStorage, func()) {
	tempBaseDir, err := ioutil.TempDir("", "notary-test-")
	require.NoError(t, err)

	gormDB, dbStore := SetupSQLDB(t, "sqlite3", tempBaseDir+"test_db")
	var cleanup = func() { os.RemoveAll(tempBaseDir) }
	return gormDB, dbStore, cleanup
}

func init() {
	sqldbSetup = sqlite3Setup
}

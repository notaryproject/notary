//go:build mysqldb || postgresqldb
// +build mysqldb postgresqldb

package storage

import (
	"os"
	"sync"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/theupdateframework/notary/tuf/data"
)

func init() {
	// Get the connection string from an environment variable
	dburl := os.Getenv("DBURL")
	if dburl == "" {
		logrus.Fatal("DBURL environment variable not set")
	}

	for i := 0; i <= 30; i++ {
		gormDB, err := gorm.Open(backend, dburl)
		if err == nil {
			err := gormDB.DB().Ping()
			if err == nil {
				break
			}
		}
		if i == 30 {
			logrus.Fatalf("Unable to connect to %s after 60 seconds", dburl)
		}
		time.Sleep(2 * time.Second)
	}

	sqldbSetup = func(t *testing.T) (*SQLStorage, func()) {
		var dropTables = func(gormDB *gorm.DB) {
			// drop all tables, if they exist
			gormDB.DropTable(&TUFFile{})
			gormDB.DropTable(&SQLChange{})
		}
		gormDB, err := gorm.Open(backend, dburl)
		require.NoError(t, err)
		dropTables(gormDB)
		gormDB.Close()
		dbStore := SetupSQLDB(t, backend, dburl)
		return dbStore, func() {
			dropTables(dbStore.DB)
			dbStore.Close()
		}
	}
}

// TestSQLUpdateManyConcurrentConflictRollback asserts that if several concurrent requests
// to store tuf files with the same gun, role, and version, that exactly one of them succeeds.
func TestSQLUpdateManyConcurrentConflictRollback(t *testing.T) {
	dbStore, cleanup := sqldbSetup(t)
	defer cleanup()

	var gun data.GUN = "testGUN"
	concurrency := 50
	var wg sync.WaitGroup

	errCh := make(chan error)

	for i := 0; i < concurrency; i++ {
		tufObj := SampleCustomTUFObj(gun, data.CanonicalRootRole, 1, []byte{byte(i)})
		updates := []MetaUpdate{MakeUpdate(tufObj)}
		wg.Add(1)
		go func() {
			defer wg.Done()
			errCh <- dbStore.UpdateMany(gun, updates)
		}()
	}

	go func() {
		wg.Wait()
		close(errCh)
	}()

	successes := 0
	for err := range errCh {
		if err == nil {
			successes++
		}
	}

	require.Equal(t, 1, successes)
}

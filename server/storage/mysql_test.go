//go:build mysqldb
// +build mysqldb

// Initializes a MySQL DB for testing purposes

package storage

import (
	"os"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/theupdateframework/notary"
)

func init() {
	// Get the MYSQL connection string from an environment variable
	dburl := os.Getenv("DBURL")
	if dburl == "" {
		logrus.Fatal("MYSQL environment variable not set")
	}

	for i := 0; i <= 30; i++ {
		gormDB, err := gorm.Open("mysql", dburl)
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
		gormDB, err := gorm.Open(notary.MySQLBackend, dburl)
		require.NoError(t, err)
		dropTables(gormDB)
		gormDB.Close()
		dbStore := SetupSQLDB(t, notary.MySQLBackend, dburl)
		return dbStore, func() {
			dropTables(&dbStore.DB)
			dbStore.Close()
		}
	}
}

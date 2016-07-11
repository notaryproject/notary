// +build mysqldb

// Initializes a MySQL DB for testing purposes

package storage

import (
	"os"
	"testing"
	"time"

	"github.com/Sirupsen/logrus"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/require"
)

func init() {
	// Get the MYSQL connection string from an environment variable
	dburl := os.Getenv("MYSQL")
	if dburl == "" {
		logrus.Fatal("MYSQL environment variable not set")
	}

	for i := 0; i < 30; i++ {
		gormDB, err := gorm.Open("mysql", dburl)
		if err == nil {
			err := gormDB.DB().Ping()
			if err == nil {
				break
			}
		}
		if i == 29 {
			logrus.Fatalf("Unable to connect to %s after 60 seconds", dburl)
		}
		time.Sleep(2)
	}

	sqldbSetup = func(t *testing.T) (*SQLStorage, func()) {
		var cleanup = func() {
			gormDB, err := gorm.Open("mysql", dburl)
			require.NoError(t, err)

			// drop all tables, if they exist
			gormDB.DropTable(&TUFFile{})
			gormDB.DropTable(&Key{})
		}
		cleanup()
		dbStore := SetupSQLDB(t, "mysql", dburl)
		return dbStore, cleanup
	}
}

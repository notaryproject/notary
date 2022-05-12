//go:build mysqldb
// +build mysqldb

// Initializes a MySQL DB for testing purposes

package storage

import (
	_ "github.com/go-sql-driver/mysql"
	"github.com/theupdateframework/notary"
)

var backend = notary.MySQLBackend

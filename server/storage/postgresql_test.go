//go:build postgresqldb
// +build postgresqldb

// Initializes a PostgreSQL DB for testing purposes

package storage

import (
	_ "github.com/lib/pq"
	"github.com/theupdateframework/notary"
)

var backend = notary.PostgresBackend

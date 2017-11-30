package couchdb

import (
	"context"
	"testing"

	"github.com/go-kivik/kiviktest/kt"
)

func connect(dsn string, t *testing.T) *client {
	couch := &Couch{}
	driverClient, err := couch.NewClient(context.Background(), dsn)
	if err != nil {
		t.Fatalf("Failed to connect to '%s': %s", dsn, err)
	}
	return driverClient.(*client)
}

func getClient(t *testing.T) *client {
	return connect(kt.DSN(t), t)
}

func getNoAuthClient(t *testing.T) *client {
	return connect(kt.NoAuthDSN(t), t)
}

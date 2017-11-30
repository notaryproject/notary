package couchdb

import (
	"context"
	"testing"

	"github.com/go-kivik/kiviktest/kt"
)

func TestAllDBs(t *testing.T) {
	client := getClient(t)
	_, err := client.AllDBs(context.Background(), nil)
	if err != nil {
		t.Fatalf("Failed: %s", err)
	}
}

func TestDBExists(t *testing.T) {
	client := getClient(t)
	exists, err := client.DBExists(context.Background(), "_users", nil)
	if err != nil {
		t.Fatalf("Failed: %s", err)
	}
	if !exists {
		t.Error("Expected _users to exist")
	}
}

func TestCreateAndDestroyDB(t *testing.T) {
	client := getClient(t)
	dbName := kt.TestDBName(t)
	defer client.DestroyDB(context.Background(), dbName, nil)
	if err := client.CreateDB(context.Background(), dbName, nil); err != nil {
		t.Errorf("Create failed: %s", err)
	}
	if err := client.DestroyDB(context.Background(), dbName, nil); err != nil {
		t.Errorf("Destroy failed: %s", err)
	}
}

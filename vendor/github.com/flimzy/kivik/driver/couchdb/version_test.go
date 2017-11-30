package couchdb

import (
	"context"
	"testing"
)

func TestVersion(t *testing.T) {
	client := getClient(t)
	_, err := client.Version(context.Background())
	if err != nil {
		t.Fatalf("Failed to get server info: %s", err)
	}
}

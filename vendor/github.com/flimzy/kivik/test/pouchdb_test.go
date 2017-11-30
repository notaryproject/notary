// +build js

package test

import (
	"testing"

	_ "github.com/go-kivik/pouchdb"
	"github.com/go-kivik/pouchdb/test"
)

func TestPouchLocal(t *testing.T) {
	test.PouchLocalTest(t)
}

func TestPouchRemote(t *testing.T) {
	test.PouchRemoteTest(t)
}

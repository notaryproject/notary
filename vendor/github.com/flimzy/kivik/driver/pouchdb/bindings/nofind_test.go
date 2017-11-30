package bindings

import (
	"context"
	"testing"

	"github.com/gopherjs/gopherjs/js"

	"github.com/flimzy/kivik"
)

func init() {
	memPouch := js.Global.Get("PouchDB").Call("defaults", map[string]interface{}{
		"db": js.Global.Call("require", "memdown"),
	})
	js.Global.Set("PouchDB", memPouch)
}

// TestNoFind tests that Find() properly returns NotImplemented when the
// pouchdb-find plugin is not loaded.
func TestNoFindPlugin(t *testing.T) {
	t.Run("FindLoaded", func(t *testing.T) {
		db := GlobalPouchDB().New("foo", nil)
		_, err := db.Find(context.Background(), "")
		if kivik.StatusCode(err) == kivik.StatusNotImplemented {
			t.Errorf("Got StatusNotImplemented when pouchdb-find should be loaded")
		}
	})
	t.Run("FindNotLoaded", func(t *testing.T) {
		db := GlobalPouchDB().New("foo", nil)
		db.Object.Set("find", nil) // Fake it
		_, err := db.Find(context.Background(), "")
		if code := kivik.StatusCode(err); code != kivik.StatusNotImplemented {
			t.Errorf("Expected %d error, got %d/%s\n", kivik.StatusNotImplemented, code, err)
		}
	})
}

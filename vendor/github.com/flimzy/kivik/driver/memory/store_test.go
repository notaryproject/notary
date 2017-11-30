package memory

import (
	"strings"
	"testing"

	"github.com/flimzy/diff"
)

func TestRandStr(t *testing.T) {
	str := randStr()
	if len(str) != 32 {
		t.Errorf("Expected 32-char string, got %d", len(str))
	}
}

func TestToCouchDoc(t *testing.T) {
	type tcdTest struct {
		Name     string
		Input    interface{}
		Expected couchDoc
		Error    string
	}
	tests := []tcdTest{
		{
			Name:     "Map",
			Input:    map[string]interface{}{"foo": "bar"},
			Expected: couchDoc{"foo": "bar"},
		},
		{
			Name:     "CouchDoc",
			Input:    couchDoc{"foo": "bar"},
			Expected: couchDoc{"foo": "bar"},
		},
		{
			Name:  "Unmarshalable",
			Input: make(chan int),
			Error: "json: unsupported type: chan int",
		},
		{
			Name:     "Marshalable",
			Input:    map[string]string{"foo": "bar"},
			Expected: couchDoc{"foo": "bar"},
		},
	}
	for _, test := range tests {
		func(test tcdTest) {
			t.Run(test.Name, func(t *testing.T) {
				result, err := toCouchDoc(test.Input)
				var msg string
				if err != nil {
					msg = err.Error()
				}
				if msg != test.Error {
					t.Errorf("Unexpected error: %s", msg)
				}
				if d := diff.Interface(test.Expected, result); d != nil {
					t.Error(d)
				}
			})
		}(test)
	}
}

func TestAddRevision(t *testing.T) {
	d := &database{
		docs: make(map[string]*document),
	}
	r := d.addRevision(couchDoc{"_id": "bar"})
	if !strings.HasPrefix(r, "1-") {
		t.Errorf("Expected initial revision to start with '1-', but got '%s'", r)
	}
	if len(r) != 34 {
		t.Errorf("rev (%s) is %d chars long, expected 34", r, len(r))
	}
	r = d.addRevision(couchDoc{"_id": "bar"})
	if !strings.HasPrefix(r, "2-") {
		t.Errorf("Expected second revision to start with '2-', but got '%s'", r)
	}
	if len(r) != 34 {
		t.Errorf("rev (%s) is %d chars long, expected 34", r, len(r))
	}
	t.Run("NoID", func(t *testing.T) {
		r := func() (i interface{}) {
			defer func() {
				i = recover()
			}()
			d.addRevision(nil)
			return nil
		}()
		if r == nil {
			t.Errorf("addRevision without ID should panic")
		}
	})
	t.Run("InvalidJSON", func(t *testing.T) {
		r := func() (i interface{}) {
			defer func() {
				i = recover()
			}()
			d.addRevision(couchDoc{"_id": "foo", "invalid": make(chan int)})
			return nil
		}()
		if r == nil {
			t.Errorf("unmarshalable objects should panic")
		}
	})
}

func TestAddLocalRevision(t *testing.T) {
	d := &database{
		docs: make(map[string]*document),
	}
	r := d.addRevision(couchDoc{"_id": "_local/foo"})
	if r != "1-0" {
		t.Errorf("Expected local revision, got %s", r)
	}
	r = d.addRevision(couchDoc{"_id": "_local/foo"})
	if r != "1-0" {
		t.Errorf("Expected local revision, got %s", r)
	}
}

func TestGetRevisionMissing(t *testing.T) {
	d := &database{
		docs: make(map[string]*document),
	}
	_, found := d.getRevision("foo", "bar")
	if found {
		t.Errorf("Should not have found missing revision")
	}
}

func TestGetRevisionFound(t *testing.T) {
	d := &database{
		docs: make(map[string]*document),
	}
	r := d.addRevision(map[string]interface{}{"_id": "foo", "a": 1})
	_ = d.addRevision(map[string]interface{}{"_id": "foo", "a": 2})
	result, found := d.getRevision("foo", r)
	if !found {
		t.Errorf("Should have found revision")
	}
	expected := map[string]interface{}{"_id": "foo", "a": 1, "_rev": r}
	if d := diff.AsJSON(expected, result.data); d != nil {
		t.Error(d)
	}
}

func TestRev(t *testing.T) {
	t.Run("Missing", func(t *testing.T) {
		d := couchDoc{}
		if d.Rev() != "" {
			t.Errorf("Rev should be missing, but got %s", d.Rev())
		}
	})
	t.Run("Set", func(t *testing.T) {
		d := couchDoc{"_rev": "foo"}
		if d.Rev() != "foo" {
			t.Errorf("Rev should be foo, but got %s", d.Rev())
		}
	})
	t.Run("NonString", func(t *testing.T) {
		d := couchDoc{"_rev": true}
		if d.Rev() != "" {
			t.Errorf("Rev should be missing, but got %s", d.Rev())
		}
	})
}

func TestID(t *testing.T) {
	t.Run("Missing", func(t *testing.T) {
		d := couchDoc{}
		if d.ID() != "" {
			t.Errorf("ID should be missing, but got %s", d.ID())
		}
	})
	t.Run("Set", func(t *testing.T) {
		d := couchDoc{"_id": "foo"}
		if d.ID() != "foo" {
			t.Errorf("ID should be foo, but got %s", d.ID())
		}
	})
	t.Run("NonString", func(t *testing.T) {
		d := couchDoc{"_id": true}
		if d.ID() != "" {
			t.Errorf("ID should be missing, but got %s", d.ID())
		}
	})
}

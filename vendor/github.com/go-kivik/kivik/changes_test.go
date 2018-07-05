package kivik

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/flimzy/diff"
	"github.com/flimzy/testy"
	"github.com/go-kivik/kivik/driver"
	"github.com/go-kivik/kivik/mock"
)

func TestChangesNext(t *testing.T) {
	tests := []struct {
		name     string
		changes  *Changes
		expected bool
	}{
		{
			name: "nothing more",
			changes: &Changes{
				iter: &iter{closed: true},
			},
			expected: false,
		},
		{
			name: "more",
			changes: &Changes{
				iter: &iter{
					feed: &mockIterator{
						NextFunc: func(_ interface{}) error { return nil },
					},
				},
			},
			expected: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := test.changes.Next()
			if result != test.expected {
				t.Errorf("Unexpected result: %v", result)
			}
		})
	}
}

func TestChangesErr(t *testing.T) {
	expected := "foo error"
	c := &Changes{
		iter: &iter{lasterr: errors.New(expected)},
	}
	err := c.Err()
	testy.Error(t, expected, err)
}

func TestChangesClose(t *testing.T) {
	expected := "close error"
	c := &Changes{
		iter: &iter{
			feed: &mockIterator{CloseFunc: func() error { return errors.New(expected) }},
		},
	}
	err := c.Close()
	testy.Error(t, expected, err)
}

func TestChangesIteratorNext(t *testing.T) {
	expected := "foo error"
	c := &changesIterator{
		Changes: &mock.Changes{
			NextFunc: func(_ *driver.Change) error { return errors.New(expected) },
		},
	}
	var i driver.Change
	err := c.Next(&i)
	testy.Error(t, expected, err)
}

func TestChangesIteratorNew(t *testing.T) {
	ch := newChanges(context.Background(), &mock.Changes{})
	expected := &Changes{
		iter: &iter{
			feed: &changesIterator{
				Changes: &mock.Changes{},
			},
			curVal: &driver.Change{},
		},
		changesi: &mock.Changes{},
	}
	ch.cancel = nil // determinism
	if d := diff.Interface(expected, ch); d != nil {
		t.Error(d)
	}
}

func TestChangesGetters(t *testing.T) {
	c := &Changes{
		iter: &iter{
			curVal: &driver.Change{
				ID:      "foo",
				Deleted: true,
				Changes: []string{"1", "2", "3"},
			},
		},
	}

	t.Run("Changes", func(t *testing.T) {
		expected := []string{"1", "2", "3"}
		result := c.Changes()
		if d := diff.Interface(expected, result); d != nil {
			t.Error(d)
		}
	})

	t.Run("Deleted", func(t *testing.T) {
		expected := true
		result := c.Deleted()
		if expected != result {
			t.Errorf("Unexpected result: %v", result)
		}
	})

	t.Run("ID", func(t *testing.T) {
		expected := "foo"
		result := c.ID()
		if expected != result {
			t.Errorf("Unexpected result: %v", result)
		}
	})
}

func TestChangesScanDoc(t *testing.T) {
	tests := []struct {
		name     string
		changes  *Changes
		expected interface{}
		status   int
		err      string
	}{
		{
			name: "success",
			changes: &Changes{
				iter: &iter{
					ready: true,
					curVal: &driver.Change{
						Doc: []byte(`{"foo":123.4}`),
					},
				},
			},
			expected: map[string]interface{}{"foo": 123.4},
		},
		{
			name: "closed",
			changes: &Changes{
				iter: &iter{
					closed: true,
				},
			},
			status: StatusIteratorUnusable,
			err:    "kivik: Iterator is closed",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var result interface{}
			err := test.changes.ScanDoc(&result)
			testy.StatusError(t, test.err, test.status, err)
			if d := diff.Interface(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestChanges(t *testing.T) {
	tests := []struct {
		name     string
		db       *DB
		opts     Options
		expected *Changes
		status   int
		err      string
	}{
		{
			name: "db error",
			db: &DB{
				driverDB: &mock.DB{
					ChangesFunc: func(_ context.Context, _ map[string]interface{}) (driver.Changes, error) {
						return nil, errors.New("db error")
					},
				},
			},
			status: 500,
			err:    "db error",
		},
		{
			name: "success",
			db: &DB{
				driverDB: &mock.DB{
					ChangesFunc: func(_ context.Context, opts map[string]interface{}) (driver.Changes, error) {
						expectedOpts := map[string]interface{}{"foo": 123.4}
						if d := diff.Interface(expectedOpts, opts); d != nil {
							return nil, fmt.Errorf("Unexpected options:\n%s", d)
						}
						return &mock.Changes{}, nil
					},
				},
			},
			opts: map[string]interface{}{"foo": 123.4},
			expected: &Changes{
				iter: &iter{
					feed: &changesIterator{
						Changes: &mock.Changes{},
					},
					curVal: &driver.Change{},
				},
				changesi: &mock.Changes{},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.db.Changes(context.Background(), test.opts)
			testy.StatusError(t, test.err, test.status, err)
			result.cancel = nil // Determinism
			if d := diff.Interface(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}

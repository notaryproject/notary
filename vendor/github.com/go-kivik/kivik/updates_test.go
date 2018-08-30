package kivik

import (
	"context"
	"errors"
	"testing"

	"github.com/flimzy/diff"
	"github.com/flimzy/testy"
	"github.com/go-kivik/kivik/driver"
	"github.com/go-kivik/kivik/mock"
)

func TestDBUpdatesNext(t *testing.T) {
	tests := []struct {
		name     string
		updates  *DBUpdates
		expected bool
	}{
		{
			name: "nothing more",
			updates: &DBUpdates{
				iter: &iter{closed: true},
			},
			expected: false,
		},
		{
			name: "more",
			updates: &DBUpdates{
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
			result := test.updates.Next()
			if result != test.expected {
				t.Errorf("Unexpected result: %v", result)
			}
		})
	}
}

func TestDBUpdatesClose(t *testing.T) {
	expected := "close error"
	u := &DBUpdates{
		iter: &iter{
			feed: &mockIterator{CloseFunc: func() error { return errors.New(expected) }},
		},
	}
	err := u.Close()
	testy.Error(t, expected, err)
}

func TestDBUpdatesErr(t *testing.T) {
	expected := "foo error"
	u := &DBUpdates{
		iter: &iter{lasterr: errors.New(expected)},
	}
	err := u.Err()
	testy.Error(t, expected, err)
}

func TestDBUpdatesIteratorNext(t *testing.T) {
	expected := "foo error"
	u := &updatesIterator{
		DBUpdates: &mock.DBUpdates{
			NextFunc: func(_ *driver.DBUpdate) error { return errors.New(expected) },
		},
	}
	var i driver.DBUpdate
	err := u.Next(&i)
	testy.Error(t, expected, err)
}

func TestDBUpdatesIteratorNew(t *testing.T) {
	u := newDBUpdates(context.Background(), &mock.DBUpdates{})
	expected := &DBUpdates{
		iter: &iter{
			feed: &updatesIterator{
				DBUpdates: &mock.DBUpdates{},
			},
			curVal: &driver.DBUpdate{},
		},
		updatesi: &mock.DBUpdates{},
	}
	u.cancel = nil // determinism
	if d := diff.Interface(expected, u); d != nil {
		t.Error(d)
	}
}

func TestDBUpdateGetters(t *testing.T) {
	dbname := "foo"
	updateType := "chicken"
	seq := "abc123"
	u := &DBUpdates{
		iter: &iter{
			ready: true,
			curVal: &driver.DBUpdate{
				DBName: dbname,
				Type:   updateType,
				Seq:    seq,
			},
		},
	}

	t.Run("DBName", func(t *testing.T) {
		result := u.DBName()
		if result != dbname {
			t.Errorf("Unexpected result: %s", result)
		}
	})

	t.Run("Type", func(t *testing.T) {
		result := u.Type()
		if result != updateType {
			t.Errorf("Unexpected result: %s", result)
		}
	})

	t.Run("Seq", func(t *testing.T) {
		result := u.Seq()
		if result != seq {
			t.Errorf("Unexpected result: %s", result)
		}
	})

	t.Run("Not Ready", func(t *testing.T) {
		u.ready = false

		t.Run("DBName", func(t *testing.T) {
			result := u.DBName()
			if result != "" {
				t.Errorf("Unexpected result: %s", result)
			}
		})

		t.Run("Type", func(t *testing.T) {
			result := u.Type()
			if result != "" {
				t.Errorf("Unexpected result: %s", result)
			}
		})

		t.Run("Seq", func(t *testing.T) {
			result := u.Seq()
			if result != "" {
				t.Errorf("Unexpected result: %s", result)
			}
		})
	})
}

func TestDBUpdates(t *testing.T) {
	tests := []struct {
		name     string
		client   *Client
		expected *DBUpdates
		status   int
		err      string
	}{
		{
			name: "non-DBUpdater",
			client: &Client{
				driverClient: &mock.Client{},
			},
			status: StatusNotImplemented,
			err:    "kivik: driver does not implement DBUpdater",
		},
		{
			name: "db error",
			client: &Client{
				driverClient: &mock.DBUpdater{
					DBUpdatesFunc: func() (driver.DBUpdates, error) {
						return nil, errors.New("db error")
					},
				},
			},
			status: StatusInternalServerError,
			err:    "db error",
		},
		{
			name: "success",
			client: &Client{
				driverClient: &mock.DBUpdater{
					DBUpdatesFunc: func() (driver.DBUpdates, error) {
						return &mock.DBUpdates{ID: "a"}, nil
					},
				},
			},
			expected: &DBUpdates{
				iter: &iter{
					feed: &updatesIterator{
						DBUpdates: &mock.DBUpdates{ID: "a"},
					},
					curVal: &driver.DBUpdate{},
				},
				updatesi: &mock.DBUpdates{ID: "a"},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.client.DBUpdates()
			testy.StatusError(t, test.err, test.status, err)
			result.cancel = nil // Determinism
			if d := diff.Interface(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}

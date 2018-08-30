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

func TestFind(t *testing.T) {
	tests := []struct {
		name     string
		db       *DB
		query    interface{}
		expected *Rows
		status   int
		err      string
	}{
		{
			name: "non-finder",
			db: &DB{
				driverDB: &mock.DB{},
			},
			status: StatusNotImplemented,
			err:    "kivik: driver does not support Find interface",
		},
		{
			name: "db error",
			db: &DB{
				driverDB: &mock.Finder{
					FindFunc: func(_ context.Context, _ interface{}) (driver.Rows, error) {
						return nil, errors.New("db error")
					},
				},
			},
			status: StatusInternalServerError,
			err:    "db error",
		},
		{
			name: "success",
			db: &DB{
				driverDB: &mock.Finder{
					FindFunc: func(_ context.Context, query interface{}) (driver.Rows, error) {
						expectedQuery := int(3)
						if d := diff.Interface(expectedQuery, query); d != nil {
							return nil, fmt.Errorf("Unexpected query:\n%s", d)
						}
						return &mock.Rows{ID: "a"}, nil
					},
				},
			},
			query: int(3),
			expected: &Rows{
				iter: &iter{
					feed: &rowsIterator{
						Rows: &mock.Rows{ID: "a"},
					},
					curVal: &driver.Row{},
				},
				rowsi: &mock.Rows{ID: "a"},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.db.Find(context.Background(), test.query)
			testy.StatusError(t, test.err, test.status, err)
			result.cancel = nil // Determinism
			if d := diff.Interface(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestCreateIndex(t *testing.T) {
	tests := []struct {
		testName   string
		db         *DB
		ddoc, name string
		index      interface{}
		status     int
		err        string
	}{
		{
			testName: "non-finder",
			db: &DB{
				driverDB: &mock.DB{},
			},
			status: StatusNotImplemented,
			err:    "kivik: driver does not support Find interface",
		},
		{
			testName: "db error",
			db: &DB{
				driverDB: &mock.Finder{
					CreateIndexFunc: func(_ context.Context, _, _ string, _ interface{}) error {
						return errors.New("db error")
					},
				},
			},
			status: StatusInternalServerError,
			err:    "db error",
		},
		{
			testName: "success",
			db: &DB{
				driverDB: &mock.Finder{
					CreateIndexFunc: func(_ context.Context, ddoc, name string, index interface{}) error {
						expectedDdoc := "foo"
						expectedName := "bar"
						expectedIndex := int(3)
						if expectedDdoc != ddoc {
							return fmt.Errorf("Unexpected ddoc: %s", ddoc)
						}
						if expectedName != name {
							return fmt.Errorf("Unexpected name: %s", name)
						}
						if d := diff.Interface(expectedIndex, index); d != nil {
							return fmt.Errorf("Unexpected index:\n%s", d)
						}
						return nil
					},
				},
			},
			ddoc:  "foo",
			name:  "bar",
			index: int(3),
		},
	}
	for _, test := range tests {
		t.Run(test.testName, func(t *testing.T) {
			err := test.db.CreateIndex(context.Background(), test.ddoc, test.name, test.index)
			testy.StatusError(t, test.err, test.status, err)
		})
	}
}

func TestDeleteIndex(t *testing.T) {
	tests := []struct {
		testName   string
		db         *DB
		ddoc, name string
		status     int
		err        string
	}{
		{
			testName: "non-finder",
			db: &DB{
				driverDB: &mock.DB{},
			},
			status: StatusNotImplemented,
			err:    "kivik: driver does not support Find interface",
		},
		{
			testName: "db error",
			db: &DB{
				driverDB: &mock.Finder{
					DeleteIndexFunc: func(_ context.Context, _, _ string) error {
						return errors.New("db error")
					},
				},
			},
			status: StatusInternalServerError,
			err:    "db error",
		},
		{
			testName: "success",
			db: &DB{
				driverDB: &mock.Finder{
					DeleteIndexFunc: func(_ context.Context, ddoc, name string) error {
						expectedDdoc := "foo"
						expectedName := "bar"
						if expectedDdoc != ddoc {
							return fmt.Errorf("Unexpected ddoc: %s", ddoc)
						}
						if expectedName != name {
							return fmt.Errorf("Unexpected name: %s", name)
						}
						return nil
					},
				},
			},
			ddoc: "foo",
			name: "bar",
		},
	}
	for _, test := range tests {
		t.Run(test.testName, func(t *testing.T) {
			err := test.db.DeleteIndex(context.Background(), test.ddoc, test.name)
			testy.StatusError(t, test.err, test.status, err)
		})
	}
}

func TestGetIndexes(t *testing.T) {
	tests := []struct {
		testName string
		db       *DB
		expected []Index
		status   int
		err      string
	}{
		{
			testName: "non-finder",
			db: &DB{
				driverDB: &mock.DB{},
			},
			status: StatusNotImplemented,
			err:    "kivik: driver does not support Find interface",
		},
		{
			testName: "db error",
			db: &DB{
				driverDB: &mock.Finder{
					GetIndexesFunc: func(_ context.Context) ([]driver.Index, error) {
						return nil, errors.New("db error")
					},
				},
			},
			status: StatusInternalServerError,
			err:    "db error",
		},
		{
			testName: "success",
			db: &DB{
				driverDB: &mock.Finder{
					GetIndexesFunc: func(_ context.Context) ([]driver.Index, error) {
						return []driver.Index{
							{Name: "a"},
							{Name: "b"},
						}, nil
					},
				},
			},
			expected: []Index{
				{
					Name: "a",
				},
				{
					Name: "b",
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.testName, func(t *testing.T) {
			result, err := test.db.GetIndexes(context.Background())
			testy.StatusError(t, test.err, test.status, err)
			if d := diff.Interface(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestExplain(t *testing.T) {
	tests := []struct {
		name     string
		db       driver.DB
		query    interface{}
		expected *QueryPlan
		status   int
		err      string
	}{
		{
			name:   "non-finder",
			db:     &mock.DB{},
			status: StatusNotImplemented,
			err:    "kivik: driver does not support Find interface",
		},
		{
			name: "explain error",
			db: &mock.Finder{
				ExplainFunc: func(_ context.Context, _ interface{}) (*driver.QueryPlan, error) {
					return nil, errors.New("explain error")
				},
			},
			status: StatusInternalServerError,
			err:    "explain error",
		},
		{
			name: "success",
			db: &mock.Finder{
				ExplainFunc: func(_ context.Context, query interface{}) (*driver.QueryPlan, error) {
					expectedQuery := int(3)
					if d := diff.Interface(expectedQuery, query); d != nil {
						return nil, fmt.Errorf("Unexpected query:\n%s", d)
					}
					return &driver.QueryPlan{DBName: "foo"}, nil
				},
			},
			query:    int(3),
			expected: &QueryPlan{DBName: "foo"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := &DB{driverDB: test.db}
			result, err := db.Explain(context.Background(), test.query)
			testy.StatusError(t, test.err, test.status, err)
			if d := diff.Interface(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}

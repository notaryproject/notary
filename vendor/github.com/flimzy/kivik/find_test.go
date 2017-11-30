package kivik

import (
	"context"
	"errors"
	"testing"

	"github.com/flimzy/diff"
	"github.com/flimzy/kivik/driver"
	"github.com/flimzy/testy"
)

func TestExplain(t *testing.T) {
	tests := []struct {
		name     string
		db       driver.DB
		expected *QueryPlan
		status   int
		err      string
	}{
		{
			name:   "non-finder",
			db:     &mockDB{},
			status: StatusNotImplemented,
			err:    "kivik: driver does not support explain",
		},
		{
			name:   "explain error",
			db:     &mockExplainer{err: errors.New("explain error")},
			status: StatusInternalServerError,
			err:    "explain error",
		},
		{
			name:     "success",
			db:       &mockExplainer{plan: &driver.QueryPlan{DBName: "foo"}},
			expected: &QueryPlan{DBName: "foo"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := &DB{driverDB: test.db}
			result, err := db.Explain(context.Background(), nil)
			testy.StatusError(t, test.err, test.status, err)
			if d := diff.Interface(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}

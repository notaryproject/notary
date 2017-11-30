package memory

import (
	"context"
	"io"
	"sort"
	"testing"

	"github.com/flimzy/diff"
	"github.com/flimzy/kivik/driver"
)

func TestAllDocsClose(t *testing.T) {
	rs := &alldocsResults{
		resultSet{
			revs:   []*revision{{}, {}}, // Two nil revisions
			docIDs: []string{"a", "b"},
		},
	}
	row := driver.Row{}
	if err := rs.Next(&row); err != nil {
		t.Fatal(err)
	}
	if err := rs.Close(); err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if err := rs.Next(&row); err != io.EOF {
		t.Errorf("Unexpected Next() error after closing: %s", err)
	}
}

func TestAllDocs(t *testing.T) {
	type adTest struct {
		Name        string
		ExpectedIDs []string
		Error       string
		DB          *db
		RowsError   string
	}
	tests := []adTest{
		{
			Name: "NoDocs",
		},
		{
			Name: "OneDoc",
			DB: func() *db {
				db := setupDB(t, nil)
				if _, err := db.Put(context.Background(), "foo", map[string]string{"foo": "bar"}); err != nil {
					t.Fatal(err)
				}
				return db
			}(),
			ExpectedIDs: []string{"foo"},
		},
		{
			Name: "Five Docs",
			DB: func() *db {
				db := setupDB(t, nil)
				for _, id := range []string{"a", "c", "z", "q", "chicken"} {
					if _, err := db.Put(context.Background(), id, map[string]string{"value": id}); err != nil {
						t.Fatal(err)
					}
				}
				return db
			}(),
			ExpectedIDs: []string{"a", "c", "chicken", "q", "z"},
		},
	}
	for _, test := range tests {
		func(test adTest) {
			t.Run(test.Name, func(t *testing.T) {
				db := test.DB
				if db == nil {
					db = setupDB(t, nil)
				}
				rows, err := db.AllDocs(context.Background(), nil)
				var msg string
				if err != nil {
					msg = err.Error()
				}
				if test.Error != msg {
					t.Errorf("Unexpected error: %s", msg)
				}
				if err != nil {
					return
				}
				checkRows(t, rows, test.ExpectedIDs, test.RowsError)
			})
		}(test)
	}
}

func checkRows(t *testing.T, rows driver.Rows, expectedIDs []string, rowsErr string) {
	var row driver.Row
	var ids []string
	msg := ""
	for {
		e := rows.Next(&row)
		if e != nil {
			if e != io.EOF {
				msg = e.Error()
			}
			break
		}
		ids = append(ids, row.ID)
	}
	if rowsErr != msg {
		t.Errorf("Unexpected rows error: %s", msg)
	}
	sort.Strings(ids)
	if d := diff.TextSlices(expectedIDs, ids); d != nil {
		t.Error(d)
	}
}

func TestAllDocsUpdateSeq(t *testing.T) {
	expected := "12345"
	rs := &alldocsResults{resultSet{updateSeq: expected}}
	if result := rs.UpdateSeq(); result != expected {
		t.Errorf("Unexpected upste seq: %s", result)
	}
}

func TestAllDocsTotalRows(t *testing.T) {
	expected := int64(123)
	rs := &alldocsResults{resultSet{totalRows: expected}}
	if result := rs.TotalRows(); result != expected {
		t.Errorf("Unexpected upste seq: %d", result)
	}
}

func TestAllDocsOffset(t *testing.T) {
	expected := int64(123)
	rs := &alldocsResults{resultSet{offset: expected}}
	if result := rs.Offset(); result != expected {
		t.Errorf("Unexpected upste seq: %d", result)
	}
}

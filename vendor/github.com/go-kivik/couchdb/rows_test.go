package couchdb

import (
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/flimzy/testy"

	"github.com/go-kivik/kivik"
	"github.com/go-kivik/kivik/driver"
)

var input = `
{
    "offset": 6,
    "rows": [
        {
            "id": "SpaghettiWithMeatballs",
            "key": "meatballs",
            "value": 1
        },
        {
            "id": "SpaghettiWithMeatballs",
            "key": "spaghetti",
            "value": 1
        },
        {
            "id": "SpaghettiWithMeatballs",
            "key": "tomato sauce",
            "value": 1
        }
    ],
    "total_rows": 3
}
`

var expectedKeys = []string{`"meatballs"`, `"spaghetti"`, `"tomato sauce"`}

func TestRowsIterator(t *testing.T) {
	rows := newRows(ioutil.NopCloser(strings.NewReader(input)))
	var count int
	for {
		row := &driver.Row{}
		err := rows.Next(row)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Next() failed: %s", err)
		}
		if string(row.Key) != expectedKeys[count] {
			t.Errorf("Expected key #%d to be %s, got %s", count, expectedKeys[count], string(row.Key))
		}
		if count++; count > 10 {
			t.Fatalf("Ran too many iterations.")
		}
	}
	if count != 3 {
		t.Errorf("Expected 3 rows, got %d", count)
	}
	if rows.TotalRows() != 3 {
		t.Errorf("Expected TotalRows of 3, got %d", rows.TotalRows())
	}
	if rows.Offset() != 6 {
		t.Errorf("Expected Offset of 6, got %d", rows.Offset())
	}
	if err := rows.Next(&driver.Row{}); err != io.EOF {
		t.Errorf("Calling Next() after end returned unexpected error: %s", err)
	}
	if err := rows.Close(); err != nil {
		t.Errorf("Error closing rows iterator: %s", err)
	}
}

func TestRowsIteratorErrors(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		status int
		err    string
	}{
		{
			name:   "empty input",
			input:  "",
			status: kivik.StatusBadResponse,
			err:    "no closing delimiter: EOF",
		},
		{
			name:   "unexpected delimiter",
			input:  "[]",
			status: kivik.StatusBadResponse,
			err:    "Unexpected JSON delimiter: [",
		},
		{
			name:   "unexpected input",
			input:  `"foo"`,
			status: kivik.StatusBadResponse,
			err:    "Unexpected token string: foo",
		},
		{
			name:   "missing closing delimiter",
			input:  `{"rows":[{"id":"1","key":"1","value":1}`,
			status: kivik.StatusBadResponse,
			err:    "no closing delimiter: EOF",
		},
		{
			name:   "unexpected key",
			input:  `{"foo":"bar"}`,
			status: kivik.StatusBadResponse,
			err:    "Unexpected key: foo",
		},
		{
			name:   "unexpected key after valid row",
			input:  `{"rows":[{"id":"1","key":"1","value":1}],"foo":"bar"}`,
			status: kivik.StatusBadResponse,
			err:    "Unexpected key: foo",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rows := newRows(ioutil.NopCloser(strings.NewReader(test.input)))
			for i := 0; i < 10; i++ {
				err := rows.Next(&driver.Row{})
				if err == nil {
					continue
				}
				testy.StatusError(t, test.err, test.status, err)
			}
		})
	}
}

var findInput = `
{"warning":"no matching index found, create an index to optimize query time",
"docs":[
{"id":"SpaghettiWithMeatballs","key":"meatballs","value":1},
{"id":"SpaghettiWithMeatballs","key":"spaghetti","value":1},
{"id":"SpaghettiWithMeatballs","key":"tomato sauce","value":1}
],
"bookmark": "nil"
}
`

func TestFindRowsIterator(t *testing.T) {
	rows := newRows(ioutil.NopCloser(strings.NewReader(findInput)))
	var count int
	for {
		row := &driver.Row{}
		err := rows.Next(row)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Next() failed: %s", err)
		}
		if count++; count > 10 {
			t.Fatalf("Ran too many iterations.")
		}
	}
	if count != 3 {
		t.Errorf("Expected 3 rows, got %d", count)
	}
	if err := rows.Next(&driver.Row{}); err != io.EOF {
		t.Errorf("Calling Next() after end returned unexpected error: %s", err)
	}
	if err := rows.Close(); err != nil {
		t.Errorf("Error closing rows iterator: %s", err)
	}
	if rows.Warning() != "no matching index found, create an index to optimize query time" {
		t.Errorf("Unexpected warning: %s", rows.Warning())
	}
	if rows.Bookmark() != "nil" {
		t.Errorf("Unexpected bookmark: %s", rows.Bookmark())
	}
}

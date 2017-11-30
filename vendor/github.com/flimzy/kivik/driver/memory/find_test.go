package memory

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/flimzy/diff"
	"github.com/flimzy/kivik/driver"
)

func TestIndexSpecUnmarshalJSON(t *testing.T) {
	type isuTest struct {
		name     string
		input    string
		expected *indexSpec
		err      string
	}
	tests := []isuTest{
		{
			name:     "ddoc only",
			input:    `"foo"`,
			expected: &indexSpec{ddoc: "foo"},
		},
		{
			name:     "ddoc and index",
			input:    `["foo","bar"]`,
			expected: &indexSpec{ddoc: "foo", index: "bar"},
		},
		{
			name:  "invalid json",
			input: "asdf",
			err:   "invalid character 'a' looking for beginning of value",
		},
		{
			name:  "extra fields",
			input: `["foo","bar","baz"]`,
			err:   "invalid index specification",
		},
		{
			name:     "One field",
			input:    `["foo"]`,
			expected: &indexSpec{ddoc: "foo"},
		},
		{
			name:  "Empty array",
			input: `[]`,
			err:   "invalid index specification",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := &indexSpec{}
			err := result.UnmarshalJSON([]byte(test.input))
			var msg string
			if err != nil {
				msg = err.Error()
			}
			if msg != test.err {
				t.Errorf("Unexpected error: %s", err)
			}
			if err != nil {
				return
			}
			if d := diff.Interface(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestCreateIndex(t *testing.T) {
	d := &db{}
	err := d.CreateIndex(context.Background(), "foo", "bar", "baz")
	if err != errFindNotImplemented {
		t.Errorf("Unexpected error: %s", err)
	}
}

func TestGetIndexes(t *testing.T) {
	d := &db{}
	_, err := d.GetIndexes(context.Background())
	if err != errFindNotImplemented {
		t.Errorf("Unexpected error: %s", err)
	}
}

func TestDeleteIndex(t *testing.T) {
	d := &db{}
	err := d.DeleteIndex(context.Background(), "foo", "bar")
	if err != errFindNotImplemented {
		t.Errorf("Unexpected error: %s", err)
	}
}

// TestFind tests selectors, to see that the proper doc IDs are returned.
func TestFind(t *testing.T) {
	type findTest struct {
		name        string
		db          *db
		query       interface{}
		expectedIDs []string
		err         string
		rowsErr     string
	}
	tests := []findTest{
		{
			name:  "invalid query",
			query: make(chan int),
			err:   "json: unsupported type: chan int",
		},
		{
			name:  "Invalid JSON query",
			query: "asdf",
			err:   "invalid character 'a' looking for beginning of value",
		},
		{
			name: "No query",
			err:  "Missing required key: selector",
		},
		{
			name:  "empty selector",
			query: `{"selector":{}}`,
			db: func() *db {
				db := setupDB(t, nil)
				for _, id := range []string{"a", "c", "z", "q", "chicken"} {
					if _, err := db.Put(context.Background(), id, map[string]string{"value": id}); err != nil {
						t.Fatal(err)
					}
				}
				return db
			}(),
			expectedIDs: []string{"a", "c", "chicken", "q", "z"},
		},
		{
			name:  "simple selector",
			query: `{"selector":{"value":"chicken"}}`,
			db: func() *db {
				db := setupDB(t, nil)
				for _, id := range []string{"a", "c", "z", "q", "chicken"} {
					if _, err := db.Put(context.Background(), id, map[string]string{"value": id}); err != nil {
						t.Fatal(err)
					}
				}
				return db
			}(),
			expectedIDs: []string{"chicken"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := test.db
			if db == nil {
				db = setupDB(t, nil)
			}
			rows, err := db.Find(context.Background(), test.query)
			var msg string
			if err != nil {
				msg = err.Error()
			}
			if msg != test.err {
				t.Errorf("Unexpected error: %s", err)
			}
			if err != nil {
				return
			}
			checkRows(t, rows, test.expectedIDs, test.rowsErr)
		})
	}
}

// TestFindDoc is the same as Testfind, but assumes only a single result
// (ignores any others), and compares the entire document.
func TestFindDoc(t *testing.T) {
	type fdTest struct {
		name     string
		db       *db
		query    interface{}
		expected interface{}
	}
	tests := []fdTest{
		{
			name:  "simple selector",
			query: `{"selector":{}}`,
			db: func() *db {
				db := setupDB(t, nil)
				id := "chicken"
				if _, err := db.Put(context.Background(), id, map[string]string{"value": id}); err != nil {
					t.Fatal(err)
				}
				return db
			}(),
			expected: map[string]interface{}{
				"_id":   "chicken",
				"_rev":  "1-xxx",
				"value": "chicken",
			},
		},
		{
			name:  "fields",
			query: `{"selector":{}, "fields":["value","_rev"]}`,
			db: func() *db {
				db := setupDB(t, nil)
				if _, _, err := db.CreateDoc(context.Background(), map[string]string{"value": "foo"}); err != nil {
					t.Fatal(err)
				}
				return db
			}(),
			expected: map[string]interface{}{
				"value": "foo",
				"_rev":  "1-xxx",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := test.db
			if db == nil {
				db = setupDB(t, nil)
			}
			rows, err := db.Find(context.Background(), test.query)
			if err != nil {
				t.Fatal(err)
			}
			var row driver.Row
			if e := rows.Next(&row); e != nil {
				t.Fatal(e)
			}
			_ = rows.Close()
			var result map[string]interface{}
			if e := json.Unmarshal(row.Doc, &result); e != nil {
				t.Fatal(e)
			}
			parts := strings.Split(result["_rev"].(string), "-")
			result["_rev"] = parts[0] + "-xxx"
			if d := diff.AsJSON(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestResultWarning(t *testing.T) {
	rows := &findResults{}
	expected := "no matching index found, create an index to optimize query time"
	if w := rows.Warning(); w != expected {
		t.Errorf("Unexpected warning: %s", w)
	}
}

func TestFilterDoc(t *testing.T) {
	type fdTest struct {
		name     string
		rows     *findResults
		data     string
		expected string
		err      string
	}
	tests := []fdTest{
		{
			name:     "no filter",
			rows:     &findResults{},
			data:     `{"foo":"bar"}`,
			expected: `{"foo":"bar"}`,
		},
		{
			name:     "with filter",
			rows:     &findResults{fields: map[string]struct{}{"foo": struct{}{}}},
			data:     `{"foo":"bar", "baz":"qux"}`,
			expected: `{"foo":"bar"}`,
		},
		{
			name: "invalid json",
			rows: &findResults{fields: map[string]struct{}{"foo": struct{}{}}},
			data: `{"foo":"bar", "baz":"qux}`,
			err:  "unexpected end of JSON input",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.rows.filterDoc([]byte(test.data))
			var msg string
			if err != nil {
				msg = err.Error()
			}
			if msg != test.err {
				t.Errorf("Unexpected error: %s", msg)
			}
			if err != nil {
				return
			}
			if d := diff.JSON([]byte(test.expected), result); d != nil {
				t.Error(d)
			}
		})
	}
}

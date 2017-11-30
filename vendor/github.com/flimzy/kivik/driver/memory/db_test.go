package memory

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/flimzy/diff"
	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/driver"
)

func TestStats(t *testing.T) {
	type statTest struct {
		Name     string
		DBName   string
		Setup    func(driver.Client)
		Expected *driver.DBStats
		Error    string
	}
	tests := []statTest{
		{
			Name:   "NoDBs",
			DBName: "foo",
			Setup: func(c driver.Client) {
				if e := c.CreateDB(context.Background(), "foo", nil); e != nil {
					panic(e)
				}
			},
			Expected: &driver.DBStats{Name: "foo"},
		},
	}
	for _, test := range tests {
		func(test statTest) {
			t.Run(test.Name, func(t *testing.T) {
				c := setup(t, test.Setup)
				db, err := c.DB(context.Background(), test.DBName, nil)
				if err != nil {
					t.Fatal(err)
				}
				result, err := db.Stats(context.Background())
				var msg string
				if err != nil {
					msg = err.Error()
				}
				if msg != test.Error {
					t.Errorf("Unexpected error: %s", msg)
				}
				if err != nil {
					return
				}
				if d := diff.Interface(test.Expected, result); d != nil {
					t.Error(d)
				}
			})
		}(test)
	}
}

func setupDB(t *testing.T, s func(driver.DB)) *db {
	c := setup(t, nil)
	if err := c.CreateDB(context.Background(), "foo", nil); err != nil {
		t.Fatal(err)
	}
	d, err := c.DB(context.Background(), "foo", nil)
	if err != nil {
		t.Fatal(err)
	}
	if s != nil {
		s(d)
	}
	return d.(*db)
}

func TestPut(t *testing.T) {
	type putTest struct {
		Name     string
		DocID    string
		Doc      interface{}
		Setup    func() driver.DB
		Expected interface{}
		Status   int
		Error    string
	}
	tests := []putTest{
		{
			Name:   "LeadingUnderscoreInID",
			DocID:  "_badid",
			Doc:    map[string]string{"_id": "_badid", "foo": "bar"},
			Status: 400,
			Error:  "Only reserved document ids may start with underscore.",
		},
		{
			Name:     "MismatchedIDs",
			DocID:    "foo",
			Doc:      map[string]string{"_id": "bar"},
			Expected: map[string]string{"_id": "foo", "_rev": "1-xxx"},
		},
		{
			Name:     "Success",
			DocID:    "foo",
			Doc:      map[string]string{"_id": "foo", "foo": "bar"},
			Expected: map[string]string{"_id": "foo", "foo": "bar", "_rev": "1-xxx"},
		},
		{
			Name:  "Conflict",
			DocID: "foo",
			Doc:   map[string]string{"_id": "foo", "_rev": "bar"},
			Setup: func() driver.DB {
				db := setupDB(t, nil)
				db.Put(context.Background(), "foo", map[string]string{"_id": "foo"})
				return db
			},
			Status: 409,
			Error:  "document update conflict",
		},
		{
			Name:  "Unmarshalable",
			DocID: "foo",
			Doc: func() interface{} {
				return map[string]interface{}{
					"channel": make(chan int),
				}
			}(),
			Status: 400,
			Error:  "json: unsupported type: chan int",
		},
		{
			Name:   "InitialRev",
			DocID:  "foo",
			Doc:    map[string]string{"_id": "foo", "_rev": "bar"},
			Status: 409,
			Error:  "document update conflict",
		},
		func() putTest {
			db := setupDB(t, nil)
			rev, err := db.Put(context.Background(), "foo", map[string]string{"_id": "foo", "foo": "bar"})
			if err != nil {
				panic(err)
			}
			return putTest{
				Name:     "Update",
				DocID:    "foo",
				Setup:    func() driver.DB { return db },
				Doc:      map[string]string{"_id": "foo", "_rev": rev},
				Expected: map[string]string{"_id": "foo", "_rev": "2-xxx"},
			}
		}(),
		{
			Name:     "DesignDoc",
			DocID:    "_design/foo",
			Doc:      map[string]string{"foo": "bar"},
			Expected: map[string]string{"_id": "_design/foo", "foo": "bar", "_rev": "1-xxx"},
		},
		{
			Name:     "LocalDoc",
			DocID:    "_local/foo",
			Doc:      map[string]string{"foo": "bar"},
			Expected: map[string]string{"_id": "_local/foo", "foo": "bar", "_rev": "1-0"},
		},
		{
			Name:     "RecreateDeleted",
			DocID:    "foo",
			Doc:      map[string]string{"foo": "bar"},
			Expected: map[string]string{"_id": "foo", "foo": "bar", "_rev": "3-xxx"},
			Setup: func() driver.DB {
				db := setupDB(t, nil)
				rev, err := db.Put(context.Background(), "foo", map[string]string{"_id": "foo"})
				if err != nil {
					t.Fatal(err)
				}
				if _, e := db.Delete(context.Background(), "foo", rev); e != nil {
					t.Fatal(e)
				}
				return db
			},
		},
		{
			Name:     "LocalDoc",
			DocID:    "_local/foo",
			Doc:      map[string]string{"foo": "baz"},
			Expected: map[string]string{"_id": "_local/foo", "foo": "baz", "_rev": "1-0"},
			Setup: func() driver.DB {
				db := setupDB(t, nil)
				_, err := db.Put(context.Background(), "_local/foo", map[string]string{"foo": "bar"})
				if err != nil {
					t.Fatal(err)
				}
				return db
			},
		},
		{
			Name:  "WithAttachments",
			DocID: "duck",
			Doc: map[string]interface{}{
				"_id":   "duck",
				"value": "quack",
				"_attachments": []map[string]interface{}{
					{"foo.css": map[string]string{
						"content_type": "text/css",
						"data":         "LyogYW4gZW1wdHkgQ1NTIGZpbGUgKi8=",
					}},
				},
			},
			Expected: map[string]string{
				"_id":   "duck",
				"_rev":  "1-xxx",
				"value": "quack",
			},
		},
	}
	for _, test := range tests {
		func(test putTest) {
			t.Run(test.Name, func(t *testing.T) {
				t.Parallel()
				var db driver.DB
				if test.Setup != nil {
					db = test.Setup()
				} else {
					db = setupDB(t, nil)
				}
				var msg string
				var status int
				if _, err := db.Put(context.Background(), test.DocID, test.Doc); err != nil {
					msg = err.Error()
					status = kivik.StatusCode(err)
				}
				if msg != test.Error {
					t.Errorf("Unexpected error: %s", msg)
				}
				if status != test.Status {
					t.Errorf("Unexpected status code: %d", status)
				}
				if msg != "" {
					return
				}
				resultJSON, err := db.Get(context.Background(), test.DocID, nil)
				if err != nil {
					t.Fatal(err)
				}
				var result map[string]interface{}
				if e := json.Unmarshal(resultJSON, &result); e != nil {
					t.Fatal(e)
				}
				if !strings.HasPrefix(test.DocID, "_local/") {
					if rev, ok := result["_rev"].(string); ok {
						parts := strings.SplitN(rev, "-", 2)
						result["_rev"] = parts[0] + "-xxx"
					}
				}
				if d := diff.AsJSON(test.Expected, result); d != nil {
					t.Error(d)
				}
			})
		}(test)
	}
}

func TestGet(t *testing.T) {
	type getTest struct {
		Name     string
		ID       string
		Opts     map[string]interface{}
		DB       driver.DB
		Status   int
		Error    string
		Expected interface{}
	}
	tests := []getTest{
		{
			Name:   "NotFound",
			ID:     "foo",
			Status: 404,
			Error:  "missing",
		},
		{
			Name: "ExistingDoc",
			ID:   "foo",
			DB: func() driver.DB {
				db := setupDB(t, nil)
				if _, err := db.Put(context.Background(), "foo", map[string]string{"_id": "foo", "foo": "bar"}); err != nil {
					panic(err)
				}
				return db
			}(),
			Expected: map[string]string{"_id": "foo", "foo": "bar"},
		},
		func() getTest {
			db := setupDB(t, nil)
			rev, err := db.Put(context.Background(), "foo", map[string]string{"_id": "foo", "foo": "Bar"})
			if err != nil {
				panic(err)
			}
			return getTest{
				Name: "SpecificRev",
				ID:   "foo",
				DB:   db,
				Opts: map[string]interface{}{
					"rev": rev,
				},
				Expected: map[string]string{"_id": "foo", "foo": "Bar"},
			}
		}(),
		func() getTest {
			db := setupDB(t, nil)
			rev, err := db.Put(context.Background(), "foo", map[string]string{"_id": "foo", "foo": "Bar"})
			if err != nil {
				panic(err)
			}
			_, err = db.Put(context.Background(), "foo", map[string]string{"_id": "foo", "foo": "baz", "_rev": rev})
			if err != nil {
				panic(err)
			}
			return getTest{
				Name: "OldRev",
				ID:   "foo",
				DB:   db,
				Opts: map[string]interface{}{
					"rev": rev,
				},
				Expected: map[string]string{"_id": "foo", "foo": "Bar"},
			}
		}(),
		{
			Name: "MissingRev",
			ID:   "foo",
			Opts: map[string]interface{}{
				"rev": "1-4c6114c65e295552ab1019e2b046b10e",
			},
			DB: func() driver.DB {
				db := setupDB(t, nil)
				_, err := db.Put(context.Background(), "foo", map[string]string{"_id": "foo", "foo": "Bar"})
				if err != nil {
					panic(err)
				}
				return db
			}(),
			Status: 404,
			Error:  "missing",
		},
		func() getTest {
			db := setupDB(t, nil)
			rev, err := db.Put(context.Background(), "foo", map[string]string{"_id": "foo"})
			if err != nil {
				panic(err)
			}
			if _, e := db.Delete(context.Background(), "foo", rev); e != nil {
				panic(e)
			}
			return getTest{
				Name:   "DeletedDoc",
				ID:     "foo",
				DB:     db,
				Status: 404,
				Error:  "missing",
			}
		}(),
	}
	for _, test := range tests {
		func(test getTest) {
			t.Run(test.Name, func(t *testing.T) {
				t.Parallel()
				db := test.DB
				if db == nil {
					db = setupDB(t, nil)
				}
				var msg string
				var status int
				docJSON, err := db.Get(context.Background(), test.ID, test.Opts)
				if err != nil {
					msg = err.Error()
					status = kivik.StatusCode(err)
				}
				if msg != test.Error {
					t.Errorf("Unexpected error: %s", msg)
				}
				if status != test.Status {
					t.Errorf("Unexpected status: %d", status)
				}
				if err != nil {
					return
				}
				var result map[string]interface{}
				if err := json.Unmarshal(docJSON, &result); err != nil {
					t.Fatal(err)
				}
				if result != nil {
					delete(result, "_rev")
				}
				if d := diff.AsJSON(test.Expected, result); d != nil {
					t.Error(d)
				}
			})
		}(test)
	}
}

func TestDeleteDoc(t *testing.T) {
	type delTest struct {
		Name   string
		ID     string
		Rev    string
		DB     driver.DB
		Status int
		Error  string
	}
	tests := []delTest{
		{
			Name:   "NonExistingDoc",
			ID:     "foo",
			Rev:    "1-4c6114c65e295552ab1019e2b046b10e",
			Status: 404,
			Error:  "missing",
		},
		func() delTest {
			db := setupDB(t, nil)
			rev, err := db.Put(context.Background(), "foo", map[string]string{"_id": "foo"})
			if err != nil {
				panic(err)
			}
			return delTest{
				Name: "Success",
				ID:   "foo",
				DB:   db,
				Rev:  rev,
			}
		}(),
		{
			Name:   "InvalidRevFormat",
			ID:     "foo",
			Rev:    "invalid rev format",
			Status: 400,
			Error:  "Invalid rev format",
		},
		{
			Name: "LocalNoRev",
			ID:   "_local/foo",
			Rev:  "",
			DB: func() driver.DB {
				db := setupDB(t, nil)
				if _, err := db.Put(context.Background(), "_local/foo", map[string]string{"foo": "bar"}); err != nil {
					panic(err)
				}
				return db
			}(),
		},
		{
			Name: "LocalWithRev",
			ID:   "_local/foo",
			Rev:  "0-1",
			DB: func() driver.DB {
				db := setupDB(t, nil)
				if _, err := db.Put(context.Background(), "_local/foo", map[string]string{"foo": "bar"}); err != nil {
					panic(err)
				}
				return db
			}(),
		},
	}
	for _, test := range tests {
		func(test delTest) {
			t.Run(test.Name, func(t *testing.T) {
				t.Parallel()
				db := test.DB
				if db == nil {
					db = setupDB(t, nil)
				}
				rev, err := db.Delete(context.Background(), test.ID, test.Rev)
				var msg string
				var status int
				if err != nil {
					msg = err.Error()
					status = kivik.StatusCode(err)
				}
				if msg != test.Error {
					t.Errorf("Unexpected error: %s", msg)
				}
				if status != test.Status {
					t.Errorf("Unexpected status: %d", status)
				}
				if err != nil {
					return
				}
				docJSON, err := db.Get(context.Background(), test.ID, map[string]interface{}{"rev": rev})
				if err != nil {
					t.Fatal(err)
				}
				var doc interface{}
				if e := json.Unmarshal(docJSON, &doc); e != nil {
					t.Fatal(e)
				}
				expected := map[string]interface{}{
					"_id":      test.ID,
					"_rev":     rev,
					"_deleted": true,
				}
				if d := diff.AsJSON(expected, doc); d != nil {
					t.Error(d)
				}

			})
		}(test)
	}
}

func TestCreateDoc(t *testing.T) {
	type cdTest struct {
		Name     string
		Doc      interface{}
		Expected map[string]interface{}
		Error    string
	}
	tests := []cdTest{
		{
			Name: "SimpleDoc",
			Doc: map[string]interface{}{
				"foo": "bar",
			},
			Expected: map[string]interface{}{
				"_rev": "1-xxx",
				"foo":  "bar",
			},
		},
	}
	for _, test := range tests {
		func(test cdTest) {
			t.Run(test.Name, func(t *testing.T) {
				db := setupDB(t, nil)
				docID, _, err := db.CreateDoc(context.Background(), test.Doc)
				var msg string
				if err != nil {
					msg = err.Error()
				}
				if msg != test.Error {
					t.Errorf("Unexpected error: %s", msg)
				}
				if err != nil {
					return
				}
				row, err := db.Get(context.Background(), docID, nil)
				if err != nil {
					t.Fatal(err)
				}
				var result map[string]interface{}
				if e := json.Unmarshal(row, &result); e != nil {
					t.Fatal(e)
				}
				if result["_id"].(string) != docID {
					t.Errorf("Unexpected id. %s != %s", result["_id"].(string), docID)
				}
				delete(result, "_id")
				if rev, ok := result["_rev"].(string); ok {
					parts := strings.SplitN(rev, "-", 2)
					result["_rev"] = parts[0] + "-xxx"
				}
				if d := diff.Interface(test.Expected, result); d != nil {
					t.Error(d)
				}
			})
		}(test)
	}
}

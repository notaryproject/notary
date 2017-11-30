package couchdb

import (
	"context"
	"io"
	"net/url"
	"testing"

	"github.com/flimzy/diff"
	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/driver"
)

func TestAllDocs(t *testing.T) {
	client := getClient(t)
	db, err := client.DB(context.Background(), "_users", kivik.Options{"force_commit": true})
	if err != nil {
		t.Fatalf("Failed to connect to db: %s", err)
	}
	rows, err := db.AllDocs(context.Background(), map[string]interface{}{
		"include_docs": true,
	})
	if err != nil {
		t.Fatalf("Failed: %s", err)
	}

	for {
		err := rows.Next(&driver.Row{})
		if err != nil {
			if err != io.EOF {
				t.Fatalf("Iteration failed: %s", err)
			}
			break
		}
	}
}

func TestDBInfo(t *testing.T) {
	client := getClient(t)
	db, err := client.DB(context.Background(), "_users", kivik.Options{"force_commit": true})
	if err != nil {
		t.Fatalf("Failed to connect to db: %s", err)
	}
	info, err := db.Stats(context.Background())
	if err != nil {
		t.Fatalf("Failed: %s", err)
	}
	if info.Name != "_users" {
		t.Errorf("Unexpected name %s", info.Name)
	}
}

func TestOptionsToParams(t *testing.T) {
	type otpTest struct {
		Name     string
		Input    map[string]interface{}
		Expected url.Values
		Error    string
	}
	tests := []otpTest{
		{
			Name:     "String",
			Input:    map[string]interface{}{"foo": "bar"},
			Expected: map[string][]string{"foo": {"bar"}},
		},
		{
			Name:     "StringSlice",
			Input:    map[string]interface{}{"foo": []string{"bar", "baz"}},
			Expected: map[string][]string{"foo": {"bar", "baz"}},
		},
		{
			Name:     "Bool",
			Input:    map[string]interface{}{"foo": true},
			Expected: map[string][]string{"foo": {"true"}},
		},
		{
			Name:     "Int",
			Input:    map[string]interface{}{"foo": 123},
			Expected: map[string][]string{"foo": {"123"}},
		},
		{
			Name:  "Error",
			Input: map[string]interface{}{"foo": []byte("foo")},
			Error: "cannot convert type []uint8 to []string",
		},
	}
	for _, test := range tests {
		func(test otpTest) {
			t.Run(test.Name, func(t *testing.T) {
				params, err := optionsToParams(test.Input)
				var msg string
				if err != nil {
					msg = err.Error()
				}
				if msg != test.Error {
					t.Errorf("Error\n\tExpected: %s\n\t  Actual: %s\n", test.Error, msg)
				}
				if d := diff.Interface(test.Expected, params); d != nil {
					t.Errorf("Params not as expected:\n%s\n", d)
				}
			})
		}(test)
	}
}

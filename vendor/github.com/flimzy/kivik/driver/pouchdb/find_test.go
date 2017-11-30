package pouchdb

import (
	"strconv"
	"testing"

	"github.com/flimzy/diff"
	"github.com/gopherjs/gopherjs/js"
)

func TestBuildIndex(t *testing.T) {
	tests := []struct {
		Ddoc     string
		Name     string
		Index    interface{}
		Expected string
	}{
		{Expected: `{}`},
		{Index: `{"fields":["foo"]}`, Expected: `{"fields":["foo"]}`},
		{Index: `{"fields":["foo"]}`, Name: "test", Expected: `{"fields":["foo"],"name":"test"}`},
		{Index: `{"fields":["foo"]}`, Name: "test", Ddoc: "_foo", Expected: `{"fields":["foo"],"name":"test","ddoc":"_foo"}`},
	}
	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			result, err := buildIndex(test.Ddoc, test.Name, test.Index)
			if err != nil {
				t.Errorf("Build Index failed: %s", err)
			}
			r := js.Global.Get("JSON").Call("stringify", result).String()
			if d := diff.JSON([]byte(test.Expected), []byte(r)); d != nil {
				t.Errorf("BuildIndex result differs:\n%s\n", d)
			}
		})
	}
}

package util

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestToJSON(t *testing.T) {
	type tjTest struct {
		Name     string
		Input    interface{}
		Expected string
	}
	tests := []tjTest{
		{
			Name:     "Null",
			Expected: "null",
		},
		{
			Name:     "String",
			Input:    `{"foo":"bar"}`,
			Expected: `{"foo":"bar"}`,
		},
		{
			Name:     "ByteSlice",
			Input:    []byte(`{"foo":"bar"}`),
			Expected: `{"foo":"bar"}`,
		},
		{
			Name:     "RawMessage",
			Input:    json.RawMessage(`{"foo":"bar"}`),
			Expected: `{"foo":"bar"}`,
		},
		{
			Name:     "Interface",
			Input:    map[string]string{"foo": "bar"},
			Expected: `{"foo":"bar"}`,
		},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			r, err := ToJSON(test.Input)
			if err != nil {
				t.Fatalf("jsonify failed: %s", err)
			}
			buf := &bytes.Buffer{}
			buf.ReadFrom(r)
			result := strings.TrimSpace(buf.String())
			if result != test.Expected {
				t.Errorf("Expected: `%s`\n  Actual: `%s`", test.Expected, result)
			}
		})
	}
}

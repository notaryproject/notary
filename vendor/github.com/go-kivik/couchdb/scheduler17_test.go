// +build go1.7,!go1.8

package couchdb

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/flimzy/diff"
	"github.com/flimzy/testy"
)

func TestNullTimeUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Time
		err      string
	}{
		{
			name:     "valid time",
			input:    `"2017-11-17T19:56:09Z"`,
			expected: parseTime(t, "2017-11-17T19:56:09Z"),
		},
		{
			name:     "null",
			input:    `null`,
			expected: time.Time{},
		},
		{
			name:  "invalid json",
			input: `invalid json`,
			err:   "invalid character 'i' looking for beginning of value",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var result nullTime
			err := json.Unmarshal([]byte(test.input), &result)
			testy.Error(t, test.err, err)
			if d := diff.Interface(test.expected, time.Time(result)); d != nil {
				t.Error(d)
			}
		})
	}
}

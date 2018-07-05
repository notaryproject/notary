package driver

import (
	"encoding/json"
	"testing"

	"github.com/flimzy/diff"
	"github.com/flimzy/testy"
)

func TestChangesUnmarshal(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected ChangedRevs
		err      string
	}{
		{
			name:  "invalid JSON",
			input: `{"foo":"bar"}`,
			err:   `json: cannot unmarshal object into Go value of type []struct { Rev string "json:\"rev\"" }`,
		},
		{
			name: "success",
			input: `[
                    {"rev": "6-460637e73a6288cb24d532bf91f32969"},
                    {"rev": "5-eeaa298781f60b7bcae0c91bdedd1b87"}
                ]`,
			expected: ChangedRevs{"6-460637e73a6288cb24d532bf91f32969", "5-eeaa298781f60b7bcae0c91bdedd1b87"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var result ChangedRevs
			err := json.Unmarshal([]byte(test.input), &result)
			testy.Error(t, test.err, err)
			if d := diff.Interface(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}

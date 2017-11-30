package driver

import (
	"encoding/json"
	"testing"

	"github.com/flimzy/diff"
)

func TestChangesUnmarshal(t *testing.T) {
	input := `[
                {"rev": "6-460637e73a6288cb24d532bf91f32969"},
                {"rev": "5-eeaa298781f60b7bcae0c91bdedd1b87"}
            ]`
	var changes ChangedRevs
	if err := json.Unmarshal([]byte(input), &changes); err != nil {
		t.Fatalf("unmarshal failed: %s", err)
	}
	if len(changes) != 2 {
		t.Errorf("Expected 2 results, got %d", len(changes))
	}
	expected := []string{"6-460637e73a6288cb24d532bf91f32969", "5-eeaa298781f60b7bcae0c91bdedd1b87"}
	if d := diff.AsJSON(expected, changes); d != nil {
		t.Errorf("Results differ from expected:\n%s\n", d)
	}
}

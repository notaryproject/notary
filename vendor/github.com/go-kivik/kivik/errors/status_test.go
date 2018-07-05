package errors

import "testing"

func TestStatusText(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		expected string
	}{
		{
			name:     "Network Error",
			code:     601,
			expected: "network_error",
		},
		{
			name:     "undefined",
			code:     999,
			expected: "unknown",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := statusText(test.code)
			if test.expected != result {
				t.Errorf("Unexpected result: %s", result)
			}
		})
	}
}

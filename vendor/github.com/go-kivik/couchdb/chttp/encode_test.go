package chttp

import "testing"

func TestEncodeDocID(t *testing.T) {
	tests := []struct {
		Input    string
		Expected string
	}{
		{Input: "foo", Expected: "foo"},
		{Input: "foo/bar", Expected: "foo%2Fbar"},
		{Input: "_design/foo", Expected: "_design/foo"},
		{Input: "_design/foo/bar", Expected: "_design/foo%2Fbar"},
		{Input: "foo@bar.com", Expected: "foo%40bar.com"},
		{Input: "foo+bar@baz.com", Expected: "foo%2Bbar%40baz.com"},
		{Input: "Is this a valid ID?", Expected: "Is+this+a+valid+ID%3F"},
		{Input: "nón-English-çharacters", Expected: "n%C3%B3n-English-%C3%A7haracters"},
		{Input: "foo+bar & páces?!*,", Expected: "foo%2Bbar+%26+p%C3%A1ces%3F%21%2A%2C"},
	}
	for _, test := range tests {
		result := EncodeDocID(test.Input)
		if result != test.Expected {
			t.Errorf("Unexpected encoded DocID from %s\n\tExpected: %s\n\t  Actual: %s\n", test.Input, test.Expected, result)
		}
	}
}

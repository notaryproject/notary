package couchdb

import (
	"encoding/json"
	"testing"
	"time"
)

func TestStateTime(t *testing.T) {
	type stTest struct {
		Name     string
		Input    string
		Error    string
		Expected string
	}
	tests := []stTest{
		{
			Name:     "Blank",
			Error:    "unexpected end of JSON input",
			Expected: "0001-01-01 00:00:00 +0000",
		},
		{
			Name:     "ValidRFC3339",
			Input:    `"2011-02-17T20:22:02+01:00"`,
			Expected: "2011-02-17 20:22:02 +0100",
		},
		{
			Name:     "ValidUnixTimestamp",
			Input:    "1492543959",
			Expected: "2017-04-18 19:32:39 +0000",
		},
		{
			Name:     "InvalidInput",
			Input:    `"foo"`,
			Error:    `kivik: '"foo"' does not appear to be a valid timestamp`,
			Expected: "0001-01-01 00:00:00 +0000",
		},
	}
	for _, test := range tests {
		func(test stTest) {
			t.Run(test.Name, func(t *testing.T) {
				var result replicationStateTime
				var errMsg string
				if err := json.Unmarshal([]byte(test.Input), &result); err != nil {
					errMsg = err.Error()
				}
				if errMsg != test.Error {
					t.Errorf("Error\nExpected: %s\n  Actual: %s\n", test.Error, errMsg)
				}
				if r := time.Time(result).Format("2006-01-02 15:04:05 -0700"); r != test.Expected {
					t.Errorf("Result\nExpected: %s\n  Actual: %s\n", test.Expected, r)
				}
			})
		}(test)
	}
}

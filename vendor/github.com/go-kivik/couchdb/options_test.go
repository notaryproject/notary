package couchdb

import (
	"testing"

	"github.com/flimzy/kivik"
	"github.com/flimzy/testy"
)

func TestFullCommit(t *testing.T) {
	tests := []struct {
		name     string
		def      bool
		input    map[string]interface{}
		expected bool
		status   int
		err      string
	}{
		{
			name:     "legacy",
			input:    map[string]interface{}{optionForceCommit: true},
			expected: true,
		},
		{
			name:   "legacy error",
			input:  map[string]interface{}{optionForceCommit: 123},
			status: kivik.StatusBadRequest,
			err:    "kivik: option 'force_commit' must be bool, not int",
		},
		{
			name:     "new",
			input:    map[string]interface{}{OptionFullCommit: true},
			expected: true,
		},
		{
			name:   "new error",
			input:  map[string]interface{}{OptionFullCommit: 123},
			status: kivik.StatusBadRequest,
			err:    "kivik: option 'X-Couch-Full-Commit' must be bool, not int",
		},
		{
			name: "new priority over old",
			input: map[string]interface{}{
				OptionFullCommit:  false,
				optionForceCommit: true,
			},
			expected: false,
		},
		{
			name:     "none",
			input:    nil,
			expected: false,
		},
		{
			name:     "true default, no option",
			def:      true,
			input:    nil,
			expected: true,
		},
		{
			name:     "override default",
			def:      true,
			input:    map[string]interface{}{OptionFullCommit: false},
			expected: false,
		},
		{
			name:     "default and option agree",
			def:      true,
			input:    map[string]interface{}{OptionFullCommit: true},
			expected: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := fullCommit(test.def, test.input)
			testy.StatusError(t, test.err, test.status, err)
			if result != test.expected {
				t.Errorf("Unexpected result: %v", result)
			}
			if _, ok := test.input[OptionFullCommit]; ok {
				t.Errorf("%s still set in options", OptionFullCommit)
			}
			if _, ok := test.input[optionForceCommit]; ok {
				t.Errorf("%s still set in options", optionForceCommit)
			}
		})
	}
}

func TestIfNoneMatch(t *testing.T) {
	tests := []struct {
		name     string
		opts     map[string]interface{}
		expected string
		status   int
		err      string
	}{
		{
			name:     "nil",
			opts:     nil,
			expected: "",
		},
		{
			name:     "inm not set",
			opts:     map[string]interface{}{"foo": "bar"},
			expected: "",
		},
		{
			name:   "wrong type",
			opts:   map[string]interface{}{OptionIfNoneMatch: 123},
			status: kivik.StatusBadRequest,
			err:    "kivik: option 'If-None-Match' must be string, not int",
		},
		{
			name:     "valid",
			opts:     map[string]interface{}{OptionIfNoneMatch: "foo"},
			expected: `"foo"`,
		},
		{
			name:     "valid, pre-quoted",
			opts:     map[string]interface{}{OptionIfNoneMatch: `"foo"`},
			expected: `"foo"`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := ifNoneMatch(test.opts)
			testy.StatusError(t, test.err, test.status, err)
			if result != test.expected {
				t.Errorf("Unexpected result: %s", result)
			}
			if _, ok := test.opts[OptionIfNoneMatch]; ok {
				t.Errorf("%s still set in options", OptionIfNoneMatch)
			}
		})
	}
}

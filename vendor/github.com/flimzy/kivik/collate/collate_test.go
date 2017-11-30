package collate

import (
	"math"
	"testing"
)

func TestComparisonString(t *testing.T) {
	type csTest struct {
		name     string
		c        comparison
		expected string
	}
	tests := []csTest{
		{
			name:     "less than",
			c:        lt,
			expected: "less than",
		},
		{
			name:     "greater than",
			c:        gt,
			expected: "greater than",
		},
		{
			name:     "equal",
			c:        eq,
			expected: "equal",
		},
		{
			name:     "-100",
			c:        -100,
			expected: "less than",
		},
		{
			name:     "100",
			c:        100,
			expected: "greater than",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := test.c.String()
			if result != test.expected {
				t.Errorf("Unexpected result: %s", result)
			}
		})
	}
}

func TestCouchTypeOf(t *testing.T) {
	type ctoTest struct {
		name     string
		input    interface{}
		expected couchType
		recovery string
	}
	tests := []ctoTest{
		{
			name:     "nil",
			expected: couchNull,
		},
		{
			name:     "int",
			input:    int(3),
			expected: couchNumber,
		},
		{
			name:     "+inifinity",
			input:    math.Inf(1),
			expected: couchNull,
		},
		{
			name:     "-infinity",
			input:    math.Inf(-1),
			expected: couchNull,
		},
		{
			name:     "NaN",
			input:    math.NaN(),
			expected: couchNull,
		},
		{
			name:     "float64",
			input:    float64(3.3),
			expected: couchNumber,
		},
		{
			name:     "string",
			input:    "foo",
			expected: couchString,
		},
		{
			name:     "slice",
			input:    []int{1, 2, 3},
			expected: couchArray,
		},
		{
			name:     "array",
			input:    [3]int{1, 2, 3},
			expected: couchArray,
		},
		{
			name:     "map",
			input:    map[string]interface{}{"foo": "bar"},
			expected: couchObject,
		},
		{
			name:     "struct",
			input:    struct{ Foo string }{"foo"},
			recovery: "unknown type",
		},
		{
			name:     "bool",
			input:    true,
			expected: couchBool,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := func() (recovery string) {
				defer func() {
					if r := recover(); r != nil {
						recovery = r.(string)
					}
				}()
				result := couchTypeOf(test.input)
				if result != test.expected {
					t.Errorf("Unexpected type: %d", result)
				}
				return
			}()
			if r != test.recovery {
				t.Errorf("Unexpected recovery: %s", r)
			}
		})
	}
}

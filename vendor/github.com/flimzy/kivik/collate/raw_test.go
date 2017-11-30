package collate

import (
	"fmt"
	"math"
	"testing"
)

func TestRawCmp(t *testing.T) {
	c := &Raw{}
	type cmpTest struct {
		name     string
		i, j     interface{}
		expected comparison
	}
	tests := []cmpTest{
		{
			name:     "both nil",
			expected: eq,
		},
		{
			name:     "nil vs number",
			j:        123,
			expected: lt,
		},
		{
			name:     "number vs nil",
			i:        123,
			expected: gt,
		},
		{
			name:     "Nan vs nil",
			i:        math.NaN(),
			expected: eq,
		},
		{
			name: "true vs false",
			i:    true, j: false,
			expected: gt,
		},
		{
			name: "false vs true",
			i:    false, j: true,
			expected: lt,
		},
		{
			name: "true vs true",
			i:    true, j: true,
			expected: eq,
		},
		{
			name: "1 vs 2",
			i:    1, j: 2,
			expected: lt,
		},
		{
			name: "123 vs 123.005",
			i:    int(123), j: float32(123.005),
			expected: lt,
		},
		{
			name: "a vs b",
			i:    "a", j: "b",
			expected: lt,
		},
		{
			name: "aaaa vs a",
			i:    "aaaa", j: "a",
			expected: gt,
		},
		{
			name: "array vs number",
			i:    []int{1, 2, 3}, j: 123,
			expected: gt,
		},
		{
			name: "2 arrays",
			i:    []int{1, 2, 3}, j: []int{2, 3},
			expected: lt,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := c.cmp(test.i, test.j)
			if result != test.expected {
				t.Errorf("Unexpected result: %s", result)
			}
		})
	}
}

func TestRawEqualityOperators(t *testing.T) {
	c := &Raw{}
	type eqTest struct {
		name string
		i, j interface{}
		lt   bool
		lte  bool
		eq   bool
		gt   bool
		gte  bool
	}
	tests := []eqTest{
		{
			name: "nil/nil",
			lte:  true, eq: true, gte: true,
		},
		{
			name: "nil/number",
			j:    123,
			lt:   true, lte: true,
		},
		{
			name: "number/nil",
			i:    123,
			gt:   true, gte: true,
		},
		{
			name: "int vs float",
			i:    int(123),
			j:    float32(400),
			lt:   true, lte: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if x := c.LT(test.i, test.j); x != test.lt {
				t.Errorf("LT returned %t", x)
			}
			if x := c.LTE(test.i, test.j); x != test.lte {
				t.Errorf("LTE returned %t", x)
			}
			if x := c.Eq(test.i, test.j); x != test.eq {
				t.Errorf("Eq returned %t", x)
			}
			if x := c.GT(test.i, test.j); x != test.gt {
				t.Errorf("GT returned %t", x)
			}
			if x := c.GTE(test.i, test.j); x != test.gte {
				t.Errorf("GTE returned %t", x)
			}
		})
	}
}

func TestNumberCmp(t *testing.T) {
	type ncTest struct {
		name     string
		i, j     interface{}
		expected comparison
	}
	tests := []ncTest{
		{
			name: "int(1) vs int(2)",
			i:    1, j: 2,
			expected: lt,
		},
		{
			name: "float32(1.12) vs int64(3)",
			i:    float32(1.12), j: int64(3),
			expected: lt,
		},
		{
			name: "int8(12) vs int16(2)",
			i:    int8(12), j: int16(2),
			expected: gt,
		},
		{
			name: "int32(1) vs float64(0.0000000001)",
			i:    int32(1), j: float64(0.0000000001),
			expected: gt,
		},
		{
			name: "uint(8) vs int(-98)",
			i:    uint(8), j: int(-98),
			expected: gt,
		},
		{
			name: "uint(1) vs uint16(1)",
			i:    uint(1), j: uint16(1),
			expected: eq,
		},
		{
			name: "uint8(1) vs uint32(7)",
			i:    uint8(1), j: uint32(7),
			expected: lt,
		},
		{
			name: "uint64(3) vs int(-10)",
			i:    uint64(3), j: int(-10),
			expected: gt,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := numberCmp(test.i, test.j)
			if result != test.expected {
				t.Errorf("Unexpected result: %s", result)
			}
		})
	}
}

func TestToFloat(t *testing.T) {
	type tfTest struct {
		i        interface{}
		expected float64
	}
	tests := []tfTest{
		{i: int(1234), expected: 1234},
		{i: int8(-60), expected: -60},
		{i: int16(-9), expected: -9},
		{i: int32(32), expected: 32},
		{i: int64(64), expected: 64},
		{i: uint(1234), expected: 1234},
		{i: uint8(100), expected: 100},
		{i: uint16(99), expected: 99},
		{i: uint32(32), expected: 32},
		{i: uint64(64), expected: 64},
		{i: float32(123), expected: 123},
		{i: float64(0.00009), expected: 0.00009},
		{i: "foo", expected: math.NaN()},
	}
	for _, test := range tests {
		t.Run(fmt.Sprintf("%T(%v)", test.i, test.i), func(t *testing.T) {
			result := toFloat(test.i)
			if result != test.expected && !(math.IsNaN(result) && math.IsNaN(test.expected)) {
				t.Errorf("Unexpected result: %v", result)
			}
		})
	}
}

func TestRawArrayCmp(t *testing.T) {
	c := &Raw{}
	type acTest struct {
		name     string
		i, j     interface{}
		expected comparison
	}
	tests := []acTest{
		{
			name: "empty slices",
			i:    []int{}, j: []int{},
			expected: eq,
		},
		{
			name: "1 item vs empty",
			i:    []int{1}, j: []int{},
			expected: gt,
		},
		{
			name: "empty vs 3 items",
			i:    []int{}, j: []int{1, 2, 3},
			expected: lt,
		},
		{
			name: "[1,2] vs [2,3]",
			i:    []int{1, 2}, j: []int{2, 3},
			expected: lt,
		},
		{
			name: "longer vs shorter",
			i:    []int{1, 2, 3}, j: []int{1, 2},
			expected: gt,
		},
		{
			name: "numbers vs strings",
			i:    []int{1, 2, 3}, j: []string{"foo", "bar"},
			expected: lt,
		},
		{
			name: "intermingled",
			i:    []interface{}{1, "foo"}, j: []interface{}{"foo", 1},
			expected: lt,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := c.arrayCmp(test.i, test.j)
			if result != test.expected {
				t.Errorf("Unexpected result: %s", result)
			}
		})
	}
}

func TestRawObjectCmp(t *testing.T) {
	c := &Raw{}
	type ocTest struct {
		name     string
		i, j     interface{}
		expected comparison
	}
	tests := []ocTest{
		{
			name: "empty objects",
			i:    map[string]interface{}{}, j: map[string]interface{}{},
			expected: eq,
		},
		{
			name: "different keys",
			i:    map[string]interface{}{"a": 1}, j: map[string]interface{}{"b": 1},
			expected: lt,
		},
		{
			name: "different values",
			i:    map[string]interface{}{"a": 2}, j: map[string]interface{}{"a": 1},
			expected: gt,
		},
		{
			name: "equal values",
			i:    map[string]interface{}{"a": 1}, j: map[string]interface{}{"a": 1},
			expected: eq,
		},
		{
			name: "longer vs shorter",
			i:    map[string]interface{}{"a": 1, "b": 100}, j: map[string]interface{}{"a": 1},
			expected: gt,
		},
		{
			name: "shorter vs",
			i:    map[string]interface{}{"a": 1}, j: map[string]interface{}{"a": 1, "b": 100},
			expected: lt,
		},
		{
			name: "string vs number",
			i:    map[string]interface{}{"a": "foo"}, j: map[string]interface{}{"a": 1},
			expected: gt,
		},
		{
			name: "nested",
			i: map[string]interface{}{
				"a": map[string]interface{}{"b": 1},
			},
			j: map[string]interface{}{
				"a": map[string]interface{}{"b": 2},
			},
			expected: lt,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := c.objectCmp(test.i, test.j)
			if result != test.expected {
				t.Errorf("Unexpected result: %s", result)
			}
		})
	}
}

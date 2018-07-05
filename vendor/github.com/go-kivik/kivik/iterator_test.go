package kivik

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/flimzy/diff"
	"github.com/flimzy/testy"
)

type TestFeed struct {
	max      int64
	i        int64
	closeErr error
}

var _ iterator = &TestFeed{}

func (f *TestFeed) Close() error { return f.closeErr }
func (f *TestFeed) Next(ifce interface{}) error {
	i, ok := ifce.(*int64)
	if ok {
		*i = f.i
		f.i++
		if f.i > f.max {
			return io.EOF
		}
		time.Sleep(5 * time.Millisecond)
		return nil
	}
	panic(fmt.Sprintf("unknown type: %T", ifce))
}

func TestIterator(t *testing.T) {
	iter := newIterator(context.Background(), &TestFeed{max: 10}, func() interface{} { var i int64; return &i }())
	expected := []int64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	result := []int64{}
	for iter.Next() {
		val, ok := iter.curVal.(*int64)
		if !ok {
			panic("Unexpected type")
		}
		result = append(result, *val)
	}
	if err := iter.Err(); err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if d := diff.AsJSON(expected, result); d != nil {
		t.Errorf("Unexpected result:\n%s\n", d)
	}
}

func TestCancelledIterator(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	iter := newIterator(ctx, &TestFeed{max: 10000}, func() interface{} { var i int64; return &i }())
	for iter.Next() {
	}
	if err := iter.Err(); err.Error() != "context deadline exceeded" {
		t.Errorf("Unexpected error: %s", err)
	}
}

func TestIteratorScan(t *testing.T) {
	type Test struct {
		name     string
		dst      interface{}
		input    json.RawMessage
		expected interface{}
		status   int
		err      string
	}
	tests := []Test{
		{
			name:   "non-pointer",
			dst:    map[string]string{},
			input:  []byte(`{"foo":123.4}`),
			status: StatusBadRequest,
			err:    "kivik: destination is not a pointer",
		},
		func() Test {
			dst := map[string]interface{}{}
			expected := map[string]interface{}{"foo": 123.4}
			return Test{
				name:     "standard unmarshal",
				dst:      &dst,
				input:    []byte(`{"foo":123.4}`),
				expected: &expected,
			}
		}(),
		func() Test {
			dst := map[string]interface{}{}
			return Test{
				name:   "invalid JSON",
				dst:    &dst,
				input:  []byte(`invalid JSON`),
				status: StatusBadResponse,
				err:    "invalid character 'i' looking for beginning of value",
			}
		}(),
		func() Test {
			var dst *json.RawMessage
			return Test{
				name:   "nil *json.RawMessage",
				dst:    dst,
				input:  []byte(`{"foo":123.4}`),
				status: StatusBadRequest,
				err:    "kivik: destination pointer is nil",
			}
		}(),
		func() Test {
			var dst *[]byte
			return Test{
				name:   "nil *[]byte",
				dst:    dst,
				input:  []byte(`{"foo":123.4}`),
				status: StatusBadRequest,
				err:    "kivik: destination pointer is nil",
			}
		}(),
		func() Test {
			dst := []byte{}
			expected := []byte(`{"foo":123.4}`)
			return Test{
				name:     "[]byte",
				dst:      &dst,
				input:    []byte(`{"foo":123.4}`),
				expected: &expected,
			}
		}(),
		func() Test {
			dst := json.RawMessage{}
			expected := json.RawMessage(`{"foo":123.4}`)
			return Test{
				name:     "json.RawMessage",
				dst:      &dst,
				input:    []byte(`{"foo":123.4}`),
				expected: &expected,
			}
		}(),
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := scan(test.dst, test.input)
			testy.StatusError(t, test.err, test.status, err)
			if d := diff.Interface(test.expected, test.dst); d != nil {
				t.Error(d)
			}
		})
	}
}

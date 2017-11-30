package kivik

import (
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/flimzy/diff"
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

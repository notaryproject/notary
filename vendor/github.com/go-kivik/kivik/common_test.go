package kivik

import (
	"errors"
	"io"
	"io/ioutil"
	"strings"
	"testing"
	"time"
)

var testOptions = map[string]interface{}{"foo": 123}

func parseTime(t *testing.T, str string) time.Time {
	ts, err := time.Parse(time.RFC3339, str)
	if err != nil {
		t.Fatal(err)
	}
	return ts
}

type errReader string

var _ io.ReadCloser = errReader("")

func (r errReader) Close() error {
	return nil
}

func (r errReader) Read(_ []byte) (int, error) {
	return 0, errors.New(string(r))
}

func body(s string) io.ReadCloser {
	return ioutil.NopCloser(strings.NewReader(s))
}

type mockIterator struct {
	NextFunc  func(interface{}) error
	CloseFunc func() error
}

var _ iterator = &mockIterator{}

func (i *mockIterator) Next(ifce interface{}) error {
	return i.NextFunc(ifce)
}

func (i *mockIterator) Close() error {
	return i.CloseFunc()
}

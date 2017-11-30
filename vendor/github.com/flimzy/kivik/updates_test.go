package kivik

import (
	"context"
	"testing"

	"github.com/flimzy/kivik/driver"
)

type mockUpdates struct{}

var _ driver.DBUpdates = &mockUpdates{}

func (u *mockUpdates) Close() error                  { return nil }
func (u *mockUpdates) Next(_ *driver.DBUpdate) error { return nil }

func TestDBUpdatesType(t *testing.T) {
	mu := &mockUpdates{}
	f := newDBUpdates(context.Background(), mu)
	t.Run("NoType", func(t *testing.T) {
		if fType := f.Type(); fType != "" {
			t.Errorf("Expected empty type, got `%s`", fType)
		}
	})
	testType := "testType"
	f.curVal = &driver.DBUpdate{
		Type: testType,
	}
	f.ready = true // Pretend we called Next()
	t.Run("TypeIsSet", func(t *testing.T) {
		if fType := f.Type(); fType != testType {
			t.Errorf("Expected '%s' type, got `%s`", testType, fType)
		}
	})
}

// +build 1.8

package couchauth

import "testing"

// Go 1.7 doesn't consider this a failure
func TestBadDSN18(t *testing.T) {
	if _, err := New("ht\\tp:/! This is really bogus!"); err == nil {
		t.Errorf("Expected error for invalid URL.")
	}
}

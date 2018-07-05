package errors

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/flimzy/diff"
	pkgErrors "github.com/pkg/errors"
)

func TestStatusf(t *testing.T) {
	e := Statusf(400, "foo %d", 123)
	result := e.(*statusError)
	expected := &statusError{
		message:    "foo 123",
		statusCode: 400,
	}
	if d := diff.Interface(expected, result); d != nil {
		t.Error(d)
	}
}

func TestWrapStatus(t *testing.T) {
	e := WrapStatus(400, errors.New("foo"))
	expected := &wrappedError{
		err:        errors.New("foo"),
		statusCode: 400,
	}
	result := e.(*wrappedError)
	if d := diff.Interface(expected, result); d != nil {
		t.Error(d)
	}

	t.Run("nil", func(t *testing.T) {
		result := WrapStatus(400, nil)
		if result != nil {
			t.Errorf("Expected nil result")
		}
	})
}

func TestErrorJSON(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "statusError not found",
			err:      &statusError{statusCode: http.StatusNotFound, message: "no_db_file"},
			expected: `{"error":"not_found", "reason":"no_db_file"}`,
		},
		{
			name:     "statusError unknown code",
			err:      &statusError{statusCode: 999, message: "somethin' bad happened"},
			expected: `{"error":"unknown", "reason": "somethin' bad happened"}`,
		},
		{
			name:     "statusError unauthorized",
			err:      &statusError{statusCode: http.StatusUnauthorized, message: "You are not a server admin."},
			expected: `{"error":"unauthorized", "reason":"You are not a server admin."}`,
		},
		{
			name:     "statusError precondition failed",
			err:      &statusError{statusCode: http.StatusPreconditionFailed, message: "The database could not be created, the file already exists."},
			expected: `{"error":"precondition_failed", "reason":"The database could not be created, the file already exists."}`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := json.Marshal(test.err)
			if err != nil {
				t.Fatal(err)
			}
			if d := diff.JSON([]byte(test.expected), result); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestStatusError(t *testing.T) {
	msg := "foo"
	status := 400
	err := &statusError{statusCode: status, message: msg}

	t.Run("Error", func(t *testing.T) {
		if result := err.Error(); result != msg {
			t.Errorf("Unexpected Error: %v", result)
		}
	})

	t.Run("StatusCode", func(t *testing.T) {
		if result := err.StatusCode(); result != status {
			t.Errorf("Unexpected StatusCode: %v", result)
		}
	})

	t.Run("Reason", func(t *testing.T) {
		if result := err.Reason(); result != msg {
			t.Errorf("Unexpected Reason: %v", result)
		}
	})
}

func TestNew(t *testing.T) {
	expected := "foo"
	expectedType := fmt.Sprintf("%T", pkgErrors.New(expected))
	err := New(expected)
	if tp := fmt.Sprintf("%T", err); tp != expectedType {
		t.Errorf("Unexpected type: %s", tp)
	}
	if e := err.Error(); e != expected {
		t.Errorf("Unexpected Error: %s", e)
	}
}

func TestStatus(t *testing.T) {
	status := 400
	msg := "foo"
	expected := &statusError{
		statusCode: status,
		message:    msg,
	}
	err := Status(status, msg)
	if d := diff.Interface(expected, err); d != nil {
		t.Error(d)
	}
}

func TestWrappedError(t *testing.T) {
	msg := "foo"
	status := 400
	e := errors.New(msg)
	err := &wrappedError{
		err:        e,
		statusCode: status,
	}

	t.Run("Error", func(t *testing.T) {
		if result := err.Error(); result != msg {
			t.Errorf("Unexpected Error: %v", result)
		}
	})

	t.Run("StatusCode", func(t *testing.T) {
		if result := err.StatusCode(); result != status {
			t.Errorf("Unexpected StatusCode: %v", result)
		}
	})

	t.Run("Cause", func(t *testing.T) {
		result := err.Cause()
		if d := diff.Interface(e, result); d != nil {
			t.Errorf("Unexpected Cause:\n%s", d)
		}
	})
}

func TestWrap(t *testing.T) {
	expected := "bar: foo"
	e := errors.New("foo")
	expectedType := fmt.Sprintf("%T", pkgErrors.Wrap(e, ""))
	err := Wrap(e, "bar")
	if tp := fmt.Sprintf("%T", err); tp != expectedType {
		t.Errorf("Unexpected type: %s", tp)
	}
	if e := err.Error(); e != expected {
		t.Errorf("Unexpected Error: %s", e)
	}
}

func TestWrapf(t *testing.T) {
	expected := "bar: foo"
	e := errors.New("foo")
	expectedType := fmt.Sprintf("%T", pkgErrors.Wrap(e, ""))
	err := Wrapf(e, "bar")
	if tp := fmt.Sprintf("%T", err); tp != expectedType {
		t.Errorf("Unexpected type: %s", tp)
	}
	if e := err.Error(); e != expected {
		t.Errorf("Unexpected Error: %s", e)
	}
}

func TestErrorf(t *testing.T) {
	expected := "foo"
	expectedType := fmt.Sprintf("%T", pkgErrors.New(expected))
	err := Errorf(expected)
	if tp := fmt.Sprintf("%T", err); tp != expectedType {
		t.Errorf("Unexpected type: %s", tp)
	}
	if e := err.Error(); e != expected {
		t.Errorf("Unexpected Error: %s", e)
	}
}

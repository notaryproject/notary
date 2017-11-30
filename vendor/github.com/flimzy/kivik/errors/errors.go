// Package errors provides convenience functions for Kivik drivers to report
// meaningful errors. This package is not conisidered part of the kivik public
// API and is subject to change without notice.
package errors // import "github.com/flimzy/kivik/errors"

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
)

// Duplicates of statuses in the kivik package, put here to avoid an import
// cycle.
const (
	statusNoError             = 0
	statusInternalServerError = 500
)

// statusError is an error message bundled with an HTTP status code.
type statusError struct {
	statusCode int
	message    string
}

// MarshalJSON satisifies the json.Marshaler interface for the statusError
// type.
func (se *statusError) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{
		"error":  statusText(se.statusCode),
		"reason": se.message,
	})
}

func (se *statusError) Error() string {
	return se.message
}

// StatusCode returns the statusError's embedded HTTP status code.
func (se *statusError) StatusCode() int {
	return se.statusCode
}

// Reason returns the error's underlying reason.
func (se *statusError) Reason() string {
	return se.message
}

// statusCoder is an optional error interface, which returns the error's
// embedded HTTP status code.
type statusCoder interface {
	StatusCode() int
}

// StatusCode extracts an embedded HTTP status code from an error.
func StatusCode(err error) int {
	if err == nil {
		return statusNoError
	}
	if sc, ok := StatusCodeOK(err); ok {
		return sc
	}
	return statusInternalServerError
}

// StatusCodeOK extracts an embedded HTTP status code from an error and returns
// it. ok will be false if the error does not embed a status code.
func StatusCodeOK(err error) (code int, ok bool) {
	if err == nil {
		return statusNoError, false
	}
	if scErr, ok := err.(statusCoder); ok {
		return scErr.StatusCode(), true
	}
	return statusNoError, false
}

// Reasoner is an interface for an error that contains a reason.
type Reasoner interface {
	Reason() string
}

// Reason returns the error's reason if there is one.
func Reason(err error) string {
	if err == nil {
		return ""
	}
	if reasoner, ok := err.(Reasoner); ok {
		return reasoner.Reason()
	}
	return ""
}

// New is a wrapper around the standard errors.New, to avoid the need for
// multiple imports.
func New(msg string) error {
	return errors.New(msg)
}

// Status returns a new error with the designated HTTP status.
func Status(status int, msg string) error {
	return &statusError{
		statusCode: status,
		message:    msg,
	}
}

// Statusf returns a new error with the designated HTTP status.
func Statusf(status int, format string, args ...interface{}) error {
	return &statusError{
		statusCode: status,
		message:    fmt.Sprintf(format, args...),
	}
}

type wrappedError struct {
	err        error
	statusCode int
}

func (e *wrappedError) Error() string {
	return e.err.Error()
}

func (e *wrappedError) StatusCode() int {
	return e.statusCode
}

func (e *wrappedError) Cause() error {
	return e.err
}

// WrapStatus bundles an existing error with a status code.
func WrapStatus(status int, err error) error {
	if err == nil {
		return nil
	}
	return &wrappedError{
		err:        err,
		statusCode: status,
	}
}

// Wrap is a wrapper around pkg/errors.Wrap()
func Wrap(err error, msg string) error {
	return errors.Wrap(err, msg)
}

// Wrapf is a wrapper around pkg/errors.Wrapf()
func Wrapf(err error, format string, args ...interface{}) error {
	return errors.Wrapf(err, format, args...)
}

// Cause is a wrapper around pkg/errors.Cause()
func Cause(err error) error {
	return errors.Cause(err)
}

// Errorf is a wrapper around pkg/errors.Errorf()
func Errorf(format string, args ...interface{}) error {
	return errors.Errorf(format, args...)
}

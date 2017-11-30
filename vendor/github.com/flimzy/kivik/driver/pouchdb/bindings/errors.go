package bindings

import (
	"fmt"

	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/errors"
	"github.com/gopherjs/gopherjs/js"
	"github.com/gopherjs/jsbuiltin"
)

type pouchError struct {
	*js.Object
	Err     string
	Message string
	Status  int
}

// NewPouchError parses a PouchDB error.
func NewPouchError(o *js.Object) error {
	if o == nil || o == js.Undefined {
		return nil
	}
	status := o.Get("status").Int()
	if status == 0 {
		status = kivik.StatusInternalServerError
	}

	var err, msg string
	switch {
	case o.Get("reason") != js.Undefined:
		msg = o.Get("reason").String()
	case o.Get("message") != js.Undefined:
		msg = o.Get("message").String()
	default:
		if jsbuiltin.InstanceOf(o, js.Global.Get("Error")) {
			return errors.Status(status, o.Get("message").String())
		}
	}
	switch {
	case o.Get("name") != js.Undefined:
		err = o.Get("name").String()
	case o.Get("error") != js.Undefined:
		err = o.Get("error").String()
	}

	if msg == "" && o.Get("errno") != js.Undefined {
		switch o.Get("errno").String() {
		case "ECONNREFUSED":
			msg = "connection refused"
		case "ECONNRESET":
			msg = "connection reset by peer"
		case "EPIPE":
			msg = "broken pipe"
		case "ETIMEDOUT", "ESOCKETTIMEDOUT":
			msg = "operation timed out"
		}
	}

	return &pouchError{
		Err:     err,
		Message: msg,
		Status:  status,
	}
}

func (e *pouchError) Error() string {
	return fmt.Sprintf("%s: %s", e.Err, e.Message)
}

func (e *pouchError) StatusCode() int {
	return e.Status
}

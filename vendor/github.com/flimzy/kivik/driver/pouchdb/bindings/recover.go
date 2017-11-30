package bindings

import (
	"fmt"

	"github.com/gopherjs/gopherjs/js"
)

// RecoverError recovers from a thrown JS error. If an error is caught, err
// is set to its value.
//
// To use, put this at the beginning of a function:
//
//     defer RecoverError(&err)
func RecoverError(err *error) {
	if r := recover(); r != nil {
		switch r.(type) {
		case *js.Object:
			*err = NewPouchError(r.(*js.Object))
		case error:
			// This shouldn't ever happen, but just in case
			*err = r.(error)
		default:
			// Catch all for everything else
			*err = fmt.Errorf("%v", r)
		}
	}
}

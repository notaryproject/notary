package kivik

type statusCoder interface {
	StatusCode() int
}

// StatusCode returns the HTTP status code embedded in the error, or 500
// (internal server error), if there was no specified status code.  If err is
// nil, StatusCode returns 0. This provides a convenient way to determine the
// precise nature of a Kivik-returned error.
//
// For example, to panic for all but NotFound errors:
//
//  row, err := db.Get(context.TODO(), "docID")
//  if kivik.StatusCode(err) == kivik.StatusNotFound {
//      return
//  }
//  if err != nil {
//      panic(err)
//  }
//
// This method uses the statusCoder interface, which is not exported by this
// package, but is considered part of the stable public API.  Driver
// implementations are expected to return errors which conform to this
// interface.
//
//  type statusCoder interface {
//      StatusCode() int
//  }
func StatusCode(err error) int {
	if err == nil {
		return 0
	}
	if coder, ok := err.(statusCoder); ok {
		return coder.StatusCode()
	}
	return StatusInternalServerError
}

type reasoner interface {
	Reason() string
}

// Reason returns the reason description for the error, or the error itself
// if none. A nil error returns an empty string.
func Reason(err error) string {
	if err == nil {
		return ""
	}
	if r, ok := err.(reasoner); ok {
		return r.Reason()
	}
	return err.Error()
}

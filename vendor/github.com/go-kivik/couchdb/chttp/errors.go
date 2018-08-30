package chttp

import (
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
)

// HTTPError is an error that represents an HTTP transport error.
type HTTPError struct {
	Code   int
	Reason string `json:"reason"`
}

func (e *HTTPError) Error() string {
	if e.Reason == "" {
		return http.StatusText(e.Code)
	}
	return fmt.Sprintf("%s: %s", http.StatusText(e.Code), e.Reason)
}

// StatusCode returns the embedded status code.
func (e *HTTPError) StatusCode() int {
	return e.Code
}

// ResponseError returns an error from an *http.Response.
func ResponseError(resp *http.Response) error {
	if resp.StatusCode < 400 {
		return nil
	}
	defer func() { _ = resp.Body.Close() }()
	httpErr := &HTTPError{}
	if resp.Request.Method != "HEAD" && resp.ContentLength != 0 {
		if ct, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type")); ct == typeJSON {
			_ = json.NewDecoder(resp.Body).Decode(httpErr)
		}
	}
	httpErr.Code = resp.StatusCode
	return httpErr
}

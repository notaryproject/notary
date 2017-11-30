package util

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
)

// ToJSON converts a string, []byte, json.RawMessage, or an arbitrary type into
// an io.Reader of JSON marshaled data.
func ToJSON(i interface{}) (io.Reader, error) {
	switch t := i.(type) {
	case string:
		return strings.NewReader(t), nil
	case []byte:
		return bytes.NewReader(t), nil
	case json.RawMessage:
		return bytes.NewReader(t), nil
	default:
		buf := &bytes.Buffer{}
		err := json.NewEncoder(buf).Encode(i)
		return buf, err
	}
}

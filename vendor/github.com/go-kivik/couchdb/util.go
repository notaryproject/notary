package couchdb

import (
	"encoding/json"

	"github.com/go-kivik/kivik"
	"github.com/go-kivik/kivik/errors"
)

// deJSONify unmarshals a string, []byte, or json.RawMessage. All other types
// are returned as-is.
func deJSONify(i interface{}) (interface{}, error) {
	var data []byte
	switch t := i.(type) {
	case string:
		data = []byte(t)
	case []byte:
		data = t
	case json.RawMessage:
		data = []byte(t)
	default:
		return i, nil
	}
	var x interface{}
	err := json.Unmarshal(data, &x)
	return x, errors.WrapStatus(kivik.StatusBadRequest, err)
}

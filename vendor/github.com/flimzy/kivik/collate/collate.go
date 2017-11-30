package collate

import (
	"math"
	"reflect"
)

// Collation provides an interface around a CouchDB collation definition.
type Collation interface {
	Eq(i, j interface{}) bool
	LT(i, j interface{}) bool
	LTE(i, j interface{}) bool
	GT(i, j interface{}) bool
	GTE(i, j interface{}) bool
}

type comparison int

const (
	lt comparison = iota - 1
	eq
	gt
)

func (c comparison) String() string {
	switch {
	case c > 0:
		return "greater than"
	case c < 0:
		return "less than"
	default:
		return "equal"
	}
}

type couchType int

const (
	couchNull couchType = iota
	couchBool
	couchNumber
	couchString
	couchArray
	couchObject
)

func couchTypeOf(i interface{}) couchType {
	if i == nil {
		return couchNull
	}
	switch t := i.(type) {
	case bool:
		return couchBool
	case float64:
		if math.IsInf(t, 0) || math.IsNaN(t) {
			return couchNull
		}
		return couchNumber
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32:
		return couchNumber
	case string:
		return couchString
	case map[string]interface{}:
		return couchObject
	}
	switch reflect.ValueOf(i).Kind() {
	case reflect.Slice, reflect.Array:
		return couchArray
	}
	panic("unknown type")
}

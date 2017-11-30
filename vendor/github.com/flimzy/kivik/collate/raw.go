package collate

import (
	"fmt"
	"math"
	"reflect"
	"sort"
)

// Raw provides raw (byte-wise) collation of strings.
type Raw struct{}

var _ Collation = &Raw{}

func (r *Raw) cmp(i, j interface{}) comparison {
	iType, jType := couchTypeOf(i), couchTypeOf(j)
	if iType < jType {
		return lt
	}
	if iType > jType {
		return gt
	}
	switch iType {
	case couchBool, couchNumber, couchString:
		if i == j {
			return eq
		}
	}
	switch iType {
	case couchNull:
		return eq
	case couchBool:
		if i.(bool) {
			return gt
		}
		return lt
	case couchNumber:
		return numberCmp(i, j)
	case couchString:
		return r.stringCmp(i.(string), j.(string))
	case couchArray:
		return r.arrayCmp(i, j)
	case couchObject:
		return r.objectCmp(i, j)
	}
	panic(fmt.Sprintf("unknown couch type: %v", iType))
}

func (r *Raw) stringCmp(i, j string) comparison {
	if i < j {
		return lt
	}
	if i > j {
		return gt
	}
	return eq
}

func (r *Raw) arrayCmp(i, j interface{}) comparison {
	iv, jv := reflect.ValueOf(i), reflect.ValueOf(j)
	maxLen := iv.Len()
	if jv.Len() < maxLen {
		maxLen = jv.Len()
	}
	for k := 0; k < maxLen; k++ {
		if cmp := r.cmp(iv.Index(k).Interface(), jv.Index(k).Interface()); cmp != eq {
			return cmp
		}
	}
	if iv.Len() == jv.Len() {
		return eq
	}
	if iv.Len() < jv.Len() {
		return lt
	}
	return gt
}

func (r *Raw) objectCmp(i, j interface{}) comparison {
	iv := i.(map[string]interface{})
	jv := j.(map[string]interface{})
	ikeys := make([]string, 0, len(iv))
	jkeys := make([]string, 0, len(jv))
	for k := range iv {
		ikeys = append(ikeys, k)
	}
	for k := range jv {
		jkeys = append(jkeys, k)
	}
	sort.Strings(ikeys)
	sort.Strings(jkeys)
	maxLen := len(ikeys)
	if maxLen > len(jkeys) {
		maxLen = len(jkeys)
	}
	for k := 0; k < maxLen; k++ {
		if cmp := r.stringCmp(ikeys[k], jkeys[k]); cmp != eq {
			return cmp
		}
		key := ikeys[k]
		if cmp := r.cmp(iv[key], jv[key]); cmp != eq {
			return cmp
		}
	}
	if len(ikeys) < len(jkeys) {
		return lt
	}
	if len(ikeys) > len(jkeys) {
		return gt
	}
	return eq
}

func numberCmp(i, j interface{}) comparison {
	fi, fj := toFloat(i), toFloat(j)
	if fi < fj {
		return lt
	}
	if fi > fj {
		return gt
	}
	return eq
}

func toFloat(i interface{}) float64 {
	switch t := i.(type) {
	case int:
		return float64(t)
	case int8:
		return float64(t)
	case int16:
		return float64(t)
	case int32:
		return float64(t)
	case int64:
		return float64(t)
	case uint:
		return float64(t)
	case uint8:
		return float64(t)
	case uint16:
		return float64(t)
	case uint32:
		return float64(t)
	case uint64:
		return float64(t)
	case float32:
		return float64(t)
	case float64:
		return t
	}
	return math.NaN()
}

// Eq returns true if i and j are equal.
func (r *Raw) Eq(i, j interface{}) bool {
	return r.cmp(i, j) == eq
}

// LT returns true if i is less than j.
func (r *Raw) LT(i, j interface{}) bool {
	return r.cmp(i, j) == lt
}

// LTE returns true if i is less than or equal to j.
func (r *Raw) LTE(i, j interface{}) bool {
	return r.cmp(i, j) <= eq
}

// GT returns true if i is greater than j.
func (r *Raw) GT(i, j interface{}) bool {
	return r.cmp(i, j) == gt
}

// GTE returns true if i is greater than or equal to j.
func (r *Raw) GTE(i, j interface{}) bool {
	return r.cmp(i, j) >= eq
}

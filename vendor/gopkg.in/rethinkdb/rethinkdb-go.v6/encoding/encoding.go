package encoding

import (
	"reflect"
	"time"
)

var (
	// type constants
	stringType = reflect.TypeOf("")
	timeType   = reflect.TypeOf(new(time.Time)).Elem()

	marshalerType   = reflect.TypeOf(new(Marshaler)).Elem()
	unmarshalerType = reflect.TypeOf(new(Unmarshaler)).Elem()

	emptyInterfaceType = reflect.TypeOf((*interface{})(nil)).Elem()
	mapInterfaceType   = reflect.TypeOf((map[string]interface{})(nil))
)

// Marshaler is the interface implemented by objects that
// can marshal themselves into a valid RQL pseudo-type.
type Marshaler interface {
	MarshalRQL() (interface{}, error)
}

// Unmarshaler is the interface implemented by objects
// that can unmarshal a pseudo-type object of themselves.
type Unmarshaler interface {
	UnmarshalRQL(interface{}) error
}

func init() {
	encoderCache.m = make(map[reflect.Type]encoderFunc)
	decoderCache.m = make(map[decoderCacheKey]decoderFunc)
}

// IgnoreType causes the encoder to ignore a type when encoding
func IgnoreType(t reflect.Type) {
	encoderCache.Lock()
	encoderCache.m[t] = doNothingEncoder
	encoderCache.Unlock()
}

func SetTypeEncoding(
	t reflect.Type,
	encode func(value interface{}) (interface{}, error),
	decode func(encoded interface{}, value reflect.Value) error,
) {
	encoderCache.Lock()
	encoderCache.m[t] = func(v reflect.Value) (interface{}, error) {
		return encode(v.Interface())
	}
	encoderCache.Unlock()

	dec := func(dv reflect.Value, sv reflect.Value) error {
		return decode(sv.Interface(), dv)
	}
	decoderCache.Lock()
	// decode as pointer
	decoderCache.m[decoderCacheKey{dt: t, st: emptyInterfaceType}] = dec
	// decode as value
	decoderCache.m[decoderCacheKey{dt: t, st: mapInterfaceType}] = dec

	if t.Kind() == reflect.Ptr {
		decoderCache.m[decoderCacheKey{dt: t.Elem(), st: emptyInterfaceType}] = dec
		decoderCache.m[decoderCacheKey{dt: t.Elem(), st: mapInterfaceType}] = dec
	}
	decoderCache.Unlock()
}

package encoding

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
)

// newTypeDecoder constructs an decoderFunc for a type.
func newTypeDecoder(dt, st reflect.Type, blank bool) decoderFunc {
	if reflect.PtrTo(dt).Implements(unmarshalerType) ||
		dt.Implements(unmarshalerType) {
		return unmarshalerDecoder
	}

	if st.Kind() == reflect.Interface {
		return newInterfaceAsTypeDecoder(blank)
	}

	switch dt.Kind() {
	case reflect.Bool:
		switch st.Kind() {
		case reflect.Bool:
			return boolAsBoolDecoder
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return intAsBoolDecoder
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			return uintAsBoolDecoder
		case reflect.Float32, reflect.Float64:
			return floatAsBoolDecoder
		case reflect.String:
			return stringAsBoolDecoder
		default:
			return decodeTypeError
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch st.Kind() {
		case reflect.Bool:
			return boolAsIntDecoder
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return intAsIntDecoder
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			return uintAsIntDecoder
		case reflect.Float32, reflect.Float64:
			return floatAsIntDecoder
		case reflect.String:
			return stringAsIntDecoder
		default:
			return decodeTypeError
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		switch st.Kind() {
		case reflect.Bool:
			return boolAsUintDecoder
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return intAsUintDecoder
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			return uintAsUintDecoder
		case reflect.Float32, reflect.Float64:
			return floatAsUintDecoder
		case reflect.String:
			return stringAsUintDecoder
		default:
			return decodeTypeError
		}
	case reflect.Float32, reflect.Float64:
		switch st.Kind() {
		case reflect.Bool:
			return boolAsFloatDecoder
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return intAsFloatDecoder
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			return uintAsFloatDecoder
		case reflect.Float32, reflect.Float64:
			return floatAsFloatDecoder
		case reflect.String:
			return stringAsFloatDecoder
		default:
			return decodeTypeError
		}
	case reflect.String:
		switch st.Kind() {
		case reflect.Bool:
			return boolAsStringDecoder
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return intAsStringDecoder
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			return uintAsStringDecoder
		case reflect.Float32, reflect.Float64:
			return floatAsStringDecoder
		case reflect.String:
			return stringAsStringDecoder
		default:
			return decodeTypeError
		}
	case reflect.Interface:
		if !st.AssignableTo(dt) {
			return decodeTypeError
		}

		return interfaceDecoder
	case reflect.Ptr:
		return newPtrDecoder(dt, st, blank)
	case reflect.Map:
		if st.AssignableTo(dt) {
			return interfaceDecoder
		}

		switch st.Kind() {
		case reflect.Map:
			return newMapAsMapDecoder(dt, st, blank)
		default:
			return decodeTypeError
		}
	case reflect.Struct:
		if st.AssignableTo(dt) {
			return interfaceDecoder
		}

		switch st.Kind() {
		case reflect.Map:
			if kind := st.Key().Kind(); kind != reflect.String && kind != reflect.Interface {
				return newDecodeTypeError(fmt.Errorf("map needs string keys"))
			}

			return newMapAsStructDecoder(dt, st, blank)
		default:
			return decodeTypeError
		}
	case reflect.Slice:
		if st.AssignableTo(dt) {
			return interfaceDecoder
		}

		switch st.Kind() {
		case reflect.Array, reflect.Slice:
			return newSliceDecoder(dt, st)
		default:
			return decodeTypeError
		}
	case reflect.Array:
		if st.AssignableTo(dt) {
			return interfaceDecoder
		}

		switch st.Kind() {
		case reflect.Array, reflect.Slice:
			return newArrayDecoder(dt, st)
		default:
			return decodeTypeError
		}
	default:
		return unsupportedTypeDecoder
	}
}

func invalidValueDecoder(dv, sv reflect.Value) error {
	dv.Set(reflect.Zero(dv.Type()))
	return nil
}

func unsupportedTypeDecoder(dv, sv reflect.Value) error {
	return &UnsupportedTypeError{dv.Type()}
}

func decodeTypeError(dv, sv reflect.Value) error {
	return &DecodeTypeError{
		DestType: dv.Type(),
		SrcType:  sv.Type(),
	}
}

func newDecodeTypeError(err error) decoderFunc {
	return func(dv, sv reflect.Value) error {
		return &DecodeTypeError{
			DestType: dv.Type(),
			SrcType:  sv.Type(),
			Reason:   err.Error(),
		}
	}
}

func interfaceDecoder(dv, sv reflect.Value) error {
	dv.Set(sv)
	return nil
}

func newInterfaceAsTypeDecoder(blank bool) decoderFunc {
	return func(dv, sv reflect.Value) error {
		if !sv.IsNil() {
			dv = indirect(dv, false)
			if blank {
				dv.Set(reflect.Zero(dv.Type()))
			}
			return decodeValue(dv, sv.Elem(), blank)
		}
		return nil
	}
}

type ptrDecoder struct {
	elemDec decoderFunc
}

func (d *ptrDecoder) decode(dv, sv reflect.Value) error {
	v := reflect.New(dv.Type().Elem())
	err := d.elemDec(v, sv)
	dv.Set(v)
	return err
}

func newPtrDecoder(dt, st reflect.Type, blank bool) decoderFunc {
	dec := &ptrDecoder{typeDecoder(dt.Elem(), st, blank)}

	return dec.decode
}

func unmarshalerDecoder(dv, sv reflect.Value) error {
	// modeled off of https://golang.org/src/encoding/json/decode.go?#L325
	if dv.Kind() != reflect.Ptr && dv.Type().Name() != "" && dv.CanAddr() {
		dv = dv.Addr()
	}

	if dv.IsNil() {
		dv.Set(reflect.New(dv.Type().Elem()))
	}

	u := dv.Interface().(Unmarshaler)
	err := u.UnmarshalRQL(sv.Interface())
	if err != nil {
		return &DecodeTypeError{dv.Type(), sv.Type(), err.Error()}
	}
	return nil
}

// Boolean decoders

func boolAsBoolDecoder(dv, sv reflect.Value) error {
	dv.SetBool(sv.Bool())
	return nil
}
func boolAsIntDecoder(dv, sv reflect.Value) error {
	if sv.Bool() {
		dv.SetInt(1)
	} else {
		dv.SetInt(0)
	}
	return nil
}
func boolAsUintDecoder(dv, sv reflect.Value) error {
	if sv.Bool() {
		dv.SetUint(1)
	} else {
		dv.SetUint(0)
	}
	return nil
}
func boolAsFloatDecoder(dv, sv reflect.Value) error {
	if sv.Bool() {
		dv.SetFloat(1)
	} else {
		dv.SetFloat(0)
	}
	return nil
}
func boolAsStringDecoder(dv, sv reflect.Value) error {
	if sv.Bool() {
		dv.SetString("1")
	} else {
		dv.SetString("0")
	}
	return nil
}

// Int decoders

func intAsBoolDecoder(dv, sv reflect.Value) error {
	dv.SetBool(sv.Int() != 0)
	return nil
}
func intAsIntDecoder(dv, sv reflect.Value) error {
	dv.SetInt(sv.Int())
	return nil
}
func intAsUintDecoder(dv, sv reflect.Value) error {
	dv.SetUint(uint64(sv.Int()))
	return nil
}
func intAsFloatDecoder(dv, sv reflect.Value) error {
	dv.SetFloat(float64(sv.Int()))
	return nil
}
func intAsStringDecoder(dv, sv reflect.Value) error {
	dv.SetString(strconv.FormatInt(sv.Int(), 10))
	return nil
}

// Uint decoders

func uintAsBoolDecoder(dv, sv reflect.Value) error {
	dv.SetBool(sv.Uint() != 0)
	return nil
}
func uintAsIntDecoder(dv, sv reflect.Value) error {
	dv.SetInt(int64(sv.Uint()))
	return nil
}
func uintAsUintDecoder(dv, sv reflect.Value) error {
	dv.SetUint(sv.Uint())
	return nil
}
func uintAsFloatDecoder(dv, sv reflect.Value) error {
	dv.SetFloat(float64(sv.Uint()))
	return nil
}
func uintAsStringDecoder(dv, sv reflect.Value) error {
	dv.SetString(strconv.FormatUint(sv.Uint(), 10))
	return nil
}

// Float decoders

func floatAsBoolDecoder(dv, sv reflect.Value) error {
	dv.SetBool(sv.Float() != 0)
	return nil
}
func floatAsIntDecoder(dv, sv reflect.Value) error {
	dv.SetInt(int64(sv.Float()))
	return nil
}
func floatAsUintDecoder(dv, sv reflect.Value) error {
	dv.SetUint(uint64(sv.Float()))
	return nil
}
func floatAsFloatDecoder(dv, sv reflect.Value) error {
	dv.SetFloat(float64(sv.Float()))
	return nil
}
func floatAsStringDecoder(dv, sv reflect.Value) error {
	dv.SetString(strconv.FormatFloat(sv.Float(), 'f', -1, 64))
	return nil
}

// String decoders

func stringAsBoolDecoder(dv, sv reflect.Value) error {
	b, err := strconv.ParseBool(sv.String())
	if err == nil {
		dv.SetBool(b)
	} else if sv.String() == "" {
		dv.SetBool(false)
	} else {
		return &DecodeTypeError{dv.Type(), sv.Type(), err.Error()}
	}
	return nil
}
func stringAsIntDecoder(dv, sv reflect.Value) error {
	i, err := strconv.ParseInt(sv.String(), 0, dv.Type().Bits())
	if err == nil {
		dv.SetInt(i)
	} else {
		return &DecodeTypeError{dv.Type(), sv.Type(), err.Error()}
	}
	return nil
}
func stringAsUintDecoder(dv, sv reflect.Value) error {
	i, err := strconv.ParseUint(sv.String(), 0, dv.Type().Bits())
	if err == nil {
		dv.SetUint(i)
	} else {
		return &DecodeTypeError{dv.Type(), sv.Type(), err.Error()}
	}
	return nil
}
func stringAsFloatDecoder(dv, sv reflect.Value) error {
	f, err := strconv.ParseFloat(sv.String(), dv.Type().Bits())
	if err == nil {
		dv.SetFloat(f)
	} else {
		return &DecodeTypeError{dv.Type(), sv.Type(), err.Error()}
	}
	return nil
}
func stringAsStringDecoder(dv, sv reflect.Value) error {
	dv.SetString(sv.String())
	return nil
}

// Slice/Array decoder

type sliceDecoder struct {
	arrayDec decoderFunc
}

func (d *sliceDecoder) decode(dv, sv reflect.Value) error {
	if dv.Kind() == reflect.Slice {
		dv.Set(reflect.MakeSlice(dv.Type(), dv.Len(), dv.Cap()))
	}

	if !sv.IsNil() {
		err := d.arrayDec(dv, sv)
		if err != nil {
			return err
		}
	}
	return nil
}

func newSliceDecoder(dt, st reflect.Type) decoderFunc {
	dec := &sliceDecoder{newArrayDecoder(dt, st)}
	return dec.decode
}

type arrayDecoder struct {
	elemDec decoderFunc
}

func (d *arrayDecoder) decode(dv, sv reflect.Value) error {
	// Iterate through the slice/array and decode each element before adding it
	// to the dest slice/array
	i := 0
	for i < sv.Len() {
		if dv.Kind() == reflect.Slice {
			// Get element of array, growing if necessary.
			if i >= dv.Cap() {
				newcap := dv.Cap() + dv.Cap()/2
				if newcap < 4 {
					newcap = 4
				}
				newdv := reflect.MakeSlice(dv.Type(), dv.Len(), newcap)
				reflect.Copy(newdv, dv)
				dv.Set(newdv)
			}
			if i >= dv.Len() {
				dv.SetLen(i + 1)
			}
		}

		if i < dv.Len() {
			// Decode into element.
			err := d.elemDec(dv.Index(i), sv.Index(i))
			if err != nil {
				return err
			}
		}

		i++
	}

	// Ensure that the destination is the correct size
	if i < dv.Len() {
		if dv.Kind() == reflect.Array {
			// Array.  Zero the rest.
			z := reflect.Zero(dv.Type().Elem())
			for ; i < dv.Len(); i++ {
				dv.Index(i).Set(z)
			}
		} else {
			dv.SetLen(i)
		}
	}
	return nil
}

func newArrayDecoder(dt, st reflect.Type) decoderFunc {
	dec := &arrayDecoder{typeDecoder(dt.Elem(), st.Elem(), true)}
	return dec.decode
}

// Map decoder

type mapAsMapDecoder struct {
	keyDec, elemDec decoderFunc
	blank           bool
}

func (d *mapAsMapDecoder) decode(dv, sv reflect.Value) error {
	dt := dv.Type()
	if d.blank {
		dv.Set(reflect.MakeMap(reflect.MapOf(dt.Key(), dt.Elem())))
	}

	var mapKey reflect.Value
	var mapElem reflect.Value

	keyType := dv.Type().Key()
	elemType := dv.Type().Elem()

	for _, sElemKey := range sv.MapKeys() {
		var dElemKey reflect.Value
		var dElemVal reflect.Value

		if !mapKey.IsValid() {
			mapKey = reflect.New(keyType).Elem()
		} else {
			mapKey.Set(reflect.Zero(keyType))
		}
		dElemKey = mapKey

		if !mapElem.IsValid() {
			mapElem = reflect.New(elemType).Elem()
		} else {
			mapElem.Set(reflect.Zero(elemType))
		}
		dElemVal = mapElem

		err := d.keyDec(dElemKey, sElemKey)
		if err != nil {
			return err
		}
		err = d.elemDec(dElemVal, sv.MapIndex(sElemKey))
		if err != nil {
			return err
		}

		dv.SetMapIndex(dElemKey, dElemVal)
	}
	return nil
}

func newMapAsMapDecoder(dt, st reflect.Type, blank bool) decoderFunc {
	d := &mapAsMapDecoder{typeDecoder(dt.Key(), st.Key(), blank), typeDecoder(dt.Elem(), st.Elem(), blank), blank}
	return d.decode
}

type mapAsStructDecoder struct {
	fields    []field
	fieldDecs []decoderFunc
	blank     bool
}

func (d *mapAsStructDecoder) decode(dv, sv reflect.Value) error {
	for _, kv := range sv.MapKeys() {
		var f *field
		var compoundFields = []*field{}
		var fieldDec decoderFunc
		key := []byte(kv.String())
		for i := range d.fields {
			ff := &d.fields[i]
			ffd := d.fieldDecs[i]

			if bytes.Equal(ff.nameBytes, key) {
				f = ff
				fieldDec = ffd
				if ff.compound {
					compoundFields = append(compoundFields, ff)
				}
			}
			if f == nil && ff.equalFold(ff.nameBytes, key) {
				f = ff
				fieldDec = ffd
				if ff.compound {
					compoundFields = append(compoundFields, ff)
				}
			}
		}

		if len(compoundFields) > 0 {
			for _, compoundField := range compoundFields {
				dElemVal := fieldByIndex(dv, compoundField.index)
				sElemVal := sv.MapIndex(kv)

				if sElemVal.Kind() == reflect.Interface {
					sElemVal = sElemVal.Elem()
				}
				sElemVal = sElemVal.Index(compoundField.compoundIndex)
				fieldDec = typeDecoder(dElemVal.Type(), sElemVal.Type(), d.blank)

				if !sElemVal.IsValid() || !dElemVal.CanSet() {
					continue
				}

				err := fieldDec(dElemVal, sElemVal)
				if err != nil {
					return err
				}
			}
		} else if f != nil {
			dElemVal := fieldByIndex(dv, f.index)
			sElemVal := sv.MapIndex(kv)

			if !sElemVal.IsValid() || !dElemVal.CanSet() {
				continue
			}

			err := fieldDec(dElemVal, sElemVal)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func newMapAsStructDecoder(dt, st reflect.Type, blank bool) decoderFunc {
	fields := cachedTypeFields(dt)
	se := &mapAsStructDecoder{
		fields:    fields,
		fieldDecs: make([]decoderFunc, len(fields)),
		blank:     blank,
	}
	for i, f := range fields {
		se.fieldDecs[i] = typeDecoder(typeByIndex(dt, f.index), st.Elem(), blank)
	}
	return se.decode
}

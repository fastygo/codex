package content

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math"
	"reflect"

	"github.com/fastygo/codex/validation"
)

func (value *MetadataValue) UnmarshalJSON(data []byte) error {
	type wireValue struct {
		Value   json.RawMessage `json:"value"`
		Private bool            `json:"private,omitempty"`
	}
	var wire wireValue
	if err := decodeJSON(data, &wire, false); err != nil {
		return err
	}
	if len(wire.Value) == 0 {
		return validation.New("content.metadata.value_required", "value", "metadata value is required")
	}
	var decoded any
	if err := decodeJSON(wire.Value, &decoded, false); err != nil {
		return err
	}
	value.Value = decoded
	value.Private = wire.Private
	return nil
}

func decodeJSON(data []byte, target any, strict bool) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if strict {
		decoder.DisallowUnknownFields()
	}
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return errors.New("multiple JSON values are not allowed")
		}
		return err
	}
	return nil
}

// ValidateJSONValue accepts JSON-native scalar, map, slice, and array values.
// Structs, pointers, channels, functions, complex numbers, and non-string map
// keys are rejected even if a custom encoder could serialize them.
func ValidateJSONValue(value any) error {
	if _, err := json.Marshal(value); err != nil {
		return validation.Wrap("content.json.marshal", "", err)
	}
	if err := validateJSONShape(reflect.ValueOf(value)); err != nil {
		return validation.Wrap("content.json.shape", "", err)
	}
	return nil
}

func validateJSONShape(value reflect.Value) error {
	if !value.IsValid() {
		return nil
	}
	if value.Kind() == reflect.Interface {
		if value.IsNil() {
			return nil
		}
		return validateJSONShape(value.Elem())
	}
	switch value.Kind() {
	case reflect.Bool, reflect.String,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return nil
	case reflect.Float32, reflect.Float64:
		if math.IsNaN(value.Float()) || math.IsInf(value.Float(), 0) {
			return errors.New("JSON number must be finite")
		}
		return nil
	case reflect.Map:
		if value.Type().Key().Kind() != reflect.String {
			return errors.New("JSON object keys must be strings")
		}
		for _, key := range value.MapKeys() {
			if err := validateJSONShape(value.MapIndex(key)); err != nil {
				return err
			}
		}
		return nil
	case reflect.Slice:
		if value.IsNil() {
			return nil
		}
		fallthrough
	case reflect.Array:
		for index := 0; index < value.Len(); index++ {
			if err := validateJSONShape(value.Index(index)); err != nil {
				return err
			}
		}
		return nil
	default:
		return errors.New("value is not a JSON-native shape")
	}
}

func cloneValue(value any) any {
	if value == nil {
		return nil
	}
	cloned := cloneJSONShape(reflect.ValueOf(value), map[cloneVisit]reflect.Value{})
	return cloned.Interface()
}

type cloneVisit struct {
	kind     reflect.Kind
	kindType reflect.Type
	pointer  uintptr
	length   int
	capacity int
}

func cloneJSONShape(value reflect.Value, visited map[cloneVisit]reflect.Value) reflect.Value {
	if value.Kind() == reflect.Interface {
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		cloned := cloneJSONShape(value.Elem(), visited)
		target := reflect.New(value.Type()).Elem()
		target.Set(cloned)
		return target
	}
	switch value.Kind() {
	case reflect.Map:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		visit := cloneVisit{kind: value.Kind(), kindType: value.Type(), pointer: uintptr(value.UnsafePointer())}
		if cloned, exists := visited[visit]; exists {
			return cloned
		}
		cloned := reflect.MakeMapWithSize(value.Type(), value.Len())
		visited[visit] = cloned
		for _, key := range value.MapKeys() {
			cloned.SetMapIndex(key, cloneJSONShape(value.MapIndex(key), visited))
		}
		return cloned
	case reflect.Slice:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		visit := cloneVisit{
			kind: value.Kind(), kindType: value.Type(), pointer: value.Pointer(),
			length: value.Len(), capacity: value.Cap(),
		}
		if cloned, exists := visited[visit]; exists {
			return cloned
		}
		cloned := reflect.MakeSlice(value.Type(), value.Len(), value.Len())
		visited[visit] = cloned
		for index := 0; index < value.Len(); index++ {
			cloned.Index(index).Set(cloneJSONShape(value.Index(index), visited))
		}
		return cloned
	case reflect.Array:
		cloned := reflect.New(value.Type()).Elem()
		for index := 0; index < value.Len(); index++ {
			cloned.Index(index).Set(cloneJSONShape(value.Index(index), visited))
		}
		return cloned
	default:
		return value
	}
}

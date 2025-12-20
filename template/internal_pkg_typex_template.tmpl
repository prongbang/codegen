package typex

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/goccy/go-json"
)

type Int64String int64

// UnmarshalJSON Supported BigInt
func (n *Int64String) UnmarshalJSON(data []byte) error {
	var temp string
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	// Parse the string to int64
	parsed, err := strconv.ParseInt(temp, 10, 64)
	if err != nil {
		return err
	}

	*n = Int64String(parsed)
	return nil
}

func SetField(value string, fieldPtr interface{}) error {
	switch ptr := fieldPtr.(type) {
	case *string:
		*ptr = value
	case *int:
		return parseNumeric(value, ptr)
	case *int64:
		return parseNumeric(value, ptr)
	case *int32:
		return parseNumeric(value, ptr)
	case *int16:
		return parseNumeric(value, ptr)
	case *int8:
		return parseNumeric(value, ptr)
	case *uint:
		return parseNumeric(value, ptr)
	case *uint64:
		return parseNumeric(value, ptr)
	case *uint32:
		return parseNumeric(value, ptr)
	case *uint16:
		return parseNumeric(value, ptr)
	case *uint8:
		return parseNumeric(value, ptr)
	case *float64:
		return parseNumeric(value, ptr)
	case *float32:
		return parseNumeric(value, ptr)
	case *bool:
		boolVal, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("failed to parse boolean '%s': %w", value, err)
		}
		*ptr = boolVal
	case *complex64:
		return parseComplex(value, ptr)
	case *complex128:
		return parseComplex(value, ptr)
	default:
		return fmt.Errorf("unsupported field type: %T", fieldPtr)
	}

	return nil
}

// Helper function for numeric parsing
func parseNumeric[T any](value string, ptr *T) error {
	var parsedValue interface{}
	var err error

	switch any(ptr).(type) {
	case *int, *int64, *int32, *int16, *int8:
		parsedValue, err = strconv.ParseInt(value, 10, 64)
	case *uint, *uint64, *uint32, *uint16, *uint8:
		parsedValue, err = strconv.ParseUint(value, 10, 64)
	case *float64, *float32:
		parsedValue, err = strconv.ParseFloat(value, 64)
	default:
		return fmt.Errorf("unsupported numeric type: %T", ptr)
	}

	if err != nil {
		return fmt.Errorf("failed to parse numeric value '%s': %w", value, err)
	}

	reflect.ValueOf(ptr).Elem().Set(reflect.ValueOf(parsedValue).Convert(reflect.TypeOf(ptr).Elem()))
	return nil
}

// Helper function for complex numbers
func parseComplex[T any](value string, ptr *T) error {
	complexVal, err := strconv.ParseComplex(value, 128)
	if err != nil {
		return fmt.Errorf("failed to parse complex number '%s': %w", value, err)
	}
	reflect.ValueOf(ptr).Elem().Set(reflect.ValueOf(complexVal).Convert(reflect.TypeOf(ptr).Elem()))
	return nil
}

package loaders

import (
	"fmt"
	"reflect"
	"strconv"
)

func setFieldValueFromInterface(field reflect.Value, value any) error {
	if value == nil {
		return nil
	}

	if reflect.TypeOf(value) == field.Type() {
		field.Set(reflect.ValueOf(value))
		return nil
	}

	if field.Kind() == reflect.Pointer {
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		return setFieldValueFromInterface(field.Elem(), value)
	}

	switch field.Kind() {
	case reflect.Slice:
		return setSliceValue(field, value)

	// TODO: add to work with nested sctructs
	// case reflect.Struct:
	// 	return setStructFromMap(field, value)

	default:
		return setFieldValue(field, fmt.Sprintf("%v", value))
	}
}

func setFieldValue(field reflect.Value, value string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("parse int: %w", err)
		}
		field.SetInt(v)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return fmt.Errorf("parse uint: %w", err)
		}
		field.SetUint(v)

	case reflect.Float32, reflect.Float64:
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("parse float: %w", err)
		}
		field.SetFloat(v)

	case reflect.Bool:
		v, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("parse bool: %w", err)
		}
		field.SetBool(v)

	case reflect.Pointer:
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		return setFieldValue(field.Elem(), value)

	default:
		return fmt.Errorf("unsupported field type: %s", field.Kind())
	}
	return nil
}

func setSliceValue(field reflect.Value, value any) error {
	elemType := field.Type().Elem()
	srcSlice := reflect.ValueOf(value)
	newSlice := reflect.MakeSlice(field.Type(), srcSlice.Len(), srcSlice.Len())

	for i := 0; i < srcSlice.Len(); i++ {
		elem := srcSlice.Index(i).Interface()
		elemVal := reflect.ValueOf(elem)

		if elemVal.Type().ConvertibleTo(elemType) {
			newSlice.Index(i).Set(elemVal.Convert(elemType))
		} else {
			if err := setFieldValueFromInterface(newSlice.Index(i), elem); err != nil {
				return fmt.Errorf("slice index %d: %w", i, err)
			}
		}
	}

	field.Set(newSlice)
	return nil
}

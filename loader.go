package yily

import (
	"fmt"
	"reflect"
	"strconv"
	"time"
)

// The functions in this file implement the loading pipeline behind
// Client.Load: inspectStruct selects the tagged fields, then decodeFields
// retrieves and converts their values. The Load method itself lives in
// client.go.

// inspectStruct validates target and returns the exported fields tagged with
// `yily`, in declaration order. It is the first step of the loading pipeline
// used by Client.Load.
//
// Target must be a non-nil pointer to a struct.
func inspectStruct(target any) ([]fieldInfo, error) {
	val := reflect.ValueOf(target)
	if val.Kind() != reflect.Pointer || val.IsNil() {
		return nil, ErrInvalidTarget
	}

	elem := val.Elem()
	if elem.Kind() != reflect.Struct {
		return nil, ErrInvalidTarget
	}

	elemType := elem.Type()
	fields := make([]fieldInfo, 0, elem.NumField())

	for i := 0; i < elem.NumField(); i++ {
		structField := elemType.Field(i)
		tag := structField.Tag.Get("yily")
		if tag == "" || tag == "-" {
			continue
		}

		// Field must be exported so it can be assigned.
		fieldVal := elem.Field(i)
		if !fieldVal.CanSet() {
			return nil, fmt.Errorf("%w: tagged field %q must be exported", ErrInvalidTag, structField.Name)
		}

		fields = append(fields, fieldInfo{
			varName:  normalizeVariableName(tag),
			fieldVal: fieldVal,
		})
	}

	return fields, nil
}

// fieldInfo is one exported struct field selected for loading.
type fieldInfo struct {
	varName  string        // Normalized Yily variable name.
	fieldVal reflect.Value // Settable value of the struct field.
}

// decodeFields is the second step of the loading pipeline used by Client.Load.
// It verifies that every required variable is present, converts its string
// value to the field type, and assigns the values only if every conversion
// succeeds.
func decodeFields(fields []fieldInfo, values map[string]string) error {
	type assignment struct {
		fieldVal reflect.Value
		val      any
	}

	assignments := make([]assignment, 0, len(fields))

	for _, f := range fields {
		raw, ok := values[f.varName]
		if !ok {
			return fmt.Errorf("%w: %s", ErrVariableNotFound, f.varName)
		}

		converted, err := convertValue(raw, f.fieldVal.Type(), f.varName)
		if err != nil {
			return err
		}

		assignments = append(assignments, assignment{
			fieldVal: f.fieldVal,
			val:      converted,
		})
	}

	// Apply assignments only after all conversions succeeded.
	for _, a := range assignments {
		switch v := a.val.(type) {
		case string:
			a.fieldVal.SetString(v)
		case int:
			a.fieldVal.SetInt(int64(v))
		case int64:
			a.fieldVal.SetInt(v)
		case time.Duration:
			a.fieldVal.SetInt(int64(v))
		case float64:
			a.fieldVal.SetFloat(v)
		case bool:
			a.fieldVal.SetBool(v)
		}
	}

	return nil
}

// convertValue converts one raw value into the target type. It is the
// per-field conversion step used by decodeFields.
func convertValue(raw string, targetType reflect.Type, varName string) (any, error) {
	if targetType == reflect.TypeFor[time.Duration]() {
		val, err := time.ParseDuration(raw)
		if err != nil {
			return nil, fmt.Errorf("%w: variable %q is not a valid time.Duration", ErrInvalidValue, varName)
		}
		return val, nil
	}

	switch targetType.Kind() {
	case reflect.String:
		return raw, nil
	case reflect.Int:
		val, err := strconv.Atoi(raw)
		if err != nil {
			return nil, fmt.Errorf("%w: variable %q is not a valid int", ErrInvalidValue, varName)
		}
		return val, nil
	case reflect.Int64:
		val, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%w: variable %q is not a valid int64", ErrInvalidValue, varName)
		}
		return val, nil
	case reflect.Float64:
		val, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return nil, fmt.Errorf("%w: variable %q is not a valid float64", ErrInvalidValue, varName)
		}
		return val, nil
	case reflect.Bool:
		val, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, fmt.Errorf("%w: variable %q is not a valid bool", ErrInvalidValue, varName)
		}
		return val, nil
	default:
		return nil, fmt.Errorf("%w: kind %s for variable %q", ErrUnsupportedType, targetType.Kind(), varName)
	}
}

package oas2

import (
	"fmt"
	"strconv"
	"strings"
)

// ConvertParameter converts parameter's value(s) according to parameter's type
// and format. Type and format MUST match OAS 2.0.
// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/2.0.md#parameterObject
func ConvertParameter(vals []string, typ, format string) (value interface{}, err error) {
	if typ == "array" {
		// TODO
		return nil, fmt.Errorf("type %s: NOT IMPLEMENTED", typ)
	}

	if typ == "file" {
		// TODO
		return nil, fmt.Errorf("type %s: NOT IMPLEMENTED", typ)
	}

	if len(vals) != 1 {
		return nil, fmt.Errorf(
			"values count is %d, want 1",
			len(vals),
		)
	}

	return ConvertPrimitive(vals[0], typ, format)
}

// ConvertPrimitive converts string values according to type and format described
// in OAS 2.0.
// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/2.0.md#parameterObject
func ConvertPrimitive(val string, typ, format string) (value interface{}, err error) {
	switch typ {
	case "string":
		return convertString(val, format)
	case "number":
		return convertNumber(val, format)
	case "integer":
		return convertInteger(val, format)
	case "boolean":
		return convertBoolean(val)
	default:
		return nil, fmt.Errorf(
			"unknown type: %s",
			typ,
		)
	}
}

var evaluatesAsTrue = map[string]struct{}{
	"true":     {},
	"1":        {},
	"yes":      {},
	"ok":       {},
	"y":        {},
	"on":       {},
	"selected": {},
	"checked":  {},
	"t":        {},
	"enabled":  {},
}

func convertString(val, format string) (interface{}, error) {
	switch format {
	case "":
		return val, nil
	default:
		// TODO: parse formats byte, binary, date, date-time
		return nil, fmt.Errorf(
			"unknown format %s for type string",
			format,
		)
	}
}

func convertInteger(val, format string) (interface{}, error) {
	switch format {
	case "int32":
		i, err := strconv.ParseInt(val, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("cannot convert %v to int32", val)
		}
		return int32(i), nil
	case "int64":
		fallthrough
	case "":
		i, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("cannot convert %v to int64", val)
		}
		return i, nil
	default:
		return nil, fmt.Errorf(
			"unknown format %s for type integer",
			format,
		)
	}
}

func convertNumber(val, format string) (interface{}, error) {
	switch format {
	case "float":
		f, err := strconv.ParseFloat(val, 32)
		if err != nil {
			return nil, fmt.Errorf("cannot convert %v to float", val)
		}
		return float32(f), nil
	case "double":
		fallthrough
	case "":
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("cannot convert %v to double", val)
		}
		return f, nil
	default:
		return nil, fmt.Errorf(
			"unknown format %s for type integer",
			format,
		)
	}
}

func convertBoolean(val string) (interface{}, error) {
	_, ok := evaluatesAsTrue[strings.ToLower(val)]
	return ok, nil
}

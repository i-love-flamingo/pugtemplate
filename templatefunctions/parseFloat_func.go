package templatefunctions

import (
	"math/big"
	"reflect"
	"strconv"
)

type (
	// ParseFloat struct for template function
	ParseFloat struct{}
)

// Func tries to parse any type into a float64, it is checking for pugjs types and also for regular types
func (p *ParseFloat) Func() interface{} {
	return func(in interface{}) float64 {
		value := reflect.ValueOf(in)

		switch value.Kind() {
		case reflect.String:
			parserFloat, err := strconv.ParseFloat(value.String(), 64)
			if err != nil {
				return 0
			}
			return parserFloat
		case reflect.Int64:
			return float64(in.(int64))
		case reflect.Int:
			return float64(in.(int))
		case reflect.Float32:
			return value.Float()
		case bigFloatKind:
			bigFloat, ok := in.(big.Float)
			if !ok {
				break
			}
			convertedFloat64, _ := bigFloat.Float64()
			return convertedFloat64
		default:
			v := reflect.Indirect(value)
			floatType := reflect.TypeOf(float64(0))
			if v.Type().ConvertibleTo(floatType) {
				valueFloat := v.Convert(floatType)
				return valueFloat.Float()
			}
		}

		return float64(0)

	}
}

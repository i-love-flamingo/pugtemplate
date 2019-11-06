package templatefunctions

import (
	"math/big"
	"reflect"
	"strconv"
)

type (
	// ParseInt struct for template function
	ParseInt struct{}
)

// big.Float isn't mapped by the reflect package, but returns a value of 25
const bigFloatKind reflect.Kind = 25

// Func tries to parse any type into an integer, it is checking for pugjs types and also for regular types
func (p *ParseInt) Func() interface{} {
	return func(in interface{}) int {

		value := reflect.ValueOf(in)
		switch value.Kind() {
		case reflect.String:
			parsedInt, err := strconv.ParseInt(value.String(), 10, 0)
			if err != nil {
				return 0
			}
			return int(parsedInt)
		case reflect.Float64:
			return int(value.Float())
		case reflect.Int:
			return int(value.Int())
		case reflect.Float32:
			return int(value.Float())
		case bigFloatKind:
			bigFloat, ok := in.(big.Float)
			if !ok {
				break
			}
			integer64, _ := bigFloat.Int64()
			return int(integer64)
		default:
			v := reflect.Indirect(value)
			intType := reflect.TypeOf(int(0))
			if v.Type().ConvertibleTo(intType) {
				valueInt := v.Convert(intType)
				return int(valueInt.Int())
			}
		}

		return 0

	}
}

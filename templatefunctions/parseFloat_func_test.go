package templatefunctions

import (
	"context"
	"math/big"
	"testing"

	"flamingo.me/pugtemplate/pugjs"
	"github.com/stretchr/testify/assert"
)

func TestParseFloat_Func(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  float64
	}{
		{
			name:  "empty string",
			input: pugjs.String(""),
			want:  0,
		},
		{
			name:  "pugjs string",
			input: pugjs.String("123"),
			want:  123,
		},
		{
			name:  " string",
			input: "123",
			want:  123,
		},
		{
			name:  "pugjs number",
			input: pugjs.Number(333),
			want:  333,
		},
		{
			name:  "int",
			input: float64(444),
			want:  444,
		},
		{
			name:  "float32",
			input: float32(445),
			want:  445,
		},
		{
			name:  "int 64",
			input: int64(654),
			want:  654,
		},
		{
			name:  "int",
			input: 555,
			want:  555,
		},
		{
			name:  "int32 - runs into default",
			input: int32(321),
			want:  321,
		},
		{
			name:  "pugjs bool",
			input: pugjs.Bool(true),
			want:  0,
		},
		{
			name:  "bool",
			input: false,
			want:  0,
		},
		{
			name:  "big float",
			input: *big.NewFloat(123),
			want:  123,
		},
	}

	for _, tt := range tests {
		tmplFunc := new(ParseFloat)
		parseFloatFunc := tmplFunc.Func(context.Background()).(func(o interface{}) float64)
		assert.Equal(t, tt.want, parseFloatFunc(tt.input), "Testcase: %v - this values should be the same", tt.name)
	}
}

package templatefunctions

import (
	"context"
	"flamingo.me/pugtemplate/pugjs"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDebugFunc_Func(t *testing.T) {
	tests := []struct {
		name      string
		input     interface{}
		allowDeep bool
		result    string
	}{
		{
			name:      "debug empty string, allow deep false",
			input:     "",
			allowDeep: false,
			result:    "\"\"",
		},
		{
			name:      "debug pugjs.Number, allow deep false",
			input:     pugjs.Number(123),
			allowDeep: false,
			result:    "123",
		},
		{
			name: "debug pugjs.Map, allow deep false",
			input: pugjs.Convert(
				map[string]pugjs.Object{
					"something": pugjs.String("string"),
					"number":    pugjs.Number(123),
				},
			).(*pugjs.Map),
			allowDeep: false,
			result:    "{\n    \"number\": 123,\n    \"something\": \"string\"\n}",
		},
	}

	for _, tt := range tests {
		tmplFunc := new(DebugFunc)
		debugFunc := tmplFunc.Func(context.Background()).(func(o interface{}, allowDeep ...bool) string)
		assert.Equal(t, tt.result, debugFunc(tt.input, tt.allowDeep))
	}
}

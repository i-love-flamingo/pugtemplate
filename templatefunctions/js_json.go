package templatefunctions

import (
	"context"
	"encoding/json"

	"flamingo.me/pugtemplate/pugjs"
)

type (
	// JsJSON is exported as a template function
	JsJSON struct{}

	// JSON is our Javascript's JSON equivalent
	JSON struct{}
)

// Func returns the JSON object
func (jl JsJSON) Func(ctx context.Context) interface{} {
	return func() JSON {
		return JSON{}
	}
}

// Stringify returns a string from the json
func (j JSON) Stringify(x interface{}) string {
	b, err := json.Marshal(x)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// Parse Stringify parses a string and returns an object
func (j JSON) Parse(x string) pugjs.Object {
	var m interface{}
	err := json.Unmarshal([]byte(x), &m)
	if err != nil {
		panic(err)
	}
	return pugjs.Convert(m)
}

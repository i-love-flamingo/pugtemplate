package templatefunctions

import (
	"context"
	"encoding/json"

	"flamingo.me/pugtemplate/pugjs"
)

type (
	// DebugFunc renders data as JSON, which allows debugging in templates
	DebugFunc struct{}
)

// Func as implementation of debug method
func (df DebugFunc) Func(ctx context.Context) interface{} {
	return func(o interface{}, allowDeep ...bool) string {
		if len(allowDeep) > 0 {
			pugjs.AllowDeep = allowDeep[0]
		}
		d, _ := json.MarshalIndent(o, "", "    ")
		pugjs.AllowDeep = true
		return string(d)
	}
}

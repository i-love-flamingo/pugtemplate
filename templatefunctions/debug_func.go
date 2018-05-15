package templatefunctions

import (
	"encoding/json"

	"go.aoe.com/flamingo/core/pugtemplate/pugjs"
)

type (
	// DebugFunc renders data as JSON, which allows debugging in templates
	// TODO move into profiler ?
	DebugFunc struct{}
)

// Name alias for use in template
func (df DebugFunc) Name() string {
	return "debug"
}

// Func as implementation of debug method
func (df DebugFunc) Func() interface{} {
	return func(o interface{}, allowDeep ...bool) string {
		if len(allowDeep) > 0 {
			pugjs.AllowDeep = allowDeep[0]
		}
		d, _ := json.MarshalIndent(o, "", "    ")
		pugjs.AllowDeep = true
		return string(d)
	}
}

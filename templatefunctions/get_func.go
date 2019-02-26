package templatefunctions

import (
	"context"

	"flamingo.me/flamingo/v3/framework/web"
	"flamingo.me/pugtemplate/pugjs"
)

type (
	// GetFunc allows templates to access the router's `get` method
	GetFunc struct {
		Router *web.Router `inject:""`
	}
)

// Func as implementation of get method
func (g *GetFunc) Func(ctx context.Context) interface{} {
	return func(what string, params ...*pugjs.Map) interface{} {
		var p = make(map[interface{}]interface{})
		if len(params) == 1 {
			for _, k := range params[0].Keys() {
				p[k] = params[0].Member(k).String()
			}
		}
		return g.Router.Data(ctx, what, p)
	}
}

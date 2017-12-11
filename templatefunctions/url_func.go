package templatefunctions

import (
	"html/template"
	"net/url"

	"go.aoe.com/flamingo/core/pugtemplate/pugjs"
	"go.aoe.com/flamingo/framework/router"
	"go.aoe.com/flamingo/framework/web"
)

type (
	// URLFunc allows templates to access the routers `URL` helper method
	URLFunc struct {
		Router *router.Router `inject:""`
	}
)

// Name alias for use in template
func (u URLFunc) Name() string {
	return "url"
}

// Func as implementation of url method
func (u *URLFunc) Func(ctx web.Context) interface{} {
	return func(where string, params ...*pugjs.Map) template.URL {
		if where == "" {
			q := ctx.Request().URL.Query()
			if len(params) == 1 {
				for k, v := range params[0].Items {
					if v.String() == "" {
						q.Del(k.String())
					} else {
						q.Set(k.String(), v.String())
					}
				}
			}
			return template.URL((&url.URL{RawQuery: q.Encode(), Path: u.Router.Base().Path + ctx.Request().URL.Path}).String())
		}

		var p = make(map[string]string)
		if len(params) == 1 {
			for k, v := range params[0].Items {
				p[k.String()] = v.String()
			}
		}
		return template.URL(u.Router.URL(where, p).String())
	}
}

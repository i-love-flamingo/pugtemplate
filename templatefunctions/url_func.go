package templatefunctions

import (
	"context"
	"html/template"
	"net/url"

	"flamingo.me/flamingo/v3/framework/web"
	"flamingo.me/pugtemplate/pugjs"
)

type (
	routerBaseURLFunc     func() *url.URL                                             // routerBaseURLFunc returns the base URL
	routerRelativeURLFunc func(to string, params map[string]string) (*url.URL, error) // routerRelativeURLFunc returns URL relative to the base URL

	// URLFunc allows templates to access the routers `URL` helper method
	URLFunc struct {
		routerBaseURL     routerBaseURLFunc
		routerRelativeURL routerRelativeURLFunc
	}
)

// Inject inject dependencies
func (u *URLFunc) Inject(router *web.Router) {
	u.routerBaseURL = func() *url.URL { return router.Base() }
	u.routerRelativeURL = func(to string, params map[string]string) (*url.URL, error) { return router.URL(to, params) }
}

// Func as implementation of url method
func (u *URLFunc) Func(ctx context.Context) interface{} {
	return func(where string, params ...*pugjs.Map) template.URL {
		request := web.RequestFromContext(ctx)
		if where == "" {
			q := request.Request().URL.Query()
			if len(params) == 1 {
				for _, k := range params[0].Keys() {
					q.Del(k)
					if arr, ok := params[0].Member(k).(*pugjs.Array); ok {
						for _, i := range arr.Items() {
							q.Add(k, i.String())
						}
					} else if params[0].Member(k).String() != "" {
						q.Set(k, params[0].Member(k).String())
					}
				}
			}
			return template.URL((&url.URL{RawQuery: q.Encode(), Path: u.routerBaseURL().Path + request.Request().URL.Path}).String())
		}

		var p = make(map[string]string)
		var q = make(map[string][]string)
		if len(params) == 1 {
			for _, k := range params[0].Keys() {
				if arr, ok := params[0].Member(k).(*pugjs.Array); ok {
					for _, i := range arr.Items() {
						q[k] = append(q[k], i.String())
					}
				} else {
					p[k] = params[0].Member(k).String()
				}
			}
		}
		url, _ := u.routerRelativeURL(where, p)
		query := url.Query()
		for k, v := range q {
			for _, i := range v {
				query.Add(k, i)
			}
		}
		url.RawQuery = query.Encode()
		return template.URL(url.String())
	}
}

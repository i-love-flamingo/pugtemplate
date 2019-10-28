package templatefunctions

import (
	"context"
	"html/template"
	"net/url"

	"flamingo.me/flamingo/v3/framework/web"
	"flamingo.me/pugtemplate/pugjs"
)

type (
	// TryURLFunc allows templates to access the routers `URL` helper method
	TryURLFunc struct {
		Router *web.Router `inject:""`
	}
)

// Func as implementation of url method
func (u *TryURLFunc) Func(ctx context.Context) interface{} {
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

			return template.URL((&url.URL{RawQuery: q.Encode(), Path: u.Router.Base().Path + request.Request().URL.Path}).String())
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

		tryURLResponse, err := u.Router.URL(where, p)
		if err != nil {
			return ""
		}

		query := tryURLResponse.Query()
		for k, v := range q {
			for _, i := range v {
				query.Add(k, i)
			}
		}
		tryURLResponse.RawQuery = query.Encode()
		return template.URL(tryURLResponse.String())
	}
}

package templatefunctions

import (
	"context"
	"flamingo.me/flamingo/v3/framework/flamingo"
	"html/template"
	"net/url"

	"flamingo.me/flamingo/v3/framework/web"
	"flamingo.me/pugtemplate/pugjs"
)

type (
	// URLFunc allows templates to access the routers `URL` helper method
	URLFunc struct {
		Router    *web.Router     `inject:""`
		Logger    flamingo.Logger `inject:""`
		DebugMode bool            `inject:"config:debug.mode"`
	}
)

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
		url, err := u.Router.Relative(where, p)
		if err != nil {
			u.panicOrError(err)
			return ""
		}
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

func (u *URLFunc) panicOrError(v interface{}) {
	if u.DebugMode {
		panic(v)
	} else {
		u.Logger.WithField(flamingo.LogKeyModule, "pugtemplate").WithField(flamingo.LogKeyCategory, "urltemplatefunc").Error(v)
	}
}

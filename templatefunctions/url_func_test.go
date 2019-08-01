package templatefunctions

import (
	"context"
	"html/template"
	"net/http"
	"net/url"
	"testing"

	"flamingo.me/flamingo/v3/framework/web"
	"github.com/stretchr/testify/assert"

	"flamingo.me/pugtemplate/pugjs"
)

func TestURLFunc_Func(t *testing.T) {
	tests := []struct {
		name                string
		baseUrl             string
		requestUrl          string
		paramWhere          string
		paramParams         []*pugjs.Map
		expectedTemplateUrl string
	}{
		{
			"should return query params: empty where, no params",
			"http://example.com",
			"/test?param1=value1",
			"",
			nil,
			"/test?param1=value1",
		},
		{
			"should return query params: sub base, empty where, no params",
			"http://example.com/sub",
			"/test?param1=value1",
			"",
			nil,
			"/sub/test?param1=value1",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			urlFunc := &URLFunc{
				routerBaseURL: func() *url.URL {
					baseURL, _ := url.Parse(test.baseUrl)
					return baseURL
				},
			}

			httpReq, _ := http.NewRequest("GET", test.requestUrl, nil)
			req := web.CreateRequest(httpReq, nil)
			ctx := web.ContextWithRequest(context.Background(), req)
			tmplFunc := urlFunc.Func(ctx).(func(string, ...*pugjs.Map) template.URL)

			resultURL := tmplFunc(test.paramWhere, test.paramParams...)
			assert.Equal(t, template.URL(test.expectedTemplateUrl), resultURL)
		})
	}
}

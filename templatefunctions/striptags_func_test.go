package templatefunctions_test

import (
	"context"
	"testing"

	"flamingo.me/flamingo/v3/framework/config"
	"flamingo.me/pugtemplate/templatefunctions"
	"github.com/stretchr/testify/assert"
)

func TestStriptagsFunc(t *testing.T) {
	tests := []struct {
		name        string
		in          string
		out         string
		allowedTags config.Slice
	}{
		{"should keep plain text", "do not modify me", "do not modify me", config.Slice{}},
		{"should keep linebreaks", "Hello\nWorld", "Hello\nWorld", config.Slice{}},
		{"should remove tags by default", "<h1>Headline<h1> <p>Paragraph</p>", "Headline Paragraph", config.Slice{}},
		{"should keep defined tags", "<h1>Headline</h1>", "<h1>Headline</h1>", config.Slice{"h1", "h2"}},
		{"should handle self-closing tags", "<h1>Hello<br />World</h1>", "<h1>Hello<br />World</h1>", config.Slice{"h1", "br"}},
		{
			"should remove non whitelisted attributes",
			"<h1 style=\"font-size: 500px\">Keep me</h1><script src=\"http://miner.tld/x.js\">",
			"<h1>Keep me</h1>",
			config.Slice{"h1"},
		},
		{
			"should keep whitelisted attributes",
			"<p>I'm a paragraph containing a <a href=\"http://tld.com\" style=\"font-size:100px\">link</a></p>",
			"<p>I&#39;m a paragraph containing a <a href=\"http://tld.com\">link</a></p>",
			config.Slice{"p", "a(href)"},
		},
		{
			"should keep multiple whitelisted attributes",
			"<a href=\"http://domain.tld\" target=\"_blank\" rel=\"nofollow\">Link with target</a>",
			"<a href=\"http://domain.tld\" target=\"_blank\">Link with target</a>",
			config.Slice{"a(href target)"},
		},
		{
			"attribute naming",
			`<div data-test:foo.bar="a">b</div>`,
			`<div data-test:foo.bar="a">b</div>`,
			config.Slice{"div(data-test:foo.bar)"},
		},
		{
			"vue.js attr",
			`<div v="menuLevel0ActiveIndex === 0 ? &#34;true&#34; : &#34;false&#34;">b</div>`,
			`<div v="menuLevel0ActiveIndex === 0 ? &#34;true&#34; : &#34;false&#34;">b</div>`,
			config.Slice{"div(v)"},
		},
		{
			"vue.js complete",
			`<div v-bind:aria-expanded="menuLevel0ActiveIndex === 0 ? &#34;true&#34; : &#34;false&#34;">b</div>`,
			`<div v-bind:aria-expanded="menuLevel0ActiveIndex === 0 ? &#34;true&#34; : &#34;false&#34;">b</div>`,
			config.Slice{"div(v-bind:aria-expanded)"},
		},
		{
			"something i found in real life",
			`<div class="miniCart" :class="{miniCartWishlistVisible: itemCount}"></div>`,
			`<div class="miniCart" :class="{miniCartWishlistVisible: itemCount}"></div>`,
			config.Slice{"div(class :class)"},
		},
		{
			"attributes without value",
			`<input disabled name="remove-me"/>`,
			`<input disabled />`,
			config.Slice{"input(disabled)"},
		},
		{
			name:        "should filter script tag with only simple html tags allowed",
			in:          "<script>alert('security!');</script>",
			out:         "alert(&#39;security!&#39;);",
			allowedTags: config.Slice{"p"},
		},
		{
			name:        "should filter script tag with escaped input",
			in:          "<p>&lt;script&gt;alert('security');&lt;/script&gt;</p>",
			out:         "<p>&lt;script&gt;alert(&#39;security&#39;);&lt;/script&gt;</p>",
			allowedTags: config.Slice{"p"},
		},
		{
			name:        "should filter script tag",
			in:          "<p><script>alert('security')</script></p>",
			out:         "<p>alert(&#39;security&#39;)</p>",
			allowedTags: config.Slice{"p"},
		},
		{
			name:        "should filter illegal tag which is escaped and surrounded by other tags",
			in:          "<p><b>test</b>&lt;script&gt;alert('security');&lt;/script&gt;<b>test</b></p>",
			out:         "<p><b>test</b>&lt;script&gt;alert(&#39;security&#39;);&lt;/script&gt;<b>test</b></p>",
			allowedTags: config.Slice{"p", "b"},
		},
	}

	var stripTagsFunc = new(templatefunctions.StriptagsFunc)
	stripTags := stripTagsFunc.Func(context.Background()).(func(htmlString string, allowedTagsConfig ...config.Slice) string)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.out, stripTags(tt.in, tt.allowedTags), tt.name)
		})
	}
}

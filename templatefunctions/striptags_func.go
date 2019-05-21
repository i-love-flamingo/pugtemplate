package templatefunctions

import (
	"context"
	"regexp"
	"strings"

	"flamingo.me/flamingo/v3/framework/config"
	"flamingo.me/pugtemplate/pugjs"
	"golang.org/x/net/html"
)

type (
	// StriptagsFunc provides template function to strip html tags
	StriptagsFunc     struct{}
	allowedAttributes map[string]struct{}
	allowedTags       map[string]allowedTag
	allowedTag        struct {
		name       string
		attributes allowedAttributes
	}
)

var (
	nameRe       = regexp.MustCompile(`[a-z0-9\-]+`)
	attributesRe = regexp.MustCompile(`([a-z-:_.0-9]+)`)
)

func createTag(definition string) allowedTag {
	definition = strings.ToLower(definition)
	attributes := make(allowedAttributes)
	for _, attr := range attributesRe.FindAllString(definition, -1) {
		attributes[attr] = struct{}{}
	}

	return allowedTag{
		nameRe.FindString(definition),
		attributes,
	}
}

// Func implements the strip tags template function
func (df StriptagsFunc) Func(ctx context.Context) interface{} {
	return func(htmlString string, allowedTagsConfig ...config.Slice) string {
		doc, err := html.ParseFragment(strings.NewReader(htmlString), nil)
		if err != nil {
			return ""
		}

		allowedTags := make(allowedTags)
		if len(allowedTagsConfig) == 1 {
			for _, item := range allowedTagsConfig[0] {
				if definition, ok := item.(string); ok {
					tag := createTag(definition)
					allowedTags[tag.name] = tag
				}
			}
		}

		res := ""
		for _, n := range doc {
			res += cleanTags(n, allowedTags)
		}
		return res
	}
}

func cleanTags(n *html.Node, allowedTags allowedTags) string {
	var allowedTag allowedTag
	res := ""

	if n.Type == html.ElementNode {
		if tag, ok := allowedTags[n.Data]; ok {
			allowedTag = tag
		}
	}

	if allowedTag.name != "" {
		res += "<"
		res += n.Data
		res += getAllowedAttributes(n.Attr, allowedTag.attributes)
		if isSelfClosingTag(n) {
			res += " /"
		}
		res += ">"
	}

	if n.Type == html.TextNode {
		res += n.Data
	}

	if n.FirstChild != nil {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			res += cleanTags(c, allowedTags)
		}
	}

	if allowedTag.name != "" && !isSelfClosingTag(n) {
		res += "</" + n.Data + ">"
	}

	return res
}

func isSelfClosingTag(n *html.Node) bool {
	if n.Type == html.ElementNode {
		if _, ok := pugjs.SelfClosingTags[n.Data]; ok {
			return true
		}
	}
	return false
}

func getAllowedAttributes(attributes []html.Attribute, allowedAttributes allowedAttributes) string {
	res := ""
	for _, attr := range attributes {
		if _, ok := allowedAttributes[attr.Key]; ok {
			if attr.Val != "" {
				res += " " + attr.Key + "=\"" + html.EscapeString(attr.Val) + "\""
			} else {
				res += " " + attr.Key
			}
		}
	}
	return res
}

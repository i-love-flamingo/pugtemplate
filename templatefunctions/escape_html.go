package templatefunctions

import (
	"context"
	"html/template"
)

type (
	// EscapeHTMLFunc is exported as a template function
	EscapeHTMLFunc struct{}
)

// Func - templatefunction to escape html strings
func (f *EscapeHTMLFunc) Func(context.Context) interface{} {
	return func(str string) string {
		return template.HTMLEscapeString(str)
	}
}

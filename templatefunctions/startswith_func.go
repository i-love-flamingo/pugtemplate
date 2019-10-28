package templatefunctions

import (
	"context"
	"strings"
)

type (
	// StartsWithFunc struct
	StartsWithFunc struct{}
)

// Func StartsWithFunc
func (s *StartsWithFunc) Func(ctx context.Context) interface{} {
	return func(haystack string, needle string) bool {
		haystack = strings.ToLower(haystack)
		needle = strings.ToLower(needle)
		return strings.HasPrefix(haystack, needle)
	}
}

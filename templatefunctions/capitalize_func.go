package templatefunctions

import (
	"context"
	"strings"
)

type (
	// CapitalizeFunc struct
	CapitalizeFunc struct{}
)

// Func to make titleCase (CapitalizeFunc)
func (s *CapitalizeFunc) Func(ctx context.Context) interface{} {
	return func(str string) string {
		return strings.Title(str)
	}
}

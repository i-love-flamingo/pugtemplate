package templatefunctions

import (
	"context"
	"strings"
)

type (
	CapitalizeFunc struct{}
)

func (s *CapitalizeFunc) Func(ctx context.Context) interface{} {
	return func(str string) string {
		return strings.Title(str)
	}
}

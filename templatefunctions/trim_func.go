package templatefunctions

import (
	"context"
	"strings"
)

type (
	TrimFunc struct{}
)

func (s *TrimFunc) Func(ctx context.Context) interface{} {
	return func(str string) string {
		return strings.TrimSpace(str)
	}
}

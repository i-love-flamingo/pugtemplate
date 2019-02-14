package templatefunctions

import "context"

type (
	TruncateFunc struct{}
)

func (s *TruncateFunc) Func(ctx context.Context) interface{} {
	return func(str string, length int) string {
		if len(str) > length {
			return str[0:length] + "..."
		}
		return str
	}
}

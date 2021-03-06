package templatefunctions

import "context"

type (
	// TruncateFunc struct
	TruncateFunc struct{}
)

// Func TruncateFunc
func (s *TruncateFunc) Func(ctx context.Context) interface{} {
	return func(str string, length int) string {
		if len(str) > length {
			return str[0:length] + "..."
		}
		return str
	}
}

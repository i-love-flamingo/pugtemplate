package templatefunctions

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTruncateFunc_Func(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		desiredLength  int
		expectedResult string
	}{
		{
			name:           "empty string",
			input:          "",
			desiredLength:  0,
			expectedResult: "",
		},
		{
			name:           "short string",
			input:          "i am shorter then the desired length",
			desiredLength:  37,
			expectedResult: "i am shorter then the desired length",
		},
		{
			name:           "short string",
			input:          "i am longer then the desired length",
			desiredLength:  34,
			expectedResult: "i am longer then the desired lengt...",
		},
	}

	for _, tt := range tests {
		tmplFunc := new(TruncateFunc)
		truncateFunc := tmplFunc.Func(context.Background()).(func(str string, length int) string)

		assert.Equal(t, tt.expectedResult, truncateFunc(tt.input, tt.desiredLength))
	}
}

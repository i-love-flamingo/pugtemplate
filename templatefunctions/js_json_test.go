package templatefunctions

import (
	"context"
	"testing"

	"flamingo.me/flamingo/v3/framework/flamingo"
	"github.com/stretchr/testify/assert"
)

func TestJsJSON(t *testing.T) {
	var jsJSON flamingo.TemplateFunc = new(JsJSON)

	json := jsJSON.Func(context.Background()).(func() JSON)()
	assert.Equal(t, `{"foo":123}`, json.Stringify(map[string]int{"foo": 123}))
}

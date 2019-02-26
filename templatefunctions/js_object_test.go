package templatefunctions

import (
	"context"
	"testing"

	"flamingo.me/flamingo/v3/framework/flamingo"
	"flamingo.me/pugtemplate/pugjs"
	"github.com/stretchr/testify/assert"
)

func TestJsObject(t *testing.T) {
	var jsObject flamingo.TemplateFunc = new(JsObject)

	object := jsObject.Func(context.Background()).(func() Object)()

	m := pugjs.Convert(make(map[pugjs.Object]pugjs.Object)).(*pugjs.Map)

	m2 := pugjs.Convert(
		map[string]pugjs.Object{
			"foo": pugjs.String("bar"),
			"asd": pugjs.String("dsa"),
		},
	).(*pugjs.Map)

	m3 := pugjs.Convert(
		map[string]pugjs.Object{
			"foo": pugjs.String("bbb"),
		},
	).(*pugjs.Map)

	mx := pugjs.Convert(
		map[string]pugjs.Object{
			"foo": pugjs.String("bbb"),
			"asd": pugjs.String("dsa"),
		},
	).(*pugjs.Map)

	object.Assign(m, m2, m3)
	assert.Len(t, m.Keys(), len(mx.Keys()), "number of keys is not equal")
	for _, expected := range mx.Keys() {
		assert.Equal(t, mx.Member(expected), m.Member(expected), "value of for key %q is not equal", expected)
	}

	arr := object.Keys(mx)
	assert.Equal(t, "asd, foo", arr.Join(", ").String())
	assert.Equal(t, "", object.Keys(nil).Join(", ").String())
}

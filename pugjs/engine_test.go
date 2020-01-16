package pugjs_test

import (
	"testing"

	"flamingo.me/dingo"
	"flamingo.me/flamingo/v3/framework"
	"flamingo.me/flamingo/v3/framework/config"
	"flamingo.me/flamingo/v3/framework/flamingo"
	"github.com/stretchr/testify/assert"

	"flamingo.me/pugtemplate"
	"flamingo.me/pugtemplate/pugjs"
)

func TestNewEngine_ratelimitFromConfig(t *testing.T) {
	cfg := config.Module{
		Map: config.Map{
			"pug_template.ratelimit": float64(42),
			"pug_template.basedir":   "",
			"debug.mode":             false,
		},
	}

	injector := dingo.NewInjector(&cfg)
	injector.Bind((*flamingo.Logger)(nil)).To(flamingo.NullLogger{})
	injector.InitModules(new(pugtemplate.Module), new(framework.InitModule))

	engine, ok := injector.GetInstance(pugjs.Engine{}).(*pugjs.Engine)
	if assert.True(t, ok) {
		assert.Equal(t, float64(42), engine.Ratelimit)
	}
}

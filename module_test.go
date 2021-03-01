package pugtemplate_test

import (
	"testing"

	"flamingo.me/flamingo/v3/framework/config"

	"flamingo.me/pugtemplate"
)

func TestModule_Configure(t *testing.T) {
	if err := config.TryModules(nil, new(pugtemplate.Module)); err != nil {
		t.Error(err)
	}
}

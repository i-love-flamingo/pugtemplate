package main

import (
	"flamingo.me/dingo"
	"flamingo.me/flamingo/v3"
	"flamingo.me/pugtemplate"
)

func main() {
	flamingo.App([]dingo.Module{
		new(pugtemplate.Module),
	})
}

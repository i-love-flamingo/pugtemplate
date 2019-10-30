package pugtemplate

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"flamingo.me/dingo"
	"flamingo.me/flamingo/v3/framework/config"
	"flamingo.me/flamingo/v3/framework/flamingo"
	"flamingo.me/flamingo/v3/framework/web"
	"flamingo.me/pugtemplate/puganalyse"
	"flamingo.me/pugtemplate/pugjs"
	"flamingo.me/pugtemplate/templatefunctions"
	"github.com/spf13/cobra"
)

type (
	// Module for framework/pug_template
	Module struct {
		DefaultMux *http.ServeMux `inject:",optional"`
		Basedir    string         `inject:"config:pug_template.basedir"`
		Whitelist  config.Slice   `inject:"config:pug_template.cors_whitelist"`
	}

	routes struct {
		controller *DebugController
		Basedir    string       `inject:"config:pug_template.basedir"`
		Whitelist  config.Slice `inject:"config:pug_template.cors_whitelist"`
	}

	assetFileSystem struct {
		fs http.FileSystem
	}
)

// Open - opens a given pass and returns a file
func (afs assetFileSystem) Open(path string) (http.File, error) {
	path = strings.Replace(path, "/assets/", "", 1)

	f, err := afs.fs.Open(path)
	if err != nil {
		return nil, err
	}

	s, err := f.Stat()
	if err == nil || s.IsDir() {
		return nil, errors.New("not allowed")
	}

	return f, nil
}

// Inject - inject func
func (r *routes) Inject(controller *DebugController) {
	r.controller = controller
}

func assetHandler(whitelisted []string) http.Handler {
	whitelist := "!" + strings.Join(whitelisted, "!") + "!"

	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		origin := req.Header.Get("Origin")
		if strings.Contains(whitelist, "!"+origin+"!") || strings.Contains(whitelist, "!*!") {
			rw.Header().Add("Access-Control-Allow-Origin", origin)
		}

		if r, e := http.Get("http://localhost:1337" + req.RequestURI); e == nil {
			copyHeaders(r, rw)
			io.Copy(rw, r.Body)
		} else {
			http.FileServer(assetFileSystem{http.Dir("frontend/dist/")}).ServeHTTP(rw, req)
		}
	})
}

// Routes define routes
func (r *routes) Routes(registry *web.RouterRegistry) {
	var whitelist []string
	r.Whitelist.MapInto(&whitelist)

	// trim whitelist of trailing slashes as urls can be configured that way
	whitelist = trimTrailingSlashes(whitelist)

	registry.Route("/_pugtpl/debug", "pugtpl.debug")
	registry.HandleGet("pugtpl.debug", r.controller.Get)

	registry.HandleAny("_static", web.WrapHTTPHandler(http.StripPrefix("/static/", assetHandler(whitelist))))
	registry.Route("/static/*n", "_static")

	registry.HandleData("page.template", func(ctx context.Context, _ *web.Request, _ web.RequestParams) interface{} {
		return ctx.Value("page.template")
	})

	registry.Route("/assets/*f", "_pugtemplate.assets")
	registry.HandleAny("_pugtemplate.assets", web.WrapHTTPHandler(assetHandler(whitelist)))
}

// Configure DI
func (m *Module) Configure(injector *dingo.Injector) {
	// We bind the Template Engine to the ChildSingleton level (in case there is different config handling
	// We use the provider to make sure both are always the same injected type
	injector.Bind(pugjs.Engine{}).In(dingo.ChildSingleton).ToProvider(pugjs.NewEngine)
	injector.Bind((*flamingo.TemplateEngine)(nil)).In(dingo.ChildSingleton).ToProvider(
		func(t *pugjs.Engine, i *dingo.Injector) flamingo.TemplateEngine {
			return flamingo.TemplateEngine(t)
		},
	)

	if m.DefaultMux != nil {
		var whitelist []string
		m.Whitelist.MapInto(&whitelist)

		m.DefaultMux.Handle("/assets/", assetHandler(whitelist))
	}

	injector.BindMap((*flamingo.TemplateFunc)(nil), "Math").To(templatefunctions.JsMath{})
	injector.BindMap((*flamingo.TemplateFunc)(nil), "Object").To(templatefunctions.JsObject{})
	injector.BindMap((*flamingo.TemplateFunc)(nil), "debug").To(templatefunctions.DebugFunc{})
	injector.BindMap((*flamingo.TemplateFunc)(nil), "JSON").To(templatefunctions.JsJSON{})
	injector.BindMap((*flamingo.TemplateFunc)(nil), "startsWith").To(templatefunctions.StartsWithFunc{})
	injector.BindMap((*flamingo.TemplateFunc)(nil), "truncate").To(templatefunctions.TruncateFunc{})
	injector.BindMap((*flamingo.TemplateFunc)(nil), "stripTags").To(templatefunctions.StriptagsFunc{})
	injector.BindMap((*flamingo.TemplateFunc)(nil), "capitalize").To(templatefunctions.CapitalizeFunc{})
	injector.BindMap((*flamingo.TemplateFunc)(nil), "trim").To(templatefunctions.TrimFunc{})

	injector.BindMap((*flamingo.TemplateFunc)(nil), "asset").To(templatefunctions.AssetFunc{})
	injector.BindMap((*flamingo.TemplateFunc)(nil), "data").To(templatefunctions.DataFunc{})
	injector.BindMap((*flamingo.TemplateFunc)(nil), "get").To(templatefunctions.GetFunc{})
	injector.BindMap((*flamingo.TemplateFunc)(nil), "tryUrl").To(templatefunctions.TryURLFunc{})
	injector.BindMap((*flamingo.TemplateFunc)(nil), "url").To(templatefunctions.URLFunc{})

	injector.BindMulti(new(cobra.Command)).ToProvider(templatecheckCmd)
	web.BindRoutes(injector, new(routes))
	flamingo.BindEventSubscriber(injector).To(pugjs.EventSubscriber{})
}

func templatecheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "templatecheck",
		Short: "run opinionated checks in frontend/src: Checks atomic design system dependencies (PUG) and js dependencies conventions",
		//Aliases: []string{"pugcheck"},
		Run: AnalyseCommand(),
	}
}

// AnalyseCommand func
func AnalyseCommand() func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		hasError := false
		if _, err := os.Stat("frontend/src"); err == nil {
			fmt.Println()
			fmt.Println("Analyse Project Design System (PUG) in frontend/src")
			fmt.Println("###################################################")
			analyser := puganalyse.NewAtomicDesignAnalyser("frontend/src")
			analyser.CheckPugImports()
			hasError = analyser.HasError
			fmt.Println(fmt.Sprintf("%v files checked", analyser.CheckCount))

			fmt.Println()
			fmt.Println("Analyse Project JS dependencies in frontend/src")
			fmt.Println("###################################################")
			jsanalyser := puganalyse.NewJsDependencyAnalyser("frontend/src")
			jsanalyser.Check()
			if !hasError {
				hasError = analyser.HasError
			}
			fmt.Println(fmt.Sprintf("%v files checked", jsanalyser.CheckCount))

		} else {
			fmt.Println("Project Design System not found in folder frontend/src")
		}

		if _, err := os.Stat("frontend/src/shared"); err == nil {
			fmt.Println()
			log.Printf("Analyse Shared Design System (PUG) in frontend/src/shared")
			fmt.Println("###################################################")
			analyser := puganalyse.NewAtomicDesignAnalyser("frontend/src/shared")
			analyser.CheckPugImports()
			fmt.Println(fmt.Sprintf("%v files checked", analyser.CheckCount))
			if !hasError {
				hasError = analyser.HasError
			}

			fmt.Println()
			fmt.Println("Analyse Shared JS dependencies in frontend/src/shared")
			fmt.Println("###################################################")
			jsanalyser := puganalyse.NewJsDependencyAnalyser("frontend/src/shared")
			jsanalyser.Check()
			if !hasError {
				hasError = analyser.HasError
			}
			fmt.Println(fmt.Sprintf("%v files checked", jsanalyser.CheckCount))

		} else {
			fmt.Println("No shared Design System not found in folder frontend/src/shared")
		}

		if hasError {
			os.Exit(-1)
		}
	}
}

// DefaultConfig for setting pug-related config options
func (m *Module) DefaultConfig() config.Map {
	return config.Map{
		"pug_template.basedir":                 "frontend/dist",
		"pug_template.debug":                   true,
		"pug_template.cors_whitelist":          config.Slice{"http://localhost:3210"},
		"imageservice.base_url":                "-",
		"imageservice.secret":                  "-",
		"opencensus.tracing.sampler.blacklist": config.Slice{"/static", "/assets"},
	}
}

func copyHeaders(r *http.Response, w http.ResponseWriter) {
	for key, values := range r.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
}

// trimTrailingSlashes remove trailing slashes from configured whitelist urls
// in case they have been configured
func trimTrailingSlashes(whitelist []string) []string {
	result := make([]string, len(whitelist))
	for i, entry := range whitelist {
		result[i] = strings.TrimRight(entry, "/")
	}
	return result
}

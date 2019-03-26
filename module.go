package pugtemplate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
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
		RootCmd        *cobra.Command      `inject:"flamingo"`
		RouterRegistry *web.RouterRegistry `inject:""`
		DefaultMux     *http.ServeMux      `inject:",optional"`
		Basedir        string              `inject:"config:pug_template.basedir"`
	}

	routes struct {
		controller *DebugController
		Basedir    string `inject:"config:pug_template.basedir"`
	}
)

func (r *routes) Inject(controller *DebugController) {
	r.controller = controller
}

func (r *routes) Routes(registry *web.RouterRegistry) {
	registry.Route("/_pugtpl/debug", "pugtpl.debug")
	registry.HandleGet("pugtpl.debug", r.controller.Get)

	//registry.HandleAny("_static", router.HTTPAction(http.StripPrefix("/static/", http.FileServer(http.Dir(r.Basedir)))))
	registry.HandleAny("_static", web.WrapHTTPHandler(http.StripPrefix("/static/", http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		origin := req.Header.Get("Origin")
		if origin != "" {
			//TODO - configure whitelist
			rw.Header().Add("Access-Control-Allow-Origin", origin)
		}
		http.ServeFile(rw, req, r.Basedir+"/"+req.URL.Path)
	}))))
	registry.Route("/static/*n", "_static")

	registry.HandleData("page.template", func(ctx context.Context, _ *web.Request, _ web.RequestParams) interface{} {
		return ctx.Value("page.template")
	})

	registry.Route("/assets/*f", "_pugtemplate.assets")
	registry.HandleAny("_pugtemplate.assets", web.WrapHTTPHandler(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		origin := req.Header.Get("Origin")
		if origin != "" {
			//TODO - configure whitelist
			rw.Header().Add("Access-Control-Allow-Origin", origin)
		}
		if r, e := http.Get("http://localhost:1337" + req.RequestURI); e == nil {
			copyHeaders(r, rw)
			io.Copy(rw, r.Body)
		} else {
			http.ServeFile(rw, req, strings.Replace(req.RequestURI, "/assets/", "frontend/dist/", 1))
		}
	})))
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
		m.DefaultMux.HandleFunc("/assets/", func(rw http.ResponseWriter, req *http.Request) {
			origin := req.Header.Get("Origin")
			if origin != "" {
				//TODO - configure whitelist
				rw.Header().Add("Access-Control-Allow-Origin", origin)
			}
			if r, e := http.Get("http://localhost:1337" + req.RequestURI); e == nil {
				copyHeaders(r, rw)
				io.Copy(rw, r.Body)
			} else {
				http.ServeFile(rw, req, strings.Replace(req.RequestURI, "/assets/", "frontend/dist/", 1))
			}
		})
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

	m.loadmock("../src/layout/*")
	m.loadmock("../src/layout/*/*")
	m.loadmock("../src/layout/*/*/*")
	m.loadmock("../src/atom/*")
	m.loadmock("../src/molecule/*/*")
	m.loadmock("../src/organism/*")
	m.loadmock("../src/page/*/*")
	m.loadmock("../src/mock")

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

// Analyse command
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
		"imageservice.base_url":                "-",
		"imageservice.secret":                  "-",
		"opencensus.tracing.sampler.blacklist": config.Slice{"/static", "/assets"},
	}
}

func (m *Module) loadmock(where string) (interface{}, error) {
	matches, err := filepath.Glob(m.Basedir + "/" + where + "/*.mock.json")
	if err != nil {
		return nil, err
	}

	for _, match := range matches {
		b, e := ioutil.ReadFile(match)
		if e != nil {
			continue
		}
		var res interface{}
		json.Unmarshal(b, &res)
		name := strings.Replace(filepath.Base(match), ".mock.json", "", 1)
		if !m.RouterRegistry.HasData(name) {
			m.RouterRegistry.HandleData(name, mockcontroller(name, res))
			log.Println("mocking because not set:", name)
		}
	}

	return nil, nil
}

func mockcontroller(name string, data interface{}) web.DataAction {
	return func(context.Context, *web.Request, web.RequestParams) interface{} {
		return data
	}
}

func copyHeaders(r *http.Response, w http.ResponseWriter) {
	for key, values := range r.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
}

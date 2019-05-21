package pugjs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"flamingo.me/flamingo/v3/framework/flamingo"
	"flamingo.me/flamingo/v3/framework/opencensus"
	"github.com/pkg/errors"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"go.opencensus.io/trace"
)

// BUG: the template loading is far from optimal, if debug is false and the loading fails we might end up in a broken situation

type (
	// RenderState holds information about the pug abstract syntax tree
	renderState struct {
		path         string
		mixin        map[string]string
		mixincalls   map[string]struct{}
		mixinorder   []string
		mixincounter int
		mixinblocks  []string
		mixinblock   string
		funcs        FuncMap
		rawmode      bool
		doctype      string
		debug        bool
		eventRouter  flamingo.EventRouter
		logger       flamingo.Logger
	}

	templateFuncProvider func() map[string]flamingo.TemplateFunc

	// Engine is the one and only javascript template engine for go ;)
	Engine struct {
		*sync.RWMutex
		Basedir        string `inject:"config:pug_template.basedir"`
		Debug          bool   `inject:"config:debug.mode"`
		Trace          bool   `inject:"config:pug_template.trace,optional"`
		Assetrewrites  map[string]string
		templateLoaded sync.Map
		templates      map[string]*Template
		TemplateCode   map[string]string
		Webpackserver  bool
		EventRouter    flamingo.EventRouter `inject:""`
		FuncProvider   templateFuncProvider `inject:""`
		Logger         flamingo.Logger      `inject:""`
	}
)

var (
	rt             = stats.Int64("flamingo/pugtemplate/render", "pugtemplate render times", stats.UnitMilliseconds)
	templateKey, _ = tag.NewKey("template")
)

func init() {
	opencensus.View("flamingo/pugtemplate/render", rt, view.Distribution(50, 100, 250, 500, 1000, 2000), templateKey)
}

// NewEngine constructor
func NewEngine() *Engine {
	return &Engine{
		RWMutex:      new(sync.RWMutex),
		TemplateCode: make(map[string]string),
	}
}

func newRenderState(path string, debug bool, eventRouter flamingo.EventRouter, logger flamingo.Logger) *renderState {
	return &renderState{
		path:        path,
		mixin:       make(map[string]string),
		mixincalls:  make(map[string]struct{}),
		debug:       debug,
		eventRouter: eventRouter,
		logger:      logger,
	}
}

// LoadTemplates with an optional filter
func (e *Engine) LoadTemplates(ctx context.Context, filtername string) error {
	e.Lock()
	defer e.Unlock()

	prevGC := debug.SetGCPercent(-1)
	defer debug.SetGCPercent(prevGC)

	start := time.Now()

	manifest, err := ioutil.ReadFile(path.Join(e.Basedir, "manifest.json"))
	if err == nil {
		json.Unmarshal(manifest, &e.Assetrewrites)
	}

	e.templates, err = e.compileDir(ctx, path.Join(e.Basedir, "template", "page"), "", filtername)
	if err != nil {
		return err
	}

	if _, err := http.Get("http://localhost:1337/assets/js/vendor.js"); err == nil {
		e.Webpackserver = true
	} else {
		e.Webpackserver = false
	}

	e.Logger.Info("Compiled templates in", time.Since(start))
	return nil
}

// compileDir returns a map of defined templates in directory dirname
func (e *Engine) compileDir(ctx context.Context, root, dirname, filtername string) (map[string]*Template, error) {
	result := make(map[string]*Template)

	dir, err := os.Open(path.Join(root, dirname))
	if err != nil {
		return nil, err
	}

	defer dir.Close()

	filenames, err := dir.Readdir(-1)
	if err != nil {
		return nil, err
	}

	for _, filename := range filenames {
		if filename.IsDir() {
			tpls, err := e.compileDir(ctx, root, path.Join(dirname, filename.Name()), filtername)
			if err != nil {
				return nil, err
			}
			for k, v := range tpls {
				if result[k] == nil {
					result[k] = v
				}
			}
		} else {
			if strings.HasSuffix(filename.Name(), ".ast.json") {
				name := path.Join(dirname, filename.Name())
				name = name[:len(name)-len(".ast.json")]

				if filtername != "" && !strings.HasPrefix(name, filtername) {
					continue
				}

				ctx, span := trace.StartSpan(ctx, "pug/template/compile")
				span.Annotate(nil, filename.Name())
				defer span.End()

				renderState := newRenderState(path.Join(e.Basedir, "template", "page"), e.Debug, e.EventRouter, e.Logger)
				renderState.funcs = FuncMap{}

				for k, f := range e.FuncProvider() {
					renderState.funcs[k] = f.Func
				}

				_, span2 := trace.StartSpan(ctx, "pug/template/parse")
				token, err := renderState.Parse(name)
				if err != nil {
					span2.End()
					return nil, err
				}
				span2.End()

				_, span3 := trace.StartSpan(ctx, "pug/template/tokenToTemplate")
				defer span3.End()
				result[name], e.TemplateCode[name], err = renderState.TokenToTemplate(name, token)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	return result, nil
}

var renderChan = make(chan struct{}, 8)

var _ flamingo.TemplateEngine = new(Engine)
var _ flamingo.PartialTemplateEngine = new(Engine)

// RenderPartials is used for progressive enhancements / rendering of partial template areas
// usually this is requested via the appropriate javascript headers and taken care of in the framework renderer
func (e *Engine) RenderPartials(ctx context.Context, templateName string, data interface{}, partials []string) (map[string]io.Reader, error) {
	res := make(map[string]io.Reader, len(partials))

	for _, partial := range partials {
		buf, err := e.Render(ctx, templateName+".partial/"+partial, data)
		if err != nil {
			return nil, err
		}
		res[partial] = buf
	}

	return res, nil
}

// Render via html/pug_template
func (e *Engine) Render(ctx context.Context, templateName string, data interface{}) (io.Reader, error) {
	ctx, span := trace.StartSpan(ctx, "pug/render")
	defer span.End()

	span.Annotate(nil, templateName)

	// block if buffered channel size is reached
	renderChan <- struct{}{}
	defer func() {
		// release one entry from channel (will release one block)
		<-renderChan
	}()

	p := strings.Split(templateName, "/")
	for i, v := range p {
		p[i] = strings.Title(v)
	}
	page := p[len(p)-1]
	if len(p) >= 2 && p[len(p)-2] != page {
		page = p[len(p)-2] + p[len(p)-1]
	}
	ctx = context.WithValue(ctx, "page.template", "page"+page)

	// recompile, make sure to fully load only once!
	_, loaded := e.templateLoaded.LoadOrStore(templateName, struct{}{})
	if !loaded || e.Debug {
		_, spanLoad := trace.StartSpan(ctx, "pug/loadTemplate")
		spanLoad.Annotate(nil, templateName)
		if err := e.LoadTemplates(ctx, templateName); err != nil {
			spanLoad.End()
			return nil, err
		}
		spanLoad.End()
	}

	// make sure template loading has finished by now!
	e.RLock()

	result := new(bytes.Buffer)

	templateInstance, ok := e.templates[templateName]
	e.RUnlock()
	if !ok {
		return nil, errors.Errorf(`Template %s not found!`, templateName)
	}

	ctx, execSpan := trace.StartSpan(ctx, "pug/execute")
	execSpan.Annotate(nil, templateName)
	start := time.Now()
	err := templateInstance.ExecuteTemplate(ctx, result, templateName, convert(data), e.Trace)
	execSpan.End()
	ctx, _ = tag.New(ctx, tag.Upsert(templateKey, templateName))
	stats.Record(ctx, rt.M(time.Since(start).Nanoseconds()/1000000))

	if err != nil {
		errstr := err.Error() + "\n"
		for i, l := range strings.Split(e.TemplateCode[templateName], "\n") {
			errstr += fmt.Sprintf("%03d: %s\n", i+1, strings.TrimSpace(strings.TrimSuffix(l, `{{- "" -}}`)))
		}
		return nil, errors.New(errstr)
	}

	return result, nil
}

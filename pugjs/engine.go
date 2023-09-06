package pugjs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"flamingo.me/flamingo/v3/framework/flamingo"
	"flamingo.me/flamingo/v3/framework/opencensus"
	"flamingo.me/flamingo/v3/framework/web"
	"github.com/pkg/errors"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"go.opencensus.io/trace"
)

// BUG: the template loading is far from optimal, if debug is false and the loading fails we might end up in a broken situation

type (
	// renderState holds information about the pug abstract syntax tree
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

	key string

	// Engine is the one and only javascript template engine for go ;)
	Engine struct {
		*sync.RWMutex
		Basedir         string `inject:"config:pug_template.basedir"`
		Debug           bool   `inject:"config:flamingo.debug.mode"`
		Trace           bool   `inject:"config:pug_template.trace,optional"`
		Assetrewrites   map[string]string
		templatesLoaded int32
		templates       map[string]*Template
		TemplateCode    map[string]string
		// Webpackserver flag
		// Deprecated: not used anymore
		Webpackserver    bool
		EventRouter      flamingo.EventRouter `inject:""`
		FuncProvider     templateFuncProvider `inject:""`
		Logger           flamingo.Logger      `inject:""`
		ratelimit        chan struct{}
		CheckWebpack1337 bool `inject:"config:pug_template.check_webpack_1337"`
	}

	// EngineOption options to configure the Engine
	EngineOption func(e *Engine)

	// EventSubscriber is the event subscriber for Engine
	EventSubscriber struct {
		engine  *Engine
		logger  flamingo.Logger
		startup *Startup
	}
)

const (
	// PageKey is used as constant in WithValue function and in module.go
	PageKey key = "page.template"
)

var (
	rt                    = stats.Int64("flamingo/pugtemplate/render", "pugtemplate render times", stats.UnitMilliseconds)
	statRateLimitWaitTime = stats.Float64("flamingo/pugtemplate/ratelimit/waittime", "pugtemplate waiting time due to rate limit", stats.UnitMilliseconds)
	templateKey, _        = tag.NewKey("template")

	debugMode      = false
	loggerInstance flamingo.Logger
)

func init() {
	_ = opencensus.View("flamingo/pugtemplate/render", rt, view.Distribution(50, 100, 250, 500, 1000, 2000), templateKey)
	_ = opencensus.View("flamingo/pugtemplate/ratelimit/waittime", statRateLimitWaitTime, view.Distribution(0.0001, 0.001, 0.01, 0.1, 1, 10, 100, 1000, 10000), templateKey)
}

// Inject injects the EventSubscibers dependencies
func (e *EventSubscriber) Inject(engine *Engine, logger flamingo.Logger, startup *Startup) {
	e.engine = engine
	e.logger = logger
	e.startup = startup
}

// Notify the event subscriber
func (e *EventSubscriber) Notify(_ context.Context, event flamingo.Event) {
	switch ev := event.(type) {
	case *web.AreaRoutedEvent:
		e.logger.Info("preloading templates on web.AreaRoutedEvent for area ", ev.ConfigArea.Name)
		e.startup.AddProcess(func() error {
			return e.engine.LoadTemplates("")
		})
	case *flamingo.ServerStartEvent:
		errs := e.startup.Finish()
		go func() {
			err := <-errs
			if err != nil {
				panic(err)
			}
		}()
	}
}

// NewEngine constructor
func NewEngine(debugsetup *struct {
	Debug  bool            `inject:"config:flamingo.debug.mode"`
	Logger flamingo.Logger `inject:""`
}) *Engine {
	if debugsetup != nil {
		setLoggerInfos(debugsetup.Logger, debugsetup.Debug)
	}

	// For backward-compatibility we set the rate limit to "8" here.
	// Also mind Engine's Inject method which also configures the instance,
	// but happens there also for call-side comparability.
	return NewEngineWithOptions(WithRateLimit(8))
}

// WithRateLimit configures the rate limit. A value of zero, disables the rate limit.
func WithRateLimit(rateLimit int) EngineOption {
	if rateLimit <= 0 {
		return func(e *Engine) {
			e.ratelimit = nil
		}
	}

	return func(e *Engine) {
		e.ratelimit = make(chan struct{}, rateLimit)
	}
}

// NewEngineWithOptions create a new Engine with options
func NewEngineWithOptions(opt ...EngineOption) *Engine {
	engine := &Engine{
		RWMutex:      new(sync.RWMutex),
		TemplateCode: make(map[string]string),
	}

	engine.applyOptions(opt...)
	return engine
}

// Inject injects dependencies
func (e *Engine) Inject(cfg *struct {
	RateLimit float64 `inject:"config:pug_template.ratelimit"`
}) {
	// Also mind NewEngine regarding instance configuration
	e.applyOptions(WithRateLimit(int(cfg.RateLimit)))
}

func (e *Engine) applyOptions(opt ...EngineOption) {
	for _, option := range opt {
		if option != nil {
			option(e)
		}
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

// GetRateLimit returns the rate limit; zero means rate limit is not activated
func (e *Engine) GetRateLimit() int {
	return cap(e.ratelimit)
}

// LoadTemplates with an optional filter
func (e *Engine) LoadTemplates(filtername string) error {
	e.Lock()
	defer e.Unlock()

	if !atomic.CompareAndSwapInt32(&e.templatesLoaded, 0, 1) && filtername == "" {
		return errors.New("Can not preload all templates again")
	}

	start := time.Now()

	manifest, err := os.ReadFile(path.Join(e.Basedir, "manifest.json"))
	if err == nil {
		_ = json.Unmarshal(manifest, &e.Assetrewrites)
	}

	e.templates, err = e.compileDir(path.Join(e.Basedir, "template", "page"), "", filtername)
	if err != nil {
		atomic.StoreInt32(&e.templatesLoaded, 0) // bail out :(
		return err
	}

	e.Webpackserver = false
	if e.CheckWebpack1337 {
		if _, err := http.Get("http://localhost:1337/assets/js/vendor.js"); err == nil {
			e.Webpackserver = true
		}
	}

	e.Logger.Info("Compiled templates in ", time.Since(start))
	return nil
}

// compileDir returns a map of defined templates in directory dirname
func (e *Engine) compileDir(root, dirname, filtername string) (map[string]*Template, error) {
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
			tpls, err := e.compileDir(root, path.Join(dirname, filename.Name()), filtername)
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

				renderState := newRenderState(path.Join(e.Basedir, "template", "page"), e.Debug, e.EventRouter, e.Logger)
				renderState.funcs = FuncMap{}

				for k, f := range e.FuncProvider() {
					renderState.funcs[k] = f.Func
				}

				token, err := renderState.Parse(name)
				if err != nil {
					return nil, err
				}
				result[name], e.TemplateCode[name], err = renderState.TokenToTemplate(name, token)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	return result, nil
}

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
	if cap(e.ratelimit) > 0 {
		start := time.Now()
		select {
		case <-ctx.Done():
			e.Logger.Debugf("template %s wait failed: %s", templateName, ctx.Err().Error())
			return nil, fmt.Errorf("template %s wait failed: %w", templateName, ctx.Err())
		case e.ratelimit <- struct{}{}:
		}

		ctx, _ = tag.New(ctx, tag.Upsert(templateKey, templateName))
		waited := float64(time.Since(start).Nanoseconds() / 1000000.0)
		stats.Record(ctx, statRateLimitWaitTime.M(waited))

		e.Logger.Debugf("template %s waited %fmsec", templateName, waited)
		defer func() {
			// release one entry from channel (will release one block)
			<-e.ratelimit
		}()
	}

	p := strings.Split(templateName, "/")
	for i, v := range p {
		p[i] = strings.Title(v)
	}
	page := p[len(p)-1]
	if len(p) >= 2 && p[len(p)-2] != page {
		page = p[len(p)-2] + p[len(p)-1]
	}
	ctx = context.WithValue(ctx, PageKey, "page"+page)

	// recompile, make sure to fully load only once!
	if atomic.LoadInt32(&e.templatesLoaded) == 0 && !e.Debug {
		_, spanLoad := trace.StartSpan(ctx, "pug/loadAllTemplates")
		if err := e.LoadTemplates(""); err != nil {
			spanLoad.End()
			return nil, err
		}
		spanLoad.End()
	} else if e.Debug {
		_, spanLoad := trace.StartSpan(ctx, "pug/loadTemplate")
		spanLoad.Annotate(nil, templateName)
		if err := e.LoadTemplates(templateName); err != nil {
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

// setLoggerInfos - used to set the package variables used in the panicOrError method
func setLoggerInfos(logger flamingo.Logger, d bool) {
	if logger == nil {
		return
	}
	debugMode = d
	loggerInstance = logger
}

func panicOrError(v interface{}) {
	if loggerInstance == nil {
		panic(v)
	}
	if debugMode {
		panic(v)
	} else {
		loggerInstance.Error(v)
	}
}

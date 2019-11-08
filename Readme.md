# Pug Template

[![Go Report Card](https://goreportcard.com/badge/github.com/i-love-flamingo/pugtemplate)](https://goreportcard.com/report/github.com/i-love-flamingo/pugtemplate) [![GoDoc](https://godoc.org/github.com/i-love-flamingo/pugtemplate?status.svg)](https://godoc.org/github.com/i-love-flamingo/pugtemplate) [![Build Status](https://travis-ci.org/i-love-flamingo/pugtemplate.svg)](https://travis-ci.org/i-love-flamingo/pugtemplate)

## Pug js

[Pug](https://pugjs.org/api/getting-started.html) is a JavaScript template rendering engine.
 
## Flamingo Pug Template

The Flamingo `flamingo.me/pugtemplate` package is a flamingo template module to use Pug templates.

Pug.js is by default compiled to JavaScript, and executed as HTML. This mechanism is used to render static prototypes for the templates, so the usual HTML prototype is
just a natural artifact of this templating, instead of an extra workflow step or custom tool.

This allows frontend developers to start templating very early with very few backend support, and without the need to rewrite chunks or even learn a new template language.

The static prototype can be used to test/analyze the UI in early project phases, when the backend might not be ready yet.

The way pug.js works is essentially the following:

```
template -[tokenizer]-> tokens -[parser]-> AST -[compiler]-> JavaScript -[runtime]-> HTML
```

To integrate it in Flamingo, we save the AST (abstract syntax tree) in a JSON representations. Pugtemplate will use a parser to build an in-memory tree of the concrete building blocks. It then will use a renderer to transform these blocks into HTML Go templates as follows:

```
AST -[parser]-> Block tree -[render]-> go template -[go-template-runtime]-> HTML
```

It is possible to view the intermediate result by https://your_flamingo_url/_pugtpl/debug?tpl=home/home

### Templating

One feature of pug.js is the possibility to use arbitrary JavaScript in cases where the template syntax does not provide the some functionality. For example, loops can be written as below:

```jade
ul
  each val, index in ['zero', 'one', 'two']
    li= index + ': ' + val
```

In this example the term `['zero', 'one', 'two']` is in JavaScript. Developers will be able to use more advanced code:

```jade
- var prefix = 'foo_'
ul
  each val, index in [prefix+'zero', prefix+'one', prefix+'two']
    li= index + ': ' + val
```

The pug_template module takes this JavaScript and uses the Go-based JS engine, otto, to parse the JavaScript and transpile it into Go code.
While this works for most standard statements and language constructs (default data types such as maps, list, etc), it does not support certain things such as Object Oriented Programming or the JavaScript standard library. Only snippets of JavaScript code can be run.

However, it is possible to recreate such functionalities in a third-party module via Flamingo's template functions.
For example pug_template itself has a substitute for the JavaScript `Math` library with the `min`, `max` and `ceil` functions. Please note that these function have to use reflection and it's up to the implementation to properly reflect the functionality and handle different inputs correctly.

Nevertheless, extensive usage of JavaScript is not advised.

## Dynamic JavaScript

The Pug Template engine compiles a subset of JavaScript (ES2015) to Go templates.
This allows frontend developers to use known a syntax and techniques, instead of learning a complete new template engine.

To make this possible Flamingo rewrites the JavaScript to Go, on the fly.

## Supported JavaScript

### Standard Datatypes

```javascript
var object = {"key": "value"}

var array = [1, 2, 3, 4, 5]

var concat_string = "string" + "another string"

var add = 1 + 2

var multiply = 15 * 8
```

## Supported Pug

### Interpolation

```jade
p Text #{variable} something #{1 + 2}
```

### Conditions
```jade
if string === "true"
 p Do something
else if string === "other"
 p Do something else
else
 p Do default
```

### Loops
```jade
each value, index in  ["a", "b", "c"]
  p value #{value} at #{index}
```

### Mixins
```jade
mixin mymixin(arg1, arg2="default)
  p.something(id=arg2)= arg1
  
+mymixing("foo")

+mymixin("foo", "bar")
```

### Includes
```jade
include /mixin/mymixin

+mymixing("foo")
```

## Debugging

Templates can be debugged via `/_pugtpl/debug?tpl=pages/product/view`


## Partials Rendering

The template engine supports rendering of partials.
The idea of partials is to support progressive enhancement by being able to request just a chunk of content from the browser. The partial will still be rendered server side and should be requested by an ajax call from the browser.

Partials are requested by setting the HTTP Header `X-Partial`

The requested partials are searched in a subfolder "{templatename}.partials"

So if you have a response that will normally render like this:
```
return cc.responder.Render("folder/template")
```

And you request that page with the Header `X-Partial: foo,bar`

The engine will search for partials in `folder/template.partials/foo.pug` and `folder/template.partials/bar.pug` and just render them and return them wrapped in a JSON response like this:
```
{
    "partials": {
        "foo": "content rendered",
        "bar": "content rendered"
    }
}
```

If you call the template function `setPartialData` in this templates, you can add additional data to the json response. For example:

```
setPartialData("cartamount", 4)
```

Will result in this response:

```
{
    "partials": {
        "foo": "content rendered",
        "bar": "content rendered"
    },
    "data" : {
        "key": 4
    }
}
```
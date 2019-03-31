# Pug Template

## pug.js

[Pug](https://pugjs.org/api/getting-started.html) is a JavaScript template rendering engine.
 
## Flamingo Pug Template

The Flamingo `flamingo.me/pugtemplate` package is a flamingo template module to use pug.js templates.

Basically pug.js is by default compiled to JavaScript, and executed as HTML.

This mechanism is used to render static prototypes for the templates, so the usual HTML prototype is
just a natural artifact of this templating, instead of an extra workflow step or custom tool.

This allows frontend developers to start templating very early with very few backend support,
yet without the need to rewrite everything or even learn a new template language.

Also the static prototype can be used to test/analyze the UI in early project phases, while the backend
might not be done yet.

The way pug.js works is essentially this:

```
template -[tokenizer]-> tokens -[parser]-> AST -[compiler]-> JavaScript -[runtime]-> HTML
```

To integrate this with Flamingo we save the AST (abstract syntax tree) in a json representation.
pug_template will use a parser to build an internal in-memory tree of the concrete building
blocks, and then use a render to transform these blocks into actual go template with HTML.

```
AST -[parser]-> Block tree -[render]-> go template -[go-template-runtime]-> HTML
```

(You can also view the intermediate result under https://flamingoURL/_pugtpl/debug?tpl=home/home)

One of the features of pug.js is the possibility to use arbitrary JavaScript in case the template syntax
does not provide the correct functions. This is used for example in loops like

```jade
ul
  each val, index in ['zero', 'one', 'two']
    li= index + ': ' + val
```

The term `['zero', 'one', 'two']` is actual JavaScript, and a developer is able to use more advanced
code like

```jade
- prefix = 'foo'
ul
  each val, index in [prefix+'zero', prefix+'one', prefix+'two']
    li= index + ': ' + val
```

The pug_template module takes this JavaScript and uses the go-based JS engine otto to parse the JavaScript
and transpile it into actual go code.
While this works for some standard statements and language constructs (default datatypes such as maps, list, etc),
this does not support certain things such as OOP or the JS stdlib.

It is, however, possible to recreate such functionality in a third-party module via Flamingo's template functions.
For example pug_template itself has a substitute for the JavaScript `Math` library with the `min`, `max` and `ceil`
functions. Please note that these function have to use reflection and it's up to the implementation to properly
reflect the functionality and handle different inputs correctly.

After all extensive usage of JavaScript is not advised.

## Dynamic JavaScript

The Pug Template engine compiles a subset of JavaScript (ES2015) to Go templates.
This allows frontend developers to use known workflows and techniques, instead of learning
a complete new template engine.

To make this possible Flamingo rewrites the JavaScript to go, on the fly.

## Supported JavaScript

### Standard Datatypes

```javascript
{"key": "value"}

"string" + "another string"

1 + 2

15 * 8

[1, 2, 3, 4, 5]
```

## Support Pug

### Interpolation

```pug
p Text #{variable} something #{1 + 2}
```

### Mixins

```pug
mixin mymixin(arg1, arg2="default)
  p.something(id=arg2)= arg1
  
+mymixing("foo")

+mymixin("foo", "bar")
```

### Loops

```pug
each value, index in  ["a", "b", "c"]
  p value #{value} at #{index}
```

## Debugging

Templates can be debugged via `/_pugtpl/debug?tpl=pages/product/view`


## Partials Rendering

The template engine supports rendering of partials.
The idea of partials is to support progressive enhancement by being able to request just a part (partial) of the content from the browser.
The partial will still be rendered serverside and should be requested by an ajax call from the browser.

Partials are requested by setting the http Header `X-Partial`

The requested partials are searched in a subfolder "{templatename}.partials"

So if you have a response that will normaly render like this:
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

If you call `setPartialData` in this templates, you can add additional data to the json response. For example:

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
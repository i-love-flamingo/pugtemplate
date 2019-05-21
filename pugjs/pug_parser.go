package pugjs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"path"
	"strings"

	"github.com/pkg/errors"
)

type (
	// Attr is a simple key-value pair
	Attr struct {
		Name       string
		Val        interface{}
		MustEscape bool
	}

	// Fileref is used by include/extends
	Fileref struct {
		Type, Path string
		Line       int
	}

	// Token defines the basic token read by the tokenizer
	// Tokens form a tree, where the beginning root node starts the document
	Token struct {
		// default
		Type, Name string
		Mode, Val  string
		Line       int

		// subblock
		Block *Token
		// subblock childs
		Nodes []*Token

		// specific information
		AttributeBlocks []*Token
		Attrs           []*Attr
		MustEscape      bool
		File            *Fileref
		Filename        string
		SelfClosing     bool
		IsInline        *bool
		Obj             string
		Key             string

		// mixin
		Call bool   // mixin call?
		Args string // call args

		// if
		Test                  string // if
		Consequent, Alternate *Token // if result

		// Interpolated
		Expr string
	}
)

// Parse parses a filename into a Token-tree
func (p *renderState) Parse(file string) (*Token, error) {
	b, err := ioutil.ReadFile(path.Join(p.path, file) + ".ast.json")

	if err != nil {
		return nil, errors.Errorf("Cannot read %q", file)
	}

	return p.ParseJSON(b, file)
}

// ParseJSON parses a json into a Token-tree
func (p *renderState) ParseJSON(bytes []byte, file string) (*Token, error) {
	token := new(Token)

	err := json.Unmarshal(bytes, token)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return token, nil
}

// TokenToTemplate gets named Template from Token
func (p *renderState) TokenToTemplate(name string, t *Token) (*Template, string, error) {
	// writeTranslations, _ = os.Create("/tmp/en-nz.page-" + strings.Replace(name, "/", "-", -1) + ".json")
	// fmt.Fprintf(writeTranslations, "[\n")
	// defer func() {
	// 	fmt.Fprintf(writeTranslations, "{}]\n")
	// 	writeTranslations.(*os.File).Close()
	// }()

	template := New(name).
		Funcs(funcmap).
		Funcs(p.funcs)

	nodes := p.build(t)
	wr := new(bytes.Buffer)

	for _, b := range nodes {
		b.Render(p, wr)
	}

	for _, b := range p.mixinblocks {
		wr.WriteString("\n" + b)
	}

	for _, b := range p.mixinorder {
		wr.WriteString("\n" + p.mixin[b])
	}

	template, err := template.Parse(wr.String())

	if err != nil {
		e := err.Error() + "\n"
		for i, l := range strings.Split(wr.String(), "\n") {
			e += fmt.Sprintf("%03d: %s\n", i+1, strings.TrimSpace(strings.TrimSuffix(l, `{{- "" -}}`)))
		}
		return nil, "", errors.New(e)
	}

	for call := range p.mixincalls {
		if _, ok := p.mixin[call]; !ok {
			if p.debug {
				return nil, "", fmt.Errorf("mixin %q called but not found", call)
			}
			p.logger.Warn(fmt.Sprintf("mixin %q called but not found", call))
		}
	}

	return template, wr.String(), nil
}

func (p *renderState) build(parent *Token) []Node {
	if parent == nil {
		return nil
	}

	var res = make([]Node, 0, len(parent.Nodes))
	for _, t := range parent.Nodes {
		node := p.buildNode(t)
		if node != nil {
			res = append(res, node)
		}
	}
	return res
}

var selfclosing = map[string]bool{
	"area":    true,
	"base":    true,
	"br":      true,
	"col":     true,
	"command": true,
	"embed":   true,
	"hr":      true,
	"img":     true,
	"input":   true,
	"keygen":  true,
	"link":    true,
	"meta":    true,
	"param":   true,
	"source":  true,
	"track":   true,
	"wbr":     true,
}

func (p *renderState) buildNode(t *Token) (res Node) {
	switch t.Type {
	case "Tag":
		tag := new(Tag)
		tag.Name = t.Name
		tag.IsInline = t.IsInline
		tag.SelfClosing = t.SelfClosing
		if len(t.AttributeBlocks) > 0 {
			tag.AttributeBlocks = make([]JavaScriptExpression, 0, len(t.AttributeBlocks))
			for _, a := range t.AttributeBlocks {
				tag.AttributeBlocks = append(tag.AttributeBlocks, JavaScriptExpression(a.Val))
			}
		}
		tag.Block = Block{Nodes: p.build(t.Block)}
		if len(t.Attrs) > 0 {
			tag.Attrs = make([]Attribute, 0, len(t.Attrs))
			for _, a := range t.Attrs {
				tag.Attrs = append(tag.Attrs, Attribute{Name: a.Name, Val: JavaScriptExpression(fmt.Sprintf("%v", a.Val)), MustEscape: a.MustEscape})
			}
		}

		tag.SelfClosing = selfclosing[tag.Name]

		return tag

	case "Mixin":
		mixin := new(Mixin)
		mixin.Block = Block{Nodes: p.build(t.Block)}
		for _, a := range t.AttributeBlocks {
			mixin.AttributeBlocks = append(mixin.AttributeBlocks, JavaScriptExpression(a.Val))
		}
		mixin.Name = JavaScriptIdentifier(t.Name)
		mixin.Args = JavaScriptExpression(`[` + t.Args + `]`)
		if len(t.Attrs) > 0 {
			mixin.Attrs = make([]Attribute, 0, len(t.Attrs))
			for _, a := range t.Attrs {
				mixin.Attrs = append(mixin.Attrs, Attribute{Name: a.Name, Val: JavaScriptExpression(fmt.Sprintf("%v", a.Val)), MustEscape: a.MustEscape})
			}
		}
		mixin.Call = t.Call

		return mixin

	case "Text":
		text := new(Text)
		t.Val = strings.Replace(t.Val, "{{", `--{{--`, -1)
		t.Val = strings.Replace(t.Val, "}}", `--}}--`, -1)
		t.Val = strings.Replace(t.Val, "--{{--", `{{"{{"}}`, -1)
		t.Val = strings.Replace(t.Val, "--}}--", `{{"}}"}}`, -1)
		text.Val = t.Val
		return text

	case "Code":
		code := new(Code)
		code.Val = t.Val
		code.Block = Block{Nodes: p.build(t.Block)}
		code.IsInline = t.IsInline
		code.MustEscape = t.MustEscape
		return code

	case "Conditional":
		cond := new(Conditional)
		cond.Test = JavaScriptExpression(t.Test)
		cond.Consequent = p.buildNode(t.Consequent)
		if t.Alternate != nil {
			cond.Alternate = p.buildNode(t.Alternate)
		}
		return cond

	case "Each":
		each := new(Each)
		each.Val = JavaScriptIdentifier(t.Val)
		each.Key = JavaScriptIdentifier(t.Key)
		each.Obj = JavaScriptExpression(t.Obj)
		each.Block = Block{Nodes: p.build(t.Block)}

		return each

	case "While":
		while := new(While)
		while.Test = JavaScriptExpression(t.Test)
		while.Block = Block{Nodes: p.build(t.Block)}

		return while

	case "Doctype":
		doctype := new(Doctype)
		doctype.Val = t.Val

		return doctype

	case "NamedBlock", "Block":
		return &Block{Nodes: p.build(t)}

	case "Comment":
		return nil

	case "BlockComment":
		return nil

	case "Case":
		cas := new(Case)
		cas.Expr = JavaScriptExpression(t.Expr)
		cas.Block = Block{Nodes: p.build(t.Block)}

		return cas

	case "When":
		when := new(When)
		when.Expr = JavaScriptExpression(t.Expr)
		when.Block = Block{Nodes: p.build(t.Block)}

		return when

	case "MixinBlock":
		return new(MixinBlock)

	case "InterpolatedTag":
		interpolatedTag := new(InterpolatedTag)
		interpolatedTag.IsInline = t.IsInline
		interpolatedTag.Expr = JavaScriptExpression(t.Expr)
		interpolatedTag.SelfClosing = t.SelfClosing
		for _, a := range t.AttributeBlocks {
			interpolatedTag.AttributeBlocks = append(interpolatedTag.AttributeBlocks, JavaScriptExpression(a.Val))
		}
		interpolatedTag.Block = Block{Nodes: p.build(t.Block)}
		for _, a := range t.Attrs {
			interpolatedTag.Attrs = append(interpolatedTag.Attrs, Attribute{Name: a.Name, Val: JavaScriptExpression(fmt.Sprintf("%v", a.Val)), MustEscape: a.MustEscape})
		}

		// todo how?
		// interpolatedTag.SelfClosing = selfclosing[interpolatedTag.Name]

		return interpolatedTag

	default:
		log.Printf("%#v\n", t)
		panic(errors.Errorf("Cannot parse Pug block %#v", t))
	}
}

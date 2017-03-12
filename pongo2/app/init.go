package pongo2

import (
	"io"
	"strings"

	p2 "github.com/flosch/pongo2"
	"github.com/revel/revel"
	"github.com/tylerb/gls"
)

// Adapter for HAML Templates.
type PongoTemplate struct {
	name     string
	template *p2.Template
	engine   *PongoEngine
	*revel.BaseTemplate
}
type Pongo2BaseTag struct {
    field string
}
func (node *Pongo2BaseTag) GetField(ctx *p2.ExecutionContext) (value interface{},found bool) {
    value,found=ctx.Public[node.field]
    if !found {
        value,found=ctx.Private[node.field]
    }
    if found {
       if wrapped,ok := value.(*p2.Value);ok {
           value =wrapped.Interface()
       }
    }
    return
}
type INodeImplied struct {
	Exec func(*p2.ExecutionContext, p2.TemplateWriter) *p2.Error
}

func (i *INodeImplied) Execute(ctx *p2.ExecutionContext, w p2.TemplateWriter) *p2.Error {
	return i.Exec(ctx, w)

}
func (tmpl PongoTemplate) Name() string {
	return tmpl.name
}
func getContext() map[string]interface{} {
	return gls.Get("data").(map[string]interface{})
}

// return a 'revel.Template' from HAML's template.
func (tmpl PongoTemplate) Render(wr io.Writer, arg interface{}) (err error) {
	gls.With(gls.Values(map[interface{}]interface{}{"data": arg}), func() {
		err = tmpl.template.ExecuteWriter(p2.Context(arg.(map[string]interface{})), wr)
		if nil != err {
			if e, ok := err.(*p2.Error); ok {
				rerr := &revel.Error{
					Title:       "Template Execution Error",
					Path:        tmpl.name,
					Description: e.Error(),
					Line:        e.Line,
					//SourceLines: tmpl.Content(),
				}
				if revel.DevMode {
					rerr.SourceLines = tmpl.Content()
				}
				err = rerr
			}
		}

	})
	return err
}

func (tmpl PongoTemplate) Content() []string {
	pa, ok := tmpl.engine.loader.TemplatePaths[tmpl.Name()]
	if !ok {
		pa, ok = tmpl.engine.loader.TemplatePaths[strings.ToLower(tmpl.Name())]
	}
	content, _ := revel.ReadLines(pa)
	return content
}

// There is only a single instance of the PongoEngine initialized
type PongoEngine struct {
	loader                *revel.TemplateLoader
	templateSetBybasePath map[string]*p2.TemplateSet
	templates             map[string]*PongoTemplate
}

func (engine *PongoEngine) ParseAndAdd(templateName string, templateSource []byte, basePath *revel.BaseTemplate) error {
	templateSet := engine.templateSetBybasePath[basePath.Location()]
	if nil == templateSet {
		templateSet = p2.NewSet(basePath.Location(), p2.MustNewLocalFileSystemLoader(basePath.Location()))
		engine.templateSetBybasePath[basePath.Location()] = templateSet
	}

	tpl, err := templateSet.FromBytes(templateSource)
	if nil != err {
		_, line, description := parsePongo2Error(err)
		return &revel.Error{
			Title:       "Template Compilation Error",
			Path:        templateName,
			Description: description,
			Line:        line,
			SourceLines: strings.Split(string(templateSource), "\n"),
		}
	}

	engine.templates[templateName] = &PongoTemplate{
		name:         templateName,
		template:     tpl,
		engine:       engine,
		BaseTemplate: basePath}
	return nil
}
func (engine *PongoEngine) Name() string {
    return "pongo2"
}

func parsePongo2Error(err error) (templateName string, line int, description string) {
	pongoError := err.(*p2.Error)
	if nil != pongoError {
		return pongoError.Filename, pongoError.Line, pongoError.Error()
	}
	return "", 0, err.Error()
}

func (engine *PongoEngine) Lookup(templateName string) revel.Template {
	tpl, found := engine.templates[strings.ToLower(templateName)]
	if !found {
		return nil
	}
	return tpl
}

func init() {
	revel.RegisterTemplateLoader("pongo2", func(loader *revel.TemplateLoader) (revel.TemplateEngine, error) {
		return &PongoEngine{
			loader:                loader,
			templateSetBybasePath: map[string]*p2.TemplateSet{},
			templates:             map[string]*PongoTemplate{},
		}, nil
	})
    /*
    // TODO Dynamically call all the built in functions
    for key,templateFunction := range revel.TemplateFuncs {
        p2.RegisterTag(key, func(doc *p2.Parser, start *p2.Token, arguments *p2.Parser) (p2.INodeTag, *p2.Error) {
            evals := []p2.IEvaluator{}
            for arguments.Remaining() > 0 {
                expr, err := arguments.ParseExpression()
                evals = append(evals, expr)
                if err != nil {
                    return  nil, err
                }
            }

        return &INodeImplied{Exec: func(ctx *p2.ExecutionContext,w p2.TemplateWriter) *p2.Error {
            args := make([]interface{}, len(evals))
            for i, ev := range evals {
                obj, err := ev.Evaluate(ctx)
                if err != nil {
                    return err
                }
                args[i] = obj
            }

            v:= &tagURLForNode{evals}
            reflect.MakeFunc ....
            return v.Execute(ctx,w)
        }}, nil

        })

    }
    */
}

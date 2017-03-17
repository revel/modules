package app

import (
	"fmt"
	"github.com/revel/revel"
	"github.com/yosssi/ace"
	"html/template"
	"io"
	"strings"
)

const ACE_TEMPLATE = "ace"

// Adapter for Go Templates.
type AceTemplate struct {
	*template.Template
	engine *AceEngine
	*revel.BaseTemplate
	File  *ace.File
	Inner *ace.File
}

// A bit trick of an implementation
// If the arg contains an ace_inner field then that will be used
// to fetch a new template
func (acetmpl AceTemplate) Render(wr io.Writer, arg interface{}) error {
	// We can redirect this render to another template if the arguments contain ace_content in them
	if argmap, ok := arg.(map[string]interface{}); ok {
		if acecontentraw, ok := argmap["ace-inner"]; ok {
            acecontent := acecontentraw.(string)
			newtemplatename := acetmpl.TemplateName + "-" + acecontent
			// Now lookup the template again
			if _, ok := acetmpl.engine.templatesByName[newtemplatename]; !ok {
				if inner, ok := acetmpl.engine.templatesByName[acecontent]; !ok {
					return fmt.Errorf("Inner content %s not found in ace templates", acecontent)
				} else {
					acetmpl.engine.templatesByName[newtemplatename] = &AceTemplate{
						File:         acetmpl.File,
						Inner:        inner.File,
						engine:       acetmpl.engine,
						BaseTemplate: acetmpl.BaseTemplate}
				}

			}
			return acetmpl.engine.templatesByName[newtemplatename].renderInternal(wr, arg)
		}
	}
	return acetmpl.renderInternal(wr, arg)
}
func (acetmpl AceTemplate) renderInternal(wr io.Writer, arg interface{}) error {
	if acetmpl.Template == nil {
		// Compile the template first
        if acetmpl.Inner==nil {
            acetmpl.Inner = ace.NewFile("",nil)
        }
		source := ace.NewSource(acetmpl.File, acetmpl.Inner, acetmpl.engine.files)
		result, err := ace.ParseSource(source, acetmpl.engine.Options)

		if err != nil {
			return err
		}
		if gtemplate, err := ace.CompileResult(acetmpl.TemplateName, result, acetmpl.engine.Options); err != nil {
			return err
		} else {
			acetmpl.Template = gtemplate
		}
	}
	return acetmpl.Execute(wr, arg)
}
func (acetmpl AceTemplate) Content() []string {
	content, _ := revel.ReadLines(acetmpl.engine.loader.TemplatePaths[acetmpl.Name()])
	return content
}

type AceEngine struct {
    revel.BaseTemplateEngine
	loader          *revel.TemplateLoader
	templatesByName map[string]*AceTemplate
	files           []*ace.File
	Options         *ace.Options
}

func (engine *AceEngine) ParseAndAdd(baseTemplate *revel.BaseTemplate) error {
//	line, err := AceValidate(baseTemplate.FileBytes)

	if baseTemplate.EngineType != ACE_TEMPLATE {
		return &revel.Error{
			Title:       "Template Compilation Error",
			Path:        baseTemplate.FilePath,
			Description: "Not correct template for engine",
			Line:        1,
			SourceLines: strings.Split(string(baseTemplate.FileBytes), "\n"),
		}
	}

	file := ace.NewFile(baseTemplate.TemplateName, baseTemplate.FileBytes)
	engine.files = append(engine.files, file)
	engine.templatesByName[strings.ToLower(baseTemplate.TemplateName)] = &AceTemplate{File: file, engine: engine, BaseTemplate: baseTemplate}
	return nil
}

func (engine *AceEngine) Lookup(templateName string) revel.Template {
	if tpl, found := engine.templatesByName[strings.ToLower(templateName)]; found {

		return tpl
	}
	return nil
}

func (engine *AceEngine) Name() string {
	return ACE_TEMPLATE
}
func (engine *AceEngine) Event(action string, i interface{})  {
	if action == revel.TEMPLATE_REFRESH {
		// At this point all the templates have been passed into the
        engine.templatesByName=map[string]*AceTemplate{}
	}
}
func init() {
	revel.RegisterTemplateLoader(ACE_TEMPLATE, func(loader *revel.TemplateLoader) (revel.TemplateEngine, error) {

		return &AceEngine{
			loader:          loader,
			templatesByName: map[string]*AceTemplate{},
            Options: &ace.Options{FuncMap:revel.TemplateFuncs},
		}, nil
	})
}

package app

import (
	"fmt"
	"github.com/revel/revel"
	"github.com/yosssi/ace"
	"html/template"
	"io"
	"log"
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
	Name  string
}

// A bit trick of an implementation
// If the arg contains an ace_inner field then that will be used
// to fetch a new template
func (acetmpl AceTemplate) Render(wr io.Writer, arg interface{}) error {
	// We can redirect this render to another temaplte if the arguments contain ace_content in them
	if argmap, ok := arg.(map[string]interface{}); ok {
		if acecontent, ok := argmap["ace_inner"]; ok {
			newtemplatename := acetmpl.Name + "-" + acecontent
			// Now lookup the template again
			if _, ok := acetmpl.engine.templatesByName[newtemplatename]; !ok {
				if inner, ok := acetmpl.engine.filesByName[acecontent]; !ok {
					return fmt.Errorf("Inner content %s not found in ace templates", acecontent)
				} else {
					acetmpl.engine.filesByName[newtemplatename] = &AceTemplate{
						File:         acetmpl.File,
						Inner:        inner,
						Name:         newtemplatename,
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
		source := ace.NewSource(acetmpl.File, acetmpl.Inner, acetmpl.engine.files)
		result, err := ace.ParseSource(source, acetmpl.engine.Options)
		if err != nil {
			return err
		}
		if gtemplate, err := ace.CompileResult(acetmpl.Name, result, acetmpl.engine.Options); err != nil {
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
	loader          *revel.TemplateLoader
	templatesByName map[string]*AceTemplate
	files           []*ace.File
	Options         *ace.Options
}

func (engine *AceEngine) ParseAndAdd(templateName string, templateSourceBytes []byte, basePath *revel.BaseTemplate) error {
	line, err := AceValidate(templateSourceBytes)
	if nil != err {
		return &revel.Error{
			Title:       "Template Compilation Error",
			Path:        templateName,
			Description: err.Error(),
			Line:        line,
			SourceLines: strings.Split(string(templateSourceBytes), "\n"),
		}
	}
	file := ace.NewFile(templateName, templateSourceBytes)
	engine.files = append(engine.files, file)
	engine.templatesByName[templateName] = &AceTemplate{File: file, Name: templateName, engine: engine, BaseTemplate: basePath}
	return nil
}

func (engine *AceEngine) Lookup(templateName string, viewArgs map[string]interface{}) revel.Template {
	// Case-insensitive matching of template file name
	if tpl, found := engine.templatesByName[templateName]; found {
		return tpl
	}
	return nil
}

func (engine *AceEngine) Name() string {
	return ACE_TEMPLATE
}
func (engine *AceEngine) Event(action string, i interface{})  {
	if action == "template-refresh" {
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

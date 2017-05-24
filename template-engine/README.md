# Templating engines
 Inside this folder are template engines made and tested by the
 revel team.

## Configuration
To add a template engine you must include the module. For example
```
module.1.pongo2=github.com/revel/modules/template-engine/pongo2
module.2.ace=github.com/revel/modules/template-engine/ace
module.3.static=github.com/revel/modules/static
module.4.jobs=github.com/revel/modules/jobs
```

This specifies that the pongo2 and ace engines are added to the
main template parser, the order of the way the modules are loaded
is important as well, since that controls what view overrides another.
The first view found is always the view used, all others will be suppressed.
The two part key allows the modules to be sorted, after sorting the 
sort key is removed and in the routes table the route is referenced 
normally. 

The configuration file also needs an list of engines to be used on the views
this is where `template.engines` key in the configuration file comes into
play like

```
template.engines=pongo2,ace,go
```

The three template engines `pongo2,ace,go` will be used when rendering
templates. 


### How Revel Picks the Right Template Engine
The `template-engine` has a method called `IsEngineFor`, which accepts
the basic template information (path, and content). The engine then
can return true or false if it can parse the file or not. How it makes
this choice is up to the parser. Revel based parsers look for a 
`shebang` at the top of the template to see which one it belongs to,
they also look for a secondary extension like `foo.ace.html` would be 
identified as an `ace` template. Finally it could try to parse the code
and if that passes it can register itself for that.

## File Path Case Sensitivity
In the past we have maintained an all lower case template path, this
works in most cases but lead to some confusion. For example if you include
a file within your template you must type out the file and file path
in lower case. Now you can now specify if 
the case sensitivity is on or off. The case sensitivity can be turned on
by setting an app configuration option per template engine like 
`go.tempate.path=case` will turn on case sensitivity on the `go` 
template engine (by default it is off). 


## Go template engine
The go template engine resides in side of revel so there is no
need to import it specifically. Below are the documents for it

- By default all views are considered compilable by go, but a template
  compile error will always display the first engine in the 
  `template.engines` path
- Go templates can be set to be case sensitive by setting
`go.tempate.path=case`, default is not case sensitive. If case sensitivity
is off internal template references must be done using lower case
- All function registered in `revel.TemplateFuncs` are available for use 
inside all templates

## Developing your own template engine
Adding a new template engine to Revel requires that you
implement the following interface

```
type TemplateEngine interface {
	// #ParseAndAdd: prase template string and add template to the set.
	//   arg: basePath *BaseTemplate
	ParseAndAdd(basePath *BaseTemplate) error

	// #Lookup: returns Template corresponding to the given templateName
	//   arg: templateName string
	Lookup(templateName string) Template

	// #Event: Fired by the template loader when events occur
	//   arg: event string
	//   arg: arg interface{}
	Event(event string,arg interface{})

	// #IsEngineFor: returns true if this engine should be used to parse the file specified in baseTemplate
	//   arg: engine The calling engine
	//   arg: baseTemplate The base template
    IsEngineFor(engine TemplateEngine, baseTemplate *BaseTemplate) bool

	// #Name: Returns the name of the engine
	Name() string
}
```

There is a `BaseTemplateEngine` class which you can use as a base class 
in your engine which implements the `IsEngineFor` function

The template returned by the Lookup call needs to implement the following
```
type Template interface {
	// #Name: The name of the template.
	Name() string      // Name of template
	// #Content: The content of the template as a string (Used in error handling).
	Content() []string // Content
	// #Render: Called by the server to render the template out the io.Writer, args contains the arguements to be passed to the template.
	//   arg: wr io.Writer
	//   arg: arg interface{}
	Render(wr io.Writer, arg interface{}) error
	// #Location: The full path to the file on the disk.
	Location() string // Disk location
}
```


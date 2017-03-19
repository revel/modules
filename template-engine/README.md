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


### How Revel Picks the Right parser
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
works in most cases but not all, you can now specify that the template
modules we provide allow you to switch path sensitivity on or off.


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

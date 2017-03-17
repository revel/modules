# Ace 
 The [ACE](https://github.com/yosssi/ace) Templating Plugin

Ace is a little different of a templating system, its output is a 
standard go template but there is no concept of template sets, 
instead you build a composite template using
 a *base* template and an *inner* template. The 
 *inner* template can only contain items like : 
   ```
= content main
  h2 Inner Template - Main : {{.Msg}}

= content sub
  h3 Inner Template - Sub : {{.Msg}}
     
   ```
The base template can contain items like 
```
= doctype html
html lang=en
  head
    meta charset=utf-8
    title Ace example
    = css
      h1 { color: blue; }
  body
    h1 Base Template : {{.Msg}}
    #container.wrapper
      = yield main
      = yield sub
      = include inc .Msg
    = javascript
      alert('{{.Msg}}');
```

You are allowed to include one *inner* template with the base template,
to do so in revel you can extend your controller from the ace controller
and call `RenderAceTemplate(base ,inner string)` which will insert
the inner template using the outer template.
 
 The ace engine requires that you explicitly set the template type on the
 template itself by either using the shebang method on the first line
 like `#! ace` or having the file name like `template.ace.html` 
 either method will work. 

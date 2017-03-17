# Ace 
 The [ACE](https://github.com/yosssi/ace) Templating Plugin

Ace is a little differnt of a templating system, its output is a 
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

Because of this it is impossible to precompile the template as one unit
until we know which pieces go together. 

So when render is called and you have 

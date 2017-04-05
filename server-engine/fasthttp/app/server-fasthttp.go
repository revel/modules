package fasthttp

import (
    "fmt"
    "github.com/revel/revel"
    "github.com/valyala/fasthttp"
    "mime/multipart"
    "net/url"
    "io"
    "time"
    "net"
    "net/http"
    "strconv"
)

type ServerFastHTTP struct {
    Server *fasthttp.Server
    ServerInit *revel.EngineInit
}

func init() {
    fastHTTP := &ServerFastHTTP{}
    revel.RegisterServerEngine(fastHTTP)
}

func (f *ServerFastHTTP) Init(init *revel.EngineInit) {
    fastHttpRequestStack       = revel.NewStackLock(revel.Config.IntDefault("server.request.stack",100), func() interface{} { return &FastHttpRequest{header: &FastHttpHeader{}} })
    fastHttpResponseStack      = revel.NewStackLock(revel.Config.IntDefault("server.response.stack",100), func() interface{} { return &FastHttpResponse{header: &FastHttpHeader{isResponse:true}} })
    fastHttpMultipartFormStack = revel.NewStackLock(revel.Config.IntDefault("server.form.stack",100), func() interface{} { return &FastHttpMultipartForm{} })

    requestHandler := func(ctx *fasthttp.RequestCtx) {
        f.RequestHandler(ctx)
    }
    f.ServerInit = init
    f.Server= &fasthttp.Server{

            ReadTimeout:  time.Duration(revel.Config.IntDefault("http.timeout.read", 0)) * time.Second,
            WriteTimeout: time.Duration(revel.Config.IntDefault("http.timeout.write", 0)) * time.Second,
            Handler:requestHandler,
            Logger:revel.ERROR,
    }

}
// Handler is assigned in the Init
func (f *ServerFastHTTP) Start() {
    go func() {
        time.Sleep(100 * time.Millisecond)
        fmt.Printf("\nListening on fasthttp %s...\n", f.ServerInit.Address)
    }()
    if revel.HTTPSsl {
        if f.ServerInit.Network != "tcp" {
            // This limitation is just to reduce complexity, since it is standard
            // to terminate SSL upstream when using unix domain sockets.
            revel.ERROR.Fatalln("SSL is only supported for TCP sockets. Specify a port to listen on.")
        }
        revel.ERROR.Fatalln("Failed to listen:",
            f.Server.ListenAndServeTLS(f.ServerInit.Address, revel.HTTPSslCert, revel.HTTPSslKey))
    } else {
        listener, err := net.Listen(f.ServerInit.Network, f.ServerInit.Address)
        if err != nil {
            revel.ERROR.Fatalln("Failed to listen:", err)
        }
        println("Listening fasthttp ", f.ServerInit.Network, f.ServerInit.Address)
        // revel.ERROR.Fatalln("Failed to serve:", f.Server.ListenAndServe(f.ServerInit.Address))
        revel.ERROR.Fatalln("Failed to serve:", f.Server.Serve(listener))
        println("***ENDING ***")
    }

}
func (f *ServerFastHTTP) RequestHandler(ctx *fasthttp.RequestCtx) {
    // TODO this
    //if maxRequestSize := int64(revel.Config.IntDefault("http.maxrequestsize", 0)); maxRequestSize > 0 {
     //   buffer := &bytes.Buffer{}
     //   err := ctx.Request.ReadLimitBody(buffer,maxRequestSize)
     //   if err!=nil {
     //       // Send the error back to the client
     //       ctx.SetStatusCode(http.StatusRequestEntityTooLarge)
     //       return
     //   }
    //}
    response := fastHttpResponseStack.Pop().(*FastHttpResponse)
    request := fastHttpRequestStack.Pop().(*FastHttpRequest)
    defer func() {
        fastHttpResponseStack.Push(response)
        fastHttpRequestStack.Push(request)
    }()
    request.Set(ctx)
    response.Set(ctx)
    f.ServerInit.Callback(response, request, nil)
}


func (f *ServerFastHTTP) Event(event int, args interface{}) {

    switch event {
    case revel.ENGINE_EVENT_PREINIT:
    case revel.ENGINE_EVENT_STARTUP:

    }

}
func (f *ServerFastHTTP) Name() string {
    return "fasthttp"
}
func (f *ServerFastHTTP) Engine() interface{} {
    return f
}
func (g *ServerFastHTTP) Stats() map[string]interface{} {
    return map[string]interface{}{
        "FastHTTP Engine Requests":fastHttpRequestStack.String(),
        "FastHTTP Engine Response":fastHttpResponseStack.String(),
        "FastHTTP Engine Forms":fastHttpMultipartFormStack.String(),
    }
}

type (
    FastHttpRequest struct {
        toQuery         bool
        query          url.Values
        Original        *fasthttp.RequestCtx
        FormParsed      bool
        form            url.Values
        MultiFormParsed bool
        // WebSocket       *websocket.Conn
        ParsedForm      *FastHttpMultipartForm
        header          *FastHttpHeader
    }

    FastHttpResponse struct {
        Original *fasthttp.RequestCtx
        header   *FastHttpHeader
        Writer   io.Writer
    }
    FastHttpMultipartForm struct {
        Form *multipart.Form
    }
    FastHttpHeader struct {
        Source     interface{}
        isResponse bool
    }
    FastHttpCookie []byte
)

var (
    fastHttpRequestStack       *revel.SimpleLockStack
    fastHttpResponseStack      *revel.SimpleLockStack
    fastHttpMultipartFormStack *revel.SimpleLockStack
)

func (r *FastHttpRequest) GetQuery() url.Values {
    if !r.toQuery {
        // Attempt to convert to query
        r.query = url.Values{}
        r.Original.QueryArgs().VisitAll(func (key, value[]byte) {
            r.query.Set(string(key),string(value))
        })
        r.toQuery = true
    }
    return r.query
}
func (r *FastHttpRequest) GetRequestURI() string {
    return string(r.Original.RequestURI())
}
func (r *FastHttpRequest) GetRemoteAddr() string {
    return r.Original.RemoteAddr().String()
}
func (r *FastHttpRequest) GetForm() (url.Values, error) {
    if !r.FormParsed {
        r.form = url.Values{}
        r.Original.PostArgs().VisitAll(func (key , value[]byte) {
            println("Set value", string(key),string(value))
            r.query.Set(string(key),string(value))
        })
        r.FormParsed = true
    }
    return r.form, nil
}
func (r *FastHttpRequest) GetMultipartForm(maxsize int64) (revel.ServerMultipartForm, error) {
    if !r.MultiFormParsed {
        form,err:= r.Original.MultipartForm()
        if err!=nil {
            return nil,err
        }

        r.ParsedForm = fastHttpMultipartFormStack.Pop().(*FastHttpMultipartForm)
        r.ParsedForm.Form = form
    }

    return r.ParsedForm, nil
}
func (r *FastHttpRequest) GetHeader() revel.ServerHeader {
    return r.header
}
func (r *FastHttpRequest) GetRaw() interface{} {
    return r.Original
}
func (r *FastHttpRequest) GetMethod() string {
    return string(r.Original.Method())
}
func (r *FastHttpRequest) GetPath() string {
    return string(r.Original.Path())
}
func (r *FastHttpRequest) GetHost() string {
    return string(r.Original.Request.Host())
}
func (r *FastHttpRequest) Set(req *fasthttp.RequestCtx) {
    r.Original = req
    r.header.Source = r

}
func (r *FastHttpRequest) Destroy() {
    r.header.Source = nil
    r.Original = nil
    r.FormParsed = false
    r.MultiFormParsed = false
    r.ParsedForm = nil
    r.toQuery = false

}
func (r *FastHttpResponse) GetWriter() io.Writer {
    return r.Writer
}
func (r *FastHttpResponse) Header() revel.ServerHeader {
    return r.header
}
func (r *FastHttpResponse) GetRaw() interface{} {
    return r.Original
}
func (r *FastHttpResponse) WriteStream(name string, contentlen int64, modtime time.Time,reader io.Reader) error {

    // do a simple io.Copy, we do it directly into the writer which may be configured to be a compressed
    // writer
    ius := r.Original.Request.Header.Peek("If-Unmodified-Since")
    if t, err := http.ParseTime(string(ius)); ius!=nil && err == nil && !modtime.IsZero() {
        // The Date-Modified header truncates sub-second precision, so
        // use mtime < t+1s instead of mtime <= t to check for unmodified.
        if modtime.Before(t.Add(1 * time.Second)) {
            h := r.Original.Response.Header
            h.Del("Content-Type")
            h.Del("Content-Length")
            if h.Peek("Etag") != nil {
                h.Del("Last-Modified")
            }
            h.SetStatusCode(http.StatusNotModified)
            return nil
        }
    }

    if contentlen != -1 {
        r.Original.Response.Header.Set("Content-Length", strconv.FormatInt(contentlen, 10))
    }
    if _, err := io.Copy(r.Writer, reader); err != nil {
       r.Original.Response.Header.SetStatusCode(http.StatusInternalServerError)
       return err
    } else {
       r.Original.Response.Header.SetStatusCode(http.StatusOK)
    }

    return nil
}
func (r *FastHttpResponse) Destroy() {
    if c,ok:=r.Writer.(io.Closer);ok {
        c.Close()
    }
    r.header.Source = nil
    r.Original = nil
    r.Writer = nil

}
func (r *FastHttpResponse) Set(w *fasthttp.RequestCtx) {
    r.Original = w
    r.header.Source = r
    r.Writer = w.Response.BodyWriter()
}
func (r *FastHttpResponse) SetWriter(writer io.Writer) {
    r.Writer = writer
}
func (r *FastHttpHeader) SetCookie(cookie string) {
    if r.isResponse {
        r.Source.(*FastHttpResponse).Original.Response.Header.Add("Set-Cookie", cookie)
    }
}
func (r *FastHttpHeader) GetCookie(key string) (value revel.ServerCookie, err error) {
    if !r.isResponse {
        var cookie []byte
        if cookie = r.Source.(*FastHttpRequest).Original.Request.Header.Cookie(key); cookie != nil {
            value = FastHttpCookie(cookie)
        } else {
            err = http.ErrNoCookie
        }

    }
    return
}
func (r *FastHttpHeader) Set(key string, value string) {
    if r.isResponse {
        r.Source.(*FastHttpResponse).Original.Response.Header.Set(key, value)
    }
}
func (r *FastHttpHeader) Add(key string, value string) {
    if r.isResponse {
        r.Source.(*FastHttpResponse).Original.Response.Header.Add(key, value)
    }
}
func (r *FastHttpHeader) Del(key string) {
    if r.isResponse {
        r.Source.(*FastHttpResponse).Original.Response.Header.Del(key)
    }
}
func (r *FastHttpHeader) Get(key string) (value string) {
    if !r.isResponse {
        value = string(r.Source.(*FastHttpRequest).Original.Request.Header.Peek(key))
    } else {
        value = string(r.Source.(*FastHttpResponse).Original.Response.Header.Peek(key))
    }
    return
}
func (r *FastHttpHeader) SetStatus(statusCode int) {
    if r.isResponse {
        r.Source.(*FastHttpResponse).Original.Response.SetStatusCode(statusCode)
    }
}
func (r FastHttpCookie) GetValue() string {
    return string(r)
}
func (f *FastHttpMultipartForm) GetFile() map[string][]*multipart.FileHeader {
    return f.Form.File
}
func (f *FastHttpMultipartForm) GetValue() url.Values {
    return url.Values(f.Form.Value)
}
func (f *FastHttpMultipartForm) RemoveAll() error {
    return f.Form.RemoveAll()
}

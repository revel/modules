package fasthttp

import (
	"bytes"
	"fmt"
	"github.com/revel/revel"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/reuseport"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
	"github.com/revel/revel/utils"
)

// The engine
type FastHTTPServer struct {
	Server           *fasthttp.Server    // The server
	ServerInit       *revel.EngineInit   // The server initialization data
	MaxMultipartSize int64               // The max form size
	HttpMuxList      revel.ServerMuxList // The list of muxers
	HasAppMux        bool                // True if has a mux
	signalChan       chan os.Signal      // The channel to stop the server
	graceful         net.Listener        // The graceful listener
}

// The server log
var serverLog = revel.AppLog

// Called to initialize
func init() {
	revel.RegisterServerEngine("fasthttp", func() revel.ServerEngine {
		return &FastHTTPServer{}
	})
	revel.RegisterModuleInit(func(m *revel.Module) {
		serverLog = m.Log
	})
}

// Called to initialize the FastHttpServer
func (f *FastHTTPServer) Init(init *revel.EngineInit) {
	f.MaxMultipartSize = int64(revel.Config.IntDefault("server.request.max.multipart.filesize", 32)) << 20 /* 32 MB */
	fastHttpContextStack = utils.NewStackLock(revel.Config.IntDefault("server.context.stack", 100),
		revel.Config.IntDefault("server.context.maxstack", 200),
		func() interface{} { return NewFastHttpContext(f) })
	fastHttpMultipartFormStack = utils.NewStackLock(revel.Config.IntDefault("server.form.stack", 100),
		revel.Config.IntDefault("server.form.maxstack", 200),
		func() interface{} { return &FastHttpMultipartForm{} })

	requestHandler := func(ctx *fasthttp.RequestCtx) {
		f.RequestHandler(ctx)
	}
	// Adds the mux list
	f.HttpMuxList = init.HTTPMuxList
	sort.Sort(f.HttpMuxList)
	f.HasAppMux = len(f.HttpMuxList) > 0

	f.signalChan = make(chan os.Signal)

	f.ServerInit = init
	f.Server = &fasthttp.Server{
		ReadTimeout:  time.Duration(revel.Config.IntDefault("http.timeout.read", 0)) * time.Second,
		WriteTimeout: time.Duration(revel.Config.IntDefault("http.timeout.write", 0)) * time.Second,
		Handler:      requestHandler,
	}

}

// Handler is assigned in the Init
func (f *FastHTTPServer) Start() {
	go func() {
		time.Sleep(100 * time.Millisecond)
		fmt.Printf("\nListening on fasthttp %s...\n", f.ServerInit.Address)
	}()
	if f.ServerInit.Network == "tcp" {
		f.ServerInit.Network = "tcp4"
	}
	listener, err := reuseport.Listen(f.ServerInit.Network, f.ServerInit.Address)
	if err != nil {
		serverLog.Fatal("Failed to listen http:", "error", err, "network", f.ServerInit.Network, "address", f.ServerInit.Address)
	}

	// create a graceful shutdown listener
	duration := 5 * time.Second
	f.graceful = NewGracefulListener(listener, duration)

	if revel.HTTPSsl {
		if f.ServerInit.Network != "tcp" {
			// This limitation is just to reduce complexity, since it is standard
			// to terminate SSL upstream when using unix domain sockets.
			serverLog.Fatal("SSL is only supported for TCP sockets. Specify a port to listen on.")
		}
		serverLog.Fatal("Failed to listen https:", "error",
			f.Server.ServeTLS(f.graceful, revel.HTTPSslCert, revel.HTTPSslKey))
	} else {

		serverLog.Info("Listening fasthttp ", f.ServerInit.Network, f.ServerInit.Address)
		serverLog.Warn("Server exiting", "error", f.Server.Serve(f.graceful))
	}
}

// The root handler
func (f *FastHTTPServer) RequestHandler(ctx *fasthttp.RequestCtx) {
	// This section is called if the developer has added custom mux to the app
	if f.HasAppMux && f.handleAppMux(ctx) {
		return
	}
	f.handleMux(ctx)
}

// Handle the request and response for the servers mux
func (f *FastHTTPServer) handleAppMux(ctx *fasthttp.RequestCtx) (result bool) {
	// Check the prefix and split them
	cpath := string(ctx.Path())
	requestPath := path.Clean(cpath)
	if handler, hasHandler := f.HttpMuxList.Find(requestPath); hasHandler {
		result = true
		clientIP := HttpClientIP(ctx)
		localLog := serverLog.New("ip", clientIP,
			"path", cpath, "method", string(ctx.Method()))
		defer func() {
			if err := recover(); err != nil {
				localLog.Error("An error was caught using the handler", "path", requestPath, "error", err)
				fmt.Fprintf(ctx, "Unable to handle response for third part mux %v", err)
				ctx.Response.SetStatusCode(http.StatusInternalServerError)
				return
			}
		}()
		start := time.Now()
		handler.(fasthttp.RequestHandler)(ctx)
		localLog.Info("Request Stats",
			"start", start,
			"duration_seconds", time.Since(start).Seconds(), "section", "requestlog",
		)
		return
	}
	return
}

// ClientIP method returns client IP address from HTTP request.
//
// Note: Set property "app.behind.proxy" to true only if Revel is running
// behind proxy like nginx, haproxy, apache, etc. Otherwise
// you may get inaccurate Client IP address. Revel parses the
// IP address in the order of X-Forwarded-For, X-Real-IP.
//
// By default revel will get http.Request's RemoteAddr
func HttpClientIP(ctx *fasthttp.RequestCtx) string {
	if revel.Config.BoolDefault("app.behind.proxy", false) {
		// Header X-Forwarded-For
		if fwdFor := strings.TrimSpace(string(ctx.Request.Header.Peek(revel.HdrForwardedFor))); fwdFor != "" {
			index := strings.Index(fwdFor, ",")
			if index == -1 {
				return fwdFor
			}
			return fwdFor[:index]
		}

		// Header X-Real-Ip
		if realIP := strings.TrimSpace(string(ctx.Request.Header.Peek(revel.HdrRealIP))); realIP != "" {
			return realIP
		}
	}

	return ctx.RemoteIP().String()
}

// Handle response
func (f *FastHTTPServer) handleMux(ctx *fasthttp.RequestCtx) {
	// TODO limit max size of body that can be read
	//if maxRequestSize := int64(revel.Config.IntDefault("http.maxrequestsize", 0)); maxRequestSize > 0 {
	//   buffer := &bytes.Buffer{}
	//   err := ctx.Request.ReadLimitBody(buffer,maxRequestSize)
	//   if err!=nil {
	//       // Send the error back to the client
	//       ctx.SetStatusCode(http.StatusRequestEntityTooLarge)
	//       return
	//   }
	//}
	context := fastHttpContextStack.Pop().(*FastHttpContext)
	defer func() {
		fastHttpContextStack.Push(context)
	}()
	context.SetContext(ctx)
	f.ServerInit.Callback(context)
}

// Handle an event generated from Revel
func (f *FastHTTPServer) Event(event revel.Event, args interface{}) revel.EventResponse {

	switch event {
	case revel.ENGINE_STARTED:
		signal.Notify(f.signalChan, os.Interrupt, os.Kill)
		go func() {
			_ = <-f.signalChan
			serverLog.Info("Received quit singal Please wait ... ")
			revel.StopServer(nil)
		}()
	case revel.ENGINE_SHUTDOWN_REQUEST:
		if err := f.graceful.Close(); err != nil {
			serverLog.Fatal("Failed to close fasthttp server gracefully, exiting using os.exit", "error", err)
		}
	default:

	}
	return 0
}

// Return the engine name
func (f *FastHTTPServer) Name() string {
	return "fasthttp"
}

// Return the engine
func (f *FastHTTPServer) Engine() interface{} {
	return f
}

// Returns stats for the engine
func (g *FastHTTPServer) Stats() map[string]interface{} {
	return map[string]interface{}{
		"FastHTTP Engine Context": fastHttpContextStack.String(),
		"FastHTTP Engine Forms":   fastHttpMultipartFormStack.String(),
	}
}

type (
	// The context
	FastHttpContext struct {
		Request  *FastHttpRequest  // The request
		Response *FastHttpResponse // The respnse
	}

	// The request
	FastHttpRequest struct {
		url             *url.URL             // The url of request
		toQuery         bool                 // True if converted to query
		query           url.Values           // The translated query
		Original        *fasthttp.RequestCtx // The original
		FormParsed      bool                 // True if the form was parsed
		form            url.Values           // The form values
		MultiFormParsed bool                 // True if multipart form
		// WebSocket       *websocket.Conn // No websocket
		ParsedForm *FastHttpMultipartForm // The parsed form
		header     *FastHttpHeader        // The request header
		Engine     *FastHTTPServer        // The response header
	}

	// The response
	FastHttpResponse struct {
		Original *fasthttp.RequestCtx // The original
		header   *FastHttpHeader      // The header
		Writer   io.Writer            // The writer
		Engine   *FastHTTPServer      // The engine
	}
	// The form
	FastHttpMultipartForm struct {
		Form *multipart.Form // The embedded form
	}
	// The header
	FastHttpHeader struct {
		Source     interface{} // The source
		isResponse bool        // True if this is a response header
	}
	// The cookie
	FastHttpCookie []byte // The cookie
)

var (
	fastHttpContextStack       *utils.SimpleLockStack // context stack
	fastHttpMultipartFormStack *utils.SimpleLockStack // form stack
)

// Create a new context
func NewFastHttpContext(instance *FastHTTPServer) *FastHttpContext {
	if instance == nil {
		instance = &FastHTTPServer{MaxMultipartSize: 32 << 20}
	}
	c := &FastHttpContext{
		Request: &FastHttpRequest{header: &FastHttpHeader{isResponse: false},
			Engine: instance},
		Response: &FastHttpResponse{header: &FastHttpHeader{isResponse: true},
			Engine: instance},
	}
	c.Response.header.Source = c.Response
	c.Request.header.Source = c.Request
	return c
}

// Called to get the request
func (c *FastHttpContext) GetRequest() revel.ServerRequest {
	return c.Request
}

// Called to get the response
func (c *FastHttpContext) GetResponse() revel.ServerResponse {
	return c.Response
}

// Called to set the context
func (c *FastHttpContext) SetContext(context *fasthttp.RequestCtx) {
	c.Response.SetContext(context)
	c.Request.SetContext(context)
}

// Called to destroy the context
func (c *FastHttpContext) Destroy() {
	c.Response.Destroy()
	c.Request.Destroy()
}

// Gets the value from the request
func (r *FastHttpRequest) Get(key int) (value interface{}, err error) {
	switch key {
	case revel.HTTP_SERVER_HEADER:
		value = r.GetHeader()
	case revel.HTTP_MULTIPART_FORM:
		value, err = r.GetMultipartForm()
	case revel.HTTP_QUERY:
		value = r.GetQuery()
	case revel.HTTP_FORM:
		value, err = r.GetForm()
	case revel.HTTP_REQUEST_URI:
		value = string(r.Original.RequestURI())
	case revel.HTTP_REMOTE_ADDR:
		value = r.Original.RemoteAddr().String()
	case revel.HTTP_METHOD:
		value = string(r.Original.Method())
	case revel.HTTP_PATH:
		value = string(r.Original.Path())
	case revel.HTTP_HOST:
		value = string(r.Original.Request.Host())
	case revel.HTTP_URL:
		if r.url == nil {
			r.url, _ = url.Parse(string(r.Original.Request.URI().FullURI()))
		}
		value = r.url
	case revel.HTTP_BODY:
		value = bytes.NewBuffer(r.Original.Request.Body())
	default:
		err = revel.ENGINE_UNKNOWN_GET
	}

	return
}

// Sets the request with the value
func (r *FastHttpRequest) Set(key int, value interface{}) bool {
	return false
}

// Returns the query string
func (r *FastHttpRequest) GetQuery() url.Values {
	if !r.toQuery {
		// Attempt to convert to query
		r.query = url.Values{}
		r.Original.QueryArgs().VisitAll(func(key, value []byte) {
			r.query.Set(string(key), string(value))
		})
		r.toQuery = true
	}
	return r.query
}

// Returns the form
func (r *FastHttpRequest) GetForm() (url.Values, error) {
	if !r.FormParsed {
		r.form = url.Values{}
		r.Original.PostArgs().VisitAll(func(key, value []byte) {
			r.query.Set(string(key), string(value))
		})
		r.FormParsed = true
	}
	return r.form, nil
}

// Returns the form
func (r *FastHttpRequest) GetMultipartForm() (revel.ServerMultipartForm, error) {
	if !r.MultiFormParsed {
		// TODO Limit size r.Engine.MaxMultipartSize
		form, err := r.Original.MultipartForm()
		if err != nil {
			return nil, err
		}

		r.ParsedForm = fastHttpMultipartFormStack.Pop().(*FastHttpMultipartForm)
		r.ParsedForm.Form = form
	}

	return r.ParsedForm, nil
}

// Returns the request header
func (r *FastHttpRequest) GetHeader() revel.ServerHeader {
	return r.header
}

// Returns the raw request
func (r *FastHttpRequest) GetRaw() interface{} {
	return r.Original
}

// Sets the context
func (r *FastHttpRequest) SetContext(req *fasthttp.RequestCtx) {
	r.Original = req

}

// Called when request is done
func (r *FastHttpRequest) Destroy() {
	r.Original = nil
	r.FormParsed = false
	r.MultiFormParsed = false
	r.ParsedForm = nil
	r.toQuery = false

}

// gets the key from the response
func (r *FastHttpResponse) Get(key int) (value interface{}, err error) {
	switch key {
	case revel.HTTP_SERVER_HEADER:
		value = r.Header()
	case revel.HTTP_STREAM_WRITER:
		value = r
	case revel.HTTP_WRITER:
		value = r.Writer
	default:
		err = revel.ENGINE_UNKNOWN_GET
	}
	return
}

// Sets the key with the value
func (r *FastHttpResponse) Set(key int, value interface{}) (set bool) {
	switch key {
	case revel.ENGINE_RESPONSE_STATUS:
		r.Header().SetStatus(value.(int))
		set = true
	case revel.HTTP_WRITER:
		r.SetWriter(value.(io.Writer))
		set = true
	}
	return
}

// Return the response writer
func (r *FastHttpResponse) GetWriter() io.Writer {
	return r.Writer
}

// Return the header
func (r *FastHttpResponse) Header() revel.ServerHeader {
	return r.header
}

// Returns the raw response
func (r *FastHttpResponse) GetRaw() interface{} {
	return r.Original
}

// Writes a stream to the response
func (r *FastHttpResponse) WriteStream(name string, contentlen int64, modtime time.Time, reader io.Reader) error {

	// do a simple io.Copy, we do it directly into the writer which may be configured to be a compressed
	// writer
	ius := r.Original.Request.Header.Peek("If-Unmodified-Since")
	if t, err := http.ParseTime(string(ius)); ius != nil && err == nil && !modtime.IsZero() {
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

// Called to reset this response
func (r *FastHttpResponse) Destroy() {
	if c, ok := r.Writer.(io.Closer); ok {
		c.Close()
	}
	r.Original = nil
	r.Writer = nil

}

// Sets the context
func (r *FastHttpResponse) SetContext(w *fasthttp.RequestCtx) {
	r.Original = w
	r.Writer = w.Response.BodyWriter()
}

// Sets the writer
func (r *FastHttpResponse) SetWriter(writer io.Writer) {
	r.Writer = writer
}

// Sets a cookie
func (r *FastHttpHeader) SetCookie(cookie string) {
	if r.isResponse {
		r.Source.(*FastHttpResponse).Original.Response.Header.Add("Set-Cookie", cookie)
	}
}

// Returns a cookie
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

// Sets (replaces) a header key
func (r *FastHttpHeader) Set(key string, value string) {
	if r.isResponse {
		r.Source.(*FastHttpResponse).Original.Response.Header.Set(key, value)
	}
}

// Adds a header key
func (r *FastHttpHeader) Add(key string, value string) {
	if r.isResponse {
		r.Source.(*FastHttpResponse).Original.Response.Header.Add(key, value)
	}
}

// Deletes a header key
func (r *FastHttpHeader) Del(key string) {
	if r.isResponse {
		r.Source.(*FastHttpResponse).Original.Response.Header.Del(key)
	}
}

// Returns the header keys
func (r *FastHttpHeader) GetKeys() (value []string) {
	addValue := func(k, v []byte) {
		found := false
		key := string(k)
		for _, r := range value {
			if key == r {
				found = true
				break
			}
		}
		if !found {
			value = append(value, key)
		}

	}
	if !r.isResponse {
		r.Source.(*FastHttpRequest).Original.Request.Header.VisitAll(addValue)

	} else {
		r.Source.(*FastHttpResponse).Original.Response.Header.VisitAll(addValue)
	}
	return
}

// returns the header value
func (r *FastHttpHeader) Get(key string) (value []string) {
	if !r.isResponse {
		value = strings.Split(string(r.Source.(*FastHttpRequest).Original.Request.Header.Peek(key)), ",")
	} else {
		value = strings.Split(string(r.Source.(*FastHttpResponse).Original.Response.Header.Peek(key)), ",")
	}
	return
}

// Sets the header status
func (r *FastHttpHeader) SetStatus(statusCode int) {
	if r.isResponse {
		r.Source.(*FastHttpResponse).Original.Response.SetStatusCode(statusCode)
	}
}

// Returns the cookie value
func (r FastHttpCookie) GetValue() string {
	return string(r)
}

// Returns the files for a form
func (f *FastHttpMultipartForm) GetFiles() map[string][]*multipart.FileHeader {
	return f.Form.File
}

// Returns the values for a form
func (f *FastHttpMultipartForm) GetValues() url.Values {
	return url.Values(f.Form.Value)
}

// Remove all the vlaues from a form
func (f *FastHttpMultipartForm) RemoveAll() error {
	return f.Form.RemoveAll()
}

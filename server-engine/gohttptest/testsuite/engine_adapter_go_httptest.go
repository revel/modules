package testsuite

import (
	"fmt"
	"net/http"
	"time"

	"github.com/revel/revel"
	"github.com/revel/revel/utils"
	"golang.org/x/net/websocket"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"net/url"
	"strconv"
)

var serverLog = revel.AppLog

// Register the GoHttpServer test engine
func init() {
	revel.RegisterServerEngine("go-test", func() revel.ServerEngine {
		return &GoHttpServer{}
	})
	revel.RegisterModuleInit(func(m *revel.Module) {
		serverLog = m.Log
	})
}

type GoHttpServer struct {
	Server           *http.Server      // Although unused this is here just to support possible code requests
	ServerInit       *revel.EngineInit // The intiialization data
	MaxMultipartSize int64
	TestChannel      chan *TestRequest
	StartedChan      chan bool
}

func (g *GoHttpServer) Init(init *revel.EngineInit) {
	g.TestChannel = make(chan *TestRequest)
	g.StartedChan = make(chan bool)
	g.MaxMultipartSize = int64(revel.Config.IntDefault("server.request.max.multipart.filesize", 32)) << 20 /* 32 MB */
	goContextStack = utils.NewStackLock(revel.Config.IntDefault("server.context.stack", 100),
		revel.Config.IntDefault("server.context.maxstack", 200),
		func() interface{} {
			return NewGOContext(g)
		})
	goMultipartFormStack = utils.NewStackLock(revel.Config.IntDefault("server.form.stack", 100),
		revel.Config.IntDefault("server.form.maxstack", 200),
		func() interface{} { return &GoMultipartForm{} })
	g.ServerInit = init
	g.Server = &http.Server{
		Addr: init.Address,
		Handler: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			g.Handle(writer, request)
		}),
		ReadTimeout:  time.Duration(revel.Config.IntDefault("http.timeout.read", 0)) * time.Second,
		WriteTimeout: time.Duration(revel.Config.IntDefault("http.timeout.write", 0)) * time.Second,
	}
}

// The server is started and continues to listen on the TestChannel until it is ended
func (g *GoHttpServer) Start() {
	go func() {
		time.Sleep(100 * time.Millisecond)
		fmt.Printf("Revel Listening on %d...\n", g.ServerInit.Port)
	}()
	// The idea is for this thread to wait for requests through the channel
	g.StartedChan <- true
	for {
		task, more := <-g.TestChannel
		if more {
			task.testSuite.Response = httptest.NewRecorder()
			g.Handle(task.testSuite.Response, task.Request)
			task.testSuite.ResponseChannel <- true
		} else {
			break
		}

	}

}

func (g *GoHttpServer) Handle(w http.ResponseWriter, r *http.Request) {
	if maxRequestSize := int64(revel.Config.IntDefault("http.maxrequestsize", 0)); maxRequestSize > 0 {
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestSize)
	}

	upgrade := r.Header.Get("Upgrade")
	context := goContextStack.Pop().(*GoContext)
	defer func() {
		goContextStack.Push(context)
	}()
	context.Request.SetRequest(r)
	context.Response.SetResponse(w)

	if upgrade == "websocket" || upgrade == "Websocket" {
		websocket.Handler(func(ws *websocket.Conn) {
			//Override default Read/Write timeout with sane value for a web socket request
			if err := ws.SetDeadline(time.Now().Add(time.Hour * 24)); err != nil {
				serverLog.Error("SetDeadLine failed:", "error", err)
			}
			r.Method = "WS"
			context.Request.WebSocket = ws
			context.WebSocket = &GoWebSocket{Conn: ws, GoResponse: *context.Response}
			g.ServerInit.Callback(context)
		}).ServeHTTP(w, r)
	} else {
		g.ServerInit.Callback(context)
	}
}

const GO_NATIVE_TEST_SERVER_ENGINE = "go-test"

func (g *GoHttpServer) Name() string {
	return GO_NATIVE_TEST_SERVER_ENGINE
}

func (g *GoHttpServer) Stats() map[string]interface{} {
	return map[string]interface{}{
		"Go Engine Context": goContextStack.String(),
		"Go Engine Forms":   goMultipartFormStack.String(),
	}
}

func (g *GoHttpServer) Engine() interface{} {
	return g.Server
}

func (g *GoHttpServer) Event(event revel.Event, args interface{}) revel.EventResponse {
	return 0
}

type (
	GoContext struct {
		Request   *GoRequest
		Response  *GoResponse
		WebSocket *GoWebSocket
	}
	GoRequest struct {
		Original        *http.Request
		FormParsed      bool
		MultiFormParsed bool
		WebSocket       *websocket.Conn
		ParsedForm      *GoMultipartForm
		Goheader        *GoHeader
		Engine          *GoHttpServer
	}

	GoResponse struct {
		Original http.ResponseWriter
		Goheader *GoHeader
		Writer   io.Writer
		Request  *GoRequest
		Engine   *GoHttpServer
	}
	GoMultipartForm struct {
		Form *multipart.Form
	}
	GoHeader struct {
		Source     interface{}
		isResponse bool
	}
	GoWebSocket struct {
		Conn *websocket.Conn
		GoResponse
	}
	GoCookie http.Cookie
)

var (
	goContextStack       *utils.SimpleLockStack
	goMultipartFormStack *utils.SimpleLockStack
)

func NewGOContext(instance *GoHttpServer) *GoContext {
	if instance == nil {
		instance = &GoHttpServer{MaxMultipartSize: 32 << 20}
	}
	c := &GoContext{Request: &GoRequest{Goheader: &GoHeader{}, Engine: instance}}
	c.Response = &GoResponse{Goheader: &GoHeader{}, Request: c.Request, Engine: instance}
	return c
}
func (c *GoContext) GetRequest() revel.ServerRequest {
	return c.Request
}
func (c *GoContext) GetResponse() revel.ServerResponse {
	if c.WebSocket != nil {
		return c.WebSocket
	}
	return c.Response
}
func (c *GoContext) Destroy() {
	c.Response.Destroy()
	c.Request.Destroy()
	if c.WebSocket != nil {
		c.WebSocket.Destroy()
	}
}
func (r *GoRequest) Get(key int) (value interface{}, err error) {
	switch key {
	case revel.HTTP_SERVER_HEADER:
		value = r.GetHeader()
	case revel.HTTP_MULTIPART_FORM:
		value, err = r.GetMultipartForm()
	case revel.HTTP_QUERY:
		value = r.Original.URL.Query()
	case revel.HTTP_FORM:
		value, err = r.GetForm()
	case revel.HTTP_REQUEST_URI:
		value = r.Original.URL.RequestURI()
	case revel.HTTP_REMOTE_ADDR:
		value = r.Original.RemoteAddr
	case revel.HTTP_METHOD:
		value = r.Original.Method
	case revel.HTTP_PATH:
		value = r.Original.URL.Path
	case revel.HTTP_HOST:
		value = r.Original.Host
	default:
		err = revel.ENGINE_UNKNOWN_GET
	}

	return
}
func (r *GoRequest) Set(key int, value interface{}) bool {
	return false
}

func (r *GoRequest) GetForm() (url.Values, error) {
	if !r.FormParsed {
		if e := r.Original.ParseForm(); e != nil {
			return nil, e
		}
		r.FormParsed = true
	}
	return r.Original.Form, nil
}
func (r *GoRequest) GetMultipartForm() (revel.ServerMultipartForm, error) {
	if !r.MultiFormParsed {
		if e := r.Original.ParseMultipartForm(r.Engine.MaxMultipartSize); e != nil {
			return nil, e
		}
		r.ParsedForm = goMultipartFormStack.Pop().(*GoMultipartForm)
		r.ParsedForm.Form = r.Original.MultipartForm
	}

	return r.ParsedForm, nil
}
func (r *GoRequest) GetHeader() revel.ServerHeader {
	return r.Goheader
}
func (r *GoRequest) GetRaw() interface{} {
	return r.Original
}
func (r *GoRequest) SetRequest(req *http.Request) {
	r.Original = req
	r.Goheader.Source = r
	r.Goheader.isResponse = false

}
func (r *GoRequest) Destroy() {
	r.Goheader.Source = nil
	r.Original = nil
	r.FormParsed = false
	r.MultiFormParsed = false
	r.ParsedForm = nil
}
func (r *GoResponse) Get(key int) (value interface{}, err error) {
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

// Returns list of header keys
func (r *GoHeader) GetKeys() (value []string) {
	if !r.isResponse {
		for key := range r.Source.(*GoRequest).Original.Header {
			value = append(value, key)
		}
	} else {
		for key := range r.Source.(*GoResponse).Original.Header() {
			value = append(value, key)
		}
	}
	return
}

func (r *GoResponse) Set(key int, value interface{}) (set bool) {
	switch key {
	case revel.HTTP_WRITER:
		r.SetWriter(value.(io.Writer))
		set = true
	}
	return
}

func (r *GoResponse) Header() revel.ServerHeader {
	return r.Goheader
}
func (r *GoResponse) GetRaw() interface{} {
	return r.Original
}
func (r *GoResponse) SetWriter(writer io.Writer) {
	r.Writer = writer
}
func (r *GoResponse) WriteStream(name string, contentlen int64, modtime time.Time, reader io.Reader) error {

	// Check to see if the output stream is modified, if not send it using the
	// Native writer
	if _, ok := r.Writer.(http.ResponseWriter); ok {
		if rs, ok := reader.(io.ReadSeeker); ok {
			http.ServeContent(r.Original, r.Request.Original, name, modtime, rs)
		}
	} else {
		// Else, do a simple io.Copy.
		ius := r.Request.Original.Header.Get("If-Unmodified-Since")
		if t, err := http.ParseTime(ius); err == nil && !modtime.IsZero() {
			// The Date-Modified header truncates sub-second precision, so
			// use mtime < t+1s instead of mtime <= t to check for unmodified.
			if modtime.Before(t.Add(1 * time.Second)) {
				h := r.Original.Header()
				delete(h, "Content-Type")
				delete(h, "Content-Length")
				if h.Get("Etag") != "" {
					delete(h, "Last-Modified")
				}
				r.Original.WriteHeader(http.StatusNotModified)
				return nil
			}
		}

		if contentlen != -1 {
			r.Original.Header().Set("Content-Length", strconv.FormatInt(contentlen, 10))
		}
		if _, err := io.Copy(r.Writer, reader); err != nil {
			r.Original.WriteHeader(http.StatusInternalServerError)
			return err
		} else {
			r.Original.WriteHeader(http.StatusOK)
		}
	}
	return nil
}

func (r *GoResponse) Destroy() {
	if c, ok := r.Writer.(io.Closer); ok {
		c.Close()
	}
	r.Goheader.Source = nil
	r.Original = nil
	r.Writer = nil
}

func (r *GoResponse) SetResponse(w http.ResponseWriter) {
	r.Original = w
	r.Writer = w
	r.Goheader.Source = r
	r.Goheader.isResponse = true

}
func (r *GoHeader) SetCookie(cookie string) {
	if r.isResponse {
		r.Source.(*GoResponse).Original.Header().Add("Set-Cookie", cookie)
	}
}
func (r *GoHeader) GetCookie(key string) (value revel.ServerCookie, err error) {
	if !r.isResponse {
		var cookie *http.Cookie
		if cookie, err = r.Source.(*GoRequest).Original.Cookie(key); err == nil {
			value = GoCookie(*cookie)

		}

	}
	return
}
func (r *GoHeader) Set(key string, value string) {
	if r.isResponse {
		r.Source.(*GoResponse).Original.Header().Set(key, value)
	}
}
func (r *GoHeader) Add(key string, value string) {
	if r.isResponse {
		r.Source.(*GoResponse).Original.Header().Add(key, value)
	}
}
func (r *GoHeader) Del(key string) {
	if r.isResponse {
		r.Source.(*GoResponse).Original.Header().Del(key)
	}
}
func (r *GoHeader) Get(key string) (value []string) {
	if !r.isResponse {
		value = r.Source.(*GoRequest).Original.Header[key]
		if len(value) == 0 {
			if ihead := r.Source.(*GoRequest).Original.Header.Get(key); ihead != "" {
				value = append(value, ihead)
			}
		}
	} else {
		value = r.Source.(*GoResponse).Original.Header()[key]
	}
	return
}
func (r *GoHeader) SetStatus(statusCode int) {
	if r.isResponse {
		r.Source.(*GoResponse).Original.WriteHeader(statusCode)
	}
}
func (r GoCookie) GetValue() string {
	return r.Value
}
func (f *GoMultipartForm) GetFiles() map[string][]*multipart.FileHeader {
	return f.Form.File
}
func (f *GoMultipartForm) GetValues() url.Values {
	return url.Values(f.Form.Value)
}
func (f *GoMultipartForm) RemoveAll() error {
	return f.Form.RemoveAll()
}
func (g *GoWebSocket) MessageSendJSON(v interface{}) error {
	return websocket.JSON.Send(g.Conn, v)
}
func (g *GoWebSocket) MessageReceiveJSON(v interface{}) error {
	return websocket.JSON.Receive(g.Conn, v)
}

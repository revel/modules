package testsuite

// The TestSuite container is a container to relay information between the server and the caller
import (
	"bytes"
	"fmt"
	"github.com/revel/revel"
	"github.com/revel/revel/session"
	"golang.org/x/net/websocket"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Start a new request, with a new session
func NewTestSuite(t *testing.T) *TestSuite {
	return NewTestSuiteEngine(revel.NewSessionCookieEngine(), t)
}

// Define a new test suite with a custom session engine
func NewTestSuiteEngine(engine revel.SessionEngine, t *testing.T) *TestSuite {
	jar, _ := cookiejar.New(nil)
	ts := &TestSuite{
		Client:        &http.Client{Jar: jar},
		Session:       session.NewSession(),
		SessionEngine: engine,
		T:             t,
		ResponseChannel:make(chan bool, 1),
	}

	return ts
}

// TestSuite container
type TestSuite struct {
	Response        *httptest.ResponseRecorder // The response recorder
	ResponseChannel chan bool                  // The response channel
	Session         session.Session            // The session
	SessionEngine   revel.SessionEngine        // The session engine
	Sent            bool                       // True if sent
	T               *testing.T                 // The test to handle any errors
	Client          *http.Client               // The client to extract the cookie data
}

// NewTestRequest returns an initialized *TestRequest. It is used for extending
// testsuite package making it possibe to define own methods. Example:
//	type MyTestSuite struct {
//		testing.TestSuite
//	}
//
//	func (t *MyTestSuite) PutFormCustom(...) {
//		req := http.NewRequest(...)
//		...
//		return t.NewTestRequest(req)
//	}
func (t *TestSuite) NewTestRequest(req *http.Request) *TestRequest {
	return &TestRequest{
		Request:   req,
		testSuite: t,
	}
}

// Host returns the address and port of the server, e.g. "127.0.0.1:8557"
func (t *TestSuite) Host() string {
	if revel.ServerEngineInit.Address[0] == ':' {
		return "127.0.0.1" + revel.ServerEngineInit.Address
	}
	return revel.ServerEngineInit.Address
}

// BaseUrl returns the base http/https URL of the server, e.g. "http://127.0.0.1:8557".
// The scheme is set to https if http.ssl is set to true in the configuration file.
func (t *TestSuite) BaseUrl() string {
	if revel.HTTPSsl {
		return "https://" + t.Host()
	}
	return "http://" + t.Host()
}

// WebSocketUrl returns the base websocket URL of the server, e.g. "ws://127.0.0.1:8557"
func (t *TestSuite) WebSocketUrl() string {
	return "ws://" + t.Host()
}

// Get issues a GET request to the given path and stores the result in Response
// and ResponseBody.
func (t *TestSuite) Get(path string) *TestSuite {
	t.GetCustom(t.BaseUrl() + path).Send()
	return t
}

// GetCustom returns a GET request to the given URI in a form of its wrapper.
func (t *TestSuite) GetCustom(uri string) *TestRequest {
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		panic(err)
	}
	return t.NewTestRequest(req)
}

// Delete issues a DELETE request to the given path and stores the result in
// Response and ResponseBody.
func (t *TestSuite) Delete(path string) {
	t.DeleteCustom(t.BaseUrl() + path).Send()
}

// DeleteCustom returns a DELETE request to the given URI in a form of its
// wrapper.
func (t *TestSuite) DeleteCustom(uri string) *TestRequest {
	req, err := http.NewRequest("DELETE", uri, nil)
	if err != nil {
		panic(err)
	}
	return t.NewTestRequest(req)
}

// Put issues a PUT request to the given path, sending the given Content-Type
// and data, storing the result in Response and ResponseBody. "data" may be nil.
func (t *TestSuite) Put(path string, contentType string, reader io.Reader) {
	t.PutCustom(t.BaseUrl()+path, contentType, reader).Send()
}

// PutCustom returns a PUT request to the given URI with specified Content-Type
// and data in a form of wrapper. "data" may be nil.
func (t *TestSuite) PutCustom(uri string, contentType string, reader io.Reader) *TestRequest {
	req, err := http.NewRequest("PUT", uri, reader)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", contentType)
	return t.NewTestRequest(req)
}

// PutForm issues a PUT request to the given path as a form put of the given key
// and values, and stores the result in Response and ResponseBody.
func (t *TestSuite) PutForm(path string, data url.Values) *TestSuite {
	t.PutFormCustom(t.BaseUrl()+path, data).Send()
	return t
}

// PutFormCustom returns a PUT request to the given URI as a form put of the
// given key and values. The request is in a form of TestRequest wrapper.
func (t *TestSuite) PutFormCustom(uri string, data url.Values) *TestRequest {
	return t.PutCustom(uri, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
}

// Patch issues a PATCH request to the given path, sending the given
// Content-Type and data, and stores the result in Response and ResponseBody.
// "data" may be nil.
func (t *TestSuite) Patch(path string, contentType string, reader io.Reader) *TestSuite {
	t.PatchCustom(t.BaseUrl()+path, contentType, reader).Send()
	return t
}

// PatchCustom returns a PATCH request to the given URI with specified
// Content-Type and data in a form of wrapper. "data" may be nil.
func (t *TestSuite) PatchCustom(uri string, contentType string, reader io.Reader) *TestRequest {
	req, err := http.NewRequest("PATCH", uri, reader)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", contentType)
	return t.NewTestRequest(req)
}

// Post issues a POST request to the given path, sending the given Content-Type
// and data, storing the result in Response and ResponseBody. "data" may be nil.
func (t *TestSuite) Post(path string, contentType string, reader io.Reader) *TestSuite {
	t.PostCustom(t.BaseUrl()+path, contentType, reader).Send()
	return t
}

// PostCustom returns a POST request to the given URI with specified
// Content-Type and data in a form of wrapper. "data" may be nil.
func (t *TestSuite) PostCustom(uri string, contentType string, reader io.Reader) *TestRequest {
	req, err := http.NewRequest("POST", uri, reader)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", contentType)
	return t.NewTestRequest(req)
}

// PostForm issues a POST request to the given path as a form post of the given
// key and values, and stores the result in Response and ResponseBody.
func (t *TestSuite) PostForm(path string, data url.Values) *TestSuite {
	t.PostFormCustom(t.BaseUrl()+path, data).Send()
	return t
}

// PostFormCustom returns a POST request to the given URI as a form post of the
// given key and values. The request is in a form of TestRequest wrapper.
func (t *TestSuite) PostFormCustom(uri string, data url.Values) *TestRequest {
	return t.PostCustom(uri, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
}

// PostFile issues a multipart request to the given path sending given params
// and files, and stores the result in Response and ResponseBody.
func (t *TestSuite) PostFile(path string, params url.Values, filePaths url.Values) *TestSuite {
	t.PostFileCustom(t.BaseUrl()+path, params, filePaths).Send()
	return t
}

// PostFileCustom returns a multipart request to the given URI in a form of its
// wrapper with the given params and files.
func (t *TestSuite) PostFileCustom(uri string, params url.Values, filePaths url.Values) *TestRequest {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for key, values := range filePaths {
		for _, value := range values {
			createFormFile(writer, key, value)
		}
	}

	for key, values := range params {
		for _, value := range values {
			err := writer.WriteField(key, value)
			t.AssertEqual(nil, err)
		}
	}
	err := writer.Close()
	t.AssertEqual(nil, err)

	return t.PostCustom(uri, writer.FormDataContentType(), body)
}

// Send issues any request and reads the response. If successful, the caller may
// examine the Response and ResponseBody properties. Session data will be
// added to the request cookies for you.
func (r *TestRequest) Send() *TestRequest {
	writer := httptest.NewRecorder()
	context := revel.NewGoContext(nil)
	context.Request.SetRequest(r.Request)
	context.Response.SetResponse(writer)
	controller := revel.NewController(context)
	controller.Session = r.testSuite.Session

	r.testSuite.SessionEngine.Encode(controller)
	response := http.Response{Header: writer.Header()}
	cookies := response.Cookies()
	for _, c := range cookies {
		r.AddCookie(c)
	}

	r.MakeRequest()
	return r
}

// MakeRequest issues any request and read the response. If successful, the
// caller may examine the Response and ResponseBody properties. You will need to
// manage session / cookie data manually
func (r *TestRequest) MakeRequest() *TestRequest {
	revel.CurrentEngine.(*GoHttpServer).TestChannel <- r
	<-r.testSuite.ResponseChannel
	r.Sent = true

	// Create the controller again to receive the response for processing.
	context := revel.NewGoContext(nil)
	// Set the request with the header from the response..
	newRequest := &http.Request{URL: r.URL, Header: r.testSuite.Response.Header()}
	for _, cookie := range r.testSuite.Client.Jar.Cookies(r.Request.URL) {
		newRequest.AddCookie(cookie)
	}
	context.Request.SetRequest(newRequest)
	context.Response.SetResponse(httptest.NewRecorder())
	controller := revel.NewController(context)

	// Decode the session data from the controller and assign it to the session
	r.testSuite.SessionEngine.Decode(controller)
	r.testSuite.Session = controller.Session

	return r
}

// WebSocket creates a websocket connection to the given path and returns it
func (t *TestSuite) WebSocket(path string) *websocket.Conn {
	t.Assertf(true, "Web Socket Not implemented at this time")
	origin := t.BaseUrl() + "/"
	url := t.WebSocketUrl() + path
	ws, err := websocket.Dial(url, "", origin)
	if err != nil {
		panic(err)
	}
	return ws
}

func (t *TestSuite) AssertOk() {
	t.AssertStatus(http.StatusOK)
}

func (t *TestSuite) AssertNotFound() {
	t.AssertStatus(http.StatusNotFound)
}

func (t *TestSuite) AssertStatus(status int) {
	if t.Response.Code != status {
		panic(fmt.Errorf("Status: (expected) %d != %d (actual)", status, t.Response.Code))
	}
}

func (t *TestSuite) AssertContentType(contentType string) {
	t.AssertHeader("Content-Type", contentType)
}

func (t *TestSuite) AssertHeader(name, value string) {
	actual := t.Response.HeaderMap.Get(name)
	if actual != value {
		panic(fmt.Errorf("Header %s: (expected) %s != %s (actual)", name, value, actual))
	}
}

func (t *TestSuite) AssertEqual(expected, actual interface{}) {
	if !revel.Equal(expected, actual) {
		panic(fmt.Errorf("(expected) %v != %v (actual)", expected, actual))
	}
}

func (t *TestSuite) AssertNotEqual(expected, actual interface{}) {
	if revel.Equal(expected, actual) {
		panic(fmt.Errorf("(expected) %v == %v (actual)", expected, actual))
	}
}

func (t *TestSuite) Assert(exp bool) {
	t.Assertf(exp, "Assertion failed")
}

func (t *TestSuite) Assertf(exp bool, formatStr string, args ...interface{}) {
	if !exp {
		panic(fmt.Errorf(formatStr, args...))
	}
}

// AssertContains asserts that the response contains the given string.
func (t *TestSuite) AssertContains(s string) {
	if !bytes.Contains(t.Response.Body.Bytes(), []byte(s)) {
		panic(fmt.Errorf("Assertion failed. Expected response to contain %s", s))
	}
}

// AssertNotContains asserts that the response does not contain the given string.
func (t *TestSuite) AssertNotContains(s string) {
	if bytes.Contains(t.Response.Body.Bytes(), []byte(s)) {
		panic(fmt.Errorf("Assertion failed. Expected response not to contain %s", s))
	}
}

// AssertContainsRegex asserts that the response matches the given regular expression.
func (t *TestSuite) AssertContainsRegex(regex string) {
	r := regexp.MustCompile(regex)

	if !r.Match(t.Response.Body.Bytes()) {
		panic(fmt.Errorf("Assertion failed. Expected response to match regexp %s", regex))
	}
}

func createFormFile(writer *multipart.Writer, fieldname, filename string) {
	// Try to open the file.
	file, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = file.Close()
	}()

	// Create a new form-data header with the provided field name and file name.
	// Determine Content-Type of the file by its extension.
	h := textproto.MIMEHeader{}
	h.Set("Content-Disposition", fmt.Sprintf(
		`form-data; name="%s"; filename="%s"`,
		escapeQuotes(fieldname),
		escapeQuotes(filepath.Base(filename)),
	))
	h.Set("Content-Type", "application/octet-stream")
	if ct := mime.TypeByExtension(filepath.Ext(filename)); ct != "" {
		h.Set("Content-Type", ct)
	}
	part, err := writer.CreatePart(h)
	if err != nil {
		panic(err)
	}

	// Copy the content of the file we have opened not reading the whole
	// file into memory.
	_, err = io.Copy(part, file)
	if err != nil {
		panic(err)
	}
}

var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

// This function was borrowed from mime/multipart package.
func escapeQuotes(s string) string {
	return quoteEscaper.Replace(s)
}

type TestRequest struct {
	*http.Request
	testSuite *TestSuite
	Sent      bool
}

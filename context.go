package httpserve

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/vroomy/httpserve/form"
)

const formContentType = "application/x-www-form-urlencoded"

// bufPool pools byte buffers used for JSON decoding and encoding,
// eliminating per-request allocations from json.NewDecoder / json.NewEncoder.
var bufPool = sync.Pool{
	New: func() interface{} { return new(bytes.Buffer) },
}

// ctxPool pools Context objects to avoid per-request heap allocation
// of the Context struct, its Storage map, and its Params slice.
var ctxPool = sync.Pool{
	New: func() interface{} {
		c := &Context{}
		c.Params = make(Params, 0, 4)
		return c
	},
}

// acquireContext retrieves a Context from the pool and resets it for reuse.
func acquireContext(w http.ResponseWriter, r *http.Request) *Context {
	c := ctxPool.Get().(*Context)
	c.writer = w
	c.request = r
	c.completed = false
	c.statusCode = 0
	c.errorFn = nil
	// Clear storage without re-allocating the map.
	for k := range c.s {
		delete(c.s, k)
	}
	// Reset slices to zero length while keeping backing arrays.
	c.hooks = c.hooks[:0]
	c.Params = c.Params[:0]
	return c
}

// releaseContext clears held references and returns the Context to the pool.
func releaseContext(c *Context) {
	c.writer = nil
	c.request = nil
	c.errorFn = nil
	ctxPool.Put(c)
}

// newContext will initialize and return a new Context.
// Kept for test compatibility; production code uses acquireContext/releaseContext.
func newContext(w http.ResponseWriter, r *http.Request, p Params) *Context {
	var c Context
	c.writer = w
	c.request = r
	c.Params = p
	return &c
}

// Context is the request context
type Context struct {
	errorFn func(error)

	// Internal context storage, used by Context.Get and Context.Put.
	// Lazily initialized on first Put to avoid allocation on requests that
	// never use storage.
	s Storage
	// hooks are a list of hook functions added during the lifespan of the context
	hooks []Hook

	// Whether or not the context has been completed
	completed bool
	// Status code of response
	statusCode int

	writer  http.ResponseWriter
	request *http.Request

	Params Params
}

// Bind is a helper function which binds the request body to a provided value to be parsed as the inbound content type
func (c *Context) Bind(value interface{}) (err error) {
	defer c.request.Body.Close()
	contentType := c.request.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, formContentType) {
		return form.NewDecoder(c.request.Body).Decode(value)
	}

	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)
	if _, err = buf.ReadFrom(c.request.Body); err != nil {
		return
	}
	return json.Unmarshal(buf.Bytes(), value)
}

// BindJSON is a helper function which binds the request body to a provided value to be parsed as JSON
func (c *Context) BindJSON(value interface{}) (err error) {
	defer c.request.Body.Close()
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)
	if _, err = buf.ReadFrom(c.request.Body); err != nil {
		return
	}
	return json.Unmarshal(buf.Bytes(), value)
}

// BindForm is a helper function which binds the request body to a provided value to be parsed as an HTML form
func (c *Context) BindForm(value interface{}) (err error) {
	defer c.request.Body.Close()
	return form.NewDecoder(c.request.Body).Decode(value)
}

// AddHook will add a hook function to be ran after the context has completed
func (c *Context) AddHook(fn Hook) {
	c.hooks = append(c.hooks, fn)
}

// Param will return the associated parameter value with the provided key
func (c *Context) Param(key string) (value string) {
	return c.Params.ByName(key)
}

// Get will retrieve a value for a provided key from the Context's internal storage
func (c *Context) Get(key string) (value string) {
	// nil map reads are safe in Go and return the zero value
	return c.s[key]
}

// Put will set a value for a provided key into the Context's internal storage
func (c *Context) Put(key, value string) {
	if c.s == nil {
		c.s = make(Storage)
	}
	c.s[key] = value
}

// WriteString will write a string
func (c *Context) WriteString(statusCode int, contentType, str string) {
	c.WriteBytes(statusCode, contentType, []byte(str))
}

// WriteBytes will write a byte slice
func (c *Context) WriteBytes(statusCode int, contentType string, bs []byte) {
	if c.completed {
		c.errorFn(ErrContextIsClosed)
		return
	}
	defer c.close()

	if redirected := c.tryRedirect(statusCode); redirected {
		// Request was redirected, return
		return
	}

	// Set content type
	c.setContentType(contentType)
	// Set status code
	c.setStatusCode(statusCode)

	if _, err := c.writer.Write(bs); err != nil {
		c.errorFn(err)
		return
	}
}

// WriteReader will copy reader bytes to the http response body
func (c *Context) WriteReader(statusCode int, contentType string, r io.Reader) {
	if c.completed {
		c.errorFn(ErrContextIsClosed)
		return
	}
	defer c.close()

	if redirected := c.tryRedirect(statusCode); redirected {
		// Request was redirected, return
		return
	}

	// Set content type
	c.setContentType(contentType)
	// Set status code
	c.setStatusCode(statusCode)

	// Copy reader bytes to writer
	if _, err := io.Copy(c.writer, r); err != nil {
		c.errorFn(err)
		return
	}
}

// WriteJSON will write JSON bytes to the http response body
func (c *Context) WriteJSON(statusCode int, value interface{}) {
	if c.completed {
		c.errorFn(ErrContextIsClosed)
		return
	}
	defer c.close()

	if redirected := c.tryRedirect(statusCode); redirected {
		// Request was redirected, return
		return
	}

	var (
		resp JSONValue
		err  error
	)

	if resp, err = makeJSONValue(statusCode, value); err != nil {
		c.errorFn(err)
		return
	}

	// Encode into pooled buffer first so we don't commit headers if encoding fails.
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)
	if err = json.NewEncoder(buf).Encode(resp); err != nil {
		c.errorFn(err)
		return
	}

	// Set content type
	c.setContentType("application/json")
	// Set status code
	c.setStatusCode(statusCode)

	if _, err = buf.WriteTo(c.writer); err != nil {
		c.errorFn(err)
	}
}

// WriteNoContent will write a no content response
func (c *Context) WriteNoContent() {
	if c.completed {
		c.errorFn(ErrContextIsClosed)
		return
	}
	defer c.close()

	if redirected := c.tryRedirect(204); redirected {
		// Request was redirected, return
		return
	}

	c.setStatusCode(204)
}

// Redirect will redirect the client to the provided destination
func (c *Context) Redirect(statusCode int, destination string) {
	if c.completed {
		c.errorFn(ErrContextIsClosed)
		return
	}
	defer c.close()

	c.redirect(statusCode, destination)
}

// Writer will return the underlying http.ResponseWriter
func (c *Context) Writer() http.ResponseWriter {
	return c.writer
}

// Request will return the underlying http.Request
func (c *Context) Request() *http.Request {
	return c.request
}

func (c *Context) setStatusCode(statusCode int) {
	// Write status code to header
	c.writer.WriteHeader(statusCode)
	// Set context status code as the provided status code
	c.statusCode = statusCode
}

func (c *Context) setContentType(contentType string) {
	// Get reference to header
	header := c.writer.Header()
	// Set content type
	header.Set("Content-Type", contentType)
}

func (c *Context) processHandlers(hs []Handler) {
	// Iterate through the provided handlers
	for _, h := range hs {
		if h(c); c.completed {
			return
		}
	}
}

func (c *Context) processHooks() {
	for i := len(c.hooks) - 1; i > -1; i-- {
		c.hooks[i](c.statusCode, c)
	}
}

func (c *Context) getRedirect(statusCode int) (redirectTo string, ok bool) {
	if statusCode < 200 || statusCode >= 300 {
		return
	}

	if c.request.Method == http.MethodGet && statusCode != 204 {
		return
	}

	accept := c.request.Header.Get("Accept")
	firstAccept := strings.SplitN(accept, ",", 2)[0]
	if firstAccept != "text/html" {
		return
	}

	var rq redirectQuery
	if err := form.Unmarshal(c.request.URL.RawQuery, &rq); err != nil {
		err = fmt.Errorf("httpserve.Context.getRedirect(): error unmarshaling form: %v", err)
		c.errorFn(err)
		return
	}

	if redirectTo = rq.Redirect; len(redirectTo) == 0 {
		return
	}

	ok = true
	return
}

func (c *Context) tryRedirect(statusCode int) (ok bool) {
	var redirectTo string
	if redirectTo, ok = c.getRedirect(statusCode); !ok {
		return
	}

	c.redirect(http.StatusFound, redirectTo)
	return
}

func (c *Context) redirect(statusCode int, destination string) {
	c.writer.Header().Add("Location", destination)
	c.setStatusCode(statusCode)
}

func (c *Context) close() {
	c.completed = true
}

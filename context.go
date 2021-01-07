package httpserve

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/vroomy/common"
	"github.com/vroomy/httpserve/form"
)

var _ common.Context = &Context{}

// newContext will initialize and return a new Context
func newContext(w http.ResponseWriter, r *http.Request, p Params) *Context {
	var c Context
	// Initialize internal storage
	c.s = make(Storage)
	// Associate provided http.ResponseWriter
	c.writer = w
	// Associate provided *http.Request
	c.request = r
	// Associate provided httprouter.Params
	c.Params = p
	return &c
}

// Context is the request context
type Context struct {
	// Internal context storage, used by Context.Get and Context.Put
	s Storage
	// hooks are a list of hook functions added during the lifespam of the context
	hooks []common.Hook

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
	switch c.request.Header.Get("Content-Type") {
	case "application/x-www-form-urlencoded":
		return form.NewDecoder(c.request.Body).Decode(value)
	default:
		return json.NewDecoder(c.request.Body).Decode(value)
	}
}

// BindJSON is a helper function which binds the request body to a provided value to be parsed as JSON
func (c *Context) BindJSON(value interface{}) (err error) {
	defer c.request.Body.Close()
	return json.NewDecoder(c.request.Body).Decode(value)
}

// BindForm is a helper function which binds the request body to a provided value to be parsed as an HTML form
func (c *Context) BindForm(value interface{}) (err error) {
	defer c.request.Body.Close()
	return form.NewDecoder(c.request.Body).Decode(value)
}

// AddHook will add a hook function to be ran after the context has completed
func (c *Context) AddHook(fn common.Hook) {
	c.hooks = append(c.hooks, fn)
}

// Param will return the associated parameter value with the provided key
func (c *Context) Param(key string) (value string) {
	return c.Params.ByName(key)
}

// Get will retrieve a value for a provided key from the Context's internal storage
func (c *Context) Get(key string) (value string) {
	return c.s[key]
}

// Put will set a value for a provided key into the Context's internal storage
func (c *Context) Put(key, value string) {
	c.s[key] = value
}

// WriteString will write a string
func (c *Context) WriteString(statusCode int, contentType, str string) (err error) {
	return c.WriteBytes(statusCode, contentType, []byte(str))
}

// WriteBytes will write a byte slice
func (c *Context) WriteBytes(statusCode int, contentType string, bs []byte) (err error) {
	if c.completed {
		return ErrContextIsClosed
	}
	defer c.close()

	if redirected := c.tryRedirect(statusCode); redirected {
		// Request was redirected, return
		return
	}

	// Set content type
	c.setContentType(contentType)

	_, err = c.writer.Write(bs)
	return
}

// WriteReader will copy reader bytes to the http response body
func (c *Context) WriteReader(statusCode int, contentType string, r io.Reader) (err error) {
	if c.completed {
		return ErrContextIsClosed
	}
	defer c.close()

	if redirected := c.tryRedirect(statusCode); redirected {
		// Request was redirected, return
		return
	}

	// Set content type
	c.setContentType(contentType)

	// Copy reader bytes to writer
	_, err = io.Copy(c.writer, r)
	return
}

// WriteJSON will write JSON bytes to the http response body
func (c *Context) WriteJSON(statusCode int, value interface{}) (err error) {
	if c.completed {
		return ErrContextIsClosed
	}
	defer c.close()

	if redirected := c.tryRedirect(statusCode); redirected {
		// Request was redirected, return
		return
	}

	// Set content type
	c.setContentType("application/json")
	// Encode value as JSON
	return json.NewEncoder(c.writer).Encode(value)
}

// WriteNoContent will write a no content response
func (c *Context) WriteNoContent(statusCode int, value interface{}) (err error) {
	if c.completed {
		return ErrContextIsClosed
	}
	defer c.close()

	if redirected := c.tryRedirect(statusCode); redirected {
		// Request was redirected, return
		return
	}

	c.setStatusCode(204)
	return
}

// Redirect will redirect the client to the provided destinatoin
func (c *Context) Redirect(statusCode int, destination string) (err error) {
	if c.completed {
		return ErrContextIsClosed
	}
	defer c.close()

	c.redirect(statusCode, destination)
	return
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

	accept := c.request.Header.Get("Accept")
	firstAccept := strings.SplitN(accept, ",", 2)[0]
	if firstAccept != "text/html" {
		return
	}

	var rq redirectQuery
	form.Unmarshal(c.request.URL.RawQuery, &rq)
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
	c.setStatusCode(statusCode)
	c.writer.Header().Add("Location", destination)
	return
}

func (c *Context) close() {
	c.completed = true
}

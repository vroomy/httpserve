package httpserve

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/vroomy/httpserve/form"
)

const formContentType = "application/x-www-form-urlencoded"

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
	errorFn func(error)

	// Internal context storage, used by Context.Get and Context.Put
	s Storage
	// hooks are a list of hook functions added during the lifespam of the context
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

	switch {
	case strings.Index(contentType, formContentType) == 0:
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
func (c *Context) AddHook(fn Hook) {
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

	// Set content type
	c.setContentType("application/json")
	// Set status code
	c.setStatusCode(statusCode)

	// Encode value as JSON
	if err = json.NewEncoder(c.writer).Encode(resp); err != nil {
		c.errorFn(err)
		return
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

// Redirect will redirect the client to the provided destinatoin

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

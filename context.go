package httpserve

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/hatchify/scribe"
	"github.com/vroomy/common"
)

// newContext will initialize and return a new Context
func newContext(w http.ResponseWriter, r *http.Request, p Params) *Context {
	var c Context
	// Initialize internal storage
	c.s = make(common.Storage)
	// Associate provided http.ResponseWriter
	c.Writer = w
	// Associate provided *http.Request
	c.Request = r
	// Associate provided httprouter.Params
	c.Params = p
	return &c
}

// Context is the request context
type Context struct {
	// Internal context storage, used by Context.Get and Context.Put
	s common.Storage
	// hooks are a list of hook functions added during the lifespam of the context
	hooks []common.Hook

	Writer  http.ResponseWriter
	Request *http.Request
	Params  Params
}

func (c *Context) getResponse(hs []common.Handler) (resp common.Response) {
	// Iterate through the provided handlers
	for _, h := range hs {
		if h == nil {
			scribe.New("httpserve").Errorf("nil handler for route: %s", c.GetRequest().URL.Path)
		}
		// Call handler and pass Context
		if resp = h(c); resp != nil {
			// A non-nil response was provided, return
			return
		}
	}

	return
}

func (c *Context) respond(resp common.Response) {
	// Response is nil, no further action is needed
	if resp == nil {
		return
	}

	if c.redirect(resp) {
		return
	}

	// Write content type!
	c.Writer.Header().Set("Content-Type", resp.ContentType())

	// Write status code to header
	c.Writer.WriteHeader(resp.StatusCode())

	// Write response to http.ResponseWriter
	if _, err := resp.WriteTo(c.Writer); err != nil {
		// Write error to stderr
		fmt.Fprintf(os.Stderr, "Error writing to http.ResponseWriter: %v\n", err)
	}
}

func (c *Context) redirect(resp common.Response) (ok bool) {
	var redirect *RedirectResponse
	if redirect, ok = resp.(*RedirectResponse); !ok {
		return
	}

	c.Writer.Header().Add("Location", redirect.url)
	c.Writer.WriteHeader(redirect.code)
	return
}

func (c *Context) wasAdopted(resp common.Response) (ok bool) {
	if _, ok = resp.(*AdoptResponse); !ok {
		return
	}

	return
}

func (c *Context) processHooks(statusCode int) {
	for i := len(c.hooks) - 1; i > -1; i-- {
		c.hooks[i](statusCode, c.s)
	}
}

// Write will write a byteslice
func (c *Context) Write(bs []byte) (n int, err error) {
	return c.Writer.Write(bs)
}

// WriteString will write a string
func (c *Context) WriteString(str string) (n int, err error) {
	return c.Writer.Write([]byte(str))
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

// GetRequest will return http.Request
func (c *Context) GetRequest() (req *http.Request) {
	return c.Request
}

// GetWriter will return http.Writer
func (c *Context) GetWriter() (writer http.ResponseWriter) {
	return c.Writer
}

// BindJSON is a helper function which binds the request body to a provided value to be parsed as JSON
func (c *Context) BindJSON(value interface{}) (err error) {
	defer c.Request.Body.Close()
	return json.NewDecoder(c.Request.Body).Decode(value)
}

// AddHook will add a hook function to be ran after the context has completed
func (c *Context) AddHook(fn common.Hook) {
	c.hooks = append(c.hooks, fn)
}

// NewAdoptResponse will return an adopt response object
func (c *Context) NewAdoptResponse() (resp common.Response) {
	return NewAdoptResponse()
}

// NewNoContentResponse will return a no content response object
func (c *Context) NewNoContentResponse() (resp common.Response) {
	return NewNoContentResponse()
}

// NewRedirectResponse will return a redirect response object
func (c *Context) NewRedirectResponse(code int, url string) (resp common.Response) {
	return NewRedirectResponse(code, url)
}

// NewJSONResponse will return a json response object
func (c *Context) NewJSONResponse(code int, value interface{}) (resp common.Response) {
	return NewJSONResponse(code, value)
}

// NewJSONPResponse will return a json response object with callback
func (c *Context) NewJSONPResponse(callback string, value interface{}) (resp common.Response) {
	return NewJSONPResponse(callback, value)
}

// NewTextResponse will return a text response object
func (c *Context) NewTextResponse(code int, body []byte) (resp common.Response) {
	return NewTextResponse(code, body)
}

// NewXMLResponse will return an xml response object
func (c *Context) NewXMLResponse(code int, body []byte) (resp common.Response) {
	return NewXMLResponse(code, body)
}

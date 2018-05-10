package httpserve

import (
	"fmt"
	"net/http"
	"os"

	"github.com/julienschmidt/httprouter"
)

// newContext will initialize and return a new Context
func newContext(w http.ResponseWriter, r *http.Request, p httprouter.Params) *Context {
	var c Context
	// Initialize internal storage
	c.s = make(Storage)
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
	s Storage

	Writer  http.ResponseWriter
	Request *http.Request
	Params  httprouter.Params
}

func (c *Context) getResponse(hs []Handler) (resp Response) {
	for _, h := range hs {
		if resp = h(c); resp != nil {
			return
		}
	}

	return
}

func (c *Context) respond(resp Response) {
	// Response is nil, no further action is needed
	if resp == nil {
		return
	}
	// Write status code to header
	c.Writer.WriteHeader(resp.StatusCode())
	// Write response to http.ResponseWriter
	if _, err := resp.WriteTo(c.Writer); err != nil {
		// Write error to stderr
		fmt.Fprintf(os.Stderr, "Error writing to http.ResponseWriter: %v\n", err)
		// TODO: Figure out how we want to handle this error
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
	return c.Param(key)
}

// Get will retrieve a value for a provided key from the Context's internal storage
func (c *Context) Get(key string) (value string) {
	return c.s[key]
}

// Put will set a value for a provided key into the Context's internal storage
func (c *Context) Put(key string) (value string) {
	return c.s[key]
}

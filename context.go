package httpserve

import (
	"encoding/json"
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
	// Iterate through the provided handlers
	for _, h := range hs {
		// Call handler and pass Context
		if resp = h(c); resp != nil {
			// A non-nil response was provided, return
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
func (c *Context) Put(key, value string) {
	c.s[key] = value
}

// BindJSON is a helper function which binds the request body to a provided value to be parsed as JSON
func (c *Context) BindJSON(value interface{}) (err error) {
	defer c.Request.Body.Close()
	return json.NewDecoder(c.Request.Body).Decode(value)
}

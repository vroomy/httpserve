package httpserve

import (
	"encoding/json"
	"fmt"
	"io"
)

// NewJSONResponse will return a new text response
func NewJSONResponse(code int, value interface{}) *JSONResponse {
	var j JSONResponse
	j.code = code
	j.val = value
	return &j
}

// JSONResponse is a basic text response
type JSONResponse struct {
	code int
	val  interface{}
}

// ContentType returns the content type
func (j *JSONResponse) ContentType() (contentType string) {
	return "application/json"
}

// StatusCode returns the status code
func (j *JSONResponse) StatusCode() (code int) {
	return j.code
}

func (j *JSONResponse) newValue() (value JSONValue, err error) {
	if j.code < 400 {
		// Status code is not an error code, set the data as the associated value and return.
		value.Data = j.val
		return
	}
	// Switch on associated value's type
	switch v := j.val.(type) {
	case error:
		// Type is a single error value, create new error slice with error as only item
		value.Errors = []error{v}
	case []error:
		// Type is an error slice, set errors as the value
		value.Errors = v
	default:
		// Invalid error value, return error
		err = fmt.Errorf("invalid type for an error response: %#v", v)
	}

	return
}

// WriteTo will write to a given io.Writer
func (j *JSONResponse) WriteTo(w io.Writer) (n int64, err error) {
	var value JSONValue
	// Initialize a new JSON value
	if value, err = j.newValue(); err != nil {
		// Error encountered while initializing responder, return early
		return
	}
	// Initialize a new JSON encoder
	enc := json.NewEncoder(w)
	// Encode the responder
	err = enc.Encode(value)
	return
}

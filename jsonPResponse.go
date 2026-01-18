package httpserve

import (
	"bytes"
	"encoding/json"
	"io"
)

var jsonPEnding = []byte(");\n")

// NewJSONPResponse will return a new text response
func NewJSONPResponse(callback string, value interface{}) *JSONPResponse {
	var j JSONPResponse
	j.callback = callback
	j.val = value
	return &j
}

// JSONPResponse is a basic text response
type JSONPResponse struct {
	callback string
	val      interface{}
}

// ContentType returns the content type
func (j *JSONPResponse) ContentType() (contentType string) {
	return "application/javascript"
}

// StatusCode returns the status code
func (j *JSONPResponse) StatusCode() (code int) {
	return 200
}

func (j *JSONPResponse) newValue() (value JSONValue) {
	// Switch on associated value's type
	switch v := j.val.(type) {
	case error:
		// Type is a single error value, create new error slice with error as only item
		value.PushErrors(v)
	case []error:
		// Type is an error slice, set errors as the value
		value.PushErrors(v...)
	default:
		value.Data = j.val
	}

	return
}

// WriteTo will write to a given io.Writer
func (j *JSONPResponse) WriteTo(w io.Writer) (n int64, err error) {
	// Initialize buffer
	buf := bytes.NewBuffer(nil)
	// Write callback func
	buf.WriteString(j.callback + "(")
	// Initialize a new JSON value
	value := j.newValue()
	// Initialize a new JSON encoder
	enc := json.NewEncoder(buf)

	// Encode the responder
	if err = enc.Encode(value); err != nil {
		return
	}

	// Remove the trailing newline
	buf.Truncate(buf.Len() - 1)
	// Write jsonP ending
	buf.Write(jsonPEnding)

	// Flush buffer to writer
	_, err = w.Write(buf.Bytes())
	return
}

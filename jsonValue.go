package httpserve

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// DecodeJSONValue will decode a JSON value
func DecodeJSONValue(r io.Reader, val interface{}) (err error) {
	var jv JSONValue
	jv.Data = val
	dec := json.NewDecoder(r)
	if err = dec.Decode(&jv); err != nil {
		return
	}

	return errors.Join(jv.Errors...)
}

// UnmarshalJSONValue will unmarshal a JSON value
func UnmarshalJSONValue(bs []byte, val interface{}) (err error) {
	var jv JSONValue
	jv.Data = val
	if err = json.Unmarshal(bs, &jv); err != nil {
		return
	}

	return errors.Join(jv.Errors...)
}

func makeJSONValue(statusCode int, data interface{}) (val JSONValue, err error) {
	if statusCode < 400 {
		// Status code is not an error code, set the data as the associated value and return.
		val.Data = data
		return
	}

	// Switch on associated value's type
	switch v := data.(type) {
	case error:
		// Type is a single error value, create new error slice with error as only item
		val.PushErrors(v)
	case []error:

		// Type is an error slice, set errors as the value
		val.PushErrors(v...)
	default:
		// Invalid error value, return error
		err = fmt.Errorf("invalid type for an error response: %#v", v)
	}

	return
}

// JSONValue represents a basic JSON value
type JSONValue struct {
	Data   interface{} `json:"data,omitempty"`
	Errors []error     `json:"errors,omitempty"`
}

func (j *JSONValue) PushErrors(errs ...error) {
	j.Errors = append(j.Errors, errs...)
}

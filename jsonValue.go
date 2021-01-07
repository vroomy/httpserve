package httpserve

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/hatchify/errors"
)

// DecodeJSONValue will decode a JSON value
func DecodeJSONValue(r io.Reader, val interface{}) (err error) {
	var jv JSONValue
	jv.Data = val
	dec := json.NewDecoder(r)
	if err = dec.Decode(&jv); err != nil {
		return
	}

	return jv.Errors.Err()
}

// UnmarshalJSONValue will unmarshal a JSON value
func UnmarshalJSONValue(bs []byte, val interface{}) (err error) {
	var jv JSONValue
	jv.Data = val
	if err = json.Unmarshal(bs, &jv); err != nil {
		return
	}

	if err = jv.Errors.Err(); err != nil {
		return
	}

	return
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
		val.Errors.Push(v)
	case []error:
		// Type is an error slice, set errors as the value
		val.Errors.Copy(v)
	default:
		// Invalid error value, return error
		err = fmt.Errorf("invalid type for an error response: %#v", v)
	}

	return
}

// JSONValue represents a basic JSON value
type JSONValue struct {
	Data   interface{}      `json:"data,omitempty"`
	Errors errors.ErrorList `json:"errors,omitempty"`
}

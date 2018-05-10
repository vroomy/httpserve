package httpserve

import (
	"encoding/json"
	"io"

	"github.com/Hatch1fy/errors"
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

	return jv.Errors.Err()
}

// JSONValue represents a basic JSON value
type JSONValue struct {
	Data   interface{}      `json:"data,omitempty"`
	Errors errors.ErrorList `json:"errors,omitempty"`
}

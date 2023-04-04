package httpserve

import (
	"bytes"
	"errors"
	"testing"
)

func TestJSONResponseData(t *testing.T) {
	var ts TestJSONStruct
	ts.Name = "John Doe"
	ts.Age = 33
	resp := NewJSONResponse(200, ts)
	buf := bytes.NewBuffer(nil)

	if _, err := resp.WriteTo(buf); err != nil {
		t.Fatalf("error writing: %v", err)
	}

	var nts TestJSONStruct
	if err := UnmarshalJSONValue(buf.Bytes(), &nts); err != nil {
		t.Fatal(err)
	}

	if ts != nts {
		t.Fatalf("invalid value, expected %#v and received %#v", ts, nts)
	}
}

func TestJSONResponseError(t *testing.T) {
	err := errors.New("test error")
	resp := NewJSONResponse(400, err)
	buf := bytes.NewBuffer(nil)

	if _, err := resp.WriteTo(buf); err != nil {
		t.Fatalf("error writing: %v", err)
	}

	var nts TestJSONStruct
	if nerr := UnmarshalJSONValue(buf.Bytes(), &nts); nerr == nil {
		t.Fatal("expected error, received nil error")
	} else if err.Error() != "test error" {
		t.Fatalf("invalid error, expected \"%s\" and received \"%s\"", err.Error(), nerr.Error())
	}
}

type TestJSONStruct struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

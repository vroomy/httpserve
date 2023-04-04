package httpserve

import (
	"bytes"
	"testing"
)

func TestTextResponse(t *testing.T) {
	value := "hello world"
	resp := NewTextResponse(200, []byte(value))
	buf := bytes.NewBuffer(nil)

	if _, err := resp.WriteTo(buf); err != nil {
		t.Fatalf("error writing: %v", err)
	}

	if buf.String() != value {
		t.Fatalf("invalid value, expected %#v and received %#v", buf.String(), value)
	}
}

package httpserve

import (
	"bytes"
	"testing"
)

func TestTextResponse(t *testing.T) {
	value := "hello world"
	resp := NewTextResponse(200, []byte(value))
	buf := bytes.NewBuffer(nil)
	resp.WriteTo(buf)

	if buf.String() != value {
		t.Fatalf("invalid value, expected %#v and received %#v", buf.String(), value)
	}
}

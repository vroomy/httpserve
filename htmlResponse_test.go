package httpserve

import (
	"bytes"
	"testing"
)

func TestHTMLResponse(t *testing.T) {
	value := "<html><head></head><body><p>hello world<p></body>"
	resp := NewHTMLResponse(200, []byte(value))
	buf := bytes.NewBuffer(nil)
	resp.WriteTo(buf)

	if buf.String() != value {
		t.Fatalf("invalid value, expected %#v and received %#v", buf.String(), value)
	}
}

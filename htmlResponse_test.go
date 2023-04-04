package httpserve

import (
	"bytes"
	"testing"
)

func TestHTMLResponse(t *testing.T) {
	value := "<html><head></head><body><p>hello world<p></body>"
	resp := NewHTMLResponse(200, []byte(value))
	buf := bytes.NewBuffer(nil)
	if _, err := resp.WriteTo(buf); err != nil {
		t.Fatalf("error writing: %v", err)
	}

	if buf.String() != value {
		t.Fatalf("invalid value, expected %#v and received %#v", buf.String(), value)
	}
}

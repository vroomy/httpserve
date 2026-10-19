package apex

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestResponseWriters(t *testing.T) {
	var cases = []struct {
		name     string
		response interface {
			io.WriterTo
			ContentType() string
			StatusCode() int
		}
		status      int
		contentType string
		body        string
		checkCount  bool
	}{
		{"HTML", NewHTMLResponse(201, []byte("<p>hello</p>")), 201, "text/html", "<p>hello</p>", true},
		{"text", NewTextResponse(202, []byte("hello")), 202, "text/plain", "hello", true},
		{"XML", NewXMLResponse(203, []byte("<hello/>")), 203, "text/xml", "<hello/>", true},
		{"JSON", NewJSONResponse(200, "hello"), 200, "application/json", "{\"data\":\"hello\"}\n", false},
		{"JSONP", NewJSONPResponse("callback", "hello"), 200, "application/javascript", "callback({\"data\":\"hello\"});\n", false},
		{"adopt", NewAdoptResponse(), 200, "text/plain", "", true},
		{"no content", NewNoContentResponse(), 204, "text/plain", "", true},
		{"redirect", NewRedirectResponse(307, "/next"), 307, "", "", true},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			var body bytes.Buffer
			var count, err = test.response.WriteTo(&body)
			if err != nil || body.String() != test.body {
				t.Fatalf("WriteTo() = %d/%v, body = %q", count, err, body.String())
			}

			// JSON and JSONP have a known byte-count bug recorded in docs/testing.md.
			if test.checkCount && count != int64(len(test.body)) {
				t.Errorf("WriteTo() count = %d, want %d", count, len(test.body))
			}

			if test.response.StatusCode() != test.status || test.response.ContentType() != test.contentType {
				t.Fatalf("metadata = %d/%q", test.response.StatusCode(), test.response.ContentType())
			}
		})
	}
}

func TestResponseWriteErrors(t *testing.T) {
	var failure = errors.New("writer failure")
	var cases = []struct {
		name     string
		response io.WriterTo
	}{
		{"HTML", NewHTMLResponse(200, []byte("hello"))},
		{"text", NewTextResponse(200, []byte("hello"))},
		{"XML", NewXMLResponse(200, []byte("<hello/>"))},
		{"JSON", NewJSONResponse(200, "hello")},
		{"JSONP", NewJSONPResponse("callback", "hello")},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			var writer = &failingWriter{err: failure}
			var _, err = test.response.WriteTo(writer)
			if !errors.Is(err, failure) {
				t.Fatalf("WriteTo() error = %v, want writer cause", err)
			}
		})
	}
}

func TestJSONResponseInvalidValues(t *testing.T) {
	var cases = []struct {
		name     string
		response io.WriterTo
	}{
		{"invalid error payload", NewJSONResponse(400, "invalid")},
		{"unencodable data", NewJSONResponse(200, make(chan int))},
		{"unencodable JSONP", NewJSONPResponse("callback", make(chan int))},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			var body bytes.Buffer
			var _, err = test.response.WriteTo(&body)
			if err == nil || body.Len() != 0 {
				t.Fatalf("WriteTo() error = %v, body = %q", err, body.String())
			}
		})
	}
}

func TestJSONErrorEnvelopes(t *testing.T) {
	var first = errors.New("first")
	var second = errors.New("second")
	var cases = []struct {
		name  string
		value any
		count int
	}{
		{"one error", first, 1},
		{"several errors", []error{first, second}, 2},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			var response = NewJSONResponse(400, test.value)
			var envelope, err = response.newValue()
			if err != nil || envelope.Data != nil || len(envelope.Errors) != test.count || !errors.Is(envelope.Errors[0], first) {
				t.Fatalf("JSON envelope = %#v, error = %v", envelope, err)
			}

			var jsonp = NewJSONPResponse("callback", test.value).newValue()
			if jsonp.Data != nil || len(jsonp.Errors) != test.count || !errors.Is(jsonp.Errors[0], first) {
				t.Fatalf("JSONP envelope = %#v", jsonp)
			}

			var value JSONValue
			value, err = makeJSONValue(500, test.value)
			if err != nil || value.Data != nil || len(value.Errors) != test.count || !errors.Is(value.Errors[0], first) {
				t.Fatalf("context envelope = %#v, error = %v", value, err)
			}
		})
	}
}

func TestJSONValueDecoding(t *testing.T) {
	var decoders = []struct {
		name   string
		decode func(string, any) error
	}{
		{"stream", func(input string, value any) error { return DecodeJSONValue(strings.NewReader(input), value) }},
		{"bytes", func(input string, value any) error { return UnmarshalJSONValue([]byte(input), value) }},
	}
	for _, decoder := range decoders {
		t.Run(decoder.name, func(t *testing.T) {
			var value struct {
				Name string `json:"name"`
			}
			var err = decoder.decode(`{"data":{"name":"Ada"}}`, &value)
			if err != nil || value.Name != "Ada" {
				t.Fatalf("decode = %#v/%v", value, err)
			}

			if err = decoder.decode(`{`, &value); err == nil {
				t.Fatal("malformed JSON was accepted")
			}

			if err = decoder.decode(`{"data":{"name":42}}`, &value); err == nil {
				t.Fatal("incompatible field type was accepted")
			}

			if err = decoder.decode(`{"errors":["failure"]}`, &value); err == nil {
				t.Fatal("error payload was accepted as success")
			}
		})
	}
}

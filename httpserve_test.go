package httpserve

import (
	"io/ioutil"
	"net/http"
	"testing"
	"time"
)

func TestServe(t *testing.T) {
	var (
		jsonVal TestJSONStruct
		resp    *http.Response
		bs      []byte
		err     error
	)

	textVal := "hello"
	jsonVal.Name = "John Doe"
	jsonVal.Age = 33

	serve := New()
	defer func() {
		if err = serve.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	derp := serve.Group("/derp")

	// Setup text resonse handler
	derp.GET("hello", func(ctx *Context) Response {
		return NewTextResponse(200, []byte(textVal))
	})

	// Setup json response handler
	derp.GET("world", func(ctx *Context) Response {
		return NewJSONResponse(200, jsonVal)
	})

	// Listen within a new goroutine
	go func() {
		if err := serve.Listen(8080); err != nil {
			t.Fatal(err)
		}
	}()

	// Sleep for 200 milliseconds to ensure we've given the serve instance enough time to listen
	time.Sleep(200 * time.Millisecond)

	if resp, err = http.Get("http://localhost:8080/derp/hello"); err != nil {
		t.Fatal(err)
	}

	if bs, err = ioutil.ReadAll(resp.Body); err != nil {
		t.Fatal(err)
	}

	if err = resp.Body.Close(); err != nil {
		t.Fatal(err)
	}

	if string(bs) != textVal {
		t.Fatalf("invalid value, expected \"%s\" and received \"%s\"", string(bs), textVal)
	}

	if resp, err = http.Get("http://localhost:8080/derp/world"); err != nil {
		t.Fatal(err)
	}

	var ts TestJSONStruct
	if err = DecodeJSONValue(resp.Body, &ts); err != nil {
		t.Fatal(err)
	}

	if err = resp.Body.Close(); err != nil {
		t.Fatal(err)
	}

	if ts != jsonVal {
		t.Fatalf("invalid value, expected \"%#v\" and received \"%#v\"", ts, jsonVal)
	}
}

package apex

import (
	"io"
	"net/http"
	"testing"
	"time"
)

func TestEngineText(t *testing.T) {
	var (
		resp *http.Response
		bs   []byte
		err  error
	)

	textVal := "hello"
	engine := New()
	defer func() {
		if err = engine.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	// Create derp group
	derp := engine.Group("/derp")

	// Setup text resonse handler
	if err = derp.GET("hello", func(ctx *Context) {
		ctx.WriteString(200, "text/plain", textVal)
	}); err != nil {
		t.Fatal(err)
	}

	errC := make(chan error, 1)
	// Listen within a new goroutine
	go func() {
		if err := engine.Listen(8080); err != nil && err != http.ErrServerClosed {
			errC <- err
			return
		}
	}()

	select {
	// Sleep for 200 milliseconds to ensure we've given the engine instance enough time to listen
	case <-time.After(200 * time.Millisecond):
	case err = <-errC:
		t.Fatal(err)
	}

	// Perform GET request
	if resp, err = http.Get("http://localhost:8080/derp/hello"); err != nil {
		t.Fatal(err)
	}

	// Read body as bytes
	if bs, err = io.ReadAll(resp.Body); err != nil {
		t.Fatal(err)
	}

	// Close response body
	if err = resp.Body.Close(); err != nil {
		t.Fatal(err)
	}

	// Ensure values are correct
	if string(bs) != textVal {
		t.Fatalf("invalid value, expected \"%s\" and received \"%s\"", textVal, string(bs))
	}
}

func TestEngineJSON(t *testing.T) {
	var (
		jsonVal TestJSONStruct
		ts      TestJSONStruct
		resp    *http.Response
		err     error
	)

	jsonVal.Name = "John Doe"
	jsonVal.Age = 33

	engine := New()
	defer func() {
		if err = engine.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	// Create derp group
	derp := engine.Group("/derp")

	// Setup json response handler
	if err = derp.GET("world", func(ctx *Context) {
		ctx.WriteJSON(200, jsonVal)
	}); err != nil {
		t.Fatal(err)
	}

	errC := make(chan error, 1)
	// Listen within a new goroutine
	go func() {
		if err := engine.Listen(8080); err != nil && err != http.ErrServerClosed {
			errC <- err
		}
	}()

	select {
	// Sleep for 200 milliseconds to ensure we've given the engine instance enough time to listen
	case <-time.After(200 * time.Millisecond):
	case err = <-errC:
		t.Fatal(err)
	}

	// Sleep for 200 milliseconds to ensure we've given the engine instance enough time to listen
	time.Sleep(200 * time.Millisecond)

	// Perform GET request
	if resp, err = http.Get("http://localhost:8080/derp/world"); err != nil {
		t.Fatal(err)
	}

	// Decode response body as TestJSONStruct
	if err = DecodeJSONValue(resp.Body, &ts); err != nil {
		t.Fatal(err)
	}

	// Close response body
	if err = resp.Body.Close(); err != nil {
		t.Fatal(err)
	}

	// Ensure values are correct
	if ts != jsonVal {
		t.Fatalf("invalid value, expected \"%#v\" and received \"%#v\"", jsonVal, ts)
	}
}

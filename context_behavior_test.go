package apex

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

type trackedBody struct {
	io.Reader
	closed bool
	err    error
}

func (b *trackedBody) Close() error {
	b.closed = true
	return b.err
}

type failingWriter struct {
	header http.Header
	err    error
	status int
}

func (w *failingWriter) Header() http.Header {
	return w.header
}

func (w *failingWriter) WriteHeader(status int) {
	w.status = status
}

func (w *failingWriter) Write([]byte) (int, error) {
	return 0, w.err
}

type failingReader struct {
	err error
}

func (r failingReader) Read([]byte) (int, error) {
	return 0, r.err
}

func TestContextBinding(t *testing.T) {
	var cases = []struct {
		name        string
		bind        func(*Context, any) error
		contentType string
		body        string
		wantError   bool
	}{
		{"automatic JSON", (*Context).Bind, "application/json", `{"name":"Ada"}`, false},
		{"default JSON", (*Context).Bind, "", `{"name":"Ada"}`, false},
		{"automatic form", (*Context).Bind, formContentType + "; charset=utf-8", "name=Ada", false},
		{"explicit JSON", (*Context).BindJSON, "text/plain", `{"name":"Ada"}`, false},
		{"explicit form", (*Context).BindForm, "text/plain", "name=Ada", false},
		{"malformed automatic JSON", (*Context).Bind, "application/json", `{`, true},
		{"malformed JSON", (*Context).BindJSON, "", `{`, true},
		{"malformed automatic form", (*Context).Bind, formContentType, "name=%zz", true},
		{"malformed form", (*Context).BindForm, "", "name=%zz", true},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			var value struct {
				Name string `json:"name" form:"name"`
			}
			var body = &trackedBody{Reader: strings.NewReader(test.body)}
			var request = httptest.NewRequest(http.MethodPost, "/", nil)
			request.Body = body
			request.Header.Set("Content-Type", test.contentType)
			var ctx = newContext(httptest.NewRecorder(), request, nil)
			var err = test.bind(ctx, &value)
			if (err != nil) != test.wantError {
				t.Fatalf("Bind() error = %v, want error %t", err, test.wantError)
			}

			if !body.closed {
				t.Error("Bind did not close the request body")
			}

			if !test.wantError && value.Name != "Ada" {
				t.Errorf("bound name = %q, want Ada", value.Name)
			}
		})
	}
}

func TestContextWrites(t *testing.T) {
	var cases = []struct {
		name        string
		write       Handler
		status      int
		contentType string
		body        string
		location    string
	}{
		{"bytes", func(c *Context) { c.WriteBytes(201, "text/custom", []byte("created")) }, 201, "text/custom", "created", ""},
		{"reader", func(c *Context) { c.WriteReader(202, "text/plain", strings.NewReader("accepted")) }, 202, "text/plain", "accepted", ""},
		{"JSON", func(c *Context) { c.WriteJSON(200, "hello") }, 200, "application/json", "{\"data\":\"hello\"}\n", ""},
		{"no content", (*Context).WriteNoContent, 204, "", "", ""},
		{"redirect", func(c *Context) { c.Redirect(307, "/next") }, 307, "", "", "/next"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			var recorder = httptest.NewRecorder()
			var ctx = newContext(recorder, httptest.NewRequest("GET", "/", nil), nil)
			var reported []error
			ctx.errorFn = func(err error) { reported = append(reported, err) }
			test.write(ctx)
			if recorder.Code != test.status || ctx.statusCode != test.status || !ctx.completed {
				t.Fatalf("write status = %d/%d, completed = %t", recorder.Code, ctx.statusCode, ctx.completed)
			}

			if recorder.Body.String() != test.body || recorder.Header().Get("Content-Type") != test.contentType || recorder.Header().Get("Location") != test.location {
				t.Fatalf("response = %v %q", recorder.Header(), recorder.Body.String())
			}

			test.write(ctx)
			if len(reported) != 1 || !errors.Is(reported[0], ErrContextIsClosed) {
				t.Fatalf("second write errors = %v, want ErrContextIsClosed", reported)
			}

			if recorder.Body.String() != test.body {
				t.Error("second write changed the body")
			}
		})
	}
}

func TestContextWriteFailures(t *testing.T) {
	var failure = errors.New("I/O failure")
	var cases = []struct {
		name      string
		write     Handler
		badWriter bool
		wantCause bool
		committed bool
	}{
		{"bytes writer", func(c *Context) { c.WriteBytes(200, "text/plain", []byte("hello")) }, true, true, true},
		{"reader writer", func(c *Context) { c.WriteReader(200, "text/plain", strings.NewReader("hello")) }, true, true, true},
		{"reader source", func(c *Context) { c.WriteReader(200, "text/plain", failingReader{failure}) }, false, true, true},
		{"JSON writer", func(c *Context) { c.WriteJSON(200, "hello") }, true, true, true},
		{"JSON invalid error value", func(c *Context) { c.WriteJSON(400, "invalid") }, false, false, false},
		{"JSON encoding", func(c *Context) { c.WriteJSON(200, make(chan int)) }, false, false, false},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			var writer http.ResponseWriter = httptest.NewRecorder()
			if test.badWriter {
				writer = &failingWriter{header: make(http.Header), err: failure}
			}

			var ctx = newContext(writer, httptest.NewRequest("GET", "/", nil), nil)
			var reported []error
			ctx.errorFn = func(err error) { reported = append(reported, err) }
			test.write(ctx)
			if len(reported) != 1 || reported[0] == nil || !ctx.completed {
				t.Fatalf("errors = %v, completed = %t", reported, ctx.completed)
			}

			if test.wantCause && !errors.Is(reported[0], failure) {
				t.Errorf("error = %v, want I/O cause", reported[0])
			}

			if (ctx.statusCode != 0) != test.committed {
				t.Errorf("committed status = %d, want committed %t", ctx.statusCode, test.committed)
			}
		})
	}
}

func TestContextAutomaticRedirects(t *testing.T) {
	var writes = []struct {
		name  string
		write Handler
	}{
		{"bytes", func(c *Context) { c.WriteBytes(201, "text/plain", []byte("unused")) }},
		{"reader", func(c *Context) { c.WriteReader(201, "text/plain", strings.NewReader("unused")) }},
		{"JSON", func(c *Context) { c.WriteJSON(201, "unused") }},
		{"no content", (*Context).WriteNoContent},
	}
	for _, test := range writes {
		t.Run(test.name, func(t *testing.T) {
			var recorder = httptest.NewRecorder()
			var request = httptest.NewRequest("POST", "/?redirect=%2Fnext", nil)
			request.Header.Set("Accept", "text/html,application/json")
			var ctx = newContext(recorder, request, nil)
			test.write(ctx)
			if recorder.Code != 302 || recorder.Header().Get("Location") != "/next" || recorder.Body.Len() != 0 || !ctx.completed {
				t.Fatalf("response = %d %v %q", recorder.Code, recorder.Header(), recorder.Body.String())
			}
		})
	}
}

func TestContextRedirectSelection(t *testing.T) {
	var cases = []struct {
		name      string
		method    string
		status    int
		accept    string
		query     string
		want      string
		wantError bool
	}{
		{"informational", "POST", 100, "text/html", "redirect=/next", "", false},
		{"error", "POST", 400, "text/html", "redirect=/next", "", false},
		{"GET content", "GET", 200, "text/html", "redirect=/next", "", false},
		{"GET no content", "GET", 204, "text/html", "redirect=/next", "/next", false},
		{"JSON preferred", "POST", 200, "application/json,text/html", "redirect=/next", "", false},
		{"missing destination", "POST", 200, "text/html", "other=value", "", false},
		{"empty destination", "POST", 200, "text/html", "redirect=", "", false},
		{"malformed query", "POST", 200, "text/html", "redirect=%zz", "", true},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			var request = httptest.NewRequest(test.method, "/?"+test.query, nil)
			request.Header.Set("Accept", test.accept)
			var ctx = newContext(httptest.NewRecorder(), request, nil)
			var reported error
			ctx.errorFn = func(err error) { reported = err }
			var destination, ok = ctx.getRedirect(test.status)
			if destination != test.want || ok != (test.want != "") || (reported != nil) != test.wantError {
				t.Fatalf("getRedirect() = %q/%t, error = %v", destination, ok, reported)
			}
		})
	}
}

func TestContextAccessorsAndHookOrder(t *testing.T) {
	var recorder = httptest.NewRecorder()
	var request = httptest.NewRequest("GET", "/", nil)
	var ctx = newContext(recorder, request, Params{{Key: "id", Value: "42"}})
	if ctx.Writer() != recorder || ctx.Request() != request || ctx.Param("id") != "42" || ctx.Param("missing") != "" || ctx.Get("missing") != "" {
		t.Fatal("context accessors returned incorrect request state")
	}

	ctx.Put("key", "first")
	ctx.Put("key", "second")
	if ctx.Get("key") != "second" {
		t.Fatal("Put did not replace the stored value")
	}

	var calls []int
	for index := 0; index < 3; index++ {
		ctx.AddHook(func(status int, received *Context) {
			if status != 202 || received != ctx {
				t.Errorf("hook received status %d, context %p", status, received)
			}

			calls = append(calls, index)
		})
	}

	ctx.statusCode = 202
	ctx.processHooks()
	if !reflect.DeepEqual(calls, []int{2, 1, 0}) {
		t.Fatalf("hook order = %v, want reverse registration order", calls)
	}
}

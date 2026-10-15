package httpserve

import (
	"net/http/httptest"
	"testing"
)

func TestContextReset(t *testing.T) {
	var (
		params = Params{{Key: "id", Value: "42"}, {Key: "hidden", Value: "retained"}}
		hooks  = []Hook{func(int, *Context) {}, func(int, *Context) {}}
		values = Storage{"key": "value"}
		ctx    = &Context{
			writer: httptest.NewRecorder(), request: httptest.NewRequest("GET", "/", nil),
			errorFn: func(error) {}, completed: true, statusCode: 204,
			Params: params[:1], hooks: hooks[:1], s: values,
		}
	)
	ctx.reset()
	if ctx.writer != nil || ctx.request != nil || ctx.errorFn != nil || ctx.completed || ctx.statusCode != 0 {
		t.Fatal("reset retained request state")
	}

	if len(ctx.Params) != 0 || cap(ctx.Params) != 2 || len(ctx.hooks) != 0 || cap(ctx.hooks) != 2 || len(values) != 0 {
		t.Fatal("reset did not empty reusable storage")
	}

	for index := range params {
		if params[index] != (Param{}) || hooks[index] != nil {
			t.Fatalf("reset retained references in slot %d", index)
		}
	}

	ctx.Put("reused", "yes")
	if values["reused"] != "yes" {
		t.Fatal("reset replaced the storage map")
	}
}

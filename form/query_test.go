package query

import (
	"net/url"
	"strings"
	"testing"
)

func TestBind(t *testing.T) {
	type testStruct struct {
		Foo string `form:"foo"`
		Bar int64  `form:"bar"`
	}

	u := make(url.Values)
	u.Set("foo", "hello world!")
	u.Set("bar", "1337")
	u.Add("multi", "1")
	u.Add("multi", "2")
	u.Add("multi", "3")

	var test testStruct
	if err := BindReader(strings.NewReader(u.Encode()), &test); err != nil {
		t.Fatal(err)
	}

	if test.Foo != "hello world!" {
		t.Fatalf("invalid value, expected \"%s\" and received \"%s\"", "hello world!", test.Foo)
	}

	if test.Bar != 1337 {
		t.Fatalf("invalid value, expected %d and received %d", 1337, test.Bar)
	}
}

package form

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

var (
	testQuery       = newTestQuery()
	testQueryString = testQuery.Encode()

	sink testStruct
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

func TestDecode(t *testing.T) {
	type testStruct struct {
		Email    string `json:"email" form:"email"`
		Password string `json:"password,omitempty" form:"password"`
	}

	var val testStruct
	str := "email=test1%40test.com&password=foobar&passwordConfirm=foobar"
	// Ensure bufio is used so every aspect is tested
	rdr := &ronly{bytes.NewBufferString(str)}
	if err := BindReader(rdr, &val); err != nil {
		t.Fatal(err)
	}

	if val.Email != "test1@test.com" {
		t.Fatalf("invalid email: %v", val.Email)
	}

	if val.Password != "foobar" {
		t.Fatalf("invalid password: %v", val.Password)
	}
}

func TestDecode_redirect(t *testing.T) {
	type testStruct struct {
		Redirect string `form:"redirect"`
	}

	var val testStruct
	str := "redirect=/dashboard"
	// Ensure bufio is used so every aspect is tested
	rdr := &ronly{bytes.NewBufferString(str)}
	if err := BindReader(rdr, &val); err != nil {
		t.Fatal(err)
	}

	if val.Redirect != "/dashboard" {
		t.Fatalf("invalid redirect: %v", val.Redirect)
	}
}

//Raw query redirect=/dashboard
//Raw query bs [114 101 100 105 114 101 99 116 61 47 100 97 115 104 98 111 97 114 100]
//RQ? {}

func BenchmarkBindReader(b *testing.B) {
	var test testStruct
	for b.Loop() {
		if err := BindReader(strings.NewReader(testQueryString), &test); err != nil {
			b.Fatal(err)
		}
	}

	b.ReportAllocs()
}

func BenchmarkBindReader_unmarshaler(b *testing.B) {
	var test testUnmarshaler
	rdr := strings.NewReader(testQueryString)
	for b.Loop() {
		if err := BindReader(rdr, &test); err != nil {
			b.Fatal(err)
		}

		if _, err := rdr.Seek(0, 0); err != nil {
			b.Fatal(err)
		}
	}

	b.ReportAllocs()
}

func BenchmarkDecoder_Decode_unmarshaler(b *testing.B) {
	var test testUnmarshaler
	rdr := strings.NewReader(testQueryString)

	for b.Loop() {
		if err := NewDecoder(rdr).Decode(&test); err != nil {
			b.Fatal(err)
		}

		if _, err := rdr.Seek(0, 0); err != nil {
			b.Fatal(err)
		}
	}

	b.ReportAllocs()
}

func BenchmarkStdlib(b *testing.B) {
	var req http.Request
	rdr := strings.NewReader(testQueryString)
	req.Body = io.NopCloser(rdr)
	req.Method = "POST"
	req.Header = make(http.Header)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	for b.Loop() {
		var err error
		if err = req.ParseForm(); err != nil {
			b.Fatal(err)
		}

		sink.Foo = req.Form.Get("foo")
		if sink.Bar, err = strconv.ParseInt(req.Form.Get("bar"), 10, 64); err != nil {
			b.Fatal(err)
		}

		req.Form = nil
		req.PostForm = nil

		if _, err := rdr.Seek(0, 0); err != nil {
			b.Fatal(err)
		}

	}

	b.ReportAllocs()
}

func newTestQuery() url.Values {
	u := make(url.Values)
	u.Set("foo", "hello world!")
	u.Set("bar", "1337")
	return u
}

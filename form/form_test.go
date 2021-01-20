package form

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

var (
	testStructSink  testStruct
	testQuery       = newTestQuery()
	testQueryString = testQuery.Encode()
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
	for i := 0; i < b.N; i++ {
		if err := BindReader(strings.NewReader(testQueryString), &test); err != nil {
			b.Fatal(err)
		}
	}

	b.ReportAllocs()
}

func BenchmarkBindReader_unmarshaler(b *testing.B) {
	var test testUnmarshaler
	rdr := strings.NewReader(testQueryString)
	for i := 0; i < b.N; i++ {
		if err := BindReader(rdr, &test); err != nil {
			b.Fatal(err)
		}

		rdr.Seek(0, 0)
	}

	b.ReportAllocs()
}

func BenchmarkDecoder_Decode_unmarshaler(b *testing.B) {
	var test testUnmarshaler
	rdr := strings.NewReader(testQueryString)

	for i := 0; i < b.N; i++ {
		if err := NewDecoder(rdr).Decode(&test); err != nil {
			b.Fatal(err)
		}

		rdr.Seek(0, 0)
	}

	b.ReportAllocs()
}

func BenchmarkStdlib(b *testing.B) {
	var (
		test testStruct
		req  http.Request
	)

	rdr := strings.NewReader(testQueryString)
	req.Body = ioutil.NopCloser(rdr)
	req.Method = "POST"
	req.Header = make(http.Header)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	for i := 0; i < b.N; i++ {
		var err error
		if err = req.ParseForm(); err != nil {
			b.Fatal(err)
		}

		test.Foo = req.Form.Get("foo")
		if test.Bar, err = strconv.ParseInt(req.Form.Get("bar"), 10, 64); err != nil {
			b.Fatal(err)
		}

		req.Form = nil
		req.PostForm = nil
		rdr.Seek(0, 0)
	}

	b.ReportAllocs()
}

func newTestQuery() url.Values {
	u := make(url.Values)
	u.Set("foo", "hello world!")
	u.Set("bar", "1337")
	return u
}

type testStruct struct {
	Foo string `form:"foo"`
	Bar int64  `form:"bar"`
}

type testUnmarshaler struct {
	testStruct
}

func (t *testUnmarshaler) UnmarshalForm(key, value string) (err error) {
	switch key {
	case "foo":
		t.Foo = value
	case "bar":
		t.Bar, err = strconv.ParseInt(value, 10, 64)
	}

	return
}

type ronly struct {
	r io.Reader
}

func (r *ronly) Read(p []byte) (n int, err error) {
	return r.r.Read(p)
}

var _ io.Reader = &ronly{}

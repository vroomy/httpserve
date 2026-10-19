package form

import (
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
)

type recordingUnmarshaler struct {
	pairs [][2]string
	err   error
}

func (u *recordingUnmarshaler) UnmarshalForm(key, value string) error {
	u.pairs = append(u.pairs, [2]string{key, value})
	return u.err
}

type failingReader struct {
	err error
}

func (r failingReader) Read([]byte) (int, error) {
	return 0, r.err
}

func TestDecoderQueryValues(t *testing.T) {
	var cases = []struct {
		name  string
		query string
		want  [][2]string
	}{
		{"empty", "", nil},
		{"empty separators", "&&", nil},
		{"key only", "flag", [][2]string{{"flag", ""}}},
		{"empty value", "key=", [][2]string{{"key", ""}}},
		{"empty key", "=value", [][2]string{{"", "value"}}},
		{"duplicate keys", "key=first&key=second&", [][2]string{{"key", "first"}, {"key", "second"}}},
		{"unescaping", "first+name=Ada+Lovelace&%E5%90%8D=%E5%80%A4&token=a%3Db%26c", [][2]string{{"first name", "Ada Lovelace"}, {"名", "値"}, {"token", "a=b&c"}}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			var target recordingUnmarshaler
			var err = NewDecoder(strings.NewReader(test.query)).Decode(&target)
			if err != nil || !reflect.DeepEqual(target.pairs, test.want) {
				t.Fatalf("Decode() = %v/%v, want %v", target.pairs, err, test.want)
			}
		})
	}
}

func TestDecoderFailures(t *testing.T) {
	var failure = errors.New("read failed")
	var cases = []struct {
		name   string
		reader io.Reader
		cause  error
	}{
		{"invalid key escape", strings.NewReader("%zz=value"), nil},
		{"invalid value escape", strings.NewReader("key=%zz"), nil},
		{"invalid middle pair", strings.NewReader("key=%zz&next=value"), nil},
		{"reader failure", failingReader{failure}, failure},
		{"reader failure after pair", io.MultiReader(strings.NewReader("key=value&partial"), failingReader{failure}), failure},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			var target recordingUnmarshaler
			var err = NewDecoder(test.reader).Decode(&target)
			if err == nil {
				t.Fatal("Decode() accepted invalid input")
			}

			if test.cause != nil && !errors.Is(err, test.cause) {
				t.Fatalf("Decode() error = %v, want reader cause", err)
			}
		})
	}
}

func TestDecoderUnmarshalerFailure(t *testing.T) {
	var failure = errors.New("unmarshal failed")
	var target = recordingUnmarshaler{err: failure}
	var err = Unmarshal("first=value&second=unused", &target)
	if !errors.Is(err, failure) || !reflect.DeepEqual(target.pairs, [][2]string{{"first", "value"}}) {
		t.Fatalf("Unmarshal() = %v/%v", target.pairs, err)
	}
}

func TestUnmarshalStructConversion(t *testing.T) {
	var value struct {
		Name string `form:"name"`
		Age  int64  `form:"age"`
	}
	var err = Unmarshal("name=Ada&age=36", &value)
	if err != nil || value.Name != "Ada" || value.Age != 36 {
		t.Fatalf("Unmarshal() = %#v/%v", value, err)
	}

	if err = Unmarshal("age=invalid", &value); err == nil {
		t.Fatal("Unmarshal accepted a non-numeric integer")
	}
}

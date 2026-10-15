package form

import "strconv"

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

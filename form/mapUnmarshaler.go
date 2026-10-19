package form

import (
	"reflect"

	"github.com/gdbu/reflectio"
)

func newMapUnmarshaler(value any) *mapUnmarshaler {
	var m mapUnmarshaler
	if m.rval = reflect.ValueOf(value); m.rval.Kind() == reflect.Pointer {
		m.rval = m.rval.Elem()
	}

	m.m = cache.Get(value, "form")
	return &m
}

type mapUnmarshaler struct {
	m    reflectio.Map
	rval reflect.Value
}

func (m *mapUnmarshaler) UnmarshalForm(key, value string) (err error) {
	return m.m.SetValueAsString(m.rval, key, value)
}

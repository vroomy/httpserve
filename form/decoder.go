package query

import (
	"bufio"
	"io"
	"net/url"
	"reflect"

	"github.com/gdbu/reflectio"
)

// NewDecoder will initialize a new decoder
func NewDecoder(r io.Reader) *Decoder {
	var d Decoder
	d.r = bufio.NewReader(r)
	return &d
}

// Decoder will decode a value
type Decoder struct {
	rval reflect.Value

	m reflectio.Map
	r *bufio.Reader

	char       rune
	seenEquals bool

	keyBuf []rune
	valBuf []rune
}

// Decode will decode a provided value
func (d *Decoder) Decode(value interface{}) (err error) {
	// Set value for decoder
	d.setValue(value)

	// Iterate through runes
	for d.char, _, err = d.r.ReadRune(); err == nil; d.char, _, err = d.r.ReadRune() {
		switch d.char {
		case '=':
			d.processEquals()
		case '&':
			err = d.processAmpersand()
		default:
			err = d.processChar()
		}
	}

	if err == io.EOF {
		err = nil
	}

	return d.processAmpersand()
}

func (d *Decoder) setValue(value interface{}) {
	if d.rval = reflect.ValueOf(value); d.rval.Kind() == reflect.Ptr {
		d.rval = d.rval.Elem()
	}

	d.m = cache.Get(value, "form")
}

func (d *Decoder) reset() {
	d.seenEquals = false
	d.keyBuf = d.keyBuf[:0]
	d.valBuf = d.valBuf[:0]
}

func (d *Decoder) processEquals() {
	d.seenEquals = true
}

func (d *Decoder) processAmpersand() (err error) {
	if len(d.keyBuf) == 0 && len(d.valBuf) == 0 {
		return
	}

	var key string
	if key, err = url.QueryUnescape(string(d.keyBuf)); err != nil {
		return
	}

	var val string
	if val, err = url.QueryUnescape(string(d.valBuf)); err != nil {
		return
	}

	d.reset()

	return d.m.SetValueAsString(d.rval, key, val)
}

func (d *Decoder) processChar() (err error) {
	if !d.seenEquals {
		d.keyBuf = append(d.keyBuf, d.char)
		return
	}

	d.valBuf = append(d.valBuf, d.char)
	return
}

package httpserve

// Params represent route parameters
type Params map[string]string

// ByName will return a value for a given key
func (p Params) ByName(key string) (value string) {
	return p[key]
}

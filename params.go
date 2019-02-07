package httpserve

// Params represent route parameters
type Params map[string]string

func (p Params) clear() {
	for key := range p {
		delete(p, key)
	}
}

// ByName will return a value for a given key
func (p Params) ByName(key string) (value string) {
	return p[key]
}

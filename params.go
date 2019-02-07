package httpserve

// Params represent route parameters
type Params []Param

// ByName will return a value for a given key
func (p Params) ByName(key string) (value string) {
	for _, kv := range p {
		if kv.Key == key {
			return kv.Value
		}
	}

	return ""
}

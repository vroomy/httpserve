package form

// Unmarshaler is an decoding helper interface
type Unmarshaler interface {
	UnmarshalForm(key, value string) error
}

package httpserve

// Group is a grouping interface
type Group interface {
	GET(route string, hs ...Handler) error
	POST(route string, hs ...Handler) error
	PUT(route string, hs ...Handler) error
	DELETE(route string, hs ...Handler) error
	OPTIONS(route string, hs ...Handler) error

	Handle(method, route string, hs ...Handler) error
	Group(route string, hs ...Handler) Group
}

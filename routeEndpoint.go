package httpserve

// routeEndpoint holds the handler and parameter names for a terminal trie match.
// Values are captured during traversal; names belong to the selected endpoint.
type routeEndpoint struct {
	handler    Handler
	paramNames []string
}

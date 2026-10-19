package apex

import (
	"fmt"
	"log"
	"net/http"
)

const colon = ':'

func newRouter() *Router {
	var r Router
	r.SetNotFound(notFoundHandler)
	r.SetPanic(r.onPanic)
	return &r
}

// Router handles routes using a compressed prefix trie for each HTTP method.
// Static branches take precedence over parameters, followed by wildcards.
// Register routes and configure handlers before serving requests; lookups can
// then run concurrently.
type Router struct {
	// rm is indexed by methodIndex for O(1) array access instead of map hashing.
	rm [numMethods]route

	notFound Handler
	panic    PanicHandler

	errorFn func(error)
}

func (r *Router) onPanic(v any) {
	log.Println("Panic encountered:", v)
}

// Match returns a matching handler and its parameters, preferring static branches
// over parameters and parameters over wildcards. Failed branches fall back to
// less specific matches. Duplicate patterns retain their first handler.
// The returned Params belong to the caller. A miss returns the not-found handler.
// Parameter storage is allocated only for a successful match with named parameters.
// Matches without named parameters and misses return nil Params.
func (r *Router) Match(method, url string) (h Handler, p Params, ok bool) {
	return r.MatchInto(method, url, nil)
}

// MatchInto matches a route using caller-owned parameter storage. It clears the
// supplied slice's existing elements and replaces them with the selected route's
// parameters. It returns an empty slice on a miss and grows storage only when a
// successful match exceeds its capacity. A nil slice is valid.
// Retain the returned slice for subsequent calls. Reusing storage overwrites prior
// results; concurrent calls must use separate backing arrays. Strings reference
// the request path and registered parameter names without copying their contents.
func (r *Router) MatchInto(method, url string, params Params) (h Handler, p Params, ok bool) {
	clear(params)
	p = params[:0]
	var idx = methodToIndex(method)
	if idx == methodUnknown {
		return r.notFound, p, false
	}

	var endpoint = r.rm[idx].find(url, &p)
	if endpoint == nil {
		return r.notFound, p, false
	}

	return endpoint.handler, p, true
}

// match is the hot-path route matcher used by ServeHTTP. It fills the provided
// Params slice in-place (from the pooled Context) instead of allocating a new
// one, eliminating a per-request heap allocation.
func (r *Router) match(method, url string, p *Params) Handler {
	var idx = methodToIndex(method)
	if idx == methodUnknown {
		return r.notFound
	}

	var endpoint = r.rm[idx].find(url, p)
	if endpoint == nil {
		if r.rm[idx].count > 0 {
			clear(*p)
			*p = (*p)[:0]
		}

		return r.notFound
	}

	return endpoint.handler
}

// SetNotFound will set the not found handler (404)
func (r *Router) SetNotFound(hs ...Handler) {
	r.notFound = newHandler(hs)
}

// SetPanic will set panic handler
func (r *Router) SetPanic(h PanicHandler) {
	r.panic = h
}

// SetOnError sets the error callback. A nil callback restores default logging.
// Configure the callback before serving requests.
func (r *Router) SetOnError(onError func(error)) {
	r.errorFn = onError
}

// Handle registers a route for a supported HTTP method. The path must be
// nonempty and start with "/". Patterns are parsed only during registration.
func (r *Router) Handle(method, url string, h Handler) (err error) {
	var idx = methodToIndex(method)
	if idx == methodUnknown {
		return fmt.Errorf("unsupported HTTP method: %s", method)
	}

	if len(url) == 0 || url[0] != '/' {
		return fmt.Errorf("error creating route for [%s] \"%s\": %v", method, url, ErrMissingLeadSlash)
	}

	var parts = getParts(url)
	r.rm[idx].add(parts, h)
	return nil
}

// GET will create a GET route
func (r *Router) GET(url string, h Handler) error {
	return r.Handle("GET", url, h)
}

// PUT will create a PUT route
func (r *Router) PUT(url string, h Handler) error {
	return r.Handle("PUT", url, h)
}

// POST will create a POST route
func (r *Router) POST(url string, h Handler) error {
	return r.Handle("POST", url, h)
}

// DELETE will create a DELETE route
func (r *Router) DELETE(url string, h Handler) error {
	return r.Handle("DELETE", url, h)
}

// OPTIONS will create an OPTIONS route
func (r *Router) OPTIONS(url string, h Handler) error {
	return r.Handle("OPTIONS", url, h)
}

func (r *Router) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	ctx := acquireContext(rw, req)
	ctx.errorFn = r.errorFn
	if ctx.errorFn == nil {
		ctx.errorFn = defaultRouterError
	}

	h := r.match(req.Method, req.URL.Path, &ctx.Params)

	// panicked starts true; set to false on clean exit so the deferred
	// recovery only fires when an actual panic occurred.
	panicked := true
	defer func() {
		if panicked {
			v := recover()
			if r.panic != nil {
				r.panic(v)
			}

			rw.WriteHeader(500)
		}

		releaseContext(ctx)
	}()

	h(ctx)
	panicked = false
}

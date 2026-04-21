package httpserve

import (
	"fmt"
	"log"
	"net/http"
)

const (
	colon        = ':'
	forwardSlash = "/"
)

// methodIndex maps HTTP method strings to a compact array index,
// replacing the map[string]routes lookup with a direct array access.
type methodIndex uint8

const (
	mGET     methodIndex = iota // 0
	mHEAD                       // 1
	mPOST                       // 2
	mPUT                        // 3
	mDELETE                     // 4
	mOPTIONS                    // 5
	numMethods = 6
)

func methodToIndex(m string) (methodIndex, bool) {
	switch m {
	case http.MethodGet:
		return mGET, true
	case http.MethodHead:
		return mHEAD, true
	case http.MethodPost:
		return mPOST, true
	case http.MethodPut:
		return mPUT, true
	case http.MethodDelete:
		return mDELETE, true
	case http.MethodOptions:
		return mOPTIONS, true
	default:
		return 0, false
	}
}

func newRouter() *Router {
	var r Router
	r.SetNotFound(notFoundHandler)
	r.SetPanic(r.onPanic)
	return &r
}

// Router handles routes
type Router struct {
	// rm is indexed by methodIndex for O(1) array access instead of map hashing.
	rm [numMethods]routes

	notFound Handler
	panic    PanicHandler

	errorFn func(error)

	maxParams int
}

func (r *Router) onPanic(v interface{}) {
	log.Println("Panic encountered:", v)
}

func (r *Router) onError(err error) {
	if r.errorFn != nil {
		r.errorFn(err)
		return
	}

	log.Println("Error encountered:", err)
}

// Match will check a url for a matching Handler, and return any associated handler and its parameters
func (r *Router) Match(method, url string) (h Handler, p Params, ok bool) {
	idx, valid := methodToIndex(method)
	if !valid {
		h = r.notFound
		return
	}

	rs := r.rm[idx]
	p = make(Params, 0, r.maxParams)
	for _, rt := range rs {
		if p, ok = rt.check(p, url); ok {
			h = rt.h
			return
		}
		p = p[:0]
	}

	h = r.notFound
	return
}

// match is the hot-path route matcher used by ServeHTTP. It fills the provided
// Params slice in-place (from the pooled Context) instead of allocating a new
// one, eliminating a per-request heap allocation.
func (r *Router) match(method, url string, p *Params) Handler {
	idx, valid := methodToIndex(method)
	if !valid {
		return r.notFound
	}

	var ok bool
	for _, rt := range r.rm[idx] {
		if *p, ok = rt.check(*p, url); ok {
			return rt.h
		}
		*p = (*p)[:0]
	}

	return r.notFound
}

// SetNotFound will set the not found handler (404)
func (r *Router) SetNotFound(hs ...Handler) {
	r.notFound = newHandler(hs)
}

// SetPanic will set panic handler
func (r *Router) SetPanic(h PanicHandler) {
	r.panic = h
}

func (r *Router) SetOnError(onError func(error)) {
	r.errorFn = onError
}

// Handle will create a route for any method
func (r *Router) Handle(method, url string, h Handler) (err error) {
	idx, valid := methodToIndex(method)
	if !valid {
		return fmt.Errorf("unsupported HTTP method: %s", method)
	}

	var rt *route
	if rt, err = newRoute(url, h, method); err != nil {
		return fmt.Errorf("error creating route for [%s] \"%s\": %v", method, url, err)
	}

	if n := rt.numParams(); n > r.maxParams {
		r.maxParams = n
	}

	r.rm[idx] = append(r.rm[idx], rt)
	return
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
	ctx.errorFn = r.onError

	h := r.match(req.Method, req.URL.Path, &ctx.Params)

	// panicked starts true; set to false on clean exit so the deferred
	// recovery only calls recover() when an actual panic occurred,
	// avoiding the closure allocation cost on every normal request.
	panicked := true
	defer func() {
		if panicked && r.panic != nil {
			r.panic(recover())
			rw.WriteHeader(500)
		}
		releaseContext(ctx)
	}()

	h(ctx)
	panicked = false
}

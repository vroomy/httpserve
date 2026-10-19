package apex

// newHandler will return a new Handler
func newHandler(hs []Handler) Handler {
	return func(ctx *Context) {
		// Process handler funcs
		ctx.processHandlers(hs)
		if ctx.completed {
			closeAndReport(ctx.request.Body)
		}

		// Process context hooks
		ctx.processHooks()
	}
}

// Handler is the HTTP handler type
type Handler func(ctx *Context)

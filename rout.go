/*
Experimental router for Go HTTP servers. Imperative control flow with
declarative syntax. Doesn't need middleware.

Very simple, small (≈300 LoC without docs), dependency-free, reasonably fast.

See `Route` for an example. See `readme.md` for additional info such as
motivation and advantages.
*/
package rout

import (
	"io"
	"net/http"
)

type (
	// Shortcut for brevity.
	R = Router

	// Non-parametrized handler func. Same as `http.HandlerFunc`. The type of
	// functions passed to `Router.Func`.
	Func = http.HandlerFunc

	// Parametrized handler func. The type of functions passed to
	// `Router.ParamFunc`. Takes additional args, produced by parenthesized
	// regexp capture groups. Args start at index 0, not 1 like in a regexp
	// match.
	ParamFunc = func(http.ResponseWriter, *http.Request, []string)

	// Short for "responder". The type of functions passed to `Router.Res`.
	Res func(*http.Request) http.Handler

	// Short for "parametrized responder". The type of functions passed to
	// `Router.ParamRes`.
	ParamRes = func(*http.Request, []string) http.Handler
)

// Implement `http.Handler`.
func (self Res) ServeHTTP(rew http.ResponseWriter, req *http.Request) {
	if self != nil {
		handler := self(req)
		if handler != nil {
			handler.ServeHTTP(rew, req)
		}
	}
}

/*
Makes a router for the given request-response. Usage:

	err := MakeRouter(rew, req).Route(myRoutes)
*/
func MakeRouter(rew http.ResponseWriter, req *http.Request) Router {
	return Router{Rew: rew, Req: req}
}

/*
Shortcut for top-level error handling.

If the error is nil, do nothing. If the error is non-nil, write its message as
plain text. By default, the HTTP status code is 500. If the error implements
`interface{ HttpStatusCode() int }` or contains `rout.Err`, HTTP status code is
derived from the error.

Example:

	rout.WriteErr(rew, rout.MakeRouter(rew, req).Route(myRoutes))
*/
func WriteErr(rew http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	rew.WriteHeader(errStatus(err))
	_, _ = io.WriteString(rew, err.Error())
}

/*
Shortcut for routing with default error handling. Same as `rout.Router.Route`,
but instead of returning an error, uses `rout.WriteErr` to write it. Example:

	rout.MakeRouter(rew, req).Serve(myRoutes)
*/
func (self Router) Serve(fun func(Router)) {
	WriteErr(self.Rew, self.Route(fun))
}

/*
Routes the given request-response, recovering from panics inherent to the
routing flow of this package. The resulting error is usually of type `Err`,
containing an appropriate HTTP status code. Your code must handle the error,
sending back an appropriate response. If routing was performed successfully,
the error is nil.

Same as `Router.Sub`, but catches panics, returning them as errors.
*/
func (self Router) Route(fun func(Router)) (err error) {
	defer rec(&err)
	self.Sub(fun)
	return
}

/*
Main router type. Should be used via `Route`, which handles panics inherent to
the routing flow.
*/
type Router struct {
	Rew http.ResponseWriter
	Req *http.Request

	pattern string
	method  string
	lax     bool
}

/*
Returns a router with the provided regexp pattern. The pattern will be used to
match `req.URL.Path`.
*/
func (self Router) Reg(val string) Router {
	self.pattern = val
	return self.Lax(false)
}

/*
Returns a router that matches only the provided method. If the method is empty,
the resulting router matches all methods, which is the default.

Note: to match multiple methods for one route, use `Router.Methods`. Otherwise,
the first mismatch generates `Err{Status: http.StatusMethodNotAllowed`}.
*/
func (self Router) Method(val string) Router {
	self.method = val
	return self
}

/*
Returns a router set to lax/permissive mode.

In strict mode (default), whenever the router matches the URL pattern but
doesn't match the HTTP method, it immediately generates a "method not allowed"
error. `Router.Reg` automatically switches the router into strict mode.

In lax mode (opt-in), if either URL pattern or HTTP status doesn't match, the
router simply proceeds to other routes, without generating an error.
`Router.Methods` automatically switches the router into lax mode.
*/
func (self Router) Lax(val bool) Router {
	self.lax = val
	return self
}

// Same as `.Method(http.MethodGet)`.
// Returns a router that matches only this HTTP method.
func (self Router) Get() Router { return self.Method(http.MethodGet) }

// Same as `.Method(http.MethodHead)`.
// Returns a router that matches only this HTTP method.
func (self Router) Head() Router { return self.Method(http.MethodHead) }

// Same as `.Method(http.MethodOptions)`.
// Returns a router that matches only this HTTP method.
func (self Router) Options() Router { return self.Method(http.MethodOptions) }

// Same as `.Method(http.MethodPost)`.
// Returns a router that matches only this HTTP method.
func (self Router) Post() Router { return self.Method(http.MethodPost) }

// Same as `.Method(http.MethodPatch)`.
// Returns a router that matches only this HTTP method.
func (self Router) Patch() Router { return self.Method(http.MethodPatch) }

// Same as `.Method(http.MethodPut)`.
// Returns a router that matches only this HTTP method.
func (self Router) Put() Router { return self.Method(http.MethodPut) }

// Same as `.Method(http.MethodDelete)`.
// Returns a router that matches only this HTTP method.
func (self Router) Delete() Router { return self.Method(http.MethodDelete) }

/*
If the router matches the request, perform sub-routing. If sub-routing doesn't
find a match, panic with `Err{Status: http.StatusNotFound}`.

If the router doesn't match the request, do nothing.
*/
func (self Router) Sub(fun func(Router)) {
	if !self.test() {
		return
	}
	if fun != nil {
		fun(self)
	}
	panic(errNotFound(self.req()))
}

/*
If the router matches the request, perform sub-routing. The router provided to
the function is automatically "lax": a mismatch in the HTTP method doesn't
immediately generate an error. However, if sub-routing doesn't find a match,
this panics with `Err{Status: http.StatusMethodNotAllowed}`.

If the router doesn't match the request, do nothing.
*/
func (self Router) Methods(fun func(Router)) {
	self.method = ``
	if !self.test() {
		return
	}

	if fun != nil {
		fun(self.Lax(true))
	}

	panic(errMethodNotAllowed(self.req()))
}

/*
Short for "recover". Recovers from a panic, and calls the provided function with
the resulting error or nil.

ALWAYS re-panics with nil to preserve the control flow. Must be used ONLY inside
routing functions:

	func routes(r rout.R) {
		defer r.Rec(writeErr)
		r.Get(`...`, someFunc)
	}

	func writeErr(rew http.ResponseWriter, req *http.Request, err error) {}

Reminiscent for middleware, which `rout` prides itself on not having. This
approach should be revised.
*/
func (self Router) Rec(fun func(http.ResponseWriter, *http.Request, error)) {
	err := toErr(recover())
	fun(self.Rew, self.Req, err)
	panic(nil)
}

/*
If the router matches the request, use the provided handler to respond.
If the router doesn't match the request, do nothing. The handler may be nil.
*/
func (self Router) Handler(val http.Handler) {
	if !self.test() {
		return
	}
	if val != nil {
		val.ServeHTTP(self.Rew, self.Req)
	}
	panic(nil)
}

/*
If the router matches the request, use the provided handler func to respond.
If the router doesn't match the request, do nothing. The func may be nil.
*/
func (self Router) Func(val Func) {
	if !self.test() {
		return
	}
	// Inline to simplify stacktraces.
	if val != nil {
		val(self.Rew, self.Req)
	}
	panic(nil)
}

/*
If the router matches the request, use the provided handler func to respond. If
the router doesn't match the request, do nothing. The func may be nil. The
additional `[]string` argument contains regexp captures from the pattern passed
to `Router.Reg`, if any.
*/
func (self Router) ParamFunc(val ParamFunc) {
	match := self.match()
	if match == nil {
		return
	}
	if val != nil {
		val(self.Rew, self.Req, match)
	}
	panic(nil)
}

/*
If the router matches the request, use the provided handler func to respond. If
the router doesn't match the request, do nothing. The func may be nil.
*/
func (self Router) Res(val Res) {
	if !self.test() {
		return
	}
	// Inline to simplify stacktraces.
	if val != nil {
		val := val(self.Req)
		if val != nil {
			val.ServeHTTP(self.Rew, self.Req)
		}
	}
	panic(nil)
}

/*
If the router matches the request, use the provided handler func to respond. If
the router doesn't match the request, do nothing. The func may be nil. The
additional `[]string` argument contains regexp captures from the pattern passed
to `Router.Reg`, if any.
*/
func (self Router) ParamRes(val ParamRes) {
	match := self.match()
	if match == nil {
		return
	}
	if val != nil {
		val := val(self.Req, match)
		if val != nil {
			val.ServeHTTP(self.Rew, self.Req)
		}
	}
	panic(nil)
}

func (self Router) testMethod() bool {
	req, method := self.Req, self.method
	return req != nil && (method == `` || method == req.Method)
}

func (self Router) testPattern() bool {
	req, pattern := self.Req, self.pattern
	return req != nil && reTest(req.URL.Path, pattern)
}

func (self Router) matchPattern() []string {
	req, pattern := self.Req, self.pattern
	if req == nil {
		return nil
	}
	return reMatch(req.URL.Path, pattern)
}

func (self Router) test() bool {
	if self.lax {
		return self.testLax()
	}
	return self.testStrict()
}

func (self Router) testLax() bool {
	return self.testMethod() && self.testPattern()
}

func (self Router) testStrict() bool {
	if !self.testPattern() {
		return false
	}
	if self.testMethod() {
		return true
	}
	panic(errMethodNotAllowed(self.req()))
}

func (self Router) match() []string {
	if self.lax {
		return self.matchLax()
	}
	return self.matchStrict()
}

func (self Router) matchLax() []string {
	if !self.testMethod() {
		return nil
	}
	return self.matchPattern()
}

func (self Router) matchStrict() []string {
	match := self.matchPattern()
	if match == nil {
		return nil
	}
	if self.testMethod() {
		return match
	}
	panic(errMethodNotAllowed(self.req()))
}

func (self Router) req() (string, string) {
	req := self.Req
	if req != nil {
		return req.Method, req.URL.Path
	}
	return ``, ``
}

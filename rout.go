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

	// Non-parametrized handler func. Same as `http.HandlerFunc`.
	// Type of functions passed to `Router.Func`.
	Func = http.HandlerFunc

	// Parametrized handler func. Type of functions passed to `Router.ParamFunc`.
	// Takes additional args, produced by parenthesized regexp capture groups.
	// Args start at index 0, not 1 like in a regexp match.
	ParamFunc = func(http.ResponseWriter, *http.Request, []string)

	// Short for "handler" or "handlerer".
	// Type of functions passed to `Router.Han`.
	// The returned `http.Handler` is used to write the response.
	// To represent responses with handlers, use "github.com/mitranim/goh".
	Han func(*http.Request) http.Handler

	// Short for "parametrized handler/handlerer".
	// Type of functions passed to `Router.ParamHan`.
	ParamHan = func(*http.Request, []string) http.Handler

	// Short for "responder". Type of functions passed to `Router.Res`.
	// The returned `*http.Response` is sent back via the function `Respond`.
	Res func(*http.Request) *http.Response

	// Short for "parametrized responder".
	// Type of functions passed to `Router.ParamRes`.
	ParamRes = func(*http.Request, []string) *http.Response
)

// Implement `http.Handler`.
func (self Han) ServeHTTP(rew http.ResponseWriter, req *http.Request) {
	if self != nil {
		han := self(req)
		if han != nil {
			han.ServeHTTP(rew, req)
		}
	}
}

// Implement `http.Handler`.
func (self Res) ServeHTTP(rew http.ResponseWriter, req *http.Request) {
	if self != nil {
		res := self(req)
		if res != nil {
			Respond(rew, res)
		}
	}
}

/*
Writes the given response. This is used internally by `Router.Res` and
`Router.ParamRes`. If either the response writer or the response is nil, this
is a nop. Uses `res.Header`, `res.StatusCode`, and `res.Body`, ignoring all
other fields of the response. The returned error, if any, always comes from
copying the body via `io.Copy`. It may be safe to ignore this error; it should
occur mostly due to client disconnect.
*/
func Respond(rew http.ResponseWriter, res *http.Response) error {
	if rew == nil || res == nil {
		return nil
	}

	head := rew.Header()
	for key, vals := range res.Header {
		head[key] = vals
	}

	status := res.StatusCode
	if status != 0 && status != http.StatusOK {
		rew.WriteHeader(status)
	}

	body := res.Body
	if body == nil {
		return nil
	}
	defer body.Close()

	_, err := io.Copy(rew, body)
	return err
}

/*
Makes a router for the given request-response. Usage:

	err := MakeRouter(rew, req).Route(myRoutes)
*/
func MakeRouter(rew http.ResponseWriter, req *http.Request) Router {
	return Router{Rew: rew, Req: req}
}

/*
Shortcut for top-level error handling. If the error is nil, do nothing. If the
error is non-nil, write its message as plain text. HTTP status code is obtained
via `rout.ErrStatus`.

Example:

	rout.WriteErr(rew, rout.MakeRouter(rew, req).Route(myRoutes))
*/
func WriteErr(rew http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	rew.WriteHeader(ErrStatus(err))
	_, _ = io.WriteString(rew, err.Error())
}

/*
Returns the underlying HTTP status code of the given error, relying on the
following hidden interface which is implemented by `rout.Err`. The interface
may be implemented by deeply-wrapped errors; this performs deep unwrapping.

	interface { HttpStatusCode() int }

If the error is nil or doesn't implement this interface, status is 500.
*/
func ErrStatus(err error) int {
	code := errStatusDeep(err)
	if code == 0 {
		return http.StatusInternalServerError
	}
	return code
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
	style   style
	lax     bool
}

/*
Takes a regexp pattern and returns a router that will use this pattern to match
`req.URL.Path`.
*/
func (self Router) Regex(val string) Router {
	return self.pat(val, styleRegex)
}

/*
Takes a string and returns a router that tests `req.URL.Path` by matching this
string exactly. Unlike `Router.Regex`, this doesn't support capture groups;
parametrized handlers will always receive empty `[]string{}`.
*/
func (self Router) Exact(val string) Router {
	return self.pat(val, styleExact)
}

/*
Takes a string and returns a router that tests `req.URL.Path` by requiring that
it has the provided prefix. Because request path always begins with `/`, the
prefix must also begin with `/`, but it doesn't have to end with `/`. Unlike
`Router.Regex`, this doesn't support capture groups; parametrized handlers will
always receive empty `[]string{}`.
*/
func (self Router) Begin(val string) Router {
	return self.pat(val, styleBegin)
}

/*
Returns a router that matches only the provided method. If the method is empty,
the resulting router matches all methods, which is the default.

Note: to match multiple methods for one route, use `Router.Methods`. Otherwise,
the first mismatch generates `Err{Status: http.StatusMethodNotAllowed}`.
*/
func (self Router) Method(val string) Router {
	self.method = val
	return self
}

/*
Returns a router set to lax/permissive mode.

In strict mode (default), whenever the router matches the URL pattern but
doesn't match the HTTP method, it immediately generates a "method not allowed"
error. `Router.Regex` automatically switches the router into strict mode.

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
	panic(NotFound(self.req()))
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

	panic(MethodNotAllowed(self.req()))
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
func (self Router) Func(fun Func) {
	if !self.test() {
		return
	}
	// Inline for shorter stacktraces.
	if fun != nil {
		fun(self.Rew, self.Req)
	}
	panic(nil)
}

/*
If the router matches the request, use the provided handler func to respond. If
the router doesn't match the request, do nothing. The func may be nil. The
additional `[]string` argument contains regexp captures from the pattern passed
to `Router.Regex`, if any.
*/
func (self Router) ParamFunc(fun ParamFunc) {
	match := self.match()
	if match == nil {
		return
	}
	if fun != nil {
		fun(self.Rew, self.Req, match)
	}
	panic(nil)
}

/*
If the router matches the request, respond by using the first non-nil handler
returned by one of the provided funcs. If the router doesn't match the request,
do nothing.
*/
func (self Router) Han(funs ...Han) {
	if !self.test() {
		return
	}

	// Inline for shorter stacktraces.
	for _, fun := range funs {
		if fun != nil {
			val := fun(self.Req)
			if val != nil {
				val.ServeHTTP(self.Rew, self.Req)
				panic(nil)
			}
		}
	}

	panic(nil)
}

/*
If the router matches the request, respond by using the first non-nil handler
returned by one of the provided funcs. If the router doesn't match the request,
do nothing. The additional `[]string` argument contains regexp captures from
the pattern passed to `Router.Regex`, if any.
*/
func (self Router) ParamHan(funs ...ParamHan) {
	match := self.match()
	if match == nil {
		return
	}

	// Inline for shorter stacktraces.
	for _, fun := range funs {
		if fun != nil {
			val := fun(self.Req, match)
			if val != nil {
				val.ServeHTTP(self.Rew, self.Req)
				panic(nil)
			}
		}
	}

	panic(nil)
}

/*
If the router matches the request, use `Respond` to write the first non-nil
response returned by one of the provided funcs. If the router doesn't match the
request, do nothing.
*/
func (self Router) Res(funs ...Res) {
	if !self.test() {
		return
	}

	// Inline for shorter stacktraces.
	for _, fun := range funs {
		if fun != nil {
			res := fun(self.Req)
			if res != nil {
				panic(Respond(self.Rew, res))
			}
		}
	}

	panic(nil)
}

/*
If the router matches the request, use the provided responder func to generate a
response, and use `Respond` to write it. If the router doesn't match the
request, do nothing. The func may be nil. The additional `[]string` argument
contains regexp captures from the pattern passed to `Router.Regex`, if any.
*/
func (self Router) ParamRes(fun ParamRes) {
	match := self.match()
	if match == nil {
		return
	}
	if fun != nil {
		panic(Respond(self.Rew, fun(self.Req, match)))
	}
	panic(nil)
}

func (self Router) pat(pattern string, style style) Router {
	self.pattern = pattern
	self.style = style
	self.lax = false
	return self
}

func (self Router) req() (string, string) {
	req := self.Req
	if req != nil {
		return req.Method, req.URL.Path
	}
	return ``, ``
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
	panic(MethodNotAllowed(self.req()))
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
	panic(MethodNotAllowed(self.req()))
}

func (self Router) testMethod() bool {
	req, method := self.Req, self.method
	return req != nil && (method == `` || method == req.Method)
}

func (self Router) testPattern() bool {
	req := self.Req
	if req == nil {
		return false
	}

	path, pattern, style := req.URL.Path, self.pattern, self.style

	switch style {
	case styleRegex:
		return testRegex(path, pattern)
	case styleExact:
		return testExact(path, pattern)
	case styleBegin:
		return testBegin(path, pattern)
	default:
		return false
	}
}

func (self Router) matchPattern() []string {
	req := self.Req
	if req == nil {
		return nil
	}

	path, pattern, style := req.URL.Path, self.pattern, self.style

	switch style {
	case styleRegex:
		return matchRegex(path, pattern)
	case styleExact:
		return matchExact(path, pattern)
	case styleBegin:
		return matchBegin(path, pattern)
	default:
		return nil
	}
}

/*
HTTP handler type that behaves similarly to `Router.Han` or `Router.ParamHan`.
Stores multiple `Han` functions, and when serving HTTP, uses the first non-nil
`http.Handler` returned by one of those functions.
*/
type Coalesce []Han

// Implement `http.Handler`.
func (self Coalesce) ServeHTTP(rew http.ResponseWriter, req *http.Request) {
	val := self.Han(req)
	if val != nil {
		val.ServeHTTP(rew, req)
	}
}

// Invokes the funcs in order, returning the first resulting non-nil handler.
func (self Coalesce) Han(req *http.Request) http.Handler {
	for _, fun := range self {
		if fun != nil {
			val := fun(req)
			if val != nil {
				return val
			}
		}
	}
	return nil
}

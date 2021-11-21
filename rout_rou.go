package rout

import (
	"net/http"
)

/*
Makes a router for the given request-response. Usage:

	ro.MakeRou(rew, req).Serve(myRoutes)

	ro.WriteErr(rew, ro.MakeRou(rew, req).Route(myRoutes))
*/
func MakeRou(rew http.ResponseWriter, req *http.Request) Rou {
	return Rou{Rew: rew, Req: req}
}

/*
Router type. Matches patterns and executes handlers. Should be used via
`Rou.Serve` or `Rou.Route`, which handles panics inherent to the routing flow.
Immutable, with a builder-style API where every method returns a modified copy.
A router is stack-allocated; its builder API incurs no allocator/GC work.

Implementation note. All "modifying" methods are defined on the value type in
order to return modified copies, but many non-modifying methods are defined on
the pointer type for marginal efficiency gains, due to the size of this
struct.
*/
type Rou struct {
	Rew http.ResponseWriter
	Req *http.Request

	Vis        Visitor
	Method     string
	Pattern    string
	Style      Match
	OnlyMethod bool
}

/*
Shortcut for routing with default error handling. Same as `rout.Rou.Route`,
but instead of returning an error, uses `rout.WriteErr` to write it. Example:

	rout.MakeRou(rew, req).Serve(myRoutes)
*/
func (self Rou) Serve(fun func(Rou)) {
	WriteErr(self.Rew, self.Route(fun))
}

/*
Routes the given request-response, recovering from panics inherent to the
routing flow of this package. The resulting error is usually of type `Err`,
containing an appropriate HTTP status code. Your code must handle the error,
sending back an appropriate response. If routing was performed successfully,
the error is nil.

Same as `Rou.Sub`, but catches panics, returning them as errors.
*/
func (self Rou) Route(fun func(Rou)) (err error) {
	defer rec(&err)
	self.Sub(fun)
	return
}

/*
Short for "regexp". Takes a regexp pattern and returns a router that will use
this pattern to match `req.URL.Path`. Regexps are compiled lazily, cached, and
reused.
*/
func (self Rou) Reg(val string) Rou {
	return self.pat(val, MatchReg)
}

/*
Short for "pattern". Takes a "path template" compatible with OpenAPI and returns
a router that will use this pattern to match `req.URL.Path`, via `Pat`.
Patterns are compiled lazily, cached, and reused.
*/
func (self Rou) Pat(val string) Rou {
	return self.pat(val, MatchPat)
}

/*
Short for "exact". Takes a string and returns a router that tests `req.URL.Path`
by matching this string exactly. Unlike `Rou.Reg`, this doesn't support capture
groups; parametrized handlers will always receive empty `[]string{}`.
*/
func (self Rou) Exa(val string) Rou {
	return self.pat(val, MatchExa)
}

/*
Short for "start" or "starts with". Takes a string and returns a router that
tests `req.URL.Path` by requiring that it has the given prefix. Because
"net/http" ensures that request path begins with `/`, the prefix should also
begin with `/`, but it doesn't need to end with `/`. Unlike `Rou.Reg`, this
doesn't support capture groups; parametrized handlers will always receive
empty `[]string{}`.
*/
func (self Rou) Sta(val string) Rou {
	return self.pat(val, MatchSta)
}

/*
Short for "method". Returns a router that matches only the given method. If the
method is empty, the resulting router matches all methods, which is the
default. Note: to match multiple methods for one route, use `Rou.Methods`.
Otherwise, the first mismatch generates `ErrMethodNotAllowed`.
*/
func (self Rou) Meth(val string) Rou {
	self.Method = val
	return self
}

/*
Returns a router set to "method only" mode.

In "normal" mode (default), whenever the router matches the URL pattern but
doesn't match the HTTP method, it immediately generates a "method not allowed"
error. All pattern-modifying router methods, such as `Rou.Exa`, automatically
switch the router into "normal" mode.

In "method only" mode (opt-in), the router tests ONLY the HTTP method. The URL
pattern is considered an automatic match. Additionally, a mismatch doesn't
generate an error. This is used by `Rou.Methods`, which automatically switches
the router into "method only" mode.
*/
func (self Rou) MethodOnly() Rou {
	self.OnlyMethod = true
	return self
}

/*
Same as `.Meth(http.MethodGet)`.
Returns a router that matches only this HTTP method.
*/
func (self Rou) Get() Rou { return self.Meth(http.MethodGet) }

/*
Same as `.Meth(http.MethodHead)`.
Returns a router that matches only this HTTP method.
*/
func (self Rou) Head() Rou { return self.Meth(http.MethodHead) }

/*
Same as `.Meth(http.MethodOptions)`.
Returns a router that matches only this HTTP method.
*/
func (self Rou) Options() Rou { return self.Meth(http.MethodOptions) }

/*
Same as `.Meth(http.MethodPost)`.
Returns a router that matches only this HTTP method.
*/
func (self Rou) Post() Rou { return self.Meth(http.MethodPost) }

/*
Same as `.Meth(http.MethodPatch)`.
Returns a router that matches only this HTTP method.
*/
func (self Rou) Patch() Rou { return self.Meth(http.MethodPatch) }

/*
Same as `.Meth(http.MethodPut)`.
Returns a router that matches only this HTTP method.
*/
func (self Rou) Put() Rou { return self.Meth(http.MethodPut) }

/*
Same as `.Meth(http.MethodDelete)`.
Returns a router that matches only this HTTP method.
*/
func (self Rou) Delete() Rou { return self.Meth(http.MethodDelete) }

/*
Same as `.Meth(http.MethodConnect)`.
Returns a router that matches only this HTTP method.
*/
func (self Rou) Connect() Rou { return self.Meth(http.MethodConnect) }

/*
Same as `.Meth(http.MethodTrace)`.
Returns a router that matches only this HTTP method.
*/
func (self Rou) Trace() Rou { return self.Meth(http.MethodTrace) }

/*
If the router matches the request, perform sub-routing. If sub-routing doesn't
find a match, panic with `ErrNotFound`. If the router doesn't match the
request, do nothing.
*/
func (self Rou) Sub(fun func(Rou)) {
	if self.real() && !self.Match() {
		return
	}
	if fun != nil {
		fun(self)
	}
	if self.real() {
		panic(NotFound(self.req()))
	}
}

/*
If the router matches the request, perform sub-routing. The router provided to
the function is set to "method only" mode: a mismatch in the HTTP method
doesn't immediately generate an error. However, if sub-routing doesn't find a
match, this panics with `ErrMethodNotAllowed`. If the router doesn't match the
request, do nothing.
*/
func (self Rou) Methods(fun func(Rou)) {
	if self.real() && !self.matchPattern() {
		return
	}
	if fun != nil {
		fun(self.MethodOnly())
	}
	if self.real() {
		panic(MethodNotAllowed(self.req()))
	}
}

/*
If the router matches the request, use the given handler to respond. If the
router doesn't match the request, do nothing. The handler may be nil. In
"dry run" mode via `Visit`, this invokes a visitor for the current endpoint.
*/
func (self Rou) Handler(val http.Handler) {
	if self.vis(val) || !self.Match() {
		return
	}
	if val != nil {
		val.ServeHTTP(self.Rew, self.Req)
	}
	panic(nil)
}

/*
If the router matches the request, use the given handler func to respond.
If the router doesn't match the request, do nothing. The func may be nil. In
"dry run" mode via `Visit`, this invokes a visitor for the current endpoint.
*/
func (self Rou) Func(fun Func) {
	if self.vis(fun) || !self.Match() {
		return
	}
	if fun != nil {
		fun(self.Rew, self.Req)
	}
	panic(nil)
}

/*
If the router matches the request, use the given handler func to respond. If the
router doesn't match the request, do nothing. The func may be nil. The
additional `[]string` argument contains regexp captures from the pattern passed
to `Rou.Reg`, if any. In "dry run" mode via `Visit`, this invokes a visitor for
the current endpoint.
*/
func (self Rou) ParamFunc(fun ParamFunc) {
	if self.vis(fun) {
		return
	}
	args := self.Submatch()
	if args == nil {
		return
	}
	if fun != nil {
		fun(self.Rew, self.Req, args)
	}
	panic(nil)
}

/*
If the router matches the request, respond by using the handler returned by the
given function. If the router doesn't match the request, do nothing. In "dry
run" mode via `Visit`, this invokes a visitor for the current endpoint.
*/
func (self Rou) Han(fun Han) {
	if self.vis(fun) || !self.Match() {
		return
	}

	if fun != nil {
		val := fun(self.Req)
		if val != nil {
			val.ServeHTTP(self.Rew, self.Req)
		}
	}

	panic(nil)
}

/*
If the router matches the request, respond by using the handler returned by the
given function. If the router doesn't match the request, do nothing. The
additional `[]string` argument contains regexp captures from the pattern passed
to `Rou.Reg`, if any. In "dry run" mode via `Visit`, this invokes a visitor for
the current endpoint.
*/
func (self Rou) ParamHan(fun ParamHan) {
	if self.vis(fun) {
		return
	}
	args := self.Submatch()
	if args == nil {
		return
	}

	if fun != nil {
		val := fun(self.Req, args)
		if val != nil {
			val.ServeHTTP(self.Rew, self.Req)
		}
	}

	panic(nil)
}

/*
If the router matches the request, use `Respond` to write the response returned
by the given function. If the router doesn't match the request, do nothing.
In "dry run" mode via `Visit`, this invokes a visitor for the current endpoint.
*/
func (self Rou) Res(fun Res) {
	if self.vis(fun) || !self.Match() {
		return
	}
	if fun != nil {
		panic(Respond(self.Rew, fun(self.Req)))
	}
	panic(nil)
}

/*
If the router matches the request, use the given responder func to generate a
response, and use `Respond` to write it. If the router doesn't match the
request, do nothing. The func may be nil. The additional `[]string` argument
contains regexp captures from the pattern passed to `Rou.Reg`, if any. In "dry
run" mode via `Visit`, this invokes a visitor for the current endpoint.
*/
func (self Rou) ParamRes(fun ParamRes) {
	if self.vis(fun) {
		return
	}
	args := self.Submatch()
	if args == nil {
		return
	}
	if fun != nil {
		panic(Respond(self.Rew, fun(self.Req, args)))
	}
	panic(nil)
}

/*
Mostly for internal use. True if the router matches the request. If
`.OnlyMethod` is true, matches only the request's method. Otherwise matches
both the pattern and the method. If the pattern matches but the method doesn't,
panics with `ErrMethodNotAllowed`; the panic is normally caught and returned
via `Rou.Route`.
*/
func (self *Rou) Match() bool {
	if self.OnlyMethod {
		return self.matchMethod()
	}
	return self.matchStrict()
}

/*
Mostly for internal use. Like `Rou.Match`, but instead of a boolean, returns a
slice with captured args. If there's no match, the slice is nil. Otherwise, the
slice is non-nil, and its length equals the amount of capture groups in the
current pattern. If the pattern matches but the method doesn't, panics with
`ErrMethodNotAllowed`; the panic is normally caught and returned via
`Rou.Route`.
*/
func (self *Rou) Submatch() []string {
	if self.OnlyMethod {
		return self.submatchOnlyMethod()
	}
	return self.submatchStrict()
}

func (self *Rou) matchMethod() bool {
	return self.Method == `` || self.Method == self.meth()
}

func (self *Rou) matchPattern() bool {
	return self.Style.Match(self.Pattern, self.path())
}

func (self *Rou) submatchPattern() []string {
	return self.Style.Submatch(self.Pattern, self.path())
}

func (self Rou) pat(pattern string, style Match) Rou {
	self.Pattern = pattern
	self.Style = style
	self.OnlyMethod = false
	return self
}

func (self *Rou) req() (string, string) {
	return self.meth(), self.path()
}

func (self *Rou) meth() string {
	req := self.Req
	if req != nil {
		return req.Method
	}
	return ``
}

func (self *Rou) path() string {
	req := self.Req
	if req != nil && req.URL != nil {
		return req.URL.Path
	}
	return ``
}

func (self *Rou) real() bool { return self.Vis == nil }

func (self *Rou) vis(val interface{}) bool {
	vis := self.Vis
	if vis != nil {
		vis.Endpoint(self.endpoint(val))
		return true
	}
	return false
}

func (self *Rou) endpoint(val interface{}) Endpoint {
	return Endpoint{self.Pattern, self.Style, self.Method, Ident(val)}
}

func (self *Rou) matchStrict() bool {
	if !self.matchPattern() {
		return false
	}
	if self.matchMethod() {
		return true
	}
	panic(MethodNotAllowed(self.req()))
}

func (self Rou) submatchOnlyMethod() []string {
	if self.matchMethod() {
		return []string{}
	}
	return nil
}

func (self *Rou) submatchStrict() []string {
	args := self.submatchPattern()
	if args == nil {
		return nil
	}
	if self.matchMethod() {
		return args
	}
	panic(MethodNotAllowed(self.req()))
}

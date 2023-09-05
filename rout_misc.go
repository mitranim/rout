package rout

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	r "reflect"
	u "unsafe"
)

// Shortcut for brevity.
type R = Rou

/*
Short for "routing function". Type of functions passed to the various routing
methods such as `Rou.Route`. Also implements `http.Handler`, as a shortcut for
using `MakeRou` and `Serve`, passing itself as the routing func.
*/
type RouFunc func(Rou)

// Implement `http.Handler`.
func (self RouFunc) ServeHTTP(rew http.ResponseWriter, req *http.Request) {
	if self != nil {
		MakeRou(rew, req).Serve(self)
	}
}

/*
Type of functions passed to `Rou.Func`. Non-parametrized handler func. Same
signature as `http.HandlerFunc`, but this is an anonymous type, not a typedef.
Doesn't implement `http.Handler`.
*/
type Func = func(http.ResponseWriter, *http.Request)

/*
Type of functions passed to `Rou.ParamFunc`. Parametrized handler func. Takes
additional args produced by capture groups, which are supported by `Rou.Reg`
and `Rou.Pat`. Args start at index 0, not 1 like in a regexp match.
*/
type ParamFunc = func(http.ResponseWriter, *http.Request, []string)

/*
Type of functions passed to `Rou.Han`. Short for "handler" or "handlerer". The
returned `http.Handler` is used to write the response. To represent responses
with handlers, use "github.com/mitranim/goh".
*/
type Han = func(*http.Request) http.Handler

/*
Type of functions passed to `Rou.ParamHan`. Short for "parametrized
handler/handlerer".
*/
type ParamHan = func(*http.Request, []string) http.Handler

/*
Type of functions passed to `Rou.Res`. Short for "responder". The returned
`*http.Response` is sent back via the function `Respond`.
*/
type Res = func(*http.Request) *http.Response

/*
Type of functions passed to `Rou.ParamRes`. Short for "parametrized responder".
*/
type ParamRes = func(*http.Request, []string) *http.Response

/*
Writes the given response. Used internally by `Rou.Res` and `Rou.ParamRes`. If
either the response writer or the response is nil, this is a nop. Uses
`res.Header`, `res.StatusCode`, and `res.Body`, ignoring all other fields of
the response. The returned error, if any, always comes from copying the body
via `io.Copy`, and should occur mostly due to premature client disconnect.
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
Shortcut for top-level error handling. If the error is nil, do nothing. If the
error is non-nil, write its message as plain text. HTTP status code is obtained
via `rout.ErrStatusFallback`.

Example:

	rout.WriteErr(rew, rout.MakeRou(rew, req).Route(myRoutes))
*/
func WriteErr(rew http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	rew.WriteHeader(ErrStatusFallback(err))
	_, _ = io.WriteString(rew, err.Error())
}

/*
Returns the underlying HTTP status code of the given error, relying on the
following hidden interface which is implemented by `rout.Err`. The interface
may be implemented by deeply-wrapped errors; this performs deep unwrapping.

	interface { HttpStatusCode() int }

If the error is nil or doesn't implement this interface, status is 0.
If you always want a non-zero code, use `ErrStatusFallback` which falls
back on 500.
*/
func ErrStatus(err error) int {
	code := errStatusDeep(err)
	if code != 0 {
		return code
	}
	return 0
}

/*
Convenience wrapper for `ErrStatus` that falls back on status 500 when the error
doesn't seem to contain an HTTP status, always returning a non-zero result.
*/
func ErrStatusFallback(err error) int {
	out := ErrStatus(err)
	if out == 0 {
		return http.StatusInternalServerError
	}
	return out
}

/*
HTTP handler type that stores multiple `Han` functions, and when serving HTTP,
uses the first non-nil `http.Handler` returned by one of those functions.
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

/*
Various types of pattern matching supported by this package: exact,
start/prefix, regexp, OAS-style pattern. See the comments on the constants such
as `MatchExa`.
*/
type Match byte

const (
	/**
	Short for "exact". Used by `Rou.Exa`. Compares pattern and input via `==`.
	Doesn't support capture groups; `.Submatch` returns `[]string{}` on a match.
	As a special rule, the empty pattern `` matches any input.
	*/
	MatchExa Match = iota

	/**
	Short for "start", or "starts with", or "prefix". Used by `Rou.Sta`. When
	matching, requires that the input path has the given pattern as its prefix.
	Because "net/http" ensures that request paths begin with `/`, the prefix
	should also begin with `/`, but it doesn't need to end with `/`; this
	package takes care of that. Doesn't support capture groups; `.Submatch`
	returns `[]string{}` on a match. As a special rule, the empty pattern ``
	matches any input.
	*/
	MatchSta

	/**
	Short for "regexp". Used by `Rou.Reg`. Performs matching or submatching by
	converting its pattern to `*regexp.Regexp`. Compiles each pattern only once,
	with caching and reuse. Does support capture groups. The empty pattern ``
	matches any input.
	*/
	MatchReg

	/**
	Short for "pattern", specifically path pattern compatible with OpenAPI specs.
	Used by `Rou.Pat`. Performs matching or submatching by converting its
	pattern to `Pat`, which is also exported by this package. Compiles each
	pattern only once, with caching and reuse. Does support capture groups. The
	empty pattern `` matches any input.
	*/
	MatchPat
)

// Implement `fmt.Stringer` for debug purposes.
func (self Match) String() string {
	switch self {
	case MatchExa:
		return `exa`
	case MatchSta:
		return `sta`
	case MatchReg:
		return `reg`
	case MatchPat:
		return `pat`
	default:
		return ``
	}
}

/*
True if the pattern matches the input. See the comments on the various `Match`
constants.
*/
func (self Match) Match(pat, inp string) bool {
	if pat == `` {
		return true
	}

	switch self {
	case MatchExa:
		return matchExa(pat, inp)
	case MatchSta:
		return matchSta(pat, inp)
	case MatchReg:
		return matchReg(pat, inp)
	case MatchPat:
		return matchPat(pat, inp)
	default:
		return false
	}
}

/*
If the pattern matches the input, returns a non-nil slice of captures. Otherwise
returns nil. See the comments on the various `Match` constants. Regardless of
the match implementation, captures start at index 0, not at index 1 like in
regexps.
*/
func (self Match) Submatch(pat, inp string) []string {
	if pat == `` {
		return []string{}
	}

	switch self {
	case MatchExa:
		return submatchExa(pat, inp)
	case MatchSta:
		return submatchSta(pat, inp)
	case MatchReg:
		return submatchReg(pat, inp)
	case MatchPat:
		return submatchPat(pat, inp)
	default:
		return nil
	}
}

/*
Tool for introspection. Returns the "identity" of the input: the internal
representation of the interface value that was passed in. When performing
a "dry run" via `Visit`, this function generates the identity of route
handlers. Advanced users of this package may build a registry that maps handler
identities to arbitrary metadata, and retrieve that information from visited
routes, using idents as keys.
*/
func Ident(val interface{}) [2]uintptr {
	return *(*[2]uintptr)(u.Pointer(&val))
}

/*
Tool for introspection. Returns the original `reflect.Type` of an `Ident`. If
the input is zero, the returned type is nil.
*/
func IdentType(val [2]uintptr) r.Type {
	val[1] = 0
	return r.TypeOf(*(*interface{})(u.Pointer(&val)))
}

/*
Tool for introspection. Passed to `Visitor` when performing a "dry run" via the
`Visit` function.
*/
type Endpoint struct {
	Pattern string
	Match   Match
	Method  string
	Handler [2]uintptr
}

/*
Tool for introspection. Performs a "dry run" of the given routing function,
visiting all routes without executing any handlers. During the dry run, the
`http.ResponseWriter` contained in the router is a special nop type that
discards all writes.
*/
func Visit(fun func(Rou), vis Visitor) {
	rou := MakeRou(NopRew{}, &http.Request{URL: new(url.URL)})
	rou.Vis = vis
	rou.Sub(fun)
}

/*
Tool for introspection. Used for performing a "dry run" that visits all routes
without executing the handlers. See `Visit`.
*/
type Visitor interface{ Endpoint(Endpoint) }

// Shortcut type. Implements `Visitor` by calling itself.
type VisitorFunc func(Endpoint)

// Implement `Visitor` by calling itself.
func (self VisitorFunc) Endpoint(val Endpoint) {
	if self != nil {
		self(val)
	}
}

/*
Tool for introspection. Simplified version of `Visitor` that doesn't "know"
about the multiple pattern types supported by this package. Must be wrapped by
adapters such as `RegexpVisitor` and `PatternVisitor`. WTB better name.
*/
type SimpleVisitor interface {
	Endpoint(pattern, method string, ident [2]uintptr)
}

// Shortcut type. Implements `SimpleVisitor` by calling itself.
type SimpleVisitorFunc func(pattern, method string, ident [2]uintptr)

// Implement `SimpleVisitor` by calling itself.
func (self SimpleVisitorFunc) Endpoint(pattern, method string, ident [2]uintptr) {
	if self != nil {
		self(pattern, method, ident)
	}
}

/*
Tool for introspection. Adapter between `Visitor` and `SimpleVisitor`. Converts
route patterns to regexp patterns, passing those to the inner visitor.
*/
type RegexpVisitor [1]SimpleVisitor

// Implement `Visitor`.
func (self RegexpVisitor) Endpoint(val Endpoint) {
	if self[0] == nil {
		return
	}

	switch val.Match {
	case MatchExa:
		self[0].Endpoint(exaToReg(val.Pattern), val.Method, val.Handler)

	case MatchSta:
		self[0].Endpoint(staToReg(val.Pattern), val.Method, val.Handler)

	case MatchReg:
		self[0].Endpoint(val.Pattern, val.Method, val.Handler)

	case MatchPat:
		self[0].Endpoint(patToReg(val.Pattern), val.Method, val.Handler)

	default:
		panic(fmt.Errorf(
			`[rout] unable to convert match %q for route %q %q to regex`,
			val.Match, val.Pattern, val.Method,
		))
	}
}

/*
Tool for introspection. Adapter between `Visitor` and `SimpleVisitor`. Converts
route patterns to OAS-style patterns compatible with `Pat`, passing those to
the inner visitor.
*/
type PatternVisitor [1]SimpleVisitor

// Implement `Visitor`.
func (self PatternVisitor) Endpoint(val Endpoint) {
	if self[0] == nil {
		return
	}

	switch val.Match {
	case MatchExa:
		self[0].Endpoint(exactToPat(val.Pattern), val.Method, val.Handler)

	case MatchPat:
		self[0].Endpoint(val.Pattern, val.Method, val.Handler)

	default:
		panic(fmt.Errorf(
			`[rout] unable to convert match %q for route %q %q to OAS pattern`,
			val.Match, val.Pattern, val.Method,
		))
	}
}

/*
Nop implementation of `http.ResponseWriter` used internally by `Visit`.
Exported for implementing custom variants of `Visit`.
*/
type NopRew struct{}

var _ = http.ResponseWriter(NopRew{})

func (NopRew) Header() http.Header           { return http.Header{} }
func (NopRew) WriteHeader(int)               {}
func (NopRew) Write(val []byte) (int, error) { return len(val), nil }

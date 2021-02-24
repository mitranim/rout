/*
Experimental router for Go HTTP servers. Imperative control flow with
declarative syntax. Doesn't need middleware.

Very simple, small (≈300 LoC without docs), dependency-free, reasonably fast.

See `Route` for an example. See `readme.md` for additional info such as
motivation and advantages.
*/
package rout

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
)

// For brevity.
type (
	R   = Router
	MR  = MethodRouter
	PR  = ParamRouter
	PMR = ParamMethodRouter
)

type (
	// Non-parametrized handler func. Same as `http.HandlerFunc`.
	Func = func(http.ResponseWriter, *http.Request)

	// Parametrized handler func. Takes additional args, produced by parenthesized
	// regexp capture groups. Args start at index 0, not 1 like in a regexp match.
	ParamFunc = func(http.ResponseWriter, *http.Request, []string)
)

/*
Routes the given request-response, recovering from panics inherent to the
routing flow. The resulting error is usually of type `Err`, containing an
appropriate HTTP status code. Your code must handle the error, sending back an
appropriate response. If routing was performed successfully, the error is nil.
*/
func Route(rew http.ResponseWriter, req *http.Request, fun func(Router)) (err error) {
	defer rec(&err)
	fun(Router{rew, req})
	return errNotFound(req.URL.Path, req.Method)
}

/*
Main router type. Should be used via `Route`, which takes care of panics
inherent to the routing flow.
*/
type Router struct {
	Rew http.ResponseWriter
	Req *http.Request
}

// Cost-free conversion to a parametrized router.
func (self Router) Param() ParamRouter { return ParamRouter(self) }

// Same as `.Method(http.MethodGet, pat, fun)`.
func (self Router) Get(pat string, fun Func) { self.Method(http.MethodGet, pat, fun) }

// Same as `.Method(http.MethodHead, pat, fun)`.
func (self Router) Head(pat string, fun Func) { self.Method(http.MethodHead, pat, fun) }

// Same as `.Method(http.MethodOptions, pat, fun)`.
func (self Router) Options(pat string, fun Func) { self.Method(http.MethodOptions, pat, fun) }

// Same as `.Method(http.MethodPost, pat, fun)`.
func (self Router) Post(pat string, fun Func) { self.Method(http.MethodPost, pat, fun) }

// Same as `.Method(http.MethodPatch, pat, fun)`.
func (self Router) Patch(pat string, fun Func) { self.Method(http.MethodPatch, pat, fun) }

// Same as `.Method(http.MethodPut, pat, fun)`.
func (self Router) Put(pat string, fun Func) { self.Method(http.MethodPut, pat, fun) }

// Same as `.Method(http.MethodDelete, pat, fun)`.
func (self Router) Delete(pat string, fun Func) { self.Method(http.MethodDelete, pat, fun) }

/*
Matches on both the URL pattern and the HTTP method. If the pattern matches but
the method doesn't, panics with an instance of `Error` where
`.HttpStatus == http.StatusMethodNotAllowed`.

Note: `.Method` can NOT be repeated for one pattern, because the first mismatch
generates an error. To allow multiple HTTP methods for one pattern, use
`.Methods`.
*/
func (self Router) Method(method string, pattern string, fun Func) {
	if self.matchPattern(pattern) {
		if !self.matchMethod(method) {
			panic(errMethodNotAllowed(self.Req.URL.Path, self.Req.Method))
		}
		if fun != nil {
			fun(self.Rew, self.Req)
		}
		panic(nil)
	}
}

/*
Matches only on the URL pattern. Allows any HTTP method.
*/
func (self Router) Any(pattern string, fun Func) {
	if self.matchPattern(pattern) {
		if fun != nil {
			fun(self.Rew, self.Req)
		}
		panic(nil)
	}
}

/*
Matches on the URL pattern and performs sub-routing. If the sub-route doesn't
find a match, panics with an instance of `Error` where
`.HttpStatus == http.StatusNotFound`.
*/
func (self Router) Sub(pattern string, fun func(Router)) {
	if self.matchPattern(pattern) {
		if fun != nil {
			fun(self)
		}
		panic(errNotFound(self.Req.URL.Path, self.Req.Method))
	}
}

/*
Matches on the URL pattern and performs sub-routing ONLY for HTTP methods on
this path. If the sub-route doesn't find a match, panics with an instance of
`Error` where `.HttpStatus == http.StatusMethodNotAllowed`.
*/
func (self Router) Methods(pattern string, fun func(MethodRouter)) {
	if self.matchPattern(pattern) {
		if fun != nil {
			fun(MethodRouter(self))
		}
		panic(errMethodNotAllowed(self.Req.URL.Path, self.Req.Method))
	}
}

/*
Short for "recover". Recovers from a panic, and calls the provided function with
the resulting error or nil.

ALWAYS re-panics with nil to preserve the `rout` control flow. Must be used ONLY
inside routing functions:

	func routes(r rout.R) {
		defer r.Rec(writeErr)
		r.Get(`...`, someFunc)
	}

	func writeErr(rew http.ResponseWriter, req *http.Request, err error) {}

Reminiscent of middleware, which `rout` prides itself on not having. This
approach should be revised.
*/
func (self Router) Rec(fun func(http.ResponseWriter, *http.Request, error)) {
	err := toErr(recover())
	fun(self.Rew, self.Req, err)
	panic(nil)
}

func (self Router) matchPattern(pattern string) bool {
	req := self.Req
	return req != nil && cachedRegexp(pattern).MatchString(req.URL.Path)
}

func (self Router) matchMethod(method string) bool {
	req := self.Req
	return req != nil && req.Method == method
}

/*
Variant of `Router` where route handlers take captured args as `[]string`.
Args are produced by parenthesized regexp capture groups.
*/
type ParamRouter Router

// Same as `.Method(http.MethodGet, pat, fun)`.
func (self ParamRouter) Get(pat string, fun ParamFunc) { self.Method(http.MethodGet, pat, fun) }

// Same as `.Method(http.MethodHead, pat, fun)`.
func (self ParamRouter) Head(pat string, fun ParamFunc) { self.Method(http.MethodHead, pat, fun) }

// Same as `.Method(http.MethodOptions, pat, fun)`.
func (self ParamRouter) Options(pat string, fun ParamFunc) { self.Method(http.MethodOptions, pat, fun) }

// Same as `.Method(http.MethodPost, pat, fun)`.
func (self ParamRouter) Post(pat string, fun ParamFunc) { self.Method(http.MethodPost, pat, fun) }

// Same as `.Method(http.MethodPatch, pat, fun)`.
func (self ParamRouter) Patch(pat string, fun ParamFunc) { self.Method(http.MethodPatch, pat, fun) }

// Same as `.Method(http.MethodPut, pat, fun)`.
func (self ParamRouter) Put(pat string, fun ParamFunc) { self.Method(http.MethodPut, pat, fun) }

// Same as `.Method(http.MethodDelete, pat, fun)`.
func (self ParamRouter) Delete(pat string, fun ParamFunc) { self.Method(http.MethodDelete, pat, fun) }

// Same as `Router.Method`, but the handler is parametrized and takes args
// produced by parenthesized capture groups in the regexp.
func (self ParamRouter) Method(method string, pattern string, fun ParamFunc) {
	args := self.patternMatch(pattern)
	if args != nil {
		if !Router(self).matchMethod(method) {
			panic(errMethodNotAllowed(self.Req.URL.Path, self.Req.Method))
		}
		if fun != nil {
			fun(self.Rew, self.Req, args)
		}
		panic(nil)
	}
}

// Same as `Router.Any`, but the handler is parametrized and takes args produced
// by parenthesized capture groups in the regexp.
func (self ParamRouter) Any(pattern string, fun ParamFunc) {
	args := self.patternMatch(pattern)
	if args != nil {
		if fun != nil {
			fun(self.Rew, self.Req, args)
		}
		panic(nil)
	}
}

// Same as `Router.Sub`, but parametrized.
func (self ParamRouter) Sub(pattern string, fun func(ParamRouter)) {
	if Router(self).matchPattern(pattern) {
		if fun != nil {
			fun(self)
		}
		panic(errNotFound(self.Req.URL.Path, self.Req.Method))
	}
}

// Same as `Router.Methods`, but the sub-router allows parametrized handlers
// that take args produced by parenthesized capture groups in the regexp.
func (self ParamRouter) Methods(pattern string, fun func(ParamMethodRouter)) {
	args := self.patternMatch(pattern)
	if args != nil {
		if fun != nil {
			fun(ParamMethodRouter{self, args})
		}
		panic(errMethodNotAllowed(self.Req.URL.Path, self.Req.Method))
	}
}

func (self ParamRouter) patternMatch(pattern string) []string {
	req := self.Req
	if req != nil {
		match := cachedRegexp(pattern).FindStringSubmatch(req.URL.Path)
		if match != nil {
			return match[1:]
		}
		return nil
	}
	return nil
}

/*
Variant of `Router` that matches ONLY on the HTTP method, ignoring the URL path.
`Router.Methods` passes this to its sub-routing function.
*/
type MethodRouter Router

// Same as `.Method(http.MethodGet, fun)`.
func (self MethodRouter) Get(fun Func) { self.Method(http.MethodGet, fun) }

// Same as `.Method(http.MethodHead, fun)`.
func (self MethodRouter) Head(fun Func) { self.Method(http.MethodHead, fun) }

// Same as `.Method(http.MethodOptions, fun)`.
func (self MethodRouter) Options(fun Func) { self.Method(http.MethodOptions, fun) }

// Same as `.Method(http.MethodPost, fun)`.
func (self MethodRouter) Post(fun Func) { self.Method(http.MethodPost, fun) }

// Same as `.Method(http.MethodPatch, fun)`.
func (self MethodRouter) Patch(fun Func) { self.Method(http.MethodPatch, fun) }

// Same as `.Method(http.MethodPut, fun)`.
func (self MethodRouter) Put(fun Func) { self.Method(http.MethodPut, fun) }

// Same as `.Method(http.MethodDelete, fun)`.
func (self MethodRouter) Delete(fun Func) { self.Method(http.MethodDelete, fun) }

/*
Similar to `Router.Method`, but matches ONLY on the HTTP method, ignoring the
URL path.
*/
func (self MethodRouter) Method(method string, fun Func) {
	if Router(self).matchMethod(method) {
		if fun != nil {
			fun(self.Rew, self.Req)
		}
		panic(nil)
	}
}

/*
Similar to `Router.Any`, but matches ONLY on the HTTP method, ignoring the URL
path.
*/
func (self MethodRouter) Any(fun Func) {
	if fun != nil {
		fun(self.Rew, self.Req)
	}
	panic(nil)
}

/*
Supports parametrized handlers, like `ParamRouter`, and matches ONLY on the HTTP
method, like `MethodRouter`. `ParamRouter.Methods` passes this to its
sub-routing function.
*/
type ParamMethodRouter struct {
	ParamRouter
	Args []string
}

// Same as `.Method(http.MethodGet, fun)`.
func (self ParamMethodRouter) Get(fun ParamFunc) { self.Method(http.MethodGet, fun) }

// Same as `.Method(http.MethodHead, fun)`.
func (self ParamMethodRouter) Head(fun ParamFunc) { self.Method(http.MethodHead, fun) }

// Same as `.Method(http.MethodOptions, fun)`.
func (self ParamMethodRouter) Options(fun ParamFunc) { self.Method(http.MethodOptions, fun) }

// Same as `.Method(http.MethodPost, fun)`.
func (self ParamMethodRouter) Post(fun ParamFunc) { self.Method(http.MethodPost, fun) }

// Same as `.Method(http.MethodPatch, fun)`.
func (self ParamMethodRouter) Patch(fun ParamFunc) { self.Method(http.MethodPatch, fun) }

// Same as `.Method(http.MethodPut, fun)`.
func (self ParamMethodRouter) Put(fun ParamFunc) { self.Method(http.MethodPut, fun) }

// Same as `.Method(http.MethodDelete, fun)`.
func (self ParamMethodRouter) Delete(fun ParamFunc) { self.Method(http.MethodDelete, fun) }

/*
Similar to `ParamRouter.Method`, but matches ONLY on the HTTP method, ignoring
the URL path.
*/
func (self ParamMethodRouter) Method(method string, fun ParamFunc) {
	if Router(self.ParamRouter).matchMethod(method) {
		if fun != nil {
			fun(self.Rew, self.Req, self.Args)
		}
		panic(nil)
	}
}

/*
Similar to `ParamRouter.Method`, but matches ONLY on the HTTP method, ignoring
the URL path.
*/
func (self ParamMethodRouter) Any(fun ParamFunc) {
	if fun != nil {
		fun(self.Rew, self.Req, self.Args)
	}
	panic(nil)
}

// Type of all errors generated by this package.
type Err struct {
	Cause      error `json:"cause"`
	HttpStatus int   `json:"httpStatus"`
}

// Implement `error`.
func (self Err) Error() string {
	var buf strings.Builder
	self.writeError(&buf)
	return buf.String()
}

// Implement a hidden interface in "errors".
func (self Err) Is(other error) bool {
	if self.Cause != nil {
		return errors.Is(self.Cause, other)
	}
	err, ok := other.(Err)
	return ok && self == err
}

// Implement a hidden interface in "errors".
func (self Err) Unwrap() error {
	return self.Cause
}

// Support verbose printing via `%+v`.
func (self Err) Format(fms fmt.State, verb rune) {
	if verb == 'v' && fms.Flag('+') {
		self.writeErrorVerbose(fms)
	} else {
		self.writeError(fms)
	}
}

func (self Err) writeError(out io.Writer) {
	self.writeErrorShallow(out)
	if self.Cause != nil {
		fmt.Fprintf(out, `: %v`, self.Cause)
	}
}

func (self Err) writeErrorShallow(out io.Writer) {
	fmt.Fprintf(out, `%v`, `routing error`)
	if self.HttpStatus != 0 {
		fmt.Fprintf(out, ` (HTTP status %v)`, self.HttpStatus)
	}
}

func (self Err) writeErrorVerbose(out io.Writer) {
	self.writeErrorShallow(out)
	if self.Cause != nil {
		fmt.Fprintf(out, `: %+v`, self.Cause)
	}
}

func errMethodNotAllowed(path string, method string) error {
	return Err{
		Cause:      fmt.Errorf(`method %v not allowed for path %q`, method, path),
		HttpStatus: http.StatusMethodNotAllowed,
	}
}

func errNotFound(path string, method string) error {
	return Err{
		Cause:      fmt.Errorf(`no endpoint for %v %q`, method, path),
		HttpStatus: http.StatusNotFound,
	}
}

var regexpCache sync.Map

func cachedRegexp(pattern string) *regexp.Regexp {
	val, ok := regexpCache.Load(pattern)
	if ok {
		return val.(*regexp.Regexp)
	}
	reg := regexp.MustCompile(pattern)
	regexpCache.Store(pattern, reg)
	return reg
}

func rec(ptr *error) {
	err := toErr(recover())
	if err != nil {
		*ptr = err
	}
}

/*
Should be used with `recover()`:

	err := toErr(recover())

Caution: `recover()` only works when called DIRECTLY inside a deferred function.
When called ANYWHERE ELSE, even in functions called BY a deferred function,
it's a nop.
*/
func toErr(val interface{}) error {
	if val == nil {
		return nil
	}

	err, _ := val.(error)
	if err != nil {
		return err
	}

	/**
	We're not prepared to handle non-error, non-nil panics.

	By using `recover()`, we prevent Go from displaying the automatic stacktrace
	pointing to the place where such a panic was raised. However, such
	stacktraces are only displayed for panics that crash the process. `net/http`
	also uses `recover()` for request handlers, so we're not making things any
	worse.
	*/
	panic(val)
}

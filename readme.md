## Overview

Experimental router for Go HTTP servers. Imperative control flow with declarative syntax. Doesn't need middleware.

Very simple, small, dependency-free, reasonably fast.

Recommended in conjunction with [`github.com/mitranim/goh`](https://github.com/mitranim/goh), which implements various "response" types that satisfy `http.Handler`.

API docs: https://pkg.go.dev/github.com/mitranim/rout.

Performance: a moderately-sized routing table of a production app can take a few microseconds. The only forced allocation on success is `[]string` for captured args, if any.

Examples: see below.

## TOC

* [Why](#why)
* [Usage](#usage)
* [Caveats](#caveats)

## Why

* You want a router because "manual" routing requires too much code.
* Most routing libraries are fatally flawed. They sacrifice imperative control flow, then invent "middleware" to work around the resulting problems. Imperative flow is precious. Treasure it. Don't let it go.

`rout` is an evolution of "manual" routing that avoids common router flaws:

* Control flow is still imperative. It _doesn't need middleware_: simply call A before/after B.
* No "mounting". Routing always uses full URL paths: `/a/b/c` instead of `"/a" "/b" "/c"`. This makes the code _searchable_.
* Correct "not found" and "method not allowed" semantics out of the box.
* Supports multiple ways of pattern matching: exact, prefix, OAS-style pattern, and regexp. Patterns are compiled once and cached.

The resulting code is very dense, simple, and clear.

## Usage

```golang
import (
  "net/http"

  "github.com/mitranim/goh"
  "github.com/mitranim/rout"
)

type (
  Rew = http.ResponseWriter
  Req = *http.Request
  Han = http.Handler
)

// Top-level handler, oversimplified. See docs on `Rou`.
var handler http.Handler = rout.RouFunc(routes)

/*
This is executed for every request.

Unknown paths cause the router to return error 404. Unknown methods on known
paths cause the router to return error 405. The error is handled by YOUR code,
which is an important advantage; see the handlers above.
*/
func routes(r rout.R) {
  r.Pat(`/`).Get().Han(pageIndex)
  r.Pat(`/articles`).Get().Han(pageArticles)
  r.Pat(`/articles/{}`).Get().ParamHan(pageArticle)
  r.Sta(`/api`).Sub(routesApi)
  r.Get().Handler(fileServer)
}

var fileServer = http.FileServer(http.Dir(`public`))

// This is executed for every request that gets routed to it.
func routesApi(r rout.R) {
  /**
  Enable CORS only for this route. This would usually involve middleware.
  With `rout`, you just call A before B.
  */
  allowCors(r.Rew.Header())

  r.Sta(`/api/articles`).Sub(routesApiArticles)
}

// This is executed for every request that gets routed to it.
func routesApiArticles(r rout.R) {
  r.Pat(`/api/articles`).Methods(func(r rout.R) {
    r.Get().Han(apiArticleFeed)
    r.Post().Han(apiArticleCreate)
  })
  r.Pat(`/api/articles/{}`).Methods(func(r rout.R) {
    r.Get().ParamHan(apiArticleGet)
    r.Patch().ParamHan(apiArticleUpdate)
    r.Delete().ParamHan(apiArticleDelete)
  })
}

// Oversimplified for example's sake.
func allowCors(head http.Header)                  {}
func pageIndex(req Req) Han                       { return goh.StringOk(`ok`) }
func pageArticles(req Req) Han                    { return goh.StringOk(`ok`) }
func pageArticle(req Req, args []string) Han      { return goh.StringOk(`ok`) }
func apiArticleFeed(req Req) Han                  { return goh.StringOk(`ok`) }
func apiArticleCreate(req Req) Han                { return goh.StringOk(`ok`) }
func apiArticleGet(req Req, args []string) Han    { return goh.StringOk(`ok`) }
func apiArticleUpdate(req Req, args []string) Han { return goh.StringOk(`ok`) }
func apiArticleDelete(req Req, args []string) Han { return goh.StringOk(`ok`) }
```

## Caveats

Because `rout` uses panics for control flow, error handling may involve `defer` and `recover`. Consider using [`github.com/mitranim/try`](https://github.com/mitranim/try).

## Changelog

### v0.6.0

* Support OAS-style patterns such as `/one/{}/two`.
  * Add `Pat`.
  * Add `Rou.Pat`.
* Add tools for introspection via "dry runs":
  * `Visit`
  * `Visitor`
  * `RegexpVisitor`
  * `PatternVisitor`
  * `Ident`
  * `IdentType`
  * `NopRew`
* Various breaking renamings for brevity:
  * `Router` → `Rou`.
  * `Exact` → `Exa`.
  * `Begin` → `Sta`.
  * `Regex` → `Reg`.
* Export lower-level pattern-matching tools via `Match`.

### v0.5.0

Lexicon change: "Res" → "Han" for anything that involves `http.Handler`.

Add support for `*http.Response` via `Respond`, `Router.Res`, `Router.ParamRes`. Expressing responses with `http.Handler` remains the preferred and recommended approach.

### v0.4.4

Optimize error creation: hundreds of nanoseconds → tens of nanoseconds.

### v0.4.3

Exported `ErrStatus`.

### v0.4.2

`WriteErr` and `Router.Serve` now perform deep error unwrapping to obtain the HTTP status code of an error.

### v0.4.1

`Router.Res` and `Router.ParamRes` are now variadic, accepting multiple funcs. They try the funs sequentially until one of the funcs returns a non-nil handler. Also added `Coalesce` which provides similar behavior without a router.

### v0.4.0

Support multiple ways of URL matching:

  * `Router.Regex` → by regexp (supports capture groups)
  * `Router.Exact` → by exact match (no capture groups)
  * `Router.Begin` → by prefix (no capture groups)

Breaking: renamed `Router.Reg` → `Router.Regex` for symmetry with the other path pattern methods.

Most routes can be expressed with exact or prefix matches. Regexps are needed
only for capturing args. This makes routing simpler, clearer, and less error-prone. It also significantly improves performance, compared to using regexps for everything.

### v0.3.0

Added simple shortcuts:

  * `WriteErr`
  * `Router.Route`
  * `Router.Serve`

Breaking: `Route` has been replaced with `Router.Route`.

### v0.2.1

`Res` now implements `http.Handler`. This is not used internally, but could be handy for users.

### v0.2.0

API redesign: fewer types, simpler, more flexible.

### v0.1.1

Method matching is now case-insensitive.

### v0.1.0

First tagged release.

## License

https://unlicense.org

## Misc

I'm receptive to suggestions. If this library _almost_ satisfies you but needs changes, open an issue or chat me up. Contacts: https://mitranim.com/#contacts

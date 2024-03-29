## Overview

Experimental router for Go HTTP servers. Imperative control flow with declarative syntax. Doesn't need middleware.

Very simple, small, dependency-free, reasonably fast.

Recommended in conjunction with [`github.com/mitranim/goh`](https://github.com/mitranim/goh), which implements various "response" types that satisfy `http.Handler`.

API docs: https://pkg.go.dev/github.com/mitranim/rout.

Performance: a moderately-sized routing table of a production app can take a few microseconds, with very minimal allocations.

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
func routes(rou rout.Rou) {
  rou.Pat(`/`).Get().Han(pageIndex)
  rou.Pat(`/articles`).Get().Han(pageArticles)
  rou.Pat(`/articles/{}`).Get().ParamHan(pageArticle)
  rou.Sta(`/api`).Sub(routesApi)
  rou.Get().Handler(fileServer)
}

var fileServer = http.FileServer(http.Dir(`public`))

// This is executed for every request that gets routed to it.
func routesApi(rou rout.Rou) {
  /**
  Enable CORS only for this route. This would usually involve middleware.
  With `rout`, you just call A before B.
  */
  allowCors(rou.Rew.Header())

  rou.Sta(`/api/articles`).Sub(routesApiArticles)
}

// This is executed for every request that gets routed to it.
func routesApiArticles(rou rout.Rou) {
  rou.Pat(`/api/articles`).Methods(func(rou rout.Rou) {
    rou.Get().Han(apiArticleFeed)
    rou.Post().Han(apiArticleCreate)
  })
  rou.Pat(`/api/articles/{}`).Methods(func(rou rout.Rou) {
    rou.Get().ParamHan(apiArticleGet)
    rou.Patch().ParamHan(apiArticleUpdate)
    rou.Delete().ParamHan(apiArticleDelete)
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

## Changelog

### v0.8.0

`ErrStatus` no longer falls back on status 500. Callers of `ErrStatus` must check if the status is 0 and implement their own fallback. Use the newly added `ErrStatusFallback` for the old behavior.

### v0.7.1

Renamed `Pat.Append` to `Pat.AppendTo` for consistency with other libraries.

### v0.7.0

Added `Rou.Mut` for introspection. It stores the matched `Endpoint` after a successful match. Minor breaking change: `Rou.Done` is removed, as the boolean is now part of `Mut`. There is no measurable performance regression.

### v0.6.3

Fix panic in `ErrStatus` when unwrapping non-comparable error values.

### v0.6.2

On successful match, `Rou` no longer uses panics to break the flow. Instead it continues execution, but flips a hidden flag that causes it to ignore all further routes. This avoids some weird gotchas related to nil panics previously used by this library.

Performance: this forces a single tiny allocation (`new(bool)`), but appears to marginally improve routing performance.

### v0.6.1

Bugfix for parametrized pattern matching in method-only routes.

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

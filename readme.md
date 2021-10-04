## Overview

Experimental router for Go HTTP servers. Imperative control flow with declarative syntax. Doesn't need middleware.

Very simple, small (≈300 LoC without docs), dependency-free, reasonably fast.

Recommended in conjunction with [`github.com/mitranim/goh`](https://github.com/mitranim/goh), which implements various "response" types that satisfy `http.Handler`.

API docs: https://pkg.go.dev/github.com/mitranim/rout.

Examples: see below.

## TOC

* [Why](#why)
* [Usage](#usage)
* [Caveats](#caveats)

## Why

* "Manual" routing = noisy code.
* Most routing libraries are fatally flawed:
  * They sacrifice imperative control flow, then invent "middleware" to work around the resulting problems. Imperative flow is precious. Treasure it. Don't let it go.
  * They invent a custom pattern dialect, with its own limitations and gotchas, instead of simply using regexps.
  * They tend to encourage incorrect semantics, such as 404 instead of 405.

`rout` is an evolution of "manual" routing that avoids common router flaws:

* Control flow is still imperative. It _doesn't need middleware_: simply call A before/after B.
* Uses regexps. Compared to custom pattern dialects, this is less surprising and more flexible. Regexps are compiled only once and cached.
* Routing uses full URL paths: `^/a/b/c$` instead of `"/a" "/b" "/c"`. This makes the code _searchable_, reducing the need for external docs.
* Correct "not found" and "method not allowed" semantics out of the box.

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
  Req = http.Request
  Res = http.Handler
)

var fileServer = http.FileServer(http.Dir(`public`))

// Top-level handler, simplified. Uses `rout.WriteErr` for error writing.
func handleRequestSimple(rew Rew, req *Req) {
  rout.MakeRouter(rew, req).Serve(routes)
}

// Top-level handler with arbitrary error handling.
func handleRequestAdvanced(rew Rew, req *Req) {
  // Errors are handled ONLY in app code. There are no surprises.
  err := rout.MakeRouter(rew, req).Route(routes)

  // Replace this with custom error handling.
  rout.WriteErr(rew, err)
}

// This is executed for every request.
//
// Unknown paths cause the router to return error 404. Unknown methods on known
// paths cause the router to return error 405. The error is handled by YOUR
// code, which is an important advantage; see the handlers above.
func routes(r rout.R) {
  r.Exact(`/`).Get().Res(pageIndex)
  r.Exact(`/articles`).Get().Res(pageArticles)
  r.Regex(`^/articles/([^/]+)$`).Get().ParamRes(pageArticle)
  r.Begin(`/api`).Sub(routesApi)
  r.Get().Handler(fileServer)
}

// This is executed for every request that gets routed to it.
func routesApi(r rout.R) {
  // Enable CORS only for this route. This would usually involve middleware.
  // With `rout`, you just call A before B.
  allowCors(r.Rew.Header())

  r.Begin(`/api/articles`).Sub(routesApiArticles)
}

// This is executed for every request that gets routed to it.
func routesApiArticles(r rout.R) {
  r.Exact(`/api/articles`).Methods(func(r rout.R) {
    r.Get().Res(apiArticleFeed)
    r.Post().Res(apiArticleCreate)
  })
  r.Regex(`^/api/articles/([^/]+)$`).Methods(func(r rout.R) {
    r.Get().ParamRes(apiArticleGet)
    r.Patch().ParamRes(apiArticleUpdate)
    r.Delete().ParamRes(apiArticleDelete)
  })
}

// Oversimplified for example's sake.
func allowCors(head http.Header)                   {}
func pageIndex(req *Req) Res                       { return goh.StringOk(`ok`) }
func pageArticles(req *Req) Res                    { return goh.StringOk(`ok`) }
func pageArticle(req *Req, args []string) Res      { return goh.StringOk(`ok`) }
func apiArticleFeed(req *Req) Res                  { return goh.StringOk(`ok`) }
func apiArticleCreate(req *Req) Res                { return goh.StringOk(`ok`) }
func apiArticleGet(req *Req, args []string) Res    { return goh.StringOk(`ok`) }
func apiArticleUpdate(req *Req, args []string) Res { return goh.StringOk(`ok`) }
func apiArticleDelete(req *Req, args []string) Res { return goh.StringOk(`ok`) }
```

## Caveats

Because `rout` uses panics for control flow, error handling may involve `defer` and `recover`. Consider using [`github.com/mitranim/try`](https://github.com/mitranim/try).

## Changelog

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

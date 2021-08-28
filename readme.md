## Overview

Experimental router for Go HTTP servers. Imperative control flow with declarative syntax. Doesn't need middleware.

Very simple, small (≈300 LoC without docs), dependency-free, reasonably fast.

API docs: https://pkg.go.dev/github.com/mitranim/rout.

Examples: see below.

## TOC

* [Why](#why)
* [Usage](#usage)
* [Caveats](#caveats)

## Why

* Unless your server has only 1-2 endpoints, you need routing.
  * "Manual" routing generates noisy code.
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
  "errors"
  "fmt"
  "net/http"

  "github.com/mitranim/rout"
)

type (
  Rew = http.ResponseWriter
  Req = http.Request
)

// Top-level handler that kicks off routing. Note that errors are handled ONLY
// in app code. `rout` never touches the response writer.
func handleRequest(rew Rew, req *Req) {
  err := rout.Route(rew, req, routes)
  writeErr(rew, req, err)
}

// This is not a "builder" function; it's executed for EVERY request.
//
// Unknown paths cause the router to return error 404. Unknown methods on known
// paths cause the router to return error 405. The error is handled by YOUR
// code, which is an important advantage; see `handleRequest` above.
func routes(r rout.R) {
  r.Reg(`^/$`).Get().Func(pageIndex)
  r.Reg(`^/articles$`).Get().Func(pageArticles)
  r.Reg(`^/articles/([^/]+)$`).Get().ParamFunc(pageArticle)
  r.Reg(`^/api(?:/|$)`).Sub(routesApi)
  r.Get().Handler(fileServer)
}

var fileServer = http.FileServer(http.Dir(`public`))

// This is not a "builder" function; it's executed for EVERY request that gets
// routed to it.
func routesApi(r rout.R) {
  // Enable CORS only for this route. This would usually involve middleware.
  // With `rout`, you just call A before B.
  allowCors(r.Rew.Header())

  r.Reg(`^/api/articles(?:/|$)`).Sub(routesApiArticles)
}

// This is not a "builder" function; it's executed for EVERY request that gets
// routed to it.
func routesApiArticles(r rout.R) {
  r.Reg(`^/api/articles$`).Methods(func(r rout.R) {
    r.Get().Func(apiArticleFeed)
    r.Post().Func(apiArticleCreate)
  })
  r.Reg(`^/api/articles/([^/]+)$`).Methods(func(r rout.R) {
    r.Get().ParamFunc(apiArticleGet)
    r.Patch().ParamFunc(apiArticleUpdate)
    r.Delete().ParamFunc(apiArticleDelete)
  })
}

// Oversimplified for example's sake.
func allowCors(head http.Header)                         {}
func pageIndex(rew Rew, req *Req)                        {}
func pageArticles(rew Rew, req *Req)                     {}
func pageArticle(rew Rew, req *Req, match []string)      {}
func apiArticleFeed(rew Rew, req *Req)                   {}
func apiArticleCreate(rew Rew, req *Req)                 {}
func apiArticleGet(rew Rew, req *Req, match []string)    {}
func apiArticleUpdate(rew Rew, req *Req, match []string) {}
func apiArticleDelete(rew Rew, req *Req, match []string) {}

// Oversimplified for example's sake.
func writeErr(rew Rew, req *Req, err error) {
  writeErrStatus(rew, req, err)
  fmt.Fprint(rew, err)
}

func writeErrStatus(rew Rew, _ *Req, err error) {
  var known rout.Err
  if errors.As(err, &known) && known.Status != 0 {
    rew.WriteHeader(known.Status)
  } else {
    rew.WriteHeader(http.StatusInternalServerError)
  }
}
```

## Caveats

Because `rout` uses panics for control flow, error handling may involve `defer` and `recover`. Consider using [`github.com/mitranim/try`](https://github.com/mitranim/try).

## Changelog

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

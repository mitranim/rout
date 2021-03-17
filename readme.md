## Overview

Experimental router for Go HTTP servers. Imperative control flow with declarative syntax. Doesn't need middleware.

Very simple, small (≈300 LoC without docs), dependency-free, with reasonable performance.

See API docs at https://pkg.go.dev/github.com/mitranim/rout.

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
  * They tend to route by a _combination_ of path and method, leading to incorrect semantics, such as 404 instead of 405.

`rout` is an evolution of "manual" routing that avoids common router flaws:

* Control flow is still imperative. It _doesn't need middleware_: simply call A before B.
* Uses regexps. Compared to custom pattern dialects, this is less surprising and more flexible. (Regexps are compiled only once and cached.)
* Encourages full URL paths: `^/a/b/c$` instead of `"a", "b", "c"`. This makes the code _searchable_, reducing the need for external docs.
* Correct "not found" and "method not allowed" semantics out of the box: routing is done by path, _then_ by method. If there's no method match for a method, this immediately generates an error.

Unlike manual routing, the resulting code is very dense, clear and declarative.

## Usage

```golang
import "github.com/mitranim/rout"

type (
  Rew = http.ResponseWriter
  Req = http.Request
)

// Top-level handler that kicks off routing. Note that errors are handled ONLY
// in app code. `rout` never touches the response writer.
func handleRequest(rew Rew, req *Req) {
  err := rout.Route(rew, req, routes)
  writeErrPlain(rew, req, err)
}

// This is not a "builder" function; it's executed for EVERY request.
//
// Unknown paths cause the router to return error 404. Unknown methods on known
// paths cause the router to return error 405. The error is handled by YOUR
// code, which is an important advantage; see `handleRequest` above.
func routes(r rout.R) {
  r.Get(`^/$`, pageIndex)
  r.Get(`^/articles$`, pageArticles)
  r.Param().Get(`^/articles/([^/]+)$`, pageArticle)
  r.Sub(`^/api(?:/|$)`, routesApi)
  r.Get(``, serveFiles)
}

// This is not a "builder" function; it's executed for EVERY request that gets
// routed to it.
func routesApi(r rout.R) {
  // Different error handling just for this route.
  defer r.Rec(writeErrJson)

  // Enable CORS only for this route. This would usually involve middleware.
  // With `rout`, you just call A before B.
  allowCors(r.Rew.Header())

  r.Sub(`^/api/articles(?:/|$)`, routesApiArticles)
}

// This is not a "builder" function; it's executed for EVERY request that gets
// routed to it.
func routesApiArticles(r rout.R) {
  r.Methods(`^/api/articles$`, func(r rout.MR) {
    r.Get(apiArticleFeed)
    r.Post(apiArticleCreate)
  })
  r.Param().Methods(`^/api/articles/([^/]+)$`, func(r rout.PMR) {
    r.Get(apiArticleGet)
    r.Patch(apiArticleUpdate)
    r.Delete(apiArticleDelete)
  })
}

var serveFiles = http.FileServer(http.Dir(`public`)).ServeHTTP

// Implementations elided for example's sake.
func allowCors        (head http.Header)                  {}
func pageIndex        (rew Rew, req *Req)                 {}
func pageArticles     (rew Rew, req *Req)                 {}
func pageArticle      (rew Rew, req *Req, match []string) {}
func apiArticleFeed   (rew Rew, req *Req)                 {}
func apiArticleCreate (rew Rew, req *Req)                 {}
func apiArticleGet    (rew Rew, req *Req, match []string) {}
func apiArticleUpdate (rew Rew, req *Req, match []string) {}
func apiArticleDelete (rew Rew, req *Req, match []string) {}

// Oversimplified for example's sake.
func writeErrJson(rew Rew, req *Req, err error) {
  if err == nil {
    return
  }
  body, encodeErr := json.Marshal(err)
  if encodeErr != nil {
    log.Error(encodeErr)
    writeErrPlain(rew, req, err)
  } else {
    writeErrStatus(rew, req, err)
    rew.Write(body)
  }
}

func writeErrPlain(rew Rew, req *Req, err error) {
  if err == nil {
    return
  }
  writeErrStatus(rew, req, err)
  rew.Write([]byte(err.Error()))
}

func writeErrStatus(rew Rew, _ *Req, err error) {
  known, _ := err.(rout.Err)
  if known.HttpStatus != 0 {
    rew.WriteHeader(known.HttpStatus)
  } else {
    rew.WriteHeader(http.StatusInternalServerError)
  }
}
```

## Caveats

* New and immature.

* Probably less optimizable than some alternatives. The performance is reasonable and shouldn't be your bottleneck.

* When handling errors in subroutes, either use `r.Rec`, or take care to _re-panic_:

```golang
import "github.com/mitranim/try"

func routes(r rout.R) {
  // Recommended error handling.
  defer r.Rec(writeErr)

  r.Get(`...`, someFunc)
}

func routes(r rout.R) {
  // Lower-level error handling.
  defer func() {
    err := try.Err(recover()) // Takes care of non-error, non-nil panics.

    writeErr(r.Rew, r.Req, err)

    // This is essential. Without it, `rout` would continue to the next route.
    // `rout.Route` will catch this nil panic and treat it as "ok".
    panic(nil)
  }()

  r.Get(`...`, someFunc)
}
```

## Changelog

### v0.1.1

Method matching is now case-insensitive.

### v0.1.0

First tagged release.

## License

https://unlicense.org

## Misc

I'm receptive to suggestions. If this library _almost_ satisfies you but needs changes, open an issue or chat me up. Contacts: https://mitranim.com/#contacts

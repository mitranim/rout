package rout_test

import (
	"io"
	"net/http"

	"github.com/mitranim/rout"
)

type (
	Rew = http.ResponseWriter
	Req = *http.Request
	Res = *http.Response
	Han = http.Handler
)

var fileServer = http.FileServer(http.Dir(`public`))

func ExampleRouter_Route() {
	handleRequest(nil, nil)
}

// Top-level request handler.
func handleRequest(rew Rew, req Req) {
	// Errors are handled ONLY in app code. There are no surprises.
	err := rout.MakeRouter(rew, req).Route(routes)

	// Replace this with custom error handling.
	rout.WriteErr(rew, err)
}

// This is executed for every request.
//
// Unknown paths cause the router to return error 404. Unknown methods on known
// paths cause the router to return error 405. The error is handled by YOUR
// code, which is an important advantage; see `handleRequest` above.
func routes(r rout.R) {
	r.Exact(`/`).Get().Han(pageIndex)
	r.Exact(`/articles`).Get().Han(pageArticles)
	r.Regex(`^/articles/([^/]+)$`).Get().ParamHan(pageArticle)
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
		r.Get().Han(apiArticleFeed)
		r.Post().Han(apiArticleCreate)
	})
	r.Regex(`^/api/articles/([^/]+)$`).Methods(func(r rout.R) {
		r.Get().ParamHan(apiArticleGet)
		r.Patch().ParamHan(apiArticleUpdate)
		r.Delete().ParamHan(apiArticleDelete)
	})
}

// Oversimplified for example's sake.
func allowCors(head http.Header)                  {}
func pageIndex(req Req) Han                       { return Str(`ok`) }
func pageArticles(req Req) Han                    { return Str(`ok`) }
func pageArticle(req Req, args []string) Han      { return Str(`ok`) }
func apiArticleFeed(req Req) Han                  { return Str(`ok`) }
func apiArticleCreate(req Req) Han                { return Str(`ok`) }
func apiArticleGet(req Req, args []string) Han    { return Str(`ok`) }
func apiArticleUpdate(req Req, args []string) Han { return Str(`ok`) }
func apiArticleDelete(req Req, args []string) Han { return Str(`ok`) }

type Str string

func (self Str) ServeHTTP(rew Rew, _ Req) { io.WriteString(rew, string(self)) }

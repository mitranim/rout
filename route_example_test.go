package rout_test

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

func ExampleRouter_Route() {
	handleRequest(nil, nil)
}

// Top-level request handler.
func handleRequest(rew Rew, req *Req) {
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

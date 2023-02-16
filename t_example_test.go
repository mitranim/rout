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

func ExampleRou_Route() {
	handleRequest(nil, nil)
}

// Top-level request handler.
func handleRequest(rew Rew, req Req) {
	// Errors are handled ONLY in app code. There are no surprises.
	err := rout.MakeRou(rew, req).Route(routes)

	// Replace this with custom error handling.
	rout.WriteErr(rew, err)
}

/*
This is executed for every request.

Unknown paths cause the router to return error 404. Unknown methods on known
paths cause the router to return error 405. The error is handled by YOUR code,
which is an important advantage; see `handleRequest` above.
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
func pageIndex(req Req) Han                       { return Str(`ok`) }
func pageArticles(req Req) Han                    { return Str(`ok`) }
func pageArticle(req Req, args []string) Han      { return Str(`ok`) }
func apiArticleFeed(req Req) Han                  { return Str(`ok`) }
func apiArticleCreate(req Req) Han                { return Str(`ok`) }
func apiArticleGet(req Req, args []string) Han    { return Str(`ok`) }
func apiArticleUpdate(req Req, args []string) Han { return Str(`ok`) }
func apiArticleDelete(req Req, args []string) Han { return Str(`ok`) }

type Str string

func (self Str) ServeHTTP(rew Rew, _ Req) { _, _ = io.WriteString(rew, string(self)) }
func (self Str) Ptr() *Str                { return &self }

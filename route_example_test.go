package rout_test

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/mitranim/rout"
)

func ExampleRoute() {
	handleRequest(nil, nil)
}

type (
	Rew = http.ResponseWriter
	Req = http.Request
	Res = http.Handler
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

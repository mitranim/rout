package rout_test

import (
	"encoding/json"
	"log"
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
func writeErrJson(rew Rew, req *Req, err error) {
	if err == nil {
		return
	}
	body, encodeErr := json.Marshal(err)
	if encodeErr != nil {
		log.Printf("%+v", encodeErr)
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

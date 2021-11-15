package rout_test

import (
	"fmt"
	"net/http"
	ht "net/http/httptest"
	"net/url"
	"testing"

	"github.com/mitranim/rout"
)

var (
	stringNop = func(string) {}
	errorNop  = func(error) {}
)

func BenchmarkRoute(b *testing.B) {
	rew := ht.NewRecorder()
	req := makeReq()

	b.ResetTimer()

	for range counter(b.N) {
		serve(rew, req)
	}
}

func makeReq() Req {
	return ht.NewRequest(http.MethodPost, `/api/match/0e60feee70b241d38aa37ab55378f926`, nil)
}

func serve(rew Rew, req Req) {
	try(rout.MakeRouter(rew, req).Route(benchRoutes))
}

func benchRoutes(r rout.R) {
	r.Begin(`/api`).Sub(benchRoutesApi)
}

func benchRoutesApi(r rout.R) {
	r.Begin(`/api/9bbb5`).Sub(unreachableRoute)
	r.Begin(`/api/3b002`).Sub(unreachableRoute)
	r.Begin(`/api/ac134`).Sub(unreachableRoute)
	r.Begin(`/api/e7c64`).Sub(unreachableRoute)
	r.Begin(`/api/424da`).Sub(unreachableRoute)
	r.Begin(`/api/4cddb`).Sub(unreachableRoute)
	r.Begin(`/api/fabe0`).Sub(unreachableRoute)
	r.Begin(`/api/210c4`).Sub(unreachableRoute)
	r.Begin(`/api/c4abd`).Sub(unreachableRoute)
	r.Begin(`/api/82863`).Sub(unreachableRoute)
	r.Begin(`/api/9ef98`).Sub(unreachableRoute)
	r.Begin(`/api/f565f`).Sub(unreachableRoute)
	r.Begin(`/api/f82b7`).Sub(unreachableRoute)
	r.Begin(`/api/d7403`).Sub(unreachableRoute)
	r.Begin(`/api/21838`).Sub(unreachableRoute)
	r.Begin(`/api/1acff`).Sub(unreachableRoute)
	r.Begin(`/api/a0771`).Sub(unreachableRoute)
	r.Begin(`/api/c2bce`).Sub(unreachableRoute)
	r.Begin(`/api/24bef`).Sub(unreachableRoute)
	r.Begin(`/api/091ee`).Sub(unreachableRoute)
	r.Begin(`/api/782d4`).Res(unreachableRes)
	r.Begin(`/api/eeabb`).Res(unreachableRes)
	r.Begin(`/api/5ffc7`).Res(unreachableRes)
	r.Begin(`/api/0f265`).Res(unreachableRes)
	r.Begin(`/api/2c970`).Res(unreachableRes)
	r.Begin(`/api/ac36c`).Res(unreachableRes)
	r.Begin(`/api/8b8d8`).Res(unreachableRes)
	r.Begin(`/api/3faf4`).Res(unreachableRes)
	r.Begin(`/api/65ddd`).Res(unreachableRes)
	r.Begin(`/api/34f35`).Res(unreachableRes)
	r.Begin(`/api/f74f2`).Res(unreachableRes)
	r.Begin(`/api/8031d`).Res(unreachableRes)
	r.Begin(`/api/9bfb8`).Res(unreachableRes)
	r.Begin(`/api/cf538`).Res(unreachableRes)
	r.Begin(`/api/becce`).Res(unreachableRes)
	r.Begin(`/api/183f4`).Res(unreachableRes)
	r.Begin(`/api/3cafa`).Res(unreachableRes)
	r.Begin(`/api/05453`).Res(unreachableRes)
	r.Begin(`/api/f25c7`).Res(unreachableRes)
	r.Begin(`/api/2e1f1`).Res(unreachableRes)
	r.Begin(`/api/match`).Sub(reachableRoute)
	panic("unreachable")
}

func reachableRoute(r rout.R) {
	r.Exact(`/api/match`).Methods(unreachableRoute)

	r.Regex(`^/api/match/([^/]+)$`).Methods(func(r rout.R) {
		r.Get().Res(unreachableRes)
		r.Put().Res(unreachableRes)
		r.Post().Func(reachableFunc)
		r.Delete().Res(unreachableRes)
	})
}

func reachableFunc(rew Rew, _ Req) {
	rew.WriteHeader(201)
}

func unreachableRoute(rout.R) { panic("unreachable") }
func unreachableRes(Req) Res  { panic("unreachable") }

func Benchmark_error_ErrNotFound_string(b *testing.B) {
	for range counter(b.N) {
		stringNop(rout.NotFound(http.MethodPost, `/some/path`).Error())
	}
}

func Benchmark_error_ErrNotFound_interface(b *testing.B) {
	for range counter(b.N) {
		errorNop(rout.NotFound(http.MethodPost, `/some/path`))
	}
}

func Benchmark_error_fmt_Errorf(b *testing.B) {
	for range counter(b.N) {
		errorNop(fmt.Errorf(
			`[rout] routing error (HTTP status 404): no such endpoint: %q %q`,
			http.MethodPost, `/some/path`,
		))
	}
}

func Benchmark_error_fmt_Sprintf(b *testing.B) {
	for range counter(b.N) {
		stringNop(fmt.Sprintf(
			`[rout] routing error (HTTP status 404): no such endpoint: %q %q`,
			http.MethodPost, `/some/path`,
		))
	}
}

func Benchmark_error_fmt_Sprintf_ErrNotFound(b *testing.B) {
	for range counter(b.N) {
		errorNop(rout.ErrNotFound(fmt.Sprintf(
			`[rout] routing error (HTTP status 404): no such endpoint: %q %q`,
			http.MethodPost, `/some/path`,
		)))
	}
}

func Benchmark_bound_methods(b *testing.B) {
	for range counter(b.N) {
		benchBoundMethod()
	}
}

func benchBoundMethod() {
	_ = rout.MakeRouter(nil, staticReq).Route(staticState.Route)
}

var staticReq = &http.Request{
	Method: http.MethodPatch,
	URL:    &url.URL{Path: `/patch`},
}

var staticState State

type State struct{ _ map[string]string }

func (self *State) Route(r rout.R) {
	r.Exact(`/get`).Get().Func(self.Get)
	r.Exact(`/post`).Post().Res(self.Post)
	r.Exact(`/patch`).Patch().Res(self.Patch)
}

func (self *State) Get(http.ResponseWriter, *http.Request) { panic(`unreachable`) }
func (self *State) Post(*http.Request) http.Handler        { panic(`unreachable`) }
func (self *State) Patch(*http.Request) http.Handler       { return nil }

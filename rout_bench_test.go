package rout_test

import (
	"fmt"
	"net/http"
	ht "net/http/httptest"
	"reflect"
	"testing"

	"github.com/mitranim/rout"
)

func TestRoute(t *testing.T) {
	rew := ht.NewRecorder()
	req := makeReq()

	serve(rew, req)
	eq(201, rew.Code)
}

func BenchmarkRoute(b *testing.B) {
	rew := ht.NewRecorder()
	req := makeReq()

	b.ResetTimer()

	for range counter(b.N) {
		serve(rew, req)
	}
}

func makeReq() *Req {
	return ht.NewRequest(http.MethodPost, `/api/match/0e60feee70b241d38aa37ab55378f926`, nil)
}

func serve(rew Rew, req *Req) {
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
	r.Begin(`/api/782d4`).Sub(unreachableRoute)
	r.Begin(`/api/eeabb`).Sub(unreachableRoute)
	r.Begin(`/api/5ffc7`).Sub(unreachableRoute)
	r.Begin(`/api/0f265`).Sub(unreachableRoute)
	r.Begin(`/api/2c970`).Sub(unreachableRoute)
	r.Begin(`/api/ac36c`).Sub(unreachableRoute)
	r.Begin(`/api/8b8d8`).Sub(unreachableRoute)
	r.Begin(`/api/3faf4`).Sub(unreachableRoute)
	r.Begin(`/api/65ddd`).Sub(unreachableRoute)
	r.Begin(`/api/34f35`).Sub(unreachableRoute)
	r.Begin(`/api/f74f2`).Sub(unreachableRoute)
	r.Begin(`/api/8031d`).Sub(unreachableRoute)
	r.Begin(`/api/9bfb8`).Sub(unreachableRoute)
	r.Begin(`/api/cf538`).Sub(unreachableRoute)
	r.Begin(`/api/becce`).Sub(unreachableRoute)
	r.Begin(`/api/183f4`).Sub(unreachableRoute)
	r.Begin(`/api/3cafa`).Sub(unreachableRoute)
	r.Begin(`/api/05453`).Sub(unreachableRoute)
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

func reachableFunc(rew Rew, _ *Req) {
	rew.WriteHeader(201)
}

func unreachableRoute(rout.R) { panic("unreachable") }
func unreachableRes(*Req) Res { panic("unreachable") }

func eq(exp, act interface{}) {
	if !reflect.DeepEqual(exp, act) {
		panic(fmt.Errorf("expected:\n%#v\ngot:\n%#v\n", exp, act))
	}
}

func counter(n int) []struct{} { return make([]struct{}, n) }

func try(err error) {
	if err != nil {
		panic(err)
	}
}

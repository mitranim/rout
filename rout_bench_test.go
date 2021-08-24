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
	try(rout.Route(rew, req, bRoutes))
}

func bRoutes(r rout.R) {
	r.Sub(`^/api(?:/|$)`, bRoutesApi)
}

func bRoutesApi(r rout.R) {
	r.Sub(`^/api/9bbb5(?:/|$)`, unreachableRoute)
	r.Sub(`^/api/3b002(?:/|$)`, unreachableRoute)
	r.Sub(`^/api/ac134(?:/|$)`, unreachableRoute)
	r.Sub(`^/api/e7c64(?:/|$)`, unreachableRoute)
	r.Sub(`^/api/424da(?:/|$)`, unreachableRoute)
	r.Sub(`^/api/4cddb(?:/|$)`, unreachableRoute)
	r.Sub(`^/api/fabe0(?:/|$)`, unreachableRoute)
	r.Sub(`^/api/210c4(?:/|$)`, unreachableRoute)
	r.Sub(`^/api/c4abd(?:/|$)`, unreachableRoute)
	r.Sub(`^/api/82863(?:/|$)`, unreachableRoute)
	r.Sub(`^/api/9ef98(?:/|$)`, unreachableRoute)
	r.Sub(`^/api/f565f(?:/|$)`, unreachableRoute)
	r.Sub(`^/api/f82b7(?:/|$)`, unreachableRoute)
	r.Sub(`^/api/d7403(?:/|$)`, unreachableRoute)
	r.Sub(`^/api/21838(?:/|$)`, unreachableRoute)
	r.Sub(`^/api/1acff(?:/|$)`, unreachableRoute)
	r.Sub(`^/api/a0771(?:/|$)`, unreachableRoute)
	r.Sub(`^/api/c2bce(?:/|$)`, unreachableRoute)
	r.Sub(`^/api/24bef(?:/|$)`, unreachableRoute)
	r.Sub(`^/api/091ee(?:/|$)`, unreachableRoute)
	r.Sub(`^/api/782d4(?:/|$)`, unreachableRoute)
	r.Sub(`^/api/eeabb(?:/|$)`, unreachableRoute)
	r.Sub(`^/api/5ffc7(?:/|$)`, unreachableRoute)
	r.Sub(`^/api/0f265(?:/|$)`, unreachableRoute)
	r.Sub(`^/api/2c970(?:/|$)`, unreachableRoute)
	r.Sub(`^/api/ac36c(?:/|$)`, unreachableRoute)
	r.Sub(`^/api/8b8d8(?:/|$)`, unreachableRoute)
	r.Sub(`^/api/3faf4(?:/|$)`, unreachableRoute)
	r.Sub(`^/api/65ddd(?:/|$)`, unreachableRoute)
	r.Sub(`^/api/34f35(?:/|$)`, unreachableRoute)
	r.Sub(`^/api/f74f2(?:/|$)`, unreachableRoute)
	r.Sub(`^/api/8031d(?:/|$)`, unreachableRoute)
	r.Sub(`^/api/9bfb8(?:/|$)`, unreachableRoute)
	r.Sub(`^/api/cf538(?:/|$)`, unreachableRoute)
	r.Sub(`^/api/becce(?:/|$)`, unreachableRoute)
	r.Sub(`^/api/183f4(?:/|$)`, unreachableRoute)
	r.Sub(`^/api/3cafa(?:/|$)`, unreachableRoute)
	r.Sub(`^/api/05453(?:/|$)`, unreachableRoute)
	r.Sub(`^/api/match(?:/|$)`, apiTwenty)
	panic("unreachable")
}

func apiTwenty(r rout.R) {
	r.Methods(`^/api/match$`, unreachableMethods)
	r.Param().Methods(`^/api/match/([^/]+)$`, apiTwentyParam)
}

func apiTwentyParam(r rout.PMR) {
	r.Get(unreachableParamFunc)
	r.Put(unreachableParamFunc)
	r.Post(apiTwentyParamPost)
	r.Delete(unreachableParamFunc)
}

func apiTwentyParamPost(rew Rew, req *Req, _ []string) {
	rew.WriteHeader(201)
}

func unreachableRoute(rout.R)                  { panic("unreachable") }
func unreachableMethods(rout.MR)               { panic("unreachable") }
func unreachableParamFunc(Rew, *Req, []string) { panic("unreachable") }

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

package rout

import (
	"fmt"
	"net/http"
	ht "net/http/httptest"
	"net/url"
	"regexp"
	"testing"
)

var (
	stringNop  = func(string) {}
	stringsNop = func([]string) {}
	errorNop   = func(error) {}
	boolNop    = func(bool) {}
)

func BenchmarkRoute(b *testing.B) {
	rew := ht.NewRecorder()
	req := tReqSpecific()

	b.ResetTimer()

	for range iter(b.N) {
		tServe(rew, req)
	}
}

func tRou(meth, path string) Rou {
	return Rou{Method: meth, Pattern: path}
}

func tReqRou(meth, path string) Rou {
	return Rou{Req: tReq(meth, path)}
}

func tReq(meth, path string) hreq {
	return &http.Request{
		Method: meth,
		URL:    &url.URL{Path: path},
	}
}

func tReqSpecific() hreq {
	return tReq(http.MethodPost, `/api/match/0e60feee70b241d38aa37ab55378f926`)
}

func tServe(rew hrew, req hreq) {
	try(MakeRou(rew, req).Route(benchRoutes))
}

func benchRoutes(r R) {
	r.Sta(`/api`).Sub(benchRoutesApi)
}

func benchRoutesApi(r R) {
	r.Sta(`/api/9bbb5`).Sub(unreachableRoute)
	r.Sta(`/api/3b002`).Sub(unreachableRoute)
	r.Sta(`/api/ac134`).Sub(unreachableRoute)
	r.Sta(`/api/e7c64`).Sub(unreachableRoute)
	r.Sta(`/api/424da`).Sub(unreachableRoute)
	r.Sta(`/api/4cddb`).Sub(unreachableRoute)
	r.Sta(`/api/fabe0`).Sub(unreachableRoute)
	r.Sta(`/api/210c4`).Sub(unreachableRoute)
	r.Sta(`/api/c4abd`).Sub(unreachableRoute)
	r.Sta(`/api/82863`).Sub(unreachableRoute)
	r.Sta(`/api/9ef98`).Sub(unreachableRoute)
	r.Sta(`/api/f565f`).Sub(unreachableRoute)
	r.Sta(`/api/f82b7`).Sub(unreachableRoute)
	r.Sta(`/api/d7403`).Sub(unreachableRoute)
	r.Sta(`/api/21838`).Sub(unreachableRoute)
	r.Sta(`/api/1acff`).Sub(unreachableRoute)
	r.Sta(`/api/a0771`).Sub(unreachableRoute)
	r.Sta(`/api/c2bce`).Sub(unreachableRoute)
	r.Sta(`/api/24bef`).Sub(unreachableRoute)
	r.Sta(`/api/091ee`).Sub(unreachableRoute)
	r.Sta(`/api/782d4`).Han(unreachableHan)
	r.Sta(`/api/eeabb`).Han(unreachableHan)
	r.Sta(`/api/5ffc7`).Han(unreachableHan)
	r.Sta(`/api/0f265`).Han(unreachableHan)
	r.Sta(`/api/2c970`).Han(unreachableHan)
	r.Sta(`/api/ac36c`).Han(unreachableHan)
	r.Sta(`/api/8b8d8`).Han(unreachableHan)
	r.Sta(`/api/3faf4`).Han(unreachableHan)
	r.Sta(`/api/65ddd`).Han(unreachableHan)
	r.Sta(`/api/34f35`).Han(unreachableHan)
	r.Sta(`/api/f74f2`).Han(unreachableHan)
	r.Sta(`/api/8031d`).Han(unreachableHan)
	r.Sta(`/api/9bfb8`).Han(unreachableHan)
	r.Sta(`/api/cf538`).Han(unreachableHan)
	r.Sta(`/api/becce`).Han(unreachableHan)
	r.Sta(`/api/183f4`).Han(unreachableHan)
	r.Sta(`/api/3cafa`).Han(unreachableHan)
	r.Sta(`/api/05453`).Han(unreachableHan)
	r.Sta(`/api/f25c7`).Han(unreachableHan)
	r.Sta(`/api/2e1f1`).Han(unreachableHan)
	r.Sta(`/api/match`).Sub(reachableRoute)

	if !*r.Done {
		panic(`unexpected non-done router state`)
	}
}

func reachableRoute(r R) {
	r.Exa(`/api/match`).Methods(unreachableRoute)

	r.Pat(`/api/match/{}`).Methods(func(r R) {
		r.Get().Han(unreachableHan)
		r.Put().Han(unreachableHan)
		r.Post().Func(reachableFunc)
		r.Delete().Han(unreachableHan)
	})
}

func reachableFunc(rew hrew, _ hreq) {
	rew.WriteHeader(201)
}

func unreachableRoute(R)       { panic(`unreachable`) }
func unreachableHan(hreq) hhan { panic(`unreachable`) }

func Benchmark_error_ErrNotFound_string(b *testing.B) {
	for range iter(b.N) {
		stringNop(NotFound(http.MethodPost, `/some/path`).Error())
	}
}

func Benchmark_error_ErrNotFound_interface(b *testing.B) {
	for range iter(b.N) {
		errorNop(NotFound(http.MethodPost, `/some/path`))
	}
}

func Benchmark_error_fmt_Errorf(b *testing.B) {
	for range iter(b.N) {
		errorNop(fmt.Errorf(
			`[rout] routing error (HTTP status 404): no such endpoint: %q %q`,
			http.MethodPost, `/some/path`,
		))
	}
}

func Benchmark_error_fmt_Sprintf(b *testing.B) {
	for range iter(b.N) {
		stringNop(fmt.Sprintf(
			`[rout] routing error (HTTP status 404): no such endpoint: %q %q`,
			http.MethodPost, `/some/path`,
		))
	}
}

func Benchmark_error_fmt_Sprintf_ErrNotFound(b *testing.B) {
	for range iter(b.N) {
		errorNop(ErrNotFound(fmt.Sprintf(
			`[rout] routing error (HTTP status 404): no such endpoint: %q %q`,
			http.MethodPost, `/some/path`,
		)))
	}
}

func Benchmark_bound_methods(b *testing.B) {
	for range iter(b.N) {
		benchBoundMethod()
	}
}

func benchBoundMethod() {
	try(MakeRou(nil, staticReq).Route(staticState.Route))
}

var staticState State

type State struct{ _ map[string]string }

func (self *State) Route(r R) {
	r.Exa(`/get`).Get().Func(self.Get)
	r.Exa(`/post`).Post().Han(self.Post)
	r.Exa(`/patch`).Patch().Han(self.Patch)
}

func (self *State) Get(hrew, hreq)  { panic(`unreachable`) }
func (self *State) Post(hreq) hhan  { panic(`unreachable`) }
func (self *State) Patch(hreq) hhan { return nil }

func Benchmark_regexp_MatchString_hit(b *testing.B) {
	reg := regexp.MustCompile(`^/one/two/([^/]+)/([^/]+)$`)
	b.ResetTimer()

	for range iter(b.N) {
		boolNop(reg.MatchString(
			`/one/two/24b6d268f6dd4031b58de9b30e12b0e0/5a8f3d3c357749e4980aab3deffcb840`,
		))
	}
}

func Benchmark_regexp_MatchString_miss(b *testing.B) {
	reg := regexp.MustCompile(`^/one/two/([^/]+)/([^/]+)$`)
	b.ResetTimer()

	for range iter(b.N) {
		boolNop(reg.MatchString(
			`/one/two/three/24b6d268f6dd4031b58de9b30e12b0e0/5a8f3d3c357749e4980aab3deffcb840`,
		))
	}
}

func Benchmark_regexp_FindStringSubmatch_hit(b *testing.B) {
	reg := regexp.MustCompile(`^/one/two/([^/]+)/([^/]+)$`)
	b.ResetTimer()

	for range iter(b.N) {
		stringsNop(reg.FindStringSubmatch(
			`/one/two/24b6d268f6dd4031b58de9b30e12b0e0/5a8f3d3c357749e4980aab3deffcb840`,
		))
	}
}

func Benchmark_regexp_FindStringSubmatch_miss(b *testing.B) {
	reg := regexp.MustCompile(`^/one/two/([^/]+)/([^/]+)$`)
	b.ResetTimer()

	for range iter(b.N) {
		stringsNop(reg.FindStringSubmatch(
			`/one/two/three/24b6d268f6dd4031b58de9b30e12b0e0/5a8f3d3c357749e4980aab3deffcb840`,
		))
	}
}

func Benchmark_Pat_Match_hit(b *testing.B) {
	var pat Pat
	try(pat.Parse(`/one/two/{}/{}`))
	b.ResetTimer()

	for range iter(b.N) {
		boolNop(pat.Match(
			`/one/two/24b6d268f6dd4031b58de9b30e12b0e0/5a8f3d3c357749e4980aab3deffcb840`,
		))
	}
}

func Benchmark_Pat_Match_miss(b *testing.B) {
	var pat Pat
	try(pat.Parse(`/one/two/{}/{}`))
	b.ResetTimer()

	for range iter(b.N) {
		boolNop(pat.Match(
			`/one/two/three/24b6d268f6dd4031b58de9b30e12b0e0/5a8f3d3c357749e4980aab3deffcb840`,
		))
	}
}

func Benchmark_Pat_Submatch_hit(b *testing.B) {
	var pat Pat
	try(pat.Parse(`/one/two/{}/{}`))
	b.ResetTimer()

	for range iter(b.N) {
		stringsNop(pat.Submatch(
			`/one/two/24b6d268f6dd4031b58de9b30e12b0e0/5a8f3d3c357749e4980aab3deffcb840`,
		))
	}
}

func Benchmark_Pat_Submatch_miss(b *testing.B) {
	var pat Pat
	try(pat.Parse(`/one/two/{}/{}`))
	b.ResetTimer()

	for range iter(b.N) {
		stringsNop(pat.Submatch(
			`/one/two/three/24b6d268f6dd4031b58de9b30e12b0e0/5a8f3d3c357749e4980aab3deffcb840`,
		))
	}
}

func Benchmark_Pat_Exact_hit(b *testing.B) {
	pat := Pat{`/one/two/24b6d268f6dd4031b58de9b30e12b0e0`}
	b.ResetTimer()
	for range iter(b.N) {
		boolNop(pat.Match(`/one/two/24b6d268f6dd4031b58de9b30e12b0e0`))
	}
}

func Benchmark_Pat_Exact_miss(b *testing.B) {
	pat := Pat{`/one/two/24b6d268f6dd4031b58de9b30e12b0e0`}
	b.ResetTimer()
	for range iter(b.N) {
		boolNop(pat.Match(`/one/two/5a8f3d3c357749e4980aab3deffcb840`))
	}
}

func Benchmark_regexp_MustCompile(b *testing.B) {
	for range iter(b.N) {
		_ = regexp.MustCompile(`^/one/two/([^/]+)/([^/]+)$`)
	}
}

func Benchmark_Pat_Parse(b *testing.B) {
	for range iter(b.N) {
		var pat Pat
		try(pat.Parse(`/one/two/{}/{}`))
	}
}

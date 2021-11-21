package rout

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	r "reflect"
	"runtime"
	"strings"
	"testing"
)

type (
	hrew = http.ResponseWriter
	hreq = *http.Request
	hres = *http.Response
	hhan = http.Handler
)

var (
	staticHandlerVar hhan = Str(`hello world`)
	staticHandlerPtr hhan = Str(`hello world`).Ptr()

	staticReq = &http.Request{
		Method: http.MethodPatch,
		URL:    &url.URL{Path: `/patch`},
	}
)

const (
	staticHandlerConst = Str(`hello world`)
)

func eq(t testing.TB, exp, act interface{}) {
	t.Helper()
	if !r.DeepEqual(exp, act) {
		t.Fatalf(`
expected (detailed):
	%#[1]v
actual (detailed):
	%#[2]v
expected (simple):
	%[1]v
actual (simple):
	%[2]v
`, exp, act)
	}
}

func notEq(t testing.TB, exp, act interface{}) {
	t.Helper()
	if r.DeepEqual(exp, act) {
		t.Fatalf(`
unexpected equality (detailed):
	%#[1]v
unexpected equality (simple):
	%[1]v
	`, exp, act)
	}
}

func errs(t testing.TB, msg string, err error) {
	if err == nil {
		t.Fatalf(`expected an error with %q, got none`, msg)
	}

	str := err.Error()
	if !strings.Contains(str, msg) {
		t.Fatalf(`expected an error with a message containing %q, got %q`, msg, str)
	}
}

func panics(t testing.TB, msg string, fun func()) {
	t.Helper()
	val := catchAny(fun)

	if val == nil {
		t.Fatalf(`expected %v to panic, found no panic`, funcName(fun))
	}

	str := fmt.Sprint(val)
	if !strings.Contains(str, msg) {
		t.Fatalf(`
expected %v to panic with a message containing:
	%v
found the following message:
	%v
`, funcName(fun), msg, str)
	}
}

func funcName(val interface{}) string {
	return runtime.FuncForPC(r.ValueOf(val).Pointer()).Name()
}

func catchAny(fun func()) (val interface{}) {
	defer recAny(&val)
	fun()
	return
}

func recAny(ptr *interface{}) { *ptr = recover() }

func iter(count int) []struct{} { return make([]struct{}, count) }

const (
	// Must not be included in `tMethods`.
	tNonMethod = `PUT`

	// Must not be included in `tPaths`.
	tNonPath = `/one/two/three/four`
)

var (
	tMethods    = []string{`GET`, `POST`}
	tPaths      = []string{`/`, `/one`, `/one/two`}
	tAnyMethods = append(tMethods, ``)
	tAnyPaths   = append(tPaths, ``)
)

type Str string

func (self Str) ServeHTTP(rew hrew, _ hreq) { _, _ = io.WriteString(rew, string(self)) }
func (self Str) Ptr() *Str                  { return &self }

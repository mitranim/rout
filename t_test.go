package rout

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	ht "net/http/httptest"
	r "reflect"
	"strings"
	"testing"
)

func TestPat_Parse(t *testing.T) {
	fail := func(src string) {
		errs(t, `[rout] invalid OAS-style pattern`, new(Pat).Parse(src))
	}

	fail(`?`)
	fail(`#`)
	fail(`{`)
	fail(`}`)
	fail(`{{}}`)
	fail(`{?}`)
	fail(`{#}`)
	fail(`{}}`)
	fail(`{}}`)
	fail(`{}{`)

	errs(
		t,
		`[rout] invalid OAS-style pattern "{}{}{}{}{}{}{}{}{}": found 9 template expressions which exceeds limit 8`,
		new(Pat).Parse(`{}{}{}{}{}{}{}{}{}`),
	)

	errs(
		t,
		`[rout] invalid OAS-style pattern "/{}/{}/{}/{}/{}/{}/{}/{}/{}": found 9 template expressions which exceeds limit 8`,
		new(Pat).Parse(`/{}/{}/{}/{}/{}/{}/{}/{}/{}`),
	)

	test := func(exp Pat, src string) {
		t.Helper()
		var tar Pat
		try(tar.Parse(src))
		eq(t, exp, tar)
	}

	test(nil, ``)
	test(Pat{`/`}, `/`)
	test(Pat{`//`}, `//`)
	test(Pat{`one`}, `one`)
	test(Pat{`one_two_three`}, `one_two_three`)
	test(Pat{`one two three`}, `one two three`)
	test(Pat{`/one`}, `/one`)
	test(Pat{`/one/`}, `/one/`)
	test(Pat{`/one/two`}, `/one/two`)
	test(Pat{` `}, ` `)
	test(Pat{``}, `{}`)
	test(Pat{``}, `{one}`)
	test(Pat{``}, `{one_two_three}`)
	test(Pat{``}, `{one two three}`)
	test(Pat{`/`, ``}, `/{}`)
	test(Pat{`/`, ``}, `/{one}`)
	test(Pat{``, `/`}, `{}/`)
	test(Pat{``, `/`}, `{one}/`)
	test(Pat{``, ``}, `{}{}`)
	test(Pat{``, ``}, `{one}{two}`)
	test(Pat{``, `/`, ``}, `{}/{}`)
	test(Pat{``, `/`, ``}, `{one}/{two}`)
	test(Pat{`/`, ``, `/`, ``}, `/{}/{}`)
	test(Pat{`/`, ``, `/`, ``}, `/{one}/{two}`)
	test(Pat{`/`, ``, `/`, ``}, `/{}/{}`)
	test(Pat{`/`, ``, `/`, ``}, `/{one}/{two}`)
}

func TestPat_Reg(t *testing.T) {
	test := func(exp string, src Pat) {
		t.Helper()
		eq(t, exp, src.Reg())
	}

	test(`^$`, Pat{})
	test(`^\^\$$`, Pat{`^$`})
	test(`^/$`, Pat{`/`})
	test(`^one$`, Pat{`one`})
	test(`^\.$`, Pat{`.`})
	test(`^\.\.\.$`, Pat{`...`})
	test(`^\.\.\.$`, Pat{`.`, `.`, `.`})
	test(`^\./\./\.$`, Pat{`.`, `/`, `.`, `/`, `.`})
	test(`^\.([^/?#]+)\.([^/?#]+)\.$`, Pat{`.`, ``, `.`, ``, `.`})
	test(`^([^/?#]+)$`, Pat{``})
	test(`^/one/([^/?#]+)/two/([^/?#]+)$`, Pat{`/one/`, ``, `/two/`, ``})
}

func TestPat_Num(t *testing.T) {
	test := func(exp int, pat Pat) {
		t.Helper()
		eq(t, exp, pat.Num())
	}

	test(0, nil)
	test(0, Pat{})
	test(0, Pat{`one`})
	test(0, Pat{`one`, `two`})
	test(0, Pat{`one`, `two`, `three`})
	test(1, Pat{``})
	test(1, Pat{``, `one`})
	test(1, Pat{`one`, ``})
	test(1, Pat{`one`, ``, `two`})
	test(2, Pat{``, ``})
	test(2, Pat{``, `one`, ``})
	test(2, Pat{`one`, ``, ``})
	test(2, Pat{`one`, ``, `two`, ``})
	test(2, Pat{`one`, ``, `two`, ``, `three`})
	test(2, Pat{`one`, ``, ``, `two`})
	test(2, Pat{`one`, ``, ``, `two`, `three`})
	test(2, Pat{``, ``, `one`, `two`, `three`})
	test(2, Pat{``, `one`, `two`, `three`, ``})
}

func TestPat_Match(t *testing.T) {
	test := func(exp bool, inp string, pat Pat) {
		t.Helper()
		eq(t, exp, pat.Match(inp))
	}

	test(true, ``, Pat{})
	test(false, ``, Pat{``})
	test(false, `/`, Pat{})
	test(false, ``, Pat{`/`})
	test(true, ` `, Pat{` `})
	test(true, `  `, Pat{`  `})
	test(true, `{}`, Pat{`{}`})
	test(false, `one`, Pat{`{}`})
	test(true, `one`, Pat{``})
	test(true, `one`, Pat{`one`})
	test(false, `two`, Pat{`one`})
	test(true, `/one`, Pat{`/`, ``})
	test(false, `/one`, Pat{`/`, ``, ``})
	test(false, `/one`, Pat{``, `/`})
	test(false, `/one`, Pat{``, `/`, ``})
	test(true, `/one`, Pat{`/`, `o`, `n`, `e`})
	test(true, `/one`, Pat{`/`, `on`, `e`})
	test(true, `/one`, Pat{`/`, `o`, `ne`})
	test(true, `/one`, Pat{`/o`, `n`, `e`})
	test(false, `/one/`, Pat{`/`, `o`, `n`, `e`})
	test(true, `/one/`, Pat{`/`, `o`, `n`, `e`, `/`})
	test(true, `/one/two`, Pat{`/one/two`})
	test(true, `/one/two`, Pat{`/`, `one`, `/`, `two`})
	test(true, `/one/two`, Pat{`/`, ``, `/`, ``})
	test(false, `/one/two`, Pat{``, `one`, ``, `two`})
	test(true, `/one/two`, Pat{`/one/`, ``})
	test(true, `/one/two`, Pat{`/`, ``, `/two`})
	test(true, `/one/two_three`, Pat{`/one/two_`, ``})
	test(true, `/one/two_three.four`, Pat{`/one/two_`, ``})
	test(false, `/one/two_three.four`, Pat{`/one/two_`, ``, `.four`})
}

func TestPat_Submatch(t *testing.T) {
	test := func(exp []string, inp string, pat Pat) {
		t.Helper()
		eq(t, exp, pat.Submatch(inp))
	}

	test([]string{}, ` `, Pat{` `})
	test([]string{}, `/`, Pat{`/`})
	test([]string{}, `/one`, Pat{`/one`})
	test([]string{`one`}, `one`, Pat{``})
	test([]string{`one`}, `/one`, Pat{`/`, ``})
	test([]string{}, `/one/two`, Pat{`/`, `one`, `/`, `two`})
	test([]string{`one`, `two`}, `/one/two`, Pat{`/`, ``, `/`, ``})
	test([]string{`one`}, `/one/two`, Pat{`/`, ``, `/two`})
	test([]string{`two`}, `/one/two`, Pat{`/one/`, ``})
	test([]string(nil), `/one/two`, Pat{`/`, ``, `/`, ``, `/`})
	test([]string{`three`}, `/one/two_three`, Pat{`/one/two_`, ``})
	test([]string{`three.four`}, `/one/two_three.four`, Pat{`/one/two_`, ``})
	test([]string(nil), `/one/two_three.four`, Pat{`/one/two_`, ``, `.four`})
}

func TestRou_matchMethod(t *testing.T) {
	test := func(exp bool, rou Rou, req hreq) {
		t.Helper()
		rou.Req = req
		eq(t, exp, rou.matchMethod())
	}

	test(true, tRou(``, ``), tReq(``, ``))
	test(true, tRou(``, ``), tReq(`GET`, ``))
	test(true, tRou(`GET`, ``), tReq(`GET`, ``))
	test(true, tRou(`POST`, ``), tReq(`POST`, ``))
	test(false, tRou(`GET`, ``), tReq(`POST`, ``))
	test(false, tRou(`POST`, ``), tReq(`GET`, ``))
	test(false, tRou(`GET`, ``), tReq(``, ``))

	for _, meth := range tAnyMethods {
		for _, path := range tAnyPaths {
			for _, pattern := range tAnyPaths {
				test(true, tRou(``, pattern), tReq(meth, path))
				test(true, tRou(meth, pattern), tReq(meth, path))
				test(false, tRou(tNonMethod, pattern), tReq(meth, path))
			}
		}
	}
}

func TestMatch_Match_MatchExa(t *testing.T) {
	test := func(exp bool, pat, inp string) {
		t.Helper()
		eq(t, exp, MatchExa.Match(pat, inp))
	}

	for _, input := range tAnyPaths {
		for _, pattern := range tAnyPaths {
			test(true, ``, input)
			test(false, tNonPath, input)
			test((pattern == `` || pattern == input), pattern, input)
		}
	}

	test(true, ``, ``)
	test(true, ``, `/`)
	test(true, ``, `/one`)
	test(true, ``, `/one/two`)
	test(true, `/`, `/`)
	test(true, `/one`, `/one`)
	test(true, `/one/two`, `/one/two`)
	test(false, `/`, ``)
	test(false, `/one`, ``)
	test(false, `/one/two`, ``)
	test(false, `/`, `/one`)
	test(false, `/one`, `/`)
	test(false, `/one/two`, `/one`)
}

func TestMatch_Match_MatchSta(t *testing.T) {
	test := func(exp bool, pat, inp string) {
		t.Helper()
		eq(t, exp, MatchSta.Match(pat, inp))
	}

	for _, path := range tAnyPaths {
		test(true, ``, path)
		test(false, tNonPath, path)
	}

	test(true, ``, ``)
	test(true, ``, `/`)
	test(true, ``, `/one`)
	test(true, ``, `/one/`)
	test(true, ``, `/onetwo`)
	test(true, ``, `/onetwo/`)
	test(true, ``, `/one/two`)
	test(true, ``, `/one/two/`)
	test(true, ``, `/one/twothree`)
	test(true, ``, `/one/twothree/`)

	test(false, `/`, ``)
	test(false, `/`, `one`)
	test(false, `/`, `one/`)

	test(false, `//`, ``)
	test(false, `//`, `/`)
	test(true, `//`, `//`)
	test(false, `//`, `/one`)
	test(false, `//`, `/one/`)

	test(false, `/`, ``)
	test(true, `/`, `/`)
	test(true, `/`, `/one`)
	test(true, `/`, `/one/`)
	test(true, `/`, `/onetwo`)
	test(true, `/`, `/onetwo/`)
	test(true, `/`, `/one/two`)
	test(true, `/`, `/one/two/`)
	test(true, `/`, `/one/twothree`)
	test(true, `/`, `/one/twothree/`)

	test(false, `/one`, ``)
	test(false, `/one`, `/`)
	test(true, `/one`, `/one`)
	test(true, `/one`, `/one/`)
	test(false, `/one`, `/onetwo`)
	test(false, `/one`, `/onetwo/`)
	test(true, `/one`, `/one/two`)
	test(true, `/one`, `/one/two/`)
	test(true, `/one`, `/one/twothree`)
	test(true, `/one`, `/one/twothree/`)

	test(false, `/one/`, ``)
	test(false, `/one/`, `/`)
	test(false, `/one/`, `/one`)
	test(true, `/one/`, `/one/`)
	test(false, `/one/`, `/onetwo`)
	test(false, `/one/`, `/onetwo/`)
	test(true, `/one/`, `/one/two`)
	test(true, `/one/`, `/one/two/`)
	test(true, `/one/`, `/one/twothree`)
	test(true, `/one/`, `/one/twothree/`)

	test(false, `/one/two`, ``)
	test(false, `/one/two`, `/`)
	test(false, `/one/two`, `/one`)
	test(false, `/one/two`, `/one/`)
	test(false, `/one/two`, `/onetwo`)
	test(false, `/one/two`, `/onetwo/`)
	test(true, `/one/two`, `/one/two`)
	test(true, `/one/two`, `/one/two/`)
	test(false, `/one/two`, `/one/twothree`)
	test(false, `/one/two`, `/one/twothree/`)

	test(false, `/one/two/`, ``)
	test(false, `/one/two/`, `/`)
	test(false, `/one/two/`, `/one`)
	test(false, `/one/two/`, `/one/`)
	test(false, `/one/two/`, `/onetwo`)
	test(false, `/one/two/`, `/onetwo/`)
	test(false, `/one/two/`, `/one/two`)
	test(true, `/one/two/`, `/one/two/`)
	test(false, `/one/two/`, `/one/twothree`)
	test(false, `/one/two/`, `/one/twothree/`)
}

func TestMatch_Match_MatchReg(t *testing.T) {
	test := func(exp bool, pat, inp string) {
		t.Helper()
		eq(t, exp, MatchReg.Match(pat, inp))
	}

	for _, path := range tAnyPaths {
		test(true, ``, path)
		test(false, tNonPath, path)
	}

	test(true, ``, ``)
	test(true, ``, `/`)
	test(true, ``, `/one`)
	test(true, `^$`, ``)
	test(true, `(?:)`, ``)
	test(true, `\s`, ` `)
	test(true, `^/$`, `/`)
	test(true, `/one`, `/one`)
	test(true, `/one`, `/onetwo`)
	test(true, `/one`, `/one/two`)
	test(true, `/two`, `/one/two`)
	test(true, `^/one$`, `/one`)

	test(false, `^$`, ` `)
	test(false, `^/$`, `/one`)
	test(false, `^/one$`, `/one/`)
	test(false, `^/one$`, `/one/two`)
	test(false, `^/two$`, `/one/two`)
}

// Delegates to `Pat.Match`, which is tested separately.
// This needs to check only the basics.
func TestMatch_Match_MatchPat(t *testing.T) {
	test := func(exp bool, pat, inp string) {
		t.Helper()
		eq(t, exp, MatchPat.Match(pat, inp))
	}

	for _, path := range tAnyPaths {
		test(true, ``, path)
		test(false, tNonPath, path)
	}

	test(true, ``, ``)
	test(true, ``, `/`)
	test(true, ``, `/one`)
	test(true, ``, `/one/`)
	test(true, ``, `/one/two`)

	test(false, `{}`, ``)
	test(false, `{}`, `/`)
	test(true, `{}`, `one`)
	test(false, `{}`, `one/`)
	test(false, `{}`, `/one`)
	test(false, `{}`, `/one/`)
	test(false, `{}`, `/one/two`)
	test(false, `{}`, `one/two`)
	test(false, `{}`, `one/two/`)

	test(false, `/`, ``)
	test(true, `/`, `/`)
	test(false, `/`, `one`)
	test(false, `/`, `/one`)
	test(false, `/`, `one/`)
	test(false, `/`, `/one/`)
	test(false, `/`, `/one/two`)
	test(false, `/`, `one/two`)
	test(false, `/`, `one/two/`)

	test(true, `/{}`, `/one`)
	test(true, `/{}`, `/two`)
	test(false, `/{}`, `/one/`)
	test(false, `/{}`, `/one/two`)
	test(true, `/one/{}`, `/one/two`)
	test(true, `/one/{}`, `/one/three`)
	test(false, `/one/{}`, `/one/two/`)
	test(false, `/one/{}`, `/one/two/three`)
	test(true, `/{}/{}`, `/one/two`)
	test(true, `/{}/{}`, `/two/three`)
	test(false, `/{}/{}`, `/one/two/`)
}

// Delegates to exact match.
// We only need to check the basics.
func TestMatch_Submatch_MatchExa(t *testing.T) {
	test := func(exp []string, pat, inp string) {
		t.Helper()
		eq(t, exp, MatchExa.Submatch(pat, inp))
	}

	test([]string{}, ``, ``)
	test([]string{}, ``, `/`)
	test([]string{}, ``, `/one`)
	test([]string{}, ``, `/one/two`)
	test([]string{}, `/`, `/`)
	test([]string{}, `/one`, `/one`)
	test([]string{}, `/one/two`, `/one/two`)

	test(nil, `/`, ``)
	test(nil, `/`, `/one`)
	test(nil, `/`, `/one/two`)
	test(nil, `/one`, `/`)
	test(nil, `/one`, `/one/two`)
	test(nil, `/one/two`, `/`)
	test(nil, `/one/two`, `/one`)
}

// Delegates to `.Match` with `MatchSta` (more or less).
// We only need to check the basics.
func TestMatch_Submatch_MatchSta(t *testing.T) {
	test := func(exp []string, pat, inp string) {
		t.Helper()
		eq(t, exp, MatchSta.Submatch(pat, inp))
	}

	test([]string{}, ``, ``)
	test([]string{}, ``, `/`)
	test([]string{}, ``, `/one`)
	test([]string{}, ``, `/one/two`)
	test([]string{}, `/`, `/`)
	test([]string{}, `/`, `/one`)
	test([]string{}, `/`, `/one/`)
	test([]string{}, `/`, `/one/two`)
	test([]string{}, `/`, `/one/two/`)
	test([]string{}, `/one`, `/one`)
	test([]string{}, `/one`, `/one/`)
	test([]string{}, `/one`, `/one/two`)
	test([]string{}, `/one`, `/one/two/`)
	test([]string{}, `/one/two`, `/one/two`)
	test([]string{}, `/one/two`, `/one/two/`)

	test(nil, `/`, ``)
	test(nil, `/one`, `/`)
	test(nil, `/one/`, `/one`)
	test(nil, `/one/two`, `/one`)
}

func TestMatch_Submatch_MatchReg(t *testing.T) {
	test := func(exp []string, pat, inp string) {
		t.Helper()
		eq(t, exp, MatchReg.Submatch(pat, inp))
	}

	for _, path := range tAnyPaths {
		test(nil, tNonPath, path)
	}
	for _, path := range tAnyPaths {
		test([]string{}, ``, path)
	}
	for _, path := range tAnyPaths {
		test([]string{}, `(?:)`, path)
	}
	for _, path := range tAnyPaths {
		test([]string{}, `^`+path+`$`, path)
	}
	for _, path := range tAnyPaths {
		test([]string{``}, `()`, path)
	}
	for _, path := range tAnyPaths {
		test([]string{``, ``}, `()()`, path)
	}
	for _, path := range tAnyPaths {
		test([]string{``}, `^`+path+`$()`, path)
	}
	for _, path := range tAnyPaths {
		test([]string{path}, `(`+path+`)`, path)
	}
	for _, path := range tAnyPaths {
		test([]string{path}, `^(`+path+`)$`, path)
	}
	for _, path := range tAnyPaths {
		test([]string{``, path, ``}, `()^(`+path+`)$()`, path)
	}

	test(
		[]string{`24b6d268f6dd4031b58de9b30e12b0e0`},
		`^/one/two/([^/]+)$`,
		`/one/two/24b6d268f6dd4031b58de9b30e12b0e0`,
	)

	test(
		[]string{`24b6d268f6dd4031b58de9b30e12b0e0`, `5a8f3d3c357749e4980aab3deffcb840`},
		`^/one/two/([^/]+)/([^/]+)$`,
		`/one/two/24b6d268f6dd4031b58de9b30e12b0e0/5a8f3d3c357749e4980aab3deffcb840`,
	)
}

// Delegates to `Pat.Submatch`, which is tested separately.
// This needs to check only the basics.
func TestMatch_Submatch_MatchPat(t *testing.T) {
	test := func(exp []string, pat, inp string) {
		t.Helper()
		eq(t, exp, MatchPat.Submatch(pat, inp))
	}

	test(
		nil,
		`/one/two/{}`,
		`/one/two/24b6d268f6dd4031b58de9b30e12b0e0/5a8f3d3c357749e4980aab3deffcb840`,
	)

	test(
		[]string{`24b6d268f6dd4031b58de9b30e12b0e0`},
		`/one/two/{}`,
		`/one/two/24b6d268f6dd4031b58de9b30e12b0e0`,
	)

	test(
		[]string{`24b6d268f6dd4031b58de9b30e12b0e0`, `5a8f3d3c357749e4980aab3deffcb840`},
		`/one/two/{}/{}`,
		`/one/two/24b6d268f6dd4031b58de9b30e12b0e0/5a8f3d3c357749e4980aab3deffcb840`,
	)
}

func TestRou_Match_OnlyMethod(t *testing.T) {
	test := func(exp bool, meth, pat string, req hreq) {
		t.Helper()
		rou := Rou{Req: req, Method: meth, Pattern: pat, OnlyMethod: true}
		eq(t, exp, rou.Match())
	}

	test(true, ``, ``, tReq(``, ``))
	test(true, ``, ``, tReq(``, `/one/two`))
	test(true, ``, ``, tReq(`POST`, ``))
	test(true, ``, ``, tReq(`POST`, `/one/two`))
	test(true, ``, `/one/two`, tReq(``, ``))
	test(true, ``, `/one/two`, tReq(``, `/one/two`))
	test(true, ``, `/one/two`, tReq(`POST`, ``))
	test(true, ``, `/one/two`, tReq(`POST`, `/one/two`))
	test(true, `POST`, ``, tReq(`POST`, ``))
	test(true, `POST`, ``, tReq(`POST`, `/one/two`))
	test(true, `POST`, `/one/two`, tReq(`POST`, ``))
	test(true, `POST`, `/one/two`, tReq(`POST`, `/one/two`))

	test(false, `POST`, ``, tReq(``, ``))
	test(false, `POST`, ``, tReq(`GET`, ``))
	test(false, `GET`, ``, tReq(`POST`, ``))
	test(false, `POST`, `/one/two`, tReq(``, `/one/two`))
	test(false, `POST`, `/one/two`, tReq(`GET`, `/one/two`))
	test(false, `GET`, `/one/two`, tReq(`POST`, `/one/two`))
}

func TestRou_Submatch_OnlyMethod_Exa(t *testing.T) {
	test := func(exp []string, meth, pat string, req hreq) {
		t.Helper()
		rou := Rou{Req: req, Method: meth, Pattern: pat, OnlyMethod: true}
		eq(t, exp, rou.Submatch())
	}

	test([]string{}, ``, ``, tReq(``, ``))
	test([]string{}, ``, ``, tReq(``, `/one/two`))
	test([]string{}, ``, ``, tReq(`POST`, ``))
	test([]string{}, ``, ``, tReq(`POST`, `/one/two`))
	test([]string{}, ``, `/one/two`, tReq(``, `/one/two`))
	test([]string{}, ``, `/one/two`, tReq(`POST`, `/one/two`))
	test([]string{}, `POST`, ``, tReq(`POST`, ``))
	test([]string{}, `POST`, ``, tReq(`POST`, `/one/two`))
	test([]string{}, `POST`, `/one/two`, tReq(`POST`, `/one/two`))

	test(nil, ``, `/one/two`, tReq(``, ``))
	test(nil, ``, `/one/two`, tReq(`POST`, ``))
	test(nil, `POST`, `/one/two`, tReq(`POST`, ``))
	test(nil, `POST`, ``, tReq(``, ``))
	test(nil, `POST`, ``, tReq(`GET`, ``))
	test(nil, `GET`, ``, tReq(`POST`, ``))
	test(nil, `POST`, `/one/two`, tReq(``, `/one/two`))
	test(nil, `POST`, `/one/two`, tReq(`GET`, `/one/two`))
	test(nil, `GET`, `/one/two`, tReq(`POST`, `/one/two`))
}

func TestRou_Submatch_OnlyMethod_Pat(t *testing.T) {
	test := func(exp []string, rou Rou) {
		t.Helper()
		eq(t, exp, rou.Submatch())
	}

	test([]string{}, tReqRou(``, ``).Pat(``).MethodOnly())
	test([]string{}, tReqRou(`GET`, ``).Pat(``).MethodOnly())
	test([]string{}, tReqRou(`GET`, ``).Pat(``).MethodOnly().Get())
	test([]string(nil), tReqRou(``, ``).Pat(``).MethodOnly().Get())
	test([]string(nil), tReqRou(`GET`, ``).Pat(``).MethodOnly().Post())

	test([]string{}, tReqRou(``, `/one/two`).Pat(``).MethodOnly())
	test([]string{}, tReqRou(`GET`, `/one/two`).Pat(``).MethodOnly())
	test([]string{}, tReqRou(`GET`, `/one/two`).Pat(``).MethodOnly().Get())
	test([]string(nil), tReqRou(``, `/one/two`).Pat(``).MethodOnly().Get())
	test([]string(nil), tReqRou(`GET`, `/one/two`).Pat(``).MethodOnly().Post())

	test([]string{`two`}, tReqRou(``, `/one/two`).Pat(`/one/{}`).MethodOnly())
	test([]string{`two`}, tReqRou(`GET`, `/one/two`).Pat(`/one/{}`).MethodOnly())
	test([]string{`two`}, tReqRou(`GET`, `/one/two`).Pat(`/one/{}`).MethodOnly().Get())
	test([]string(nil), tReqRou(``, `/one/two`).Pat(`/one/{}`).MethodOnly().Get())
	test([]string(nil), tReqRou(`GET`, `/one/two`).Pat(`/one/{}`).MethodOnly().Post())
}

// Oversimplified. Needs more tests.
func TestRoute(t *testing.T) {
	rew := ht.NewRecorder()
	rou := MakeRou(rew, tReqSpecific())
	try(rou.Route(benchRoutes))

	eq(t, 201, rew.Code)

	eq(
		t,
		Mut{
			Endpoint: Endpoint{
				Pattern: `/api/match/{}`,
				Match:   MatchPat,
				Method:  http.MethodPost,
				Handler: Ident(reachableFunc),
			},
			Done: true,
		},
		*rou.Mut,
	)
}

func TestErrStatus(t *testing.T) {
	test := func(exp int, err error) {
		t.Helper()
		eq(t, exp, ErrStatus(err))
	}

	test(0, nil)
	test(0, io.EOF)
	test(http.StatusNotFound, NotFound(``, ``))
	test(http.StatusMethodNotAllowed, MethodNotAllowed(``, ``))
	test(http.StatusNotFound, fmt.Errorf(`wrapped: %w`, NotFound(``, ``)))

	// Must avoid a runtime panic due to `==` on uncomparable error values.
	test(http.StatusNotFound, ErrUncomparable{ErrUncomparable{ErrUncomparable{NotFound(``, ``)}}})

	// Must avoid an infinite loop when an error unwraps to itself.
	test(0, ErrUnwrapCyclic{NotFound(``, ``)})
	test(0, ErrUnwrapCyclic{})
}

func TestErrStatusFallback(t *testing.T) {
	test := func(exp int, err error) {
		t.Helper()
		eq(t, exp, ErrStatusFallback(err))
	}

	test(http.StatusInternalServerError, nil)
	test(http.StatusInternalServerError, io.EOF)
	test(http.StatusNotFound, NotFound(``, ``))
	test(http.StatusMethodNotAllowed, MethodNotAllowed(``, ``))
	test(http.StatusNotFound, fmt.Errorf(`wrapped: %w`, NotFound(``, ``)))

	// Must avoid a runtime panic due to `==` on uncomparable error values.
	test(http.StatusNotFound, ErrUncomparable{ErrUncomparable{ErrUncomparable{NotFound(``, ``)}}})

	// Must avoid an infinite loop when an error unwraps to itself.
	test(http.StatusInternalServerError, ErrUnwrapCyclic{NotFound(``, ``)})
	test(http.StatusInternalServerError, ErrUnwrapCyclic{})
}

func TestWriteErr(t *testing.T) {
	test := func(exp int, err error) {
		t.Helper()
		rew := ht.NewRecorder()
		WriteErr(rew, err)

		if err == nil {
			eq(t, 0, len(rew.Body.Bytes()))
		} else {
			eq(t, err.Error(), rew.Body.String())
		}

		eq(t, exp, rew.Code)
	}

	test(http.StatusOK, nil)
	test(http.StatusInternalServerError, io.EOF)
	test(http.StatusNotFound, NotFound(``, ``))
	test(http.StatusMethodNotAllowed, MethodNotAllowed(``, ``))
	test(http.StatusNotFound, fmt.Errorf(`wrapped: %w`, NotFound(``, ``)))
}

func TestRespond(t *testing.T) {
	eq(t, nil, Respond(nil, nil))
	eq(t, nil, Respond(nil, new(http.Response)))
	eq(t, nil, Respond(ht.NewRecorder(), nil))

	rew := ht.NewRecorder()
	rew.Header().Set(`One`, `two`)

	try(Respond(rew, &http.Response{
		StatusCode: http.StatusBadRequest,
		Header:     http.Header{`Three`: {`four`}},
		Body:       io.NopCloser(strings.NewReader(`hello world`)),
	}))
	res := rew.Result()

	eq(t, http.StatusBadRequest, res.StatusCode)
	eq(t, http.Header{`One`: {`two`}, `Three`: {`four`}}, res.Header)
	eq(t, io.NopCloser(bytes.NewReader([]byte(`hello world`))), res.Body)
}

/*
This investigates various quirks of conversion of non-interfaces to interfaces.
We're relying on implementation details that may be inconsistent between
different Go implementations, or even compiler versions. Might revise in the
future.
*/
func TestIdent(t *testing.T) {
	test := func(val interface{}) {
		t.Helper()
		eq(t, Ident(val), Ident(val))
		notEq(t, [2]uintptr{}, Ident(val))
		notEq(t, Ident(func() {}), Ident(val))
	}

	test(t)
	test(Ident)
	test(eq)
	test(`hello world`)
	test([]byte(`hello world`))
	test(Str(`hello world`))
	test(staticHandlerVar)
	test(staticHandlerPtr)
	test(staticHandlerConst)
	test(struct{}{})

	// Constants get deduplicated.
	eq(t, Ident(`hello world`), Ident(`hello world`))
	eq(t, Ident(Str(`hello world`)), Ident(Str(`hello world`)))
	eq(t, Ident(123.456i), Ident(123.456i))

	type Zero0 struct{}
	type Zero1 struct{}

	// Zero-sized values get deduplicated, using zerobase.
	eq(t, Ident(struct{}{}), Ident(struct{}{}))
	eq(t, Ident(Zero0{}), Ident(Zero0{}))
	eq(t, Ident(Zero1{}), Ident(Zero1{}))
	eq(t, Ident([0]byte{}), Ident([0]byte{}))
	eq(t, Ident(Zero0{})[1], Ident([0]byte{})[1])

	// Zero-sized values of different types are not the same.
	notEq(t, Ident(struct{}{}), Ident(Zero0{}))
	notEq(t, Ident(struct{}{}), Ident(Zero1{}))
	notEq(t, Ident(Zero0{}), Ident(Zero1{}))
	notEq(t, Ident(Zero0{}), Ident([0]byte{}))

	// Non-zero-sized non-constants aren't identical even if they're equal or equivalent.
	notEq(t, Ident(func() {}), Ident(func() {}))

	// These values used to be different before Go 1.18. The test is disabled
	// because this is not consistent across Go versions, and we don't actually
	// rely on this behavior. It's left here as a warning.
	//
	// notEq(t, Ident([1]byte{}), Ident([1]byte{}))

	// Non-constants with size <= word don't get copied on interface conversion.
	var char byte = 127
	eq(t, Ident(char), Ident(char))
	var fun = func() {}
	eq(t, Ident(fun), Ident(fun))

	// Non-constants larger than 1 word get copied on each interface conversion.
	var large = `hello world`
	notEq(t, Ident(large), Ident(large))

	// Pre-converting to interface allows deduplication regardless of size.
	var largish interface{} = `hello world`
	eq(t, Ident(largish), Ident(largish))
}

func TestIdentType(t *testing.T) {
	test := func(exp r.Type, typ interface{}) {
		t.Helper()
		eq(t, exp, IdentType(Ident(typ)))
	}

	test(nil, nil)
	test(r.TypeOf(0), 123)
	test(r.TypeOf(``), `str`)
	test(r.TypeOf(http.Request{}), http.Request{})
}

func TestRou_Vis(t *testing.T) {
	var (
		handlerFunc = func(hrew, hreq) { panic(`unreachable`) }
		handler     = http.HandlerFunc(handlerFunc)
		han         = func(hreq) hhan { panic(`unreachable`) }
		paramHan    = func(hreq, []string) hhan { panic(`unreachable`) }
		res         = func(hreq) hres { panic(`unreachable`) }
		paramRes    = func(hreq, []string) hres { panic(`unreachable`) }
	)

	route := func(rou Rou) {
		rou.Exa(`/handlerFunc`).Get().Func(handlerFunc)
		rou.Exa(`/handler`).Get().Handler(handler)
		rou.Exa(`/han`).Get().Han(han)
		rou.Exa(`/paramHan`).Get().ParamHan(paramHan)
		rou.Exa(`/res`).Get().Res(res)
		rou.Exa(`/paramRes`).Get().ParamRes(paramRes)

		rou.Sta(`/one`).Sub(func(rou Rou) {
			rou.Pat(`/one/handlerFunc`).Post().Func(handlerFunc)
			rou.Pat(`/one/handler`).Post().Handler(handler)
			rou.Pat(`/one/han`).Post().Han(han)
			rou.Pat(`/one/paramHan`).Post().ParamHan(paramHan)
			rou.Pat(`/one/res`).Post().Res(res)
			rou.Pat(`/one/paramRes`).Post().ParamRes(paramRes)

			rou.Reg(`^/two/([^/])$`).Methods(func(rou Rou) {
				rou.Get().Func(handlerFunc)
				rou.Get().Handler(handler)
				rou.Get().Han(han)
				rou.Get().ParamHan(paramHan)
				rou.Get().Res(res)
				rou.Get().ParamRes(paramRes)

				rou.Patch().Func(handlerFunc)
				rou.Patch().Handler(handler)
				rou.Patch().Han(han)
				rou.Patch().ParamHan(paramHan)
				rou.Patch().Res(res)
				rou.Patch().ParamRes(paramRes)
			})
		})
	}

	var endpoints []Endpoint

	Visit(route, VisitorFunc(func(val Endpoint) {
		endpoints = append(endpoints, val)
	}))

	eq(
		t,
		[]Endpoint{
			{`/handlerFunc`, MatchExa, http.MethodGet, Ident(Func(handlerFunc))},
			{`/handler`, MatchExa, http.MethodGet, Ident(http.Handler(handler))},
			{`/han`, MatchExa, http.MethodGet, Ident(Han(han))},
			{`/paramHan`, MatchExa, http.MethodGet, Ident(ParamHan(paramHan))},
			{`/res`, MatchExa, http.MethodGet, Ident(Res(res))},
			{`/paramRes`, MatchExa, http.MethodGet, Ident(ParamRes(paramRes))},

			{`/one/handlerFunc`, MatchPat, http.MethodPost, Ident(Func(handlerFunc))},
			{`/one/handler`, MatchPat, http.MethodPost, Ident(http.Handler(handler))},
			{`/one/han`, MatchPat, http.MethodPost, Ident(Han(han))},
			{`/one/paramHan`, MatchPat, http.MethodPost, Ident(ParamHan(paramHan))},
			{`/one/res`, MatchPat, http.MethodPost, Ident(Res(res))},
			{`/one/paramRes`, MatchPat, http.MethodPost, Ident(ParamRes(paramRes))},

			{`^/two/([^/])$`, MatchReg, http.MethodGet, Ident(Func(handlerFunc))},
			{`^/two/([^/])$`, MatchReg, http.MethodGet, Ident(http.Handler(handler))},
			{`^/two/([^/])$`, MatchReg, http.MethodGet, Ident(Han(han))},
			{`^/two/([^/])$`, MatchReg, http.MethodGet, Ident(ParamHan(paramHan))},
			{`^/two/([^/])$`, MatchReg, http.MethodGet, Ident(Res(res))},
			{`^/two/([^/])$`, MatchReg, http.MethodGet, Ident(ParamRes(paramRes))},

			{`^/two/([^/])$`, MatchReg, http.MethodPatch, Ident(Func(handlerFunc))},
			{`^/two/([^/])$`, MatchReg, http.MethodPatch, Ident(http.Handler(handler))},
			{`^/two/([^/])$`, MatchReg, http.MethodPatch, Ident(Han(han))},
			{`^/two/([^/])$`, MatchReg, http.MethodPatch, Ident(ParamHan(paramHan))},
			{`^/two/([^/])$`, MatchReg, http.MethodPatch, Ident(Res(res))},
			{`^/two/([^/])$`, MatchReg, http.MethodPatch, Ident(ParamRes(paramRes))},
		},
		endpoints,
	)
}

func TestRegexpVisitor(t *testing.T) {
	var (
		hanExa = func(hreq) hhan { panic(`unreachable`) }
		hanSta = func(hreq) hhan { panic(`unreachable`) }
		hanReg = func(hreq) hhan { panic(`unreachable`) }
		hanPat = func(hreq) hhan { panic(`unreachable`) }
	)

	notEq(t, Ident(hanExa), Ident(hanSta))

	route := func(rou Rou) {
		rou.Exa(`/one/exa`).Post().Han(hanExa)
		rou.Sta(`/two/sta`).Post().Han(hanSta)
		rou.Reg(`^/three/reg/([^/]+)$`).Post().Han(hanReg)
		rou.Pat(`/four/pat/{}`).Post().Han(hanPat)
	}

	var endpoints []Endpoint

	Visit(route, RegexpVisitor{SimpleVisitorFunc(func(path, meth string, ident [2]uintptr) {
		endpoints = append(endpoints, Endpoint{path, MatchReg, meth, ident})
	})})

	eq(
		t,
		[]Endpoint{
			{`^/one/exa$`, MatchReg, http.MethodPost, Ident(hanExa)},
			{`^/two/sta`, MatchReg, http.MethodPost, Ident(hanSta)},
			{`^/three/reg/([^/]+)$`, MatchReg, http.MethodPost, Ident(hanReg)},
			{`^/four/pat/([^/?#]+)$`, MatchReg, http.MethodPost, Ident(hanPat)},
		},
		endpoints,
	)
}

func TestPatternVisitor(t *testing.T) {
	var (
		hanExa = func(hreq) hhan { panic(`unreachable`) }
		hanPat = func(hreq) hhan { panic(`unreachable`) }
	)

	notEq(t, Ident(hanExa), Ident(hanPat))

	// This adapter supports only "exact" and "pattern" matches.
	route := func(rou Rou) {
		rou.Exa(`/one/exa`).Post().Han(hanExa)
		rou.Pat(`/four/pat/{}`).Post().Han(hanPat)
	}

	var endpoints []Endpoint

	vis := PatternVisitor{SimpleVisitorFunc(func(path, meth string, ident [2]uintptr) {
		endpoints = append(endpoints, Endpoint{path, MatchPat, meth, ident})
	})}

	Visit(route, vis)

	eq(
		t,
		[]Endpoint{
			{`/one/exa`, MatchPat, http.MethodPost, Ident(hanExa)},
			{`/four/pat/{}`, MatchPat, http.MethodPost, Ident(hanPat)},
		},
		endpoints,
	)

	routeReg := func(rou Rou) {
		rou.Reg(`^/three/reg/([^/]+)$`).Post().Han(nil)
	}

	panics(
		t,
		`[rout] unable to convert match "reg" for route "^/three/reg/([^/]+)$" "POST" to OAS pattern`,
		func() { Visit(routeReg, vis) },
	)

	routeSta := func(rou Rou) {
		rou.Sta(`/two/sta`).Post().Han(nil)
	}

	panics(
		t,
		`[rout] unable to convert match "sta" for route "/two/sta" "POST" to OAS pattern`,
		func() { Visit(routeSta, vis) },
	)
}

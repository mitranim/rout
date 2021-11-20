package rout_test

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	ht "net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/mitranim/rout"
)

// Incomplete, needs more tests. For now, the router is tested by running it in
// production. 😅
func TestRoute(t *testing.T) {
	rew := ht.NewRecorder()
	req := makeReq()

	serve(rew, req)
	eq(t, 201, rew.Code)
}

func TestErrStatus(t *testing.T) {
	test := func(exp int, err error) {
		t.Helper()
		eq(t, exp, rout.ErrStatus(err))
	}

	test(http.StatusInternalServerError, nil)
	test(http.StatusInternalServerError, io.EOF)
	test(http.StatusNotFound, rout.NotFound(``, ``))
	test(http.StatusMethodNotAllowed, rout.MethodNotAllowed(``, ``))
	test(http.StatusNotFound, fmt.Errorf(`wrapped: %w`, rout.NotFound(``, ``)))
}

func TestWriteErr(t *testing.T) {
	test := func(exp int, err error) {
		t.Helper()
		rew := ht.NewRecorder()
		rout.WriteErr(rew, err)

		if err == nil {
			eq(t, 0, len(rew.Body.Bytes()))
		} else {
			eq(t, err.Error(), rew.Body.String())
		}

		eq(t, exp, rew.Code)
	}

	test(http.StatusOK, nil)
	test(http.StatusInternalServerError, io.EOF)
	test(http.StatusNotFound, rout.NotFound(``, ``))
	test(http.StatusMethodNotAllowed, rout.MethodNotAllowed(``, ``))
	test(http.StatusNotFound, fmt.Errorf(`wrapped: %w`, rout.NotFound(``, ``)))
}

func TestRespond(t *testing.T) {
	rout.Respond(nil, nil)
	rout.Respond(nil, new(http.Response))

	rew := ht.NewRecorder()
	rew.Header().Set(`One`, `two`)

	rout.Respond(rew, &http.Response{
		StatusCode: http.StatusBadRequest,
		Header:     http.Header{`Three`: {`four`}},
		Body:       io.NopCloser(strings.NewReader(`hello world`)),
	})
	res := rew.Result()

	eq(t, http.StatusBadRequest, res.StatusCode)
	eq(t, http.Header{`One`: {`two`}, `Three`: {`four`}}, res.Header)
	eq(t, io.NopCloser(bytes.NewReader([]byte(`hello world`))), res.Body)
}

func testRes(t testing.TB, exp Res, rew *ht.ResponseRecorder) {
	t.Helper()
	eq(t, exp, rew.Result())
}

func eq(t testing.TB, exp, act interface{}) {
	if !reflect.DeepEqual(exp, act) {
		panic(fmt.Errorf("expected:\n%#v\ngot:\n%#v\n", exp, act))
	}
}

func iter(n int) []struct{} { return make([]struct{}, n) }

func try(err error) {
	if err != nil {
		panic(err)
	}
}

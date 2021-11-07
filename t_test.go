package rout_test

import (
	"fmt"
	"io"
	"net/http"
	ht "net/http/httptest"
	"reflect"
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
	test(http.StatusInternalServerError, rout.Err{})
	test(http.StatusBadRequest, rout.Err{Status: http.StatusBadRequest})
	test(http.StatusBadRequest, fmt.Errorf(`wrapped: %w`, rout.Err{Status: http.StatusBadRequest}))
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
	test(http.StatusInternalServerError, rout.Err{})
	test(http.StatusBadRequest, rout.Err{Status: http.StatusBadRequest})
	test(http.StatusBadRequest, fmt.Errorf(`wrapped: %w`, rout.Err{Status: http.StatusBadRequest}))
}

func eq(t testing.TB, exp, act interface{}) {
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

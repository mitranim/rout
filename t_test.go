package rout_test

import (
	"fmt"
	ht "net/http/httptest"
	"reflect"
	"testing"
)

// Incomplete, needs more tests. For now, the router is tested by running it in
// production. 😅
func TestRoute(t *testing.T) {
	rew := ht.NewRecorder()
	req := makeReq()

	serve(rew, req)
	eq(t, 201, rew.Code)
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

package rout

import (
	"errors"
	"net/http"
	"regexp"
	"sync"
)

var regexpCache sync.Map

func cachedRegexp(pattern string) *regexp.Regexp {
	val, ok := regexpCache.Load(pattern)
	if ok {
		return val.(*regexp.Regexp)
	}
	reg := regexp.MustCompile(pattern)
	regexpCache.Store(pattern, reg)
	return reg
}

func rec(ptr *error) {
	err := toErr(recover())
	if err != nil {
		*ptr = err
	}
}

/*
Should be used with `recover()`:

	err := toErr(recover())

Caution: `recover()` only works when called DIRECTLY inside a deferred function.
When called ANYWHERE ELSE, even in functions called BY a deferred function,
it's a nop.
*/
func toErr(val interface{}) error {
	if val == nil {
		return nil
	}

	err, _ := val.(error)
	if err != nil {
		return err
	}

	/**
	We're not prepared to handle non-error, non-nil panics.

	By using `recover()`, we prevent Go from displaying the automatic stacktrace
	pointing to the place where such a panic was raised. However, such
	stacktraces are only displayed for panics that crash the process. `net/http`
	also uses `recover()` for request handlers, so we're not making things any
	worse.
	*/
	panic(val)
}

func reTest(str, pattern string) bool {
	return pattern == `` || cachedRegexp(pattern).MatchString(str)
}

func reMatch(str, pattern string) []string {
	if pattern == `` {
		return []string{}
	}

	match := cachedRegexp(pattern).FindStringSubmatch(str)
	if len(match) >= 1 {
		return match[1:]
	}
	return nil
}

func errStatus(err error) (code int) {
	impl, _ := err.(interface{ HttpStatusCode() int })
	if impl != nil {
		code = impl.HttpStatusCode()
	} else {
		var impl Err
		if errors.As(err, &impl) {
			code = impl.Status
		}
	}

	if code == 0 {
		code = http.StatusInternalServerError
	}
	return
}

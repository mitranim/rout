package rout

import (
	"errors"
	"regexp"
	"strings"
	"sync"
)

type style byte

const (
	styleRegex style = iota
	styleExact
	styleBegin
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

func testRegex(path, pattern string) bool {
	return pattern == `` || cachedRegexp(pattern).MatchString(path)
}

func testExact(path, pattern string) bool {
	return path == pattern
}

func testBegin(path, pattern string) bool {
	if strings.HasPrefix(path, pattern) {
		return len(path) == len(pattern) ||
			hasSlashSuffix(pattern) ||
			hasSlashPrefix(path[len(pattern):])
	}
	return false
}

func matchRegex(path, pattern string) []string {
	if pattern == `` {
		return []string{}
	}

	match := cachedRegexp(pattern).FindStringSubmatch(path)
	if len(match) >= 1 {
		return match[1:]
	}
	return nil
}

func matchExact(path, pattern string) []string {
	if testExact(path, pattern) {
		return []string{}
	}
	return nil
}

func matchBegin(path, pattern string) []string {
	if testBegin(path, pattern) {
		return []string{}
	}
	return nil
}

func hasSlashPrefix(val string) bool {
	return len(val) > 0 && val[0] == '/'
}

func hasSlashSuffix(val string) bool {
	return len(val) > 0 && val[len(val)-1] == '/'
}

func errStatusDeep(err error) int {
	for err != nil {
		impl, _ := err.(interface{ HttpStatusCode() int })
		if impl != nil {
			return impl.HttpStatusCode()
		}

		un := errors.Unwrap(err)
		if un == err {
			return 0
		}
		err = un
	}
	return 0
}

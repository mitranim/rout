package rout

import (
	"fmt"
	"net/http"
	"strconv"
)

var ErrInit = fmt.Errorf(
	`[rout] routing error: the router wasn't properly initialized; please use "rout.MakeRou"`,
)

// Error type returned by `rout.Route` for requests with a known path and an
// unknown method.
type ErrMethodNotAllowed string

// Implement a hidden interface supported by `rout.ErrStatus`.
// Always returns `http.StatusMethodNotAllowed`.
func (ErrMethodNotAllowed) HttpStatusCode() int { return http.StatusMethodNotAllowed }

// Implement `error` by returning self.
func (self ErrMethodNotAllowed) Error() string { return string(self) }

// Error type returned by `rout.Route` for requests with an unknown path.
type ErrNotFound string

// Implement a hidden interface supported by `rout.ErrStatus`.
// Always returns `http.StatusNotFound`.
func (ErrNotFound) HttpStatusCode() int { return http.StatusNotFound }

// Implement `error` by returning self.
func (self ErrNotFound) Error() string { return string(self) }

// Generates an appropriate `ErrMethodNotAllowed`. Used internally.
func MethodNotAllowed(meth, path string) ErrMethodNotAllowed {
	return ErrMethodNotAllowed(Err(
		`method not allowed`, ErrMethodNotAllowed(``).HttpStatusCode(), meth, path,
	))
}

// Generates an appropriate `ErrNotFound`. Used internally.
func NotFound(meth, path string) ErrNotFound {
	return ErrNotFound(Err(
		`no such endpoint`, ErrNotFound(``).HttpStatusCode(), meth, path,
	))
}

/*
Generates a routing error message including the given status, method and path.
More efficient than equivalent `fmt.Sprintf` or `fmt.Errorf`.
*/
func Err(msg string, status int, meth, path string) string {
	const (
		preface      = `[rout] routing error`
		statusPrefix = ` (HTTP status `
		statusSuffix = `)`
		colon        = `: `
		quote        = `"`
		quoteInfix   = `" "`
	)

	buf := make(
		[]byte,
		0,
		len(preface)+
			len(statusPrefix)+
			intLen(status)+
			len(statusSuffix)+
			len(colon)+
			len(msg)+
			len(colon)+
			len(quote)+
			len(meth)+
			len(quoteInfix)+
			len(path)+
			len(quote),
	)

	buf = append(buf, preface...)
	if status != 0 {
		buf = append(buf, statusPrefix...)
		buf = strconv.AppendInt(buf, int64(status), 10)
		buf = append(buf, statusSuffix...)
	}
	buf = append(buf, colon...)
	buf = append(buf, msg...)
	buf = append(buf, colon...)
	buf = append(buf, quote...)
	buf = append(buf, meth...)
	buf = append(buf, quoteInfix...)
	buf = append(buf, path...)
	buf = append(buf, quote...)

	return bytesString(buf)
}

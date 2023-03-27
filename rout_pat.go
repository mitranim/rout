package rout

import (
	"fmt"
	"regexp"
	"strings"
)

/*
Short for "pattern", specifically templated URL pattern compatible with
OpenAPI:

	/
	/{}
	/one/{id}
	/one/{}/two/{}
	/one/{id}/two/{action}
	...
	https://spec.openapis.org/oas/v3.1.0#path-templating

Supports parsing, matching, and capturing. MUCH more efficient than equivalent
Go regexps. Once parsed, the pattern is safe for concurrent use by multiple
goroutines.

This structure represents a pattern parsed via `(*Pat).Parse`. Empty strings
represent capture groups, which are called "template expression" in OpenAPI.
Non-empty strings represent exact matches. Template expressions / capture
groups can't overlap. The current implementation allows only up to 8 capture
groups; this is easy to optimize and serves the common case well. The
limitation could be lifted if there was any demand.

Rules:

	* A non-empty segment matches and consumes the exact same string from the
	  start of the input (a prefix), without capturing.

	* An empty segment matches, consumes, and captures an equivalent of the
	  regular expression `([^/?#]+)`.

	* The pattern matches the entire input, behaving like a regexp wrapped in `^$`.

Just like `*regexp.Regexp`, `Pat` allows names in capture groups, such
as "{id}", but discards them when parsing. Submatching is positional, by index.
*/
type Pat []string

/*
Like `(*regexp.Regexp).MatchString`: returns true if the input matches the
pattern, without capturing.
*/
func (self Pat) Match(inp string) bool {
	return self.match(inp, nil)
}

/*
Similar to `(*regexp.Regexp).FindStringSubmatch`: returns nil or positional
captures. Unlike regexps, the resulting slice has ONLY the captures, without
the matched string. On success, slice length equals `pat.Num()`. Allows only a
limited number of captures; see the comment on the type.
*/
func (self Pat) Submatch(inp string) []string {
	buf := []string{}
	if self.match(inp, &buf) {
		return buf
	}
	return nil
}

func (self Pat) match(rem string, out *[]string) bool {
	var subs subs

outer:
	for _, seg := range self {
		if seg != `` {
			if !strings.HasPrefix(rem, seg) {
				return false
			}
			rem = rem[len(seg):]
			continue
		}

		var ind int = -1
		var char rune

		for ind, char = range rem {
			if char == '/' || char == '?' || char == '#' {
				if !subs.add(strPop(&rem, ind)) {
					return false
				}
				continue outer
			}
		}

		if !subs.add(strPop(&rem, ind+1)) {
			return false
		}
	}

	if len(rem) != 0 {
		return false
	}

	if out != nil {
		*out = append(*out, subs.slice()...)
	}
	return true
}

// Parses the pattern from a string, appending to the receiver.
func (self *Pat) Parse(src string) error {
	/**
	TODO add tests and benchmarks to lock down this behavior. It must allocate
	exactly as much as needed, and avoid allocating if the receiver already has
	enough capacity.
	*/
	buf := self.grow(patLen(src))

	var template bool
	var cursor int
	var templates int

	for ind, char := range src {
		if char == '?' || char == '#' {
			return fmt.Errorf(
				`[rout] invalid OAS-style pattern %q: unexpected %q`,
				src, char,
			)
		}

		if template {
			if char == '}' {
				buf = append(buf, ``)
				cursor = ind + 1
				template = false
				templates++

				if templates > subsCap {
					return fmt.Errorf(
						`[rout] invalid OAS-style pattern %q: found %v template expressions which exceeds limit %v`,
						src, templates, subsCap,
					)
				}

				continue
			}

			if char == '/' {
				return fmt.Errorf(
					`[rout] invalid OAS-style pattern %q: unexpected %q in the middle of a template expression`,
					src, char,
				)
			}
			continue
		}

		if char == '{' {
			prev := src[cursor:ind]
			if prev != `` {
				buf = append(buf, prev)
			}
			cursor = ind
			template = true
			continue
		}

		if char == '}' {
			return fmt.Errorf(
				`[rout] invalid OAS-style pattern %q: unexpected %q outside of template expression`,
				src, char,
			)
		}
	}

	if template {
		return fmt.Errorf(
			`[rout] invalid OAS-style pattern %q: unclosed template expression`,
			src,
		)
	}

	prev := src[cursor:]
	if prev != `` {
		buf = append(buf, prev)
	}

	*self = buf
	return nil
}

/*
Implement `fmt.Stringer` for debug purposes. For patterns parsed from a string,
the resulting representation is functionally equivalent to the original, but
capture groups are now anonymous (their inner text is lost).
*/
func (self Pat) String() string { return bytesString(self.AppendTo(nil)) }

/*
Appends a text representation, same as `.String`. Sometimes allows more
efficient encoding.
*/
func (self Pat) AppendTo(buf []byte) []byte {
	buf = growBytes(buf, self.strLen())
	for _, val := range self {
		if val == `` {
			buf = append(buf, segmentTemplate...)
		} else {
			buf = append(buf, val...)
		}
	}
	return buf
}

/*
Implement `encoding.TextUnmarshaler`, allowing automatic decoding from text,
such as from JSON.
*/
func (self *Pat) UnmarshalText(src []byte) error {
	return self.Parse(string(src))
}

/*
Implement `encoding.TextMarshaler`, allowing automatic encoding to text,
such as for JSON.
*/
func (self Pat) MarshalText() ([]byte, error) {
	return self.AppendTo(nil), nil
}

/*
Same as `(*regexp.Regexp).NumSubexp`. Returns the amount of "capture groups"
by counting empty segments.
*/
func (self Pat) Num() int {
	var num int
	for _, val := range self {
		if val == `` {
			num++
		}
	}
	return num
}

/*
Returns a string representing a regexp pattern that should be equivalent to the
given OAS pattern. The pattern is enclosed in `^$`. Template expressions such
as "{}" or "{id}" are represented with `([^/?#]+)`. Because the pattern type
has no way to store the text inside template expressions, the capture groups in
the resulting regexp are anonymous.
*/
func (self Pat) Reg() string {
	buf := make([]byte, 0, self.regLen())
	buf = append(buf, `^`...)

	for _, val := range self {
		if val == `` {
			buf = append(buf, segmentPattern...)
		} else {
			buf = append(buf, regexp.QuoteMeta(val)...)
		}
	}

	buf = append(buf, `$`...)
	return bytesString(buf)
}

// Approximate estimate of resulting length of `Pat.Reg`.
func (self Pat) regLen() (out int) {
	for _, val := range self {
		if val == `` {
			out += len(segmentPattern)
		} else {
			out += len(val) // Not exact. Escapes require more space.
		}
	}
	out += len(`^$`)
	return
}

func (self Pat) strLen() (out int) {
	for _, val := range self {
		if val == `` {
			out += len(segmentTemplate)
		} else {
			out += len(val)
		}
	}
	return
}

func (self Pat) grow(size int) Pat {
	len, cap := len(self), cap(self)
	if cap-len >= size {
		return self
	}

	next := make(Pat, len, cap+size)
	copy(next, self)
	return next
}

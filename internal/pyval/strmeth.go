// Port of the CPython 3.12 str methods pyparsing uses: Objects/unicodeobject.c
// (unicode_expandtabs_impl, do_strip/do_argstrip, split/splitlines, do_title, find/count
// with ADJUST_INDICES) (PSF License Version 2; Copyright (c) 2001-2024 Python Software
// Foundation).

package pyval

import (
	"math"
	"strings"
	"unicode/utf8"

	"github.com/pyparsing/pyparsing-go/internal/pyuni"
)

// Len is len(s): the number of code points.
func Len(s string) int { return utf8.RuneCountInString(s) }

// ExpandTabs is s.expandtabs(tabsize): each tab becomes the spaces up to the next multiple of
// tabsize, counting code points from the last \n or \r; tabsize <= 0 removes tabs.
func ExpandTabs(s string, tabsize int) string {
	if strings.IndexByte(s, '\t') < 0 {
		return s
	}
	var b strings.Builder
	b.Grow(len(s) + 8)
	col := 0
	for _, r := range s {
		if r == '\t' {
			if tabsize > 0 {
				incr := tabsize - col%tabsize
				col += incr
				b.WriteString(strings.Repeat(" ", incr))
			}
			continue
		}
		col++
		b.WriteRune(r)
		if r == '\n' || r == '\r' {
			col = 0
		}
	}
	return b.String()
}

// IsSpaceRune is str.isspace() for one code point (Py_UNICODE_ISSPACE; includes \x1c-\x1f,
// \x85 and U+00A0).
func IsSpaceRune(r rune) bool { return isSpaceRune(r) }

// IsSpace is s.isspace(): s is non-empty and every code point is whitespace.
func IsSpace(s string) bool { return allRunes(s, isSpaceRune) }

// IsDigit is s.isdigit().
func IsDigit(s string) bool { return allRunes(s, pyuni.IsDigit) }

// IsDecimal is s.isdecimal().
func IsDecimal(s string) bool { return allRunes(s, pyuni.IsDecimal) }

// IsAlpha is s.isalpha().
func IsAlpha(s string) bool { return allRunes(s, pyuni.IsAlpha) }

// IsAlnum is s.isalnum().
func IsAlnum(s string) bool { return allRunes(s, pyuni.IsAlnum) }

// IsPrintable is s.isprintable() (true for the empty string).
func IsPrintable(s string) bool {
	for _, r := range s {
		if !pyuni.IsPrintable(r) {
			return false
		}
	}
	return true
}

func allRunes(s string, pred func(rune) bool) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !pred(r) {
			return false
		}
	}
	return true
}

// Strip is s.strip(chars) with an explicit chars argument: it removes leading and trailing
// code points that occur in chars. An empty chars removes nothing (Python's s.strip("")).
// For s.strip() with no argument (whitespace) use StripSpace.
func Strip(s, chars string) string { return RStrip(LStrip(s, chars), chars) }

// LStrip is s.lstrip(chars) with an explicit chars argument.
func LStrip(s, chars string) string {
	return strings.TrimLeftFunc(s, func(r rune) bool { return strings.ContainsRune(chars, r) })
}

// RStrip is s.rstrip(chars) with an explicit chars argument.
func RStrip(s, chars string) string {
	return strings.TrimRightFunc(s, func(r rune) bool { return strings.ContainsRune(chars, r) })
}

// StripSpace is s.strip() (also s.strip(None)): it removes leading and trailing Python
// whitespace (str.isspace, including \x1c-\x1f and \x85).
func StripSpace(s string) string { return strings.TrimFunc(s, isSpaceRune) }

// LStripSpace is s.lstrip().
func LStripSpace(s string) string { return strings.TrimLeftFunc(s, isSpaceRune) }

// RStripSpace is s.rstrip().
func RStripSpace(s string) string { return strings.TrimRightFunc(s, isSpaceRune) }

// StripOpt is s.strip(chars) where chars nil means None (strip whitespace).
func StripOpt(s string, chars *string) string {
	if chars == nil {
		return StripSpace(s)
	}
	return Strip(s, *chars)
}

// Split is s.split(): the runs of non-whitespace code points (Python whitespace).
func Split(s string) []string { return SplitSpaceN(s, -1) }

// SplitSpaceN is s.split(None, maxsplit): at most maxsplit splits (maxsplit < 0 means no
// limit); the remainder keeps its trailing whitespace but loses its leading whitespace.
func SplitSpaceN(s string, maxsplit int) []string {
	if maxsplit < 0 {
		maxsplit = math.MaxInt
	}
	rs := []rune(s)
	out := []string{}
	i, n := 0, len(rs)
	for maxsplit > 0 {
		for i < n && isSpaceRune(rs[i]) {
			i++
		}
		if i == n {
			return out
		}
		j := i
		for j < n && !isSpaceRune(rs[j]) {
			j++
		}
		out = append(out, string(rs[i:j]))
		maxsplit--
		i = j
	}
	for i < n && isSpaceRune(rs[i]) {
		i++
	}
	if i < n {
		out = append(out, string(rs[i:]))
	}
	return out
}

// SplitSep is s.split(sep, maxsplit) with a str separator (maxsplit < 0 means no limit). An
// empty sep gives ValueError "empty separator".
func SplitSep(s, sep string, maxsplit int) ([]string, error) {
	if sep == "" {
		return nil, valueError("empty separator")
	}
	n := maxsplit + 1
	if maxsplit < 0 {
		n = -1
	}
	return strings.SplitN(s, sep, n), nil
}

// isLineBoundary is the str.splitlines() boundary set: \n \r \v \f \x1c \x1d \x1e \x85
// U+2028 U+2029 (\r\n counts as one boundary).
func isLineBoundary(r rune) bool {
	switch r {
	case '\n', '\r', '\v', '\f', 0x1c, 0x1d, 0x1e, 0x85, 0x2028, 0x2029:
		return true
	}
	return false
}

// SplitLines is s.splitlines(keepends).
func SplitLines(s string, keepends bool) []string {
	rs := []rune(s)
	out := []string{}
	i, n := 0, len(rs)
	for i < n {
		j := i
		for j < n && !isLineBoundary(rs[j]) {
			j++
		}
		eol := j
		if j < n {
			if rs[j] == '\r' && j+1 < n && rs[j+1] == '\n' {
				j += 2
			} else {
				j++
			}
			if keepends {
				eol = j
			}
		}
		out = append(out, string(rs[i:eol]))
		i = j
	}
	return out
}

// Upper is s.upper() (full mappings, Unicode 15.0).
func Upper(s string) string { return pyuni.Upper(s) }

// Lower is s.lower() (full mappings and the final-sigma rule, Unicode 15.0).
func Lower(s string) string { return pyuni.Lower(s) }

// CaseFold is s.casefold().
func CaseFold(s string) string { return pyuni.CaseFold(s) }

// Title is s.title(): a cased code point that follows a cased code point is lowercased (with
// the final-sigma rule), every other code point is titlecased (full mapping).
func Title(s string) string {
	rs := []rune(s)
	var b strings.Builder
	b.Grow(len(s))
	prevCased := false
	for i, r := range rs {
		if prevCased {
			if r == 0x3a3 {
				if finalSigma(rs, i) {
					b.WriteRune(0x3c2)
				} else {
					b.WriteRune(0x3c3)
				}
			} else {
				b.WriteString(string(pyuni.LowerRune(r)))
			}
		} else if t, ok := titleSpecial[r]; ok {
			b.WriteString(t)
		} else {
			b.WriteString(string(pyuni.UpperRune(r)))
		}
		prevCased = pyuni.IsCased(r)
	}
	return b.String()
}

// finalSigma is handle_capital_sigma: whether the U+03A3 at rs[i] is final (preceded by a
// cased code point, skipping case-ignorable ones, and not followed by one).
func finalSigma(rs []rune, i int) bool {
	j := i - 1
	for j >= 0 && pyuni.IsCaseIgnorable(rs[j]) {
		j--
	}
	if j < 0 || !pyuni.IsCased(rs[j]) {
		return false
	}
	j = i + 1
	for j < len(rs) && pyuni.IsCaseIgnorable(rs[j]) {
		j++
	}
	return j == len(rs) || !pyuni.IsCased(rs[j])
}

// adjustIndices is CPython's ADJUST_INDICES for a string of n code points.
func adjustIndices(start, end, n int) (int, int) {
	if end > n {
		end = n
	} else if end < 0 {
		end += n
		if end < 0 {
			end = 0
		}
	}
	if start < 0 {
		start += n
		if start < 0 {
			start = 0
		}
	}
	return start, end
}

// Find is s.find(sub, start, end) in code points (pass 0 and math.MaxInt for the defaults);
// -1 when not found.
func Find(s, sub string, start, end int) int {
	rs, sr := []rune(s), []rune(sub)
	start, end = adjustIndices(start, end, len(rs))
	if end-start < len(sr) {
		return -1
	}
	for i := start; i+len(sr) <= end; i++ {
		if runesHasPrefix(rs[i:], sr) {
			return i
		}
	}
	return -1
}

// RFind is s.rfind(sub, start, end) in code points (pass 0 and math.MaxInt for the
// defaults); -1 when not found.
func RFind(s, sub string, start, end int) int {
	rs, sr := []rune(s), []rune(sub)
	start, end = adjustIndices(start, end, len(rs))
	if end-start < len(sr) {
		return -1
	}
	for i := end - len(sr); i >= start; i-- {
		if runesHasPrefix(rs[i:], sr) {
			return i
		}
	}
	return -1
}

// Count is s.count(sub, start, end) in code points (pass 0 and math.MaxInt for the
// defaults): non-overlapping occurrences; an empty sub counts end-start+1.
func Count(s, sub string, start, end int) int {
	rs, sr := []rune(s), []rune(sub)
	start, end = adjustIndices(start, end, len(rs))
	if end-start < len(sr) {
		return 0
	}
	if len(sr) == 0 {
		return end - start + 1
	}
	c := 0
	for i := start; i+len(sr) <= end; {
		if runesHasPrefix(rs[i:], sr) {
			c++
			i += len(sr)
		} else {
			i++
		}
	}
	return c
}

func runesHasPrefix(rs, p []rune) bool {
	if len(rs) < len(p) {
		return false
	}
	for i, r := range p {
		if rs[i] != r {
			return false
		}
	}
	return true
}

// Slice is s[start:end] with Python's slice rules on code points (negative indices count
// from the end; out-of-range indices are clamped). Pass math.MaxInt for an omitted end.
func Slice(s string, start, end int) string {
	rs := []rune(s)
	n := len(rs)
	clamp := func(i int) int {
		if i < 0 {
			i += n
			if i < 0 {
				i = 0
			}
		} else if i > n {
			i = n
		}
		return i
	}
	start, end = clamp(start), clamp(end)
	if start >= end {
		return ""
	}
	return string(rs[start:end])
}

// Replace is s.replace(old, new, count) (count < 0 replaces all); an empty old inserts new
// before every code point and at the end, as Python does.
func Replace(s, old, new string, count int) string { return strings.Replace(s, old, new, count) }

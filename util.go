// Port of pyparsing/util.py (pyparsing 3.3.3+). MIT License, Copyright (c) 2003-2025  Paul McGuire; see LICENSE.

package pyparsing

import (
	"sort"
	"strings"

	"github.com/pyparsing/pyparsing-go/internal/pyre"
)

// pyIndex normalizes a Python slice bound against length n.
func pyIndex(i, n int) int {
	if i < 0 {
		i += n
		if i < 0 {
			i = 0
		}
	}
	if i > n {
		i = n
	}
	return i
}

// pySlice is s[a:b] on runes with Python slice semantics.
func pySlice(s []rune, a, b int) []rune {
	n := len(s)
	a = pyIndex(a, n)
	b = pyIndex(b, n)
	if a >= b {
		return nil
	}
	return s[a:b]
}

// rfindNL is s.rfind("\n", 0, end).
func rfindNL(s []rune, end int) int {
	end = pyIndex(end, len(s))
	for i := end - 1; i >= 0; i-- {
		if s[i] == '\n' {
			return i
		}
	}
	return -1
}

// findNL is s.find("\n", start).
func findNL(s []rune, start int) int {
	start = pyIndex(start, len(s))
	for i := start; i < len(s); i++ {
		if s[i] == '\n' {
			return i
		}
	}
	return -1
}

// colRunes is util.col.
func colRunes(loc int, s []rune) int {
	if 0 < loc && loc < len(s) && s[loc-1] == '\n' {
		return 1
	}
	return loc - rfindNL(s, loc)
}

// linenoRunes is util.lineno.
func linenoRunes(loc int, s []rune) int {
	end := pyIndex(loc, len(s))
	n := 0
	for i := 0; i < end; i++ {
		if s[i] == '\n' {
			n++
		}
	}
	return n + 1
}

// lineRunes is util.line.
func lineRunes(loc int, s []rune) string {
	lastCR := rfindNL(s, loc)
	nextCR := findNL(s, loc)
	if nextCR >= 0 {
		return string(pySlice(s, lastCR+1, nextCR))
	}
	return string(pySlice(s, lastCR+1, len(s)))
}

// runeSet is a Python set of single-character strings.
type runeSet map[rune]bool

func newRuneSet(s string) runeSet {
	m := runeSet{}
	for _, r := range s {
		m[r] = true
	}
	return m
}

func (s runeSet) copySet() runeSet {
	m := make(runeSet, len(s))
	for r := range s {
		m[r] = true
	}
	return m
}

func (s runeSet) sorted() []rune {
	out := make([]rune, 0, len(s))
	for r := range s {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func (s runeSet) equal(o runeSet) bool {
	if len(s) != len(o) {
		return false
	}
	for r := range s {
		if !o[r] {
			return false
		}
	}
	return true
}

func (s runeSet) union(o runeSet) runeSet {
	m := s.copySet()
	for r := range o {
		m[r] = true
	}
	return m
}

// escapeRegexRangeChars is util._escape_regex_range_chars.
func escapeRegexRangeChars(s string) string {
	for _, c := range []string{`\`, "^", "-", "[", "]"} {
		s = strings.ReplaceAll(s, c, `\`+c)
	}
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	return s
}

func escapeReRangeChar(c rune) string {
	if strings.ContainsRune(`\^-][`, c) {
		return `\` + string(c)
	}
	return string(c)
}

// collapseStringToRanges is util._collapse_string_to_ranges over a set of characters.
func collapseStringToRanges(chars runeSet, reEscape bool) string {
	esc := func(c rune) string {
		if reEscape {
			return escapeReRangeChar(c)
		}
		return string(c)
	}
	sc := chars.sorted()
	var b strings.Builder
	if len(sc) > 2 {
		i := 0
		for i < len(sc) {
			j := i
			for j+1 < len(sc) && sc[j+1]-sc[j] <= 1 {
				j++
			}
			first, last := sc[i], sc[j]
			if first == last {
				b.WriteString(esc(first))
			} else if last == first+1 {
				b.WriteString(esc(first) + esc(last))
			} else {
				b.WriteString(esc(first) + "-" + esc(last))
			}
			i = j + 1
		}
	} else {
		for _, c := range sc {
			b.WriteString(esc(c))
		}
	}
	return b.String()
}

// reEscape is re.escape.
func reEscape(s string) string { return pyre.Escape(s) }

// convertEscapedNumericsToChar is util._convert_escaped_numerics_to_char.
func convertEscapedNumericsToChar(s string) string {
	if s == "0" {
		return "\x00"
	}
	rs := []rune(s)
	allDigit := len(rs) > 0
	for _, r := range rs {
		if !isPyDigit(r) {
			allDigit = false
		}
	}
	if allDigit && len(rs) == 3 {
		n := 0
		for _, r := range rs {
			n = n*8 + int(r-'0')
		}
		return string(rune(n))
	}
	if strings.HasPrefix(s, "u") || strings.HasPrefix(s, "x") {
		n := 0
		for _, r := range rs[1:] {
			n = n*16 + hexVal(r)
		}
		return string(rune(n))
	}
	return s
}

func hexVal(r rune) int {
	switch {
	case r >= '0' && r <= '9':
		return int(r - '0')
	case r >= 'a' && r <= 'f':
		return int(r-'a') + 10
	case r >= 'A' && r <= 'F':
		return int(r-'A') + 10
	}
	return 0
}

// makeCompressedRe is util.make_compressed_re (level >= 1).
func makeCompressedRe(wordList []string, maxLevel int, nonCapturing bool, level int) (string, error) {
	if level == 1 {
		if len(wordList) == 0 {
			return "", pyErr("ValueError", "no words given to make_compressed_re()")
		}
		for _, w := range wordList {
			if w == "" {
				return "", pyErr("ValueError", "word list cannot contain empty string")
			}
		}
	} else if len(wordList) == 0 {
		return "", nil
	}
	// dedupe keeping order
	seen := map[string]bool{}
	var words []string
	for _, w := range wordList {
		if !seen[w] {
			seen[w] = true
			words = append(words, w)
		}
	}
	runeLen := func(s string) int { return len([]rune(s)) }
	sortByLenDesc := func(ss []string) {
		sort.SliceStable(ss, func(i, j int) bool { return runeLen(ss[i]) > runeLen(ss[j]) })
	}
	if maxLevel == 0 {
		anyLong := false
		for _, w := range words {
			if runeLen(w) > 1 {
				anyLong = true
			}
		}
		if anyLong {
			esc := make([]string, len(words))
			for i, w := range words {
				esc[i] = reEscape(w)
			}
			sortByLenDesc(esc)
			return strings.Join(esc, "|"), nil
		}
		var b strings.Builder
		for _, w := range words {
			b.WriteString(escapeRegexRangeChars(w))
		}
		return "[" + b.String() + "]", nil
	}
	var ret []string
	sep := ""
	ncgroup := ""
	if nonCapturing {
		ncgroup = "?:"
	}
	sorted := append([]string(nil), words...)
	sort.Strings(sorted)
	type group struct {
		prefix   string
		suffixes []string
	}
	var groups []group
	if len(sorted) > 1 {
		for i := 0; i < len(sorted); {
			p := firstRuneStr(sorted[i])
			var suf []string
			j := i
			for j < len(sorted) && firstRuneStr(sorted[j]) == p {
				suf = append(suf, string([]rune(sorted[j])[1:]))
				j++
			}
			sortByLenDesc(suf)
			groups = append(groups, group{p, suf})
			i = j
		}
	} else {
		rs := []rune(sorted[0])
		groups = append(groups, group{string(rs[0]), []string{string(rs[1:])}})
	}
	for _, g := range groups {
		ret = append(ret, sep)
		sep = "|"
		initial := reEscape(g.prefix)
		suffixes := g.suffixes
		trailing := ""
		for i, s := range suffixes {
			if s == "" {
				trailing = "?"
				suffixes = append(suffixes[:i:i], suffixes[i+1:]...)
				break
			}
		}
		allOne := func() bool {
			for _, s := range suffixes {
				if runeLen(s) != 1 {
					return false
				}
			}
			return true
		}
		if len(suffixes) > 1 {
			if allOne() {
				var b strings.Builder
				for _, s := range suffixes {
					b.WriteString(escapeRegexRangeChars(s))
				}
				ret = append(ret, initial+"["+b.String()+"]"+trailing)
			} else if level < maxLevel {
				ss := append([]string(nil), suffixes...)
				sort.Strings(ss)
				sub, err := makeCompressedRe(ss, maxLevel, nonCapturing, level+1)
				if err != nil {
					return "", err
				}
				ret = append(ret, initial+"("+ncgroup+sub+")"+trailing)
			} else {
				ss := append([]string(nil), suffixes...)
				sortByLenDesc(ss)
				esc := make([]string, len(ss))
				for i, s := range ss {
					esc[i] = reEscape(s)
				}
				ret = append(ret, initial+"("+ncgroup+strings.Join(esc, "|")+")"+trailing)
			}
		} else if len(suffixes) > 0 {
			suffix := reEscape(suffixes[0])
			if runeLen(suffix) > 1 && trailing != "" {
				ret = append(ret, initial+"("+ncgroup+suffix+")"+trailing)
			} else {
				ret = append(ret, initial+suffix+trailing)
			}
		} else {
			ret = append(ret, initial)
		}
	}
	return strings.Join(ret, ""), nil
}

func firstRuneStr(s string) string {
	for _, r := range s {
		return string(r)
	}
	return ""
}

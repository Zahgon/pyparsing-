// Port of pyparsing/core.py, the Token classes (pyparsing 3.3.3+). MIT License, Copyright (c) 2003-2025  Paul McGuire; see LICENSE.

package pyparsing

import (
	"math"
	"sort"
	"strings"

	"github.com/pyparsing/pyparsing-go/internal/pyre"
	"github.com/pyparsing/pyparsing-go/internal/pyuni"
)

func newToken(cls *pclass) *ParserElement { return newElement(cls, false) }

func (e *ParserElement) raise(st *pstate, loc int) error {
	return newPE(KindParseException, st.str(), loc, e.errmsg, e)
}

func runeAt(st *pstate, i int) (rune, bool) {
	n := len(st.runes)
	if i < 0 {
		i += n
	}
	if i < 0 || i >= n {
		return 0, false
	}
	return st.runes[i], true
}

// NoMatch returns NoMatch().
func NoMatch() *ParserElement {
	e := newToken(clsNoMatch)
	e.mayReturnEmpty = true
	e.mayIndexError = false
	e.setErrmsg("Unmatchable token")
	return e
}

// Literal returns Literal(match_string) (Empty for "", _SingleCharLiteral for one character).
func Literal(matchString string) *ParserElement {
	if matchString == "" {
		return Empty()
	}
	cls := clsLiteral
	if len([]rune(matchString)) == 1 {
		cls = clsSingleCharLit
	}
	return initLiteral(newToken(cls), matchString)
}

func initLiteral(e *ParserElement, matchString string) *ParserElement {
	e.match = []rune(matchString)
	e.matchLen = len(e.match)
	e.setErrmsg("Expected " + e.name())
	e.mayReturnEmpty = false
	e.mayIndexError = false
	return e
}

// Empty returns Empty().
func Empty() *ParserElement { return newEmptyCls(clsEmpty) }

func newEmptyCls(cls *pclass) *ParserElement {
	e := initLiteral(newToken(cls), "")
	e.mayReturnEmpty = true
	e.mayIndexError = false
	return e
}

func errorStop() *ParserElement {
	e := newEmptyCls(clsErrorStop)
	e.leaveWhitespace(true)
	return e
}

func (e *ParserElement) literalParseImpl(st *pstate, loc int) (int, any, error) {
	c, ok := runeAt(st, loc)
	if !ok {
		return 0, nil, errIndex
	}
	if len(e.match) > 0 && c == e.match[0] && hasPrefixAt(st.runes, e.match, loc) {
		return loc + e.matchLen, string(e.match), nil
	}
	return 0, nil, e.raise(st, loc)
}

func hasPrefixAt(s, p []rune, loc int) bool {
	if loc < 0 {
		loc += len(s)
		if loc < 0 {
			loc = 0
		}
	}
	if loc > len(s) || len(s)-loc < len(p) {
		return false
	}
	for i, r := range p {
		if s[loc+i] != r {
			return false
		}
	}
	return true
}

// defaultKeywordChars is Keyword.DEFAULT_KEYWORD_CHARS.
var defaultKeywordChars = Alphanums + "_$"

// Keyword returns Keyword(match_string, ident_chars, caseless).
func Keyword(matchString string, identChars *string, caseless bool) (*ParserElement, error) {
	return newKeyword(clsKeyword, matchString, identChars, caseless)
}

func newKeyword(cls *pclass, matchString string, identChars *string, caseless bool) (*ParserElement, error) {
	e := newToken(cls)
	ic := defaultKeywordChars
	if identChars != nil && *identChars != "" {
		ic = *identChars
	} else if identChars != nil {
		ic = *identChars
	}
	e.match = []rune(matchString)
	e.matchLen = len(e.match)
	if len(e.match) == 0 {
		return nil, pyErr("ValueError", "null string passed to Keyword; use Empty() instead")
	}
	e.setErrmsg("Expected " + cls.name + " " + e.name())
	e.mayReturnEmpty = false
	e.mayIndexError = false
	e.caseless = caseless
	if caseless {
		e.caselessmatch = pyuni.Upper(matchString)
		ic = pyuni.Upper(ic)
	}
	e.identChars = newRuneSet(ic)
	return e, nil
}

// CaselessKeyword returns CaselessKeyword(match_string, ident_chars).
func CaselessKeyword(matchString string, identChars *string) (*ParserElement, error) {
	return newKeyword(clsCaselessKeyword, matchString, identChars, true)
}

// inUpperSet is `c.upper() in set`.
func inUpperSet(c rune, set runeSet) bool {
	u := pyuni.UpperRune(c)
	return len(u) == 1 && set[u[0]]
}

func (e *ParserElement) keywordParseImpl(st *pstate, loc int) (int, any, error) {
	errmsg := e.errmsgOr()
	errloc := loc
	s := st.runes
	n := len(s)
	if e.caseless {
		if pyuni.Upper(string(pySlice(s, loc, loc+e.matchLen))) == e.caselessmatch {
			if loc == 0 || !inUpperSet(mustRune(st, loc-1), e.identChars) {
				if loc >= n-e.matchLen {
					return loc + e.matchLen, string(e.match), nil
				}
				c, ok := runeAt(st, loc+e.matchLen)
				if !ok {
					return 0, nil, errIndex
				}
				if !inUpperSet(c, e.identChars) {
					return loc + e.matchLen, string(e.match), nil
				}
				errmsg += ", was immediately followed by keyword character"
				errloc = loc + e.matchLen
			} else {
				errmsg += ", keyword was immediately preceded by keyword character"
				errloc = loc - 1
			}
		}
	} else {
		c, ok := runeAt(st, loc)
		if !ok {
			return 0, nil, errIndex
		}
		if (c == e.match[0] && e.matchLen == 1) || hasPrefixAt(s, e.match, loc) {
			if loc == 0 || !e.identChars[mustRune(st, loc-1)] {
				if loc >= n-e.matchLen {
					return loc + e.matchLen, string(e.match), nil
				}
				c2, ok := runeAt(st, loc+e.matchLen)
				if !ok {
					return 0, nil, errIndex
				}
				if !e.identChars[c2] {
					return loc + e.matchLen, string(e.match), nil
				}
				errmsg += ", keyword was immediately followed by keyword character"
				errloc = loc + e.matchLen
			} else {
				errmsg += ", keyword was immediately preceded by keyword character"
				errloc = loc - 1
			}
		}
	}
	return 0, nil, newParseExc(st.str(), errloc, errmsg, e)
}

func mustRune(st *pstate, i int) rune {
	c, _ := runeAt(st, i)
	return c
}

// CaselessLiteral returns CaselessLiteral(match_string).
func CaselessLiteral(matchString string) *ParserElement {
	e := initLiteral(newToken(clsCaselessLiteral), pyuni.Upper(matchString))
	e.returnString = matchString
	e.setErrmsg("Expected " + e.name())
	return e
}

func (e *ParserElement) caselessLiteralParseImpl(st *pstate, loc int) (int, any, error) {
	if pyuni.Upper(string(pySlice(st.runes, loc, loc+e.matchLen))) == string(e.match) {
		return loc + e.matchLen, e.returnString, nil
	}
	return 0, nil, e.raise(st, loc)
}

// CloseMatch returns CloseMatch(match_string, max_mismatches, caseless=caseless).
func CloseMatch(matchString string, maxMismatches *int, caseless bool) *ParserElement {
	e := newToken(clsCloseMatch)
	e.match = []rune(matchString)
	e.maxMismatches = 1
	if maxMismatches != nil {
		e.maxMismatches = *maxMismatches
	}
	e.setErrmsg("Expected " + Repr(matchString) + " (with up to " + itoa(e.maxMismatches) + " mismatches)")
	e.caseless = caseless
	e.mayIndexError = false
	e.mayReturnEmpty = false
	return e
}

func (e *ParserElement) closeMatchParseImpl(st *pstate, loc int) (int, any, error) {
	start := loc
	instrlen := len(st.runes)
	maxloc := start + len(e.match)
	if maxloc <= instrlen {
		matchStringloc := 0
		mismatches := []any{}
		var mm []int
		src := pySlice(st.runes, loc, maxloc)
		broke := false
		n := len(src)
		if len(e.match) < n {
			n = len(e.match)
		}
		for i := 0; i < n; i++ {
			matchStringloc = i
			a, b := string(src[i]), string(e.match[i])
			if e.caseless {
				a, b = pyuni.Lower(a), pyuni.Lower(b)
			}
			if a != b {
				mm = append(mm, i)
				if len(mm) > e.maxMismatches {
					broke = true
					break
				}
			}
		}
		if !broke {
			loc = start + matchStringloc + 1
			results := mustPR([]any{st.slice(start, loc)})
			results.setItemStr("original", string(e.match))
			for _, x := range mm {
				mismatches = append(mismatches, bigInt(x))
			}
			results.setItemStr("mismatches", mismatches)
			return loc, results, nil
		}
	}
	return 0, nil, e.raise(st, loc)
}

// Word returns Word(init_chars, body_chars, min, max, exact, as_keyword, exclude_chars).
func Word(initChars string, bodyChars *string, min, max, exact int, asKeyword bool, excludeChars *string) (*ParserElement, error) {
	return newWord(clsWord, initChars, bodyChars, min, max, exact, asKeyword, excludeChars)
}

func newWord(cls *pclass, initChars string, bodyCharsP *string, min, max, exact int, asKeyword bool, excludeChars *string) (*ParserElement, error) {
	e := newToken(cls)
	bodyChars := ""
	if bodyCharsP != nil {
		bodyChars = *bodyCharsP
	}
	if initChars == "" {
		return nil, pyErr("ValueError", "invalid "+cls.name+", initChars cannot be empty string")
	}
	initSet := newRuneSet(initChars)
	if excludeChars != nil && *excludeChars != "" {
		ex := newRuneSet(*excludeChars)
		for r := range ex {
			delete(initSet, r)
		}
		if bodyChars != "" {
			bs := newRuneSet(bodyChars)
			for r := range ex {
				delete(bs, r)
			}
			bodyChars = string(bs.sorted())
		}
	}
	e.initChars = initSet
	e.initCharsOrig = string(initSet.sorted())
	if bodyChars != "" {
		e.bodyChars = newRuneSet(bodyChars)
		rs := []rune(bodyChars)
		sort.Slice(rs, func(i, j int) bool { return rs[i] < rs[j] })
		e.bodyCharsOrig = string(rs)
	} else {
		e.bodyChars = initSet
		e.bodyCharsOrig = e.initCharsOrig
	}
	maxSpecified := max > 0
	if min < 1 {
		return nil, pyErr("ValueError", "cannot specify a minimum length < 1; use Opt(Word()) if zero-length word is permitted")
	}
	if maxSpecified && min > max {
		return nil, pyErr("ValueError", "invalid args, if min and max both specified min must be <= max (min="+itoa(min)+", max="+itoa(max)+")")
	}
	e.minLen = min
	if max > 0 {
		e.maxLen = max
	} else {
		e.maxLen = math.MaxInt
	}
	if exact > 0 {
		min, max = exact, exact
		e.maxLen = exact
		e.minLen = exact
	}
	e.setErrmsg("Expected " + e.name())
	e.mayIndexError = false
	e.asKeyword = asKeyword
	if asKeyword {
		e.setErrmsg(*e.errmsg + " as a keyword")
	}
	if !e.initChars[' '] && !e.bodyChars[' '] {
		var lead string
		if len(e.initChars) == 1 {
			lead = reEscape(e.initCharsOrig)
		} else {
			lead = "[" + collapseStringToRanges(e.initChars, true) + "]"
		}
		if e.bodyChars.equal(e.initChars) {
			var repeat string
			switch {
			case max == 0 && e.minLen == 1:
				repeat = "+"
			case max == 1:
				repeat = ""
			case e.minLen != e.maxLen:
				mx := ""
				if e.maxLen != math.MaxInt {
					mx = itoa(e.maxLen)
				}
				repeat = "{" + itoa(e.minLen) + "," + mx + "}"
			default:
				repeat = "{" + itoa(e.minLen) + "}"
			}
			e.reString = lead + repeat
		} else {
			var body, repeat string
			if max == 1 {
				body, repeat = "", ""
			} else {
				body = "[" + collapseStringToRanges(e.bodyChars, true) + "]"
				opt := func(v int) string {
					if v > 0 {
						return itoa(v)
					}
					return ""
				}
				switch {
				case max == 0 && e.minLen == 1:
					repeat = "*"
				case max == 2:
					if min <= 1 {
						repeat = "?"
					} else {
						repeat = ""
					}
				case min != max:
					repeat = "{" + opt(min-1) + "," + opt(max-1) + "}"
				default:
					repeat = "{" + opt(min-1) + "}"
				}
			}
			e.reString = lead + body + repeat
		}
		if e.asKeyword {
			e.reString = `\b` + e.reString + `\b`
		}
		if p, err := pyre.Compile(e.reString, 0); err == nil {
			e.re = p
			e.useRe = true
		}
	}
	return e, nil
}

func (e *ParserElement) wordParseImpl(st *pstate, loc int) (int, any, error) {
	if e.useRe {
		m := e.re.Match(st.runes, loc)
		if m == nil {
			return 0, nil, e.raise(st, loc)
		}
		g, _ := m.Group(0)
		return m.End(0), g, nil
	}
	c, ok := runeAt(st, loc)
	if !ok {
		return 0, nil, errIndex
	}
	if !e.initChars[c] {
		return 0, nil, e.raise(st, loc)
	}
	start := loc
	loc++
	instrlen := len(st.runes)
	maxloc := start + e.maxLen
	if e.maxLen == math.MaxInt || maxloc > instrlen {
		maxloc = instrlen
	}
	for loc < maxloc && e.bodyChars[st.runes[loc]] {
		loc++
	}
	throw := false
	if loc-start < e.minLen {
		throw = true
	} else if e.asKeyword {
		if start > 0 && e.bodyChars[st.runes[start-1]] {
			throw = true
		} else if loc < instrlen && e.bodyChars[st.runes[loc]] {
			throw = true
		}
	}
	if throw {
		return 0, nil, e.raise(st, loc)
	}
	return loc, st.slice(start, loc), nil
}

func charsAsStr(s runeSet) string {
	const maxReprLen = 16
	str := []rune(collapseStringToRanges(s, false))
	if len(str) > maxReprLen {
		return string(str[:maxReprLen-3]) + "..."
	}
	return string(str)
}

func (e *ParserElement) wordDefaultName() string {
	var base string
	if !e.initChars.equal(e.bodyChars) {
		base = "W:(" + charsAsStr(e.initChars) + ", " + charsAsStr(e.bodyChars) + ")"
	} else {
		base = "W:(" + charsAsStr(e.initChars) + ")"
	}
	if e.minLen > 1 || e.maxLen != math.MaxInt {
		if e.minLen == e.maxLen {
			if e.minLen == 1 {
				return base[2:]
			}
			return base + "{" + itoa(e.minLen) + "}"
		} else if e.maxLen == math.MaxInt {
			return base + "{" + itoa(e.minLen) + ",...}"
		}
		return base + "{" + itoa(e.minLen) + "," + itoa(e.maxLen) + "}"
	}
	return base
}

// Char returns Char(charset, as_keyword, exclude_chars).
func Char(charset string, asKeyword bool, excludeChars *string) (*ParserElement, error) {
	return newWord(clsChar, charset, nil, 1, 0, 1, asKeyword, excludeChars)
}

// Regex returns Regex(pattern, flags, as_group_list, as_match). The pattern is compiled
// here: an invalid one gives ValueError "invalid pattern (...) passed to Regex".
func Regex(pattern string, flags int, asGroupList, asMatch bool) (*ParserElement, error) {
	e := newToken(clsRegex)
	if pattern == "" {
		return nil, pyErr("ValueError", "null string passed to Regex; use Empty() instead")
	}
	e.reString = pattern
	e.pattern = pattern
	e.flags = flags
	e.setErrmsg("Expected " + e.name())
	e.mayIndexError = false
	e.asGroupList = asGroupList
	e.asMatch = asMatch
	p, err := pyre.Compile(pattern, flags)
	if err != nil {
		if _, ok := err.(*pyre.Error); ok {
			return nil, pyErr("ValueError", "invalid pattern ("+Repr(pattern)+") passed to Regex")
		}
		if pe, ok := err.(*pyre.PyError); ok {
			return nil, pyErr(pe.Class, pe.Msg)
		}
		return nil, err
	}
	e.re = p
	e.mayReturnEmpty = p.Match([]rune{}, 0) != nil
	return e, nil
}

func mustRegex(pattern string, flags int) *ParserElement {
	e, err := Regex(pattern, flags, false, false)
	if err != nil {
		panic("pyparsing: invalid builtin regex " + pattern)
	}
	return e
}

func (e *ParserElement) regexParseImpl(st *pstate, loc int) (int, any, error) {
	if loc > len(st.runes) && e.mayReturnEmpty {
		return 0, nil, e.raise(st, loc)
	}
	m := e.re.Match(st.runes, loc)
	if m == nil {
		return 0, nil, e.raise(st, loc)
	}
	end := m.End(0)
	if e.asMatch {
		return end, &RegexMatch{m: m}, nil
	}
	if e.asGroupList {
		gs := m.Groups()
		t := make(PyTuple, len(gs))
		for i, g := range gs {
			if g != nil {
				t[i] = *g
			}
		}
		return end, t, nil
	}
	g0, _ := m.Group(0)
	ret := mustPR(g0)
	for _, ng := range m.GroupDict() {
		if ng.Value == nil {
			ret.setItemStr(ng.Name, nil)
		} else {
			ret.setItemStr(ng.Name, *ng.Value)
		}
	}
	return end, ret, nil
}

// RegexSub is Regex.sub(repl): a parse action that runs re.sub (or Match.expand with
// as_match) on the matched text.
func (e *ParserElement) RegexSub(repl string) (*ParserElement, error) {
	if e == nil {
		return nil, pyErr("AttributeError", "'NoneType' object has no attribute 'sub'")
	}
	if e.cls != clsRegex {
		return nil, pyErr("AttributeError", "'"+e.cls.name+"' object has no attribute 'sub'")
	}
	if e.asGroupList {
		return nil, pyErr("TypeError", "cannot use sub() with Regex(as_group_list=True)")
	}
	var pa ParseAction
	if e.asMatch {
		pa = func(s string, l int, t *ParseResults) (any, error) {
			v, err := t.At(0)
			if err != nil {
				return nil, err
			}
			m, ok := v.(*RegexMatch)
			if !ok {
				return nil, pyErr("AttributeError", "'"+pyTypeName(v)+"' object has no attribute 'expand'")
			}
			r, err := m.m.Expand(repl)
			return r, fromInternal(err)
		}
	} else {
		re := e.re
		pa = func(s string, l int, t *ParseResults) (any, error) {
			v, err := t.At(0)
			if err != nil {
				return nil, err
			}
			str, ok := v.(string)
			if !ok {
				return nil, pyErr("TypeError", "expected string or bytes-like object, got '"+pyTypeName(v)+"'")
			}
			r, err := re.Sub(repl, []rune(str), 0)
			return r, fromInternal(err)
		}
	}
	return e.addPA(pa), nil
}

// ---------------------------------------------------------------- QuotedString

type quotedData struct {
	quoteChar, endQuoteChar []rune
	escChar                 string
	hasEscChar              bool
	escQuote                string
	unquoteResults          bool
	convertWS               bool
	multiline               bool
	unquoteScanRe           *pyre.Pattern
}

var wsMapKeys = []string{`\t`, `\n`, `\f`, `\r`}
var wsMap = map[string]string{`\t`: "\t", `\n`: "\n", `\f`: "\x0c", `\r`: "\r"}

// QuotedString returns QuotedString(quote_char, esc_char, esc_quote, multiline,
// unquote_results, end_quote_char, convert_whitespace_escapes).
func QuotedString(quoteChar string, escChar, escQuote *string, multiline, unquoteResults bool, endQuoteChar *string, convertWhitespaceEscapes bool) (*ParserElement, error) {
	e := newToken(clsQuotedString)
	if pyStrip(quoteChar) == "" {
		return nil, pyErr("ValueError", "quote_char cannot be the empty string")
	}
	eq := quoteChar
	if endQuoteChar != nil && *endQuoteChar != "" {
		eq = *endQuoteChar
		if pyStrip(eq) == "" {
			return nil, pyErr("ValueError", "end_quote_char cannot be the empty string")
		}
	} else if endQuoteChar != nil {
		return nil, pyErr("ValueError", "end_quote_char cannot be the empty string")
	}
	q := &quotedData{quoteChar: []rune(quoteChar), endQuoteChar: []rune(eq), multiline: multiline,
		unquoteResults: unquoteResults, convertWS: convertWhitespaceEscapes}
	esc := ""
	if escChar != nil {
		esc = *escChar
	}
	q.escChar = esc
	q.hasEscChar = escChar != nil && *escChar != ""
	if escQuote != nil {
		q.escQuote = *escQuote
	}
	e.qs = q
	reFlags := 0
	var inner []string
	if q.escQuote != "" {
		inner = append(inner, "(?:"+reEscape(q.escQuote)+")")
	}
	if esc != "" {
		inner = append(inner, "(?:"+reEscape(esc)+".)")
	}
	eqr := q.endQuoteChar
	if len(eqr) > 1 {
		var parts []string
		for i := len(eqr) - 1; i > 0; i-- {
			parts = append(parts, "(?:"+reEscape(string(eqr[:i]))+"(?!"+reEscape(string(eqr[i:]))+"))")
		}
		inner = append(inner, "(?:"+strings.Join(parts, "|")+")")
	}
	escPart := ""
	if q.hasEscChar {
		escPart = escapeRegexRangeChars(esc)
	}
	if multiline {
		reFlags |= MULTILINE | DOTALL
		inner = append(inner, "(?:[^"+escapeRegexRangeChars(string(eqr[0]))+escPart+"])")
	} else {
		inner = append(inner, "(?:[^"+escapeRegexRangeChars(string(eqr[0]))+`\n\r`+escPart+"])")
	}
	e.pattern = reEscape(quoteChar) + "(?:" + strings.Join(inner, "|") + ")*" + reEscape(eq)
	if unquoteResults {
		var err error
		if convertWhitespaceEscapes {
			// The original writes this pattern as an rf-string, so its "{3}", "{2}" and "{4}"
			// are replacement fields: the digits 3, 2 and 4, not repeat counts.
			ks := make([]string, len(wsMapKeys))
			for i, k := range wsMapKeys {
				ks[i] = reEscape(k)
			}
			q.unquoteScanRe, err = pyre.Compile("("+strings.Join(ks, "|")+`)|(\\[0-7]3|\\0|\\x[0-9a-fA-F]2|\\u[0-9a-fA-F]4)|(`+reEscape(esc)+`.)|(\n|.)`, reFlags)
		} else {
			q.unquoteScanRe, err = pyre.Compile("("+reEscape(esc)+`.)|(\n|.)`, reFlags)
		}
		if err != nil {
			return nil, fromInternal(err)
		}
	}
	p, err := pyre.Compile(e.pattern, reFlags)
	if err != nil {
		return nil, pyErr("ValueError", "invalid pattern "+Repr(e.pattern)+" passed to Regex")
	}
	e.re = p
	e.reString = e.pattern
	e.setErrmsg("Expected " + e.name())
	e.mayIndexError = false
	e.mayReturnEmpty = true
	return e, nil
}

func (e *ParserElement) quotedDefaultName() string {
	q := e.qs
	if string(q.quoteChar) == string(q.endQuoteChar) {
		return "string enclosed in " + Repr(string(q.quoteChar))
	}
	return "quoted string, starting with " + string(q.quoteChar) + " ending with " + string(q.endQuoteChar)
}

func (e *ParserElement) quotedParseImpl(st *pstate, loc int) (int, any, error) {
	q := e.qs
	c, ok := runeAt(st, loc)
	if !ok {
		return 0, nil, errIndex
	}
	var m *pyre.Match
	if c == q.quoteChar[0] {
		m = e.re.Match(st.runes, loc)
	}
	if m == nil {
		return 0, nil, e.raise(st, loc)
	}
	loc = m.End(0)
	ret, _ := m.Group(0)
	if q.unquoteResults {
		rs := []rune(ret)
		inner := pySlice(rs, len(q.quoteChar), len(rs)-len(q.endQuoteChar))
		var b strings.Builder
		for _, mm := range q.unquoteScanRe.FindIter(inner, 0, len(inner)) {
			if q.convertWS {
				if g, ok := mm.Group(1); ok && g != "" {
					b.WriteString(wsMap[g])
				} else if g, ok := mm.Group(2); ok && g != "" {
					b.WriteString(convertEscapedNumericsToChar(string([]rune(g)[1:])))
				} else if g, ok := mm.Group(3); ok && g != "" {
					gr := []rune(g)
					b.WriteRune(gr[len(gr)-1])
				} else {
					g4, _ := mm.Group(4)
					b.WriteString(g4)
				}
			} else {
				if g, ok := mm.Group(1); ok && g != "" {
					gr := []rune(g)
					b.WriteRune(gr[len(gr)-1])
				} else {
					g2, _ := mm.Group(2)
					b.WriteString(g2)
				}
			}
		}
		ret = b.String()
		if q.escQuote != "" {
			ret = strings.ReplaceAll(ret, q.escQuote, string(q.endQuoteChar))
		}
	}
	return loc, ret, nil
}

// ---------------------------------------------------------------- CharsNotIn, White

// CharsNotIn returns CharsNotIn(not_chars, min, max, exact).
func CharsNotIn(notChars string, min, max, exact int) (*ParserElement, error) {
	e := newToken(clsCharsNotIn)
	e.skipWhitespace = false
	e.notChars = notChars
	e.notCharsSet = newRuneSet(notChars)
	if min < 1 {
		return nil, pyErr("ValueError", "cannot specify a minimum length < 1; use Opt(CharsNotIn()) if zero-length char group is permitted")
	}
	e.minLen = min
	if max > 0 {
		e.maxLen = max
	} else {
		e.maxLen = math.MaxInt
	}
	if exact > 0 {
		e.maxLen = exact
		e.minLen = exact
	}
	e.setErrmsg("Expected " + e.name())
	e.mayReturnEmpty = e.minLen == 0
	e.mayIndexError = false
	return e, nil
}

func mustCharsNotIn(notChars string, exact int) *ParserElement {
	e, _ := CharsNotIn(notChars, 1, 0, exact)
	return e
}

func (e *ParserElement) charsNotInDefaultName() string {
	s := []rune(collapseStringToRanges(e.notCharsSet, true))
	if len(s) > 16 {
		return "!W:(" + string(pySlice([]rune(e.notChars), 0, 13)) + "...)"
	}
	return "!W:(" + e.notChars + ")"
}

func (e *ParserElement) charsNotInParseImpl(st *pstate, loc int) (int, any, error) {
	c, ok := runeAt(st, loc)
	if !ok {
		return 0, nil, errIndex
	}
	if e.notCharsSet[c] {
		return 0, nil, e.raise(st, loc)
	}
	start := loc
	loc++
	maxlen := len(st.runes)
	if e.maxLen != math.MaxInt && start+e.maxLen < maxlen {
		maxlen = start + e.maxLen
	}
	for loc < maxlen && !e.notCharsSet[st.runes[loc]] {
		loc++
	}
	if loc-start < e.minLen {
		return 0, nil, e.raise(st, loc)
	}
	return loc, st.slice(start, loc), nil
}

var whiteStrsKeys = []rune{' ', '\t', '\n', '\r', '\x0c', ' ', ' ', '᠎', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', '​', ' ', ' ', '　'}
var whiteStrs = map[rune]string{' ': "<SP>", '\t': "<TAB>", '\n': "<LF>", '\r': "<CR>", '\x0c': "<FF>", ' ': "<NBSP>", ' ': "<OGHAM_SPACE_MARK>", '᠎': "<MONGOLIAN_VOWEL_SEPARATOR>", ' ': "<EN_QUAD>", ' ': "<EM_QUAD>", ' ': "<EN_SPACE>", ' ': "<EM_SPACE>", ' ': "<THREE-PER-EM_SPACE>", ' ': "<FOUR-PER-EM_SPACE>", ' ': "<SIX-PER-EM_SPACE>", ' ': "<FIGURE_SPACE>", ' ': "<PUNCTUATION_SPACE>", ' ': "<THIN_SPACE>", ' ': "<HAIR_SPACE>", '​': "<ZERO_WIDTH_SPACE>", ' ': "<NNBSP>", ' ': "<MMSP>", '　': "<IDEOGRAPHIC_SPACE>"}

// White returns White(ws, min, max, exact). A character of ws that has no entry in
// White.whiteStrs makes the element's name raise KeyError in Python; here it is named "".
func White(ws string, min, max, exact int) *ParserElement {
	e := newToken(clsWhite)
	e.matchWhite = []rune(ws)
	var other []rune
	mw := newRuneSet(ws)
	for _, c := range whiteStrsKeys {
		if !mw[c] {
			other = append(other, c)
		}
	}
	e.SetWhitespaceChars(string(other), true)
	e.mayReturnEmpty = true
	e.setErrmsg("Expected " + e.name())
	e.minLen = min
	if max > 0 {
		e.maxLen = max
	} else {
		e.maxLen = math.MaxInt
	}
	if exact > 0 {
		e.maxLen = exact
		e.minLen = exact
	}
	return e
}

func (e *ParserElement) whiteDefaultName() string {
	var b strings.Builder
	for _, c := range e.matchWhite {
		b.WriteString(whiteStrs[c])
	}
	return b.String()
}

func (e *ParserElement) inMatchWhite(c rune) bool {
	for _, x := range e.matchWhite {
		if x == c {
			return true
		}
	}
	return false
}

func (e *ParserElement) whiteParseImpl(st *pstate, loc int) (int, any, error) {
	c, ok := runeAt(st, loc)
	if !ok {
		return 0, nil, errIndex
	}
	if !e.inMatchWhite(c) {
		return 0, nil, e.raise(st, loc)
	}
	start := loc
	loc++
	maxloc := len(st.runes)
	if e.maxLen != math.MaxInt && start+e.maxLen < maxloc {
		maxloc = start + e.maxLen
	}
	for loc < maxloc && e.inMatchWhite(st.runes[loc]) {
		loc++
	}
	if loc-start < e.minLen {
		return 0, nil, e.raise(st, loc)
	}
	return loc, st.slice(start, loc), nil
}

// ---------------------------------------------------------------- position tokens

func newPositionToken(cls *pclass) *ParserElement {
	e := newToken(cls)
	e.mayReturnEmpty = true
	e.mayIndexError = false
	return e
}

// GoToColumn returns GoToColumn(colno).
func GoToColumn(colno int) *ParserElement {
	e := newPositionToken(clsGoToColumn)
	e.col = colno
	return e
}

func (e *ParserElement) goToColumnPreParse(st *pstate, loc int) (int, error) {
	if colRunes(loc, st.runes) == e.col {
		return loc, nil
	}
	instrlen := len(st.runes)
	if len(e.ignoreExprs.items) > 0 {
		var err error
		loc, err = e.skipIgnorables(st, loc)
		if err != nil {
			return 0, err
		}
	}
	for loc < instrlen && isPySpace(st.runes[loc]) && colRunes(loc, st.runes) != e.col {
		loc++
	}
	return loc, nil
}

func (e *ParserElement) goToColumnParseImpl(st *pstate, loc int) (int, any, error) {
	thiscol := colRunes(loc, st.runes)
	if thiscol > e.col {
		return 0, nil, newParseExc(st.str(), loc, "Text not in expected column", e)
	}
	newloc := loc + e.col - thiscol
	return newloc, st.slice(loc, newloc), nil
}

// LineStart returns LineStart().
func LineStart() *ParserElement {
	e := newPositionToken(clsLineStart)
	e.leaveWhitespace(true)
	e.origWhiteChars = e.whiteChars.copySet()
	ws := e.whiteChars.copySet()
	delete(ws, '\n')
	e.whiteChars = ws
	e.skipper = Empty().setWhitespaceSet(e.whiteChars, false)
	e.SetName("start of line")
	return e
}

func (e *ParserElement) lineStartPreParse(st *pstate, loc int) (int, error) {
	if loc == 0 {
		return loc, nil
	}
	ret, err := e.skipper.preParse(st, loc)
	if err != nil {
		return 0, err
	}
	if e.origWhiteChars['\n'] {
		for string(pySlice(st.runes, ret, ret+1)) == "\n" {
			ret, err = e.skipper.preParse(st, ret+1)
			if err != nil {
				return 0, err
			}
		}
	}
	return ret, nil
}

// LineEnd returns LineEnd().
func LineEnd() *ParserElement {
	e := newPositionToken(clsLineEnd)
	ws := e.whiteChars.copySet()
	delete(ws, '\n')
	e.setWhitespaceSet(ws, false)
	e.SetName("end of line")
	return e
}

func (e *ParserElement) lineEndParseImpl(st *pstate, loc int) (int, any, error) {
	n := len(st.runes)
	if loc < n {
		if st.runes[loc] == '\n' {
			return loc + 1, "\n", nil
		}
		return 0, nil, e.raise(st, loc)
	} else if loc == n {
		return loc + 1, []any{}, nil
	}
	return 0, nil, e.raise(st, loc)
}

// StringStart returns StringStart().
func StringStart() *ParserElement {
	e := newPositionToken(clsStringStart)
	e.SetName("start of text")
	return e
}

func (e *ParserElement) stringStartParseImpl(st *pstate, loc int) (int, any, error) {
	if loc != 0 {
		p, err := e.preParse(st, 0)
		if err != nil {
			return 0, nil, err
		}
		if loc != p {
			return 0, nil, e.raise(st, loc)
		}
	}
	return loc, []any{}, nil
}

// StringEnd returns StringEnd().
func StringEnd() *ParserElement {
	e := newPositionToken(clsStringEnd)
	e.SetName("end of text")
	return e
}

func (e *ParserElement) stringEndParseImpl(st *pstate, loc int) (int, any, error) {
	n := len(st.runes)
	if loc < n {
		return 0, nil, e.raise(st, loc)
	}
	if loc == n {
		return loc + 1, []any{}, nil
	}
	return loc, []any{}, nil
}

// WordStart returns WordStart(word_chars).
func WordStart(wordChars string) *ParserElement {
	e := newPositionToken(clsWordStart)
	e.wordChars = newRuneSet(wordChars)
	e.SetName("start of a word")
	return e
}

func (e *ParserElement) wordStartParseImpl(st *pstate, loc int) (int, any, error) {
	if loc != 0 {
		p, ok := runeAt(st, loc-1)
		if !ok {
			return 0, nil, errIndex
		}
		if e.wordChars[p] {
			return 0, nil, e.raise(st, loc)
		}
		c, ok := runeAt(st, loc)
		if !ok {
			return 0, nil, errIndex
		}
		if !e.wordChars[c] {
			return 0, nil, e.raise(st, loc)
		}
	}
	return loc, []any{}, nil
}

// WordEnd returns WordEnd(word_chars).
func WordEnd(wordChars string) *ParserElement {
	e := newPositionToken(clsWordEnd)
	e.wordChars = newRuneSet(wordChars)
	e.skipWhitespace = false
	e.SetName("end of a word")
	return e
}

func (e *ParserElement) wordEndParseImpl(st *pstate, loc int) (int, any, error) {
	instrlen := len(st.runes)
	if instrlen > 0 && loc < instrlen {
		c, ok := runeAt(st, loc)
		if !ok {
			return 0, nil, errIndex
		}
		if e.wordChars[c] {
			return 0, nil, e.raise(st, loc)
		}
		p, ok := runeAt(st, loc-1)
		if !ok {
			return 0, nil, errIndex
		}
		if !e.wordChars[p] {
			return 0, nil, e.raise(st, loc)
		}
	}
	return loc, []any{}, nil
}

// Tag returns Tag(tag_name, value).
func Tag(tagName string, value any) *ParserElement {
	e := newToken(clsTag)
	e.mayReturnEmpty = true
	e.mayIndexError = false
	e.leaveWhitespace(true)
	e.tagName = tagName
	e.tagValue = value
	e.addPA(func(s string, l int, t *ParseResults) (any, error) {
		t.setItemStr(e.tagName, e.tagValue)
		return nil, nil
	})
	return e
}

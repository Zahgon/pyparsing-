// Port of pyparsing/core.py, ParserElement parsing machinery (pyparsing 3.3.3+). MIT License, Copyright (c) 2003-2025  Paul McGuire; see LICENSE.

package pyparsing

import (
	"math"

	"github.com/pyparsing/pyparsing-go/internal/pyval"
)

// pstate is one Python input string: its code points, its str and its identity for the cache.
type pstate struct {
	runes []rune
	s     string
	hasS  bool
	id    int
}

var (
	internIDs  = map[string]int{}
	internNext = 1
)

func newState(s string) *pstate {
	return newStateRunes([]rune(s), s, true)
}

func newStateRunes(rs []rune, s string, hasS bool) *pstate {
	st := &pstate{runes: rs, s: s, hasS: hasS}
	str := st.str()
	if id, ok := internIDs[str]; ok {
		st.id = id
	} else {
		st.id = internNext
		internIDs[str] = internNext
		internNext++
	}
	return st
}

func (st *pstate) str() string {
	if !st.hasS {
		st.s = string(st.runes)
		st.hasS = true
	}
	return st.s
}

func (st *pstate) slice(a, b int) string { return string(pySlice(st.runes, a, b)) }

var errIndex = &PyError{Class: "IndexError", Msg: "string index out of range"}

func isIndexError(err error) bool {
	pe, ok := err.(*PyError)
	return ok && pe.Class == "IndexError"
}

// asPE returns the error as a parse exception of the given test.
func asParseException(err error) (*ParseException, bool) {
	pe, ok := err.(*ParseException)
	return pe, ok && pe.Kind == KindParseException
}

func asFatal(err error) (*ParseException, bool) {
	pe, ok := err.(*ParseException)
	return pe, ok && pe.Kind != KindParseException
}

func asBase(err error) (*ParseException, bool) {
	pe, ok := err.(*ParseException)
	return pe, ok
}

// ---------------------------------------------------------------- _parse

// maxParseDepth bounds the nesting of _parse calls; deeper recursion (an unbounded left
// recursion, say) reports Python's RecursionError instead of exhausting the Go stack.
const maxParseDepth = 20000

var parseDepth int

func (e *ParserElement) parse(st *pstate, loc int, doActions, callPreParse bool) (int, *ParseResults, error) {
	if parseDepth >= maxParseDepth {
		return 0, nil, &PyError{Class: "RecursionError", Msg: "maximum recursion depth exceeded"}
	}
	parseDepth++
	defer func() { parseDepth-- }()
	if packratEnabled {
		return e.parseCache(st, loc, doActions, callPreParse)
	}
	return e.parseNoCache(st, loc, doActions, callPreParse)
}

func (e *ParserElement) parseNoCache(st *pstate, loc int, doActions, callPreParse bool) (int, *ParseResults, error) {
	lenS := len(st.runes)
	preLoc := loc
	var err error
	if callPreParse && e.callPreparse {
		preLoc, err = e.preParse(st, loc)
		if err != nil {
			return 0, nil, err
		}
	}
	tokensStart := preLoc
	var toks any
	if e.mayIndexError || preLoc >= lenS {
		loc, toks, err = e.parseImpl(st, preLoc, doActions)
		if err != nil && isIndexError(err) {
			err = newPE(KindParseException, st.str(), lenS, e.errmsg, e)
		}
	} else {
		loc, toks, err = e.parseImpl(st, preLoc, doActions)
	}
	if err != nil {
		return 0, nil, err
	}
	tokens, err := e.postParse(st, loc, toks)
	if err != nil {
		return 0, nil, err
	}
	retTokens, err := newPR(tokens, e.resultsName, e.saveAsList, e.modalResults)
	if err != nil {
		return 0, nil, err
	}
	if len(e.parseAction) > 0 && (doActions || e.callDuringTry) {
		s := st.str()
		for _, fn := range e.parseAction {
			t, err := fn.fn(s, tokensStart, retTokens)
			if err != nil {
				if isIndexError(err) {
					return 0, nil, newPE(KindParseException, "exception raised in parse action", 0, nil, nil)
				}
				return 0, nil, err
			}
			if t == nil {
				continue
			}
			if pr, ok := t.(*ParseResults); ok && pr == retTokens {
				continue
			}
			var nr *ParseResults
			switch tt := t.(type) {
			case *ParseResults, []any, prList:
				nr, err = newPR(tt, e.resultsName, e.saveAsList, e.modalResults)
			case PyTuple:
				if len(tt) == 0 {
					nr, err = newPR(tt, e.resultsName, false, e.modalResults)
				} else {
					nr, err = newPR([]any{tt}, e.resultsName, false, e.modalResults)
				}
			default:
				nr, err = newPR([]any{t}, e.resultsName, false, e.modalResults)
			}
			if err != nil {
				return 0, nil, err
			}
			retTokens = nr
		}
	}
	return loc, retTokens, nil
}

// tryParse is try_parse(instring, loc, raise_fatal, do_actions).
func (e *ParserElement) tryParse(st *pstate, loc int, raiseFatal, doActions bool) (int, error) {
	l, _, err := e.parse(st, loc, doActions, true)
	if err != nil {
		if _, ok := asFatal(err); ok {
			if raiseFatal {
				return 0, err
			}
			return 0, newPE(KindParseException, st.str(), loc, e.errmsg, e)
		}
		return 0, err
	}
	return l, nil
}

// canParseNext is can_parse_next.
func (e *ParserElement) canParseNext(st *pstate, loc int, doActions bool) (bool, error) {
	_, err := e.tryParse(st, loc, false, doActions)
	if err != nil {
		if _, ok := asParseException(err); ok || isIndexError(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// ---------------------------------------------------------------- packrat

type cacheKey struct {
	elem         *ParserElement
	sid          int
	loc          int
	callPreParse bool
	doActions    bool
}

type cacheVal struct {
	exc    *ParseException
	loc    int
	result *ParseResults
}

type packratCache struct {
	size    int // < 0: unbounded
	m       map[cacheKey]*cacheVal
	order   []cacheKey
	ordHead int
}

func (c *packratCache) get(k cacheKey) (*cacheVal, bool) {
	v, ok := c.m[k]
	return v, ok
}

func (c *packratCache) set(k cacheKey, v *cacheVal) {
	if _, ok := c.m[k]; !ok && c.size >= 0 {
		c.order = append(c.order, k)
	}
	c.m[k] = v
	if c.size >= 0 {
		for len(c.m) > c.size {
			for c.ordHead < len(c.order) {
				ok := c.order[c.ordHead]
				c.ordHead++
				if _, present := c.m[ok]; present {
					delete(c.m, ok)
					break
				}
			}
		}
		if c.ordHead > 1024 && c.ordHead*2 > len(c.order) {
			c.order = append([]cacheKey(nil), c.order[c.ordHead:]...)
			c.ordHead = 0
		}
	}
}

func (c *packratCache) clear() {
	c.m = map[cacheKey]*cacheVal{}
	c.order = nil
	c.ordHead = 0
}

var theCache = &packratCache{size: -1, m: map[cacheKey]*cacheVal{}}

func (e *ParserElement) parseCache(st *pstate, loc int, doActions, callPreParse bool) (int, *ParseResults, error) {
	key := cacheKey{e, st.id, loc, callPreParse, doActions}
	if v, ok := theCache.get(key); ok {
		if v.exc != nil {
			return 0, nil, v.exc
		}
		return v.loc, v.result.Copy(), nil
	}
	l, res, err := e.parseNoCache(st, loc, doActions, callPreParse)
	if err != nil {
		if pe, ok := asBase(err); ok {
			theCache.set(key, &cacheVal{exc: pe.fromArgs()})
		}
		return 0, nil, err
	}
	theCache.set(key, &cacheVal{loc: l, result: res.Copy()})
	return l, res, nil
}

// ---------------------------------------------------------------- left recursion memo

type memoKey struct {
	loc       int
	elem      *ParserElement
	doActions bool
}

type memoVal struct {
	loc    int
	result *ParseResults
	exc    *ParseException
}

type recursionMemo interface {
	get(k memoKey) (*memoVal, bool)
	set(k memoKey, v *memoVal)
	del(k memoKey)
	clear()
}

// unboundedMemo is util.UnboundedMemo (deletion is a no-op).
type unboundedMemo struct{ m map[memoKey]*memoVal }

func (u *unboundedMemo) get(k memoKey) (*memoVal, bool) { v, ok := u.m[k]; return v, ok }
func (u *unboundedMemo) set(k memoKey, v *memoVal)      { u.m[k] = v }
func (u *unboundedMemo) del(k memoKey)                  {}
func (u *unboundedMemo) clear()                         { u.m = map[memoKey]*memoVal{} }

// orderedMemo is an insertion-ordered dict.
type orderedMemo struct {
	keys []memoKey
	m    map[memoKey]*memoVal
}

func newOrderedMemo() *orderedMemo { return &orderedMemo{m: map[memoKey]*memoVal{}} }

func (o *orderedMemo) pop(k memoKey) (*memoVal, bool) {
	v, ok := o.m[k]
	if !ok {
		return nil, false
	}
	delete(o.m, k)
	for i, kk := range o.keys {
		if kk == k {
			o.keys = append(o.keys[:i:i], o.keys[i+1:]...)
			break
		}
	}
	return v, true
}

func (o *orderedMemo) put(k memoKey, v *memoVal) {
	if _, ok := o.m[k]; !ok {
		o.keys = append(o.keys, k)
	}
	o.m[k] = v
}

// lruMemo is util.LRUMemo.
type lruMemo struct {
	capacity int
	active   map[memoKey]*memoVal
	memory   *orderedMemo
}

func (l *lruMemo) get(k memoKey) (*memoVal, bool) {
	if v, ok := l.active[k]; ok {
		return v, true
	}
	v, ok := l.memory.pop(k)
	if !ok {
		return nil, false
	}
	l.memory.put(k, v)
	return v, true
}

func (l *lruMemo) set(k memoKey, v *memoVal) {
	l.memory.pop(k)
	l.active[k] = v
}

func (l *lruMemo) del(k memoKey) {
	v, ok := l.active[k]
	if !ok {
		return
	}
	delete(l.active, k)
	n := len(l.memory.keys) - (l.capacity + 1)
	if n > 0 {
		old := append([]memoKey(nil), l.memory.keys[:n]...)
		for _, ok := range old {
			l.memory.pop(ok)
		}
	}
	l.memory.put(k, v)
}

func (l *lruMemo) clear() {
	l.active = map[memoKey]*memoVal{}
	l.memory = newOrderedMemo()
}

var recursionMemos recursionMemo = &unboundedMemo{m: map[memoKey]*memoVal{}}

// ResetCache is ParserElement.reset_cache.
func ResetCache() {
	theCache.clear()
	recursionMemos.clear()
	if len(internIDs) > 4096 {
		internIDs = map[string]int{}
	}
}

// DisableMemoization is ParserElement.disable_memoization.
func DisableMemoization() {
	ResetCache()
	leftRecursion = false
	packratEnabled = false
}

// EnableLeftRecursion is ParserElement.enable_left_recursion.
func EnableLeftRecursion(cacheSizeLimit *int, force bool) error {
	if force {
		DisableMemoization()
	} else if packratEnabled {
		return pyErr("RuntimeError", "Packrat and Bounded Recursion are not compatible")
	}
	if cacheSizeLimit == nil {
		recursionMemos = &unboundedMemo{m: map[memoKey]*memoVal{}}
	} else if *cacheSizeLimit > 0 {
		recursionMemos = &lruMemo{capacity: *cacheSizeLimit, active: map[memoKey]*memoVal{}, memory: newOrderedMemo()}
	} else {
		return pyErr("NotImplementedError", "Memo size of "+itoa(*cacheSizeLimit))
	}
	leftRecursion = true
	return nil
}

// EnablePackrat is ParserElement.enable_packrat.
func EnablePackrat(cacheSizeLimit *int, force bool) error {
	if force {
		DisableMemoization()
	} else if leftRecursion {
		return pyErr("RuntimeError", "Packrat and Bounded Recursion are not compatible")
	}
	if packratEnabled {
		return nil
	}
	packratEnabled = true
	if cacheSizeLimit == nil {
		theCache = &packratCache{size: -1, m: map[cacheKey]*cacheVal{}}
	} else {
		theCache = &packratCache{size: *cacheSizeLimit, m: map[cacheKey]*cacheVal{}}
	}
	return nil
}

// ---------------------------------------------------------------- entry points

func expandTabs(s string) string { return pyval.ExpandTabs(s, 8) }

// recoverNone turns a nil-pointer failure (an expression built from None, which Python
// reports as AttributeError when it is used) into an error instead of a panic.
func recoverNone(err *error) {
	if r := recover(); r != nil {
		*err = &PyError{Class: "AttributeError", Msg: "'NoneType' object has no attribute '_parse'"}
	}
}

// ParseString is parse_string(instring, parse_all).
func (e *ParserElement) ParseString(s string, parseAll bool) (*ParseResults, error) {
	return e.parseString(s, parseAll)
}

func (e *ParserElement) parseString(s string, parseAll bool) (res *ParseResults, err error) {
	if e == nil {
		return nil, pyErr("AttributeError", "'NoneType' object has no attribute 'parse_string'")
	}
	defer recoverNone(&err)
	ResetCache()
	if !e.streamlined {
		e.streamline()
	}
	for _, x := range e.ignoreExprs.items {
		x.streamline()
	}
	if !e.keepTabs {
		s = expandTabs(s)
	}
	st := newState(s)
	var loc int
	var tokens *ParseResults
	loc, tokens, err = e.parse(st, 0, true, true)
	if err == nil && parseAll {
		loc, err = e.preParse(st, loc)
		if err == nil {
			se := Empty().Add(StringEnd())
			_, _, err = se.parse(st, loc, true, true)
		}
	}
	if err != nil {
		if pa, ok := err.(*paIndexError); ok {
			return nil, pa.exc
		}
		return nil, err
	}
	return tokens, nil
}

// ScanString is scan_string(instring, max_matches, overlap, always_skip_whitespace). On an
// exception the matches found before it are returned with the error.
func (e *ParserElement) ScanString(s string, maxMatches int, overlap, alwaysSkipWhitespace bool) ([]ScanMatch, error) {
	if e == nil {
		return nil, pyErr("AttributeError", "'NoneType' object has no attribute 'scan_string'")
	}
	var out []ScanMatch
	err := e.scanString(s, maxMatches, overlap, alwaysSkipWhitespace, func(t *ParseResults, a, b int) error {
		out = append(out, ScanMatch{t, a, b})
		return nil
	})
	return out, exportErr(err)
}

func (e *ParserElement) scanString(s string, maxMatches int, overlap, alwaysSkipWS bool, yield func(*ParseResults, int, int) error) (rerr error) {
	defer recoverNone(&rerr)
	if !e.streamlined {
		e.streamline()
	}
	for _, x := range e.ignoreExprs.items {
		x.streamline()
	}
	if !e.keepTabs {
		s = expandTabs(s)
	}
	st := newState(s)
	instrlen := len(st.runes)
	loc := 0
	var preparse func(*pstate, int) (int, error)
	if alwaysSkipWS {
		preparser := Empty()
		preparser.ignoreExprs = e.ignoreExprs
		preparser.whiteChars = e.whiteChars
		preparse = preparser.preParse
	} else {
		preparse = e.preParse
	}
	ResetCache()
	matches := 0
	preloc := 0
	for loc <= instrlen && matches < maxMatches {
		var err error
		preloc, err = preparse(st, loc)
		var nextLoc int
		var tokens *ParseResults
		if err == nil {
			nextLoc, tokens, err = e.parse(st, preloc, true, false)
		}
		if err != nil {
			if _, ok := asParseException(err); ok {
				loc = preloc + 1
				continue
			}
			return err
		}
		if nextLoc > loc {
			matches++
			if err := yield(tokens, preloc, nextLoc); err != nil {
				return err
			}
			if overlap {
				nl, err := preparse(st, loc)
				if err != nil {
					return err
				}
				if nl > loc {
					loc = nextLoc
				} else {
					loc++
				}
			} else {
				loc = nextLoc
			}
		} else {
			loc = preloc + 1
		}
	}
	return nil
}

// exportErr converts internal error types at the API boundary.
func exportErr(err error) error {
	if pa, ok := err.(*paIndexError); ok {
		return &PyError{Class: "_ParseActionIndexError", Msg: pa.Error()}
	}
	return err
}

// TransformString is transform_string(instring).
func (e *ParserElement) TransformString(s string) (string, error) {
	if e == nil {
		return "", pyErr("AttributeError", "'NoneType' object has no attribute 'transform_string'")
	}
	r, err := e.transformString(s)
	return r, exportErr(err)
}

func (e *ParserElement) transformString(s string) (string, error) {
	var out []any
	lastE := 0
	e.keepTabs = true
	rs := []rune(s)
	err := e.scanString(s, math.MaxInt, false, true, func(t *ParseResults, a, b int) error {
		if a > lastE {
			out = append(out, string(pySlice(rs, lastE, a)))
		}
		lastE = b
		if !t.Truth() {
			return nil
		}
		out = append(out, t.AsList(false)...)
		return nil
	})
	if err != nil {
		return "", err
	}
	out = append(out, string(pySlice(rs, lastE, len(rs))))
	var filt []any
	for _, o := range out {
		if pyTruth(o) {
			filt = append(filt, o)
		}
	}
	var b []byte
	for _, x := range flattenList(filt) {
		b = append(b, Str(x)...)
	}
	return string(b), nil
}

// SearchString is search_string(instring, max_matches).
func (e *ParserElement) SearchString(s string, maxMatches int) (*ParseResults, error) {
	if e == nil {
		return nil, pyErr("AttributeError", "'NoneType' object has no attribute 'search_string'")
	}
	var toks []any
	err := e.scanString(s, maxMatches, false, false, func(t *ParseResults, a, b int) error {
		toks = append(toks, t)
		return nil
	})
	if err != nil {
		return nil, exportErr(err)
	}
	return mustPR(append([]any{}, toks...)), nil
}

// Split is split(instring, maxsplit, include_separators); the pieces are str values, and the
// separators (with include_separators) are the first token of each match. On an exception the
// pieces produced before it are returned with the error.
func (e *ParserElement) Split(s string, maxsplit int, includeSeparators bool) ([]any, error) {
	if e == nil {
		return nil, pyErr("AttributeError", "'NoneType' object has no attribute 'split'")
	}
	var out []any
	last := 0
	rs := []rune(s)
	err := e.scanString(s, maxsplit, false, true, func(t *ParseResults, a, b int) error {
		out = append(out, string(pySlice(rs, last, a)))
		if includeSeparators {
			v, err := t.At(0)
			if err != nil {
				return err
			}
			out = append(out, v)
		}
		last = b
		return nil
	})
	if err != nil {
		return out, exportErr(err)
	}
	out = append(out, string(pySlice(rs, last, len(rs))))
	return out, nil
}

// Matches is matches(test_string, parse_all).
func (e *ParserElement) Matches(s string, parseAll bool) (bool, error) {
	if e == nil {
		return false, pyErr("AttributeError", "'NoneType' object has no attribute 'matches'")
	}
	_, err := e.ParseString(s, parseAll)
	if err != nil {
		if _, ok := asBase(err); ok {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

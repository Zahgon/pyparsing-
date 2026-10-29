// Port of pyparsing/core.py, ParseElementEnhance and its subclasses (pyparsing 3.3.3+). MIT License, Copyright (c) 2003-2025  Paul McGuire; see LICENSE.

package pyparsing

import (
	"math"
	"strings"
)

// newEnhance is ParseElementEnhance.__init__(expr, savelist).
func newEnhance(cls *pclass, expr any, savelist bool) *ParserElement {
	e := newElement(cls, savelist)
	var x *ParserElement
	switch v := expr.(type) {
	case string:
		if literalStringClass == nil || literalClassCls.isSub(clsToken) {
			x = literalOf(v)
		} else if cls.isSub(literalClassCls) {
			x = Literal(v)
		} else {
			x = literalOf(v)
		}
	case *ParserElement:
		x = v
	}
	e.expr = x
	if x != nil {
		e.mayIndexError = x.mayIndexError
		e.mayReturnEmpty = x.mayReturnEmptyProp()
		e.setWhitespaceSet(x.whiteChars, x.copyDefaultWhiteChars)
		e.skipWhitespace = x.skipWhitespace
		e.saveAsList = x.saveAsList
		e.callPreparse = x.callPreparse
		e.ignoreExprs.items = append(e.ignoreExprs.items, x.ignoreExprs.items...)
	}
	return e
}

func (e *ParserElement) enhanceParseImpl(st *pstate, loc int, doActions bool) (int, any, error) {
	if e.expr == nil {
		return 0, nil, newParseExc(st.str(), loc, "No expression defined", e)
	}
	l, t, err := e.expr.parse(st, loc, doActions, false)
	if err != nil {
		if pbe, ok := asBase(err); ok {
			if pbe.Kind == KindParseSyntaxException {
				return 0, nil, err
			}
			if pbe.Pstr == "" {
				pbe.Pstr = st.str()
			}
			if pbe.Loc == 0 {
				pbe.Loc = loc
			}
			if pbe.ParserElement == nil {
				pbe.ParserElement = e
			}
			if e.cls != clsForward && e.customName != nil {
				if e.errmsg != nil && *e.errmsg != "" {
					pbe.Msg = *e.errmsg
				}
			}
		}
		return 0, nil, err
	}
	return l, t, nil
}

// AtStringStart returns AtStringStart(expr).
func AtStringStart(expr any) *ParserElement {
	e := newEnhance(clsAtStringStart, expr, false)
	e.callPreparse = false
	return e
}

// AtLineStart returns AtLineStart(expr).
func AtLineStart(expr any) *ParserElement {
	e := newEnhance(clsAtLineStart, expr, false)
	e.callPreparse = false
	return e
}

// FollowedBy returns FollowedBy(expr).
func FollowedBy(expr any) *ParserElement {
	e := newEnhance(clsFollowedBy, expr, false)
	e.mayReturnEmpty = true
	return e
}

func newFB(expr *ParserElement) *ParserElement {
	e := newEnhance(clsFB, expr, false)
	e.mayReturnEmpty = true
	return e
}

// PrecededBy returns PrecededBy(expr, retreat).
func PrecededBy(expr any, retreat int) *ParserElement {
	e := newEnhance(clsPrecededBy, expr, false)
	if e.expr != nil {
		e.expr = e.expr.copy().leaveWhitespace(true)
	}
	e.mayReturnEmpty = true
	e.mayIndexError = false
	e.exact = false
	var exprStr string
	switch v := expr.(type) {
	case string:
		retreat = len([]rune(v))
		e.exact = true
		exprStr = v
	case *ParserElement:
		exprStr = v.String()
		switch {
		case v.isa(clsLiteral) || v.isa(clsKeyword):
			retreat = v.matchLen
			e.exact = true
		case (v.isa(clsWord) || v.isa(clsCharsNotIn)) && v.maxLen != math.MaxInt:
			retreat = v.maxLen
			e.exact = true
		case v.isa(clsPositionToken):
			retreat = 0
			e.exact = true
		}
	}
	e.retreat = retreat
	e.setErrmsg("not preceded by " + exprStr)
	e.skipWhitespace = false
	e.parseAction = append(e.parseAction, &action{func(s string, l int, t *ParseResults) (any, error) {
		t.delAll()
		return nil, nil
	}})
	return e
}

func (e *ParserElement) precededByParseImpl(st *pstate, loc int, doActions bool) (int, any, error) {
	if e.exact {
		if loc < e.retreat {
			return 0, nil, e.raise(st, loc)
		}
		start := loc - e.retreat
		_, ret, err := e.expr.parse(st, start, true, true)
		if err != nil {
			return 0, nil, err
		}
		return loc, ret, nil
	}
	testExpr := e.expr.Add(StringEnd())
	lo := loc - e.retreat
	if lo < 0 {
		lo = 0
	}
	sl := pySlice(st.runes, lo, loc)
	sst := newStateRunes(append([]rune(nil), sl...), "", false)
	var lastExpr error = e.raise(st, loc)
	n := loc
	if e.retreat+1 < n {
		n = e.retreat + 1
	}
	for offset := 1; offset <= n; offset++ {
		_, ret, err := testExpr.parse(sst, len(sl)-offset, true, true)
		if err != nil {
			if _, ok := asBase(err); ok {
				lastExpr = err
				continue
			}
			return 0, nil, err
		}
		return loc, ret, nil
	}
	return 0, nil, lastExpr
}

// Located returns Located(expr).
func Located(expr any) *ParserElement { return newEnhance(clsLocated, expr, false) }

func (e *ParserElement) locatedParseImpl(st *pstate, loc int, doActions bool) (int, any, error) {
	start, err := e.expr.preParse(st, loc)
	if err != nil {
		return 0, nil, err
	}
	loc, tokens, err := e.expr.parse(st, start, doActions, false)
	if err != nil {
		return 0, nil, err
	}
	bs, bl := bigInt(start), bigInt(loc)
	ret := mustPR([]any{bs, tokens, bl})
	ret.setItemStr("locn_start", bs)
	ret.setItemStr("value", tokens)
	ret.setItemStr("locn_end", bl)
	if e.resultsName != nil && *e.resultsName != "" {
		return loc, []any{ret}, nil
	}
	return loc, ret, nil
}

// NotAny returns NotAny(expr).
func NotAny(expr any) *ParserElement {
	e := newEnhance(clsNotAny, expr, false)
	e.skipWhitespace = false
	e.mayReturnEmpty = true
	e.setErrmsg("Found unwanted token, " + e.expr.String())
	return e
}

func newMultipleMatch(cls *pclass, expr any, stopOn any, max *int) (*ParserElement, error) {
	e := newEnhance(cls, expr, false)
	if max != nil && *max <= 0 {
		return nil, pyErr("ValueError", "max must be greater than 0")
	}
	e.saveAsList = true
	if max != nil {
		m := *max
		e.maxCount = &m
	}
	e.stopOn(stopOn)
	return e, nil
}

func (e *ParserElement) stopOn(ender any) *ParserElement {
	x := toElement(ender)
	if x != nil {
		e.notEnder = NotAny(x)
	} else {
		e.notEnder = nil
	}
	return e
}

// OneOrMore returns OneOrMore(expr, stop_on, max).
func OneOrMore(expr any, stopOn any, max *int) (*ParserElement, error) {
	return newMultipleMatch(clsOneOrMore, expr, stopOn, max)
}

// ZeroOrMore returns ZeroOrMore(expr, stop_on, max).
func ZeroOrMore(expr any, stopOn any, max *int) (*ParserElement, error) {
	e, err := newMultipleMatch(clsZeroOrMore, expr, stopOn, max)
	if err != nil {
		return nil, err
	}
	e.mayReturnEmpty = true
	return e, nil
}

func zeroOrMore(expr any) *ParserElement {
	e, _ := ZeroOrMore(expr, nil, nil)
	return e
}

func oneOrMore(expr any) *ParserElement {
	e, _ := OneOrMore(expr, nil, nil)
	return e
}

func (e *ParserElement) multipleMatchParseImpl(st *pstate, loc int, doActions bool) (int, any, error) {
	checkEnder := e.notEnder != nil
	if checkEnder {
		if _, err := e.notEnder.tryParse(st, loc, false, false); err != nil {
			return 0, nil, err
		}
	}
	loc, tokens, err := e.expr.parse(st, loc, doActions, true)
	if err != nil {
		return 0, nil, err
	}
	matchCount := 1
	hasIgnore := len(e.ignoreExprs.items) > 0
	for e.maxCount == nil || matchCount < *e.maxCount {
		if checkEnder {
			if _, err := e.notEnder.tryParse(st, loc, false, false); err != nil {
				if _, ok := asParseException(err); ok || isIndexError(err) {
					break
				}
				return 0, nil, err
			}
		}
		preloc := loc
		if hasIgnore {
			preloc, err = e.skipIgnorables(st, loc)
			if err != nil {
				if _, ok := asParseException(err); ok || isIndexError(err) {
					break
				}
				return 0, nil, err
			}
		}
		l, tmp, err := e.expr.parse(st, preloc, doActions, true)
		if err != nil {
			if _, ok := asParseException(err); ok || isIndexError(err) {
				break
			}
			return 0, nil, err
		}
		loc = l
		tokens.IAdd(tmp)
		matchCount++
	}
	return loc, tokens, nil
}

// DelimitedList returns DelimitedList(expr, delim, combine, min, max, allow_trailing_delim=...).
func DelimitedList(expr any, delim any, combine bool, min, max *int, allowTrailingDelim bool) (*ParserElement, error) {
	content := toElement(expr)
	if content == nil {
		return nil, pyErr("AttributeError", "'"+pyTypeName(expr)+"' object has no attribute 'streamline'")
	}
	if min != nil && *min < 1 {
		return nil, pyErr("ValueError", "min must be greater than 0")
	}
	if max != nil && min != nil && *max < *min {
		return nil, pyErr("ValueError", "max must be greater than, or equal to min")
	}
	d := &delimData{content: content, combine: combine, allowTrailingDelim: allowTrailingDelim}
	switch v := delim.(type) {
	case string:
		d.rawDelim = v
	case *ParserElement:
		d.rawDelim = v.String()
	default:
		d.rawDelim = Str(delim)
	}
	d.delim = delim
	if !combine {
		if de, ok := delim.(*ParserElement); ok && de.isa(clsSuppress) {
			d.delim = de
		} else {
			d.delim = Suppress(delim)
		}
	}
	d.min = 1
	if min != nil && *min != 0 {
		d.min = *min
	}
	d.max = max
	var tup PyTuple
	if max == nil {
		tup = PyTuple{d.min - 1, nil}
	} else {
		tup = PyTuple{d.min - 1, *max - 1}
	}
	dc := addAny(d.delim, content)
	rep, err := dc.Mul(tup)
	if err != nil {
		return nil, err
	}
	dle := content.Add(rep)
	if allowTrailingDelim {
		dle = dle.iadd(Opt(d.delim))
	}
	if combine {
		dle = Combine(dle, "", true)
	}
	e := newEnhance(clsDelimitedList, dle, true)
	e.dl = d
	return e, nil
}

// addAny is a + b for an `any` left operand (str or element).
func addAny(a any, b *ParserElement) *ParserElement {
	switch v := a.(type) {
	case *ParserElement:
		return v.Add(b)
	case string:
		return b.RAdd(v)
	}
	return b.RAdd(a)
}

// Opt returns Opt(expr, default); omitting default means "no default".
func Opt(expr any, defaultValue ...any) *ParserElement {
	e := newEnhance(clsOpt, expr, false)
	if e.expr != nil {
		e.saveAsList = e.expr.saveAsList
	}
	if len(defaultValue) > 0 {
		e.defaultValue = defaultValue[0]
	} else {
		e.defaultValue = nullToken{}
	}
	e.mayReturnEmpty = true
	return e
}

func (e *ParserElement) optParseImpl(st *pstate, loc int, doActions bool) (int, any, error) {
	l, tokens, err := e.expr.parse(st, loc, doActions, false)
	if err == nil {
		return l, tokens, nil
	}
	if _, ok := asParseException(err); !ok && !isIndexError(err) {
		return 0, nil, err
	}
	dv := e.defaultValue
	if _, notMatched := dv.(nullToken); !notMatched {
		if e.expr.resultsName != nil && *e.expr.resultsName != "" {
			t := mustPR([]any{dv})
			t.setItemStr(*e.expr.resultsName, dv)
			return loc, t, nil
		}
		return loc, []any{dv}, nil
	}
	return loc, []any{}, nil
}

// SkipTo returns SkipTo(other, include, ignore, fail_on).
func SkipTo(other any, include bool, ignore any, failOn any) *ParserElement {
	e := newEnhance(clsSkipTo, other, false)
	e.ignoreExpr = ignore
	e.mayReturnEmpty = true
	e.mayIndexError = false
	e.includeMatch = include
	e.saveAsList = false
	e.failOn = toElement(failOn)
	e.setErrmsg("No match found for " + e.expr.String())
	e.ignorer = Empty().leaveWhitespace(true)
	e.updateIgnorer()
	return e
}

func (e *ParserElement) updateIgnorer() {
	e.ignorer.ignoreExprs.items = e.ignorer.ignoreExprs.items[:0:0]
	for _, x := range append([]*ParserElement(nil), e.expr.ignoreExprs.items...) {
		e.ignorer.ignore(x)
	}
	if pyTruth(e.ignoreExpr) || e.ignoreExpr != nil && isElem(e.ignoreExpr) {
		if o := ignoreArg(e.ignoreExpr); o != nil {
			e.ignorer.ignore(o)
		}
	}
}

func isElem(v any) bool {
	_, ok := v.(*ParserElement)
	return ok
}

func (e *ParserElement) skipToParseImpl(st *pstate, loc int, doActions bool) (int, any, error) {
	startloc := loc
	instrlen := len(st.runes)
	useIgnorer := len(e.ignorer.ignoreExprs.items) > 0
	tmploc := loc
	found := false
	for tmploc <= instrlen {
		if e.failOn != nil {
			ok, err := e.failOn.canParseNext(st, tmploc, false)
			if err != nil {
				return 0, nil, err
			}
			if ok {
				found = true
				break
			}
		}
		if useIgnorer {
			prev := tmploc
			for {
				l, err := e.ignorer.tryParse(st, tmploc, false, false)
				if err != nil {
					if _, ok := asBase(err); ok {
						break
					}
					return 0, nil, err
				}
				tmploc = l
				if tmploc == prev {
					break
				}
				prev = tmploc
			}
		}
		_, _, err := e.expr.parse(st, tmploc, false, false)
		if err != nil {
			if _, ok := asParseException(err); ok || isIndexError(err) {
				tmploc++
				continue
			}
			return 0, nil, err
		}
		found = true
		break
	}
	if !found {
		return 0, nil, e.raise(st, loc)
	}
	loc = tmploc
	skipresult := mustPR(st.slice(startloc, loc))
	if e.includeMatch {
		l, mat, err := e.expr.parse(st, loc, doActions, false)
		if err != nil {
			return 0, nil, err
		}
		loc = l
		skipresult.IAdd(mat)
	}
	return loc, skipresult, nil
}

// Forward returns Forward(other) (no argument: an empty Forward).
func Forward(other ...any) *ParserElement {
	var o any
	if len(other) > 0 {
		o = other[0]
	}
	return newEnhance(clsForward, o, false)
}

func (e *ParserElement) lshift(other any) *ParserElement {
	x := toElement(other)
	if x == nil {
		return nil
	}
	e.expr = x
	e.streamlined = x.streamlined
	e.mayIndexError = x.mayIndexError
	e.mayReturnEmpty = x.mayReturnEmptyProp()
	e.setWhitespaceSet(x.whiteChars, x.copyDefaultWhiteChars)
	e.skipWhitespace = x.skipWhitespace
	e.saveAsList = x.saveAsList
	e.ignoreExprs.items = append(e.ignoreExprs.items, x.ignoreExprs.items...)
	return e
}

// LShift is forward << other; it returns nil when expr is not a Forward or other is not an
// expression or str.
func (e *ParserElement) LShift(other any) *ParserElement {
	if e == nil {
		return nil
	}
	if e.cls != clsForward {
		return nil
	}
	return e.lshift(other)
}

// ILShift is forward <<= other.
func (e *ParserElement) ILShift(other any) *ParserElement { return e.LShift(other) }

func (e *ParserElement) forwardParseImpl(st *pstate, loc int, doActions bool) (int, any, error) {
	if !leftRecursion {
		return e.enhanceParseImpl(st, loc, doActions)
	}
	memo := recursionMemos
	if v, ok := memo.get(memoKey{loc, e, doActions}); ok {
		if v.exc != nil {
			return 0, nil, v.exc
		}
		return v.loc, v.result.Copy(), nil
	}
	actKey := memoKey{loc, e, true}
	peekKey := memoKey{loc, e, false}
	init := &memoVal{loc: loc - 1, exc: newParseExc(st.str(), loc, "Forward recursion without base case", e)}
	memo.set(peekKey, init)
	prevLoc, prevPeek := init.loc, init
	if doActions {
		memo.set(actKey, init)
	}
	for {
		var newLoc int
		var newPeek *memoVal
		l, t, err := e.enhanceParseImpl(st, loc, false)
		if err != nil {
			if _, ok := asParseException(err); ok {
				if prevPeek.exc != nil {
					return 0, nil, err
				}
				newLoc, newPeek = prevLoc, prevPeek
			} else {
				return 0, nil, err
			}
		} else {
			newLoc, newPeek = l, &memoVal{loc: l, result: t.(*ParseResults)}
		}
		if newLoc <= prevLoc {
			if doActions {
				v, _ := memo.get(actKey)
				memo.set(peekKey, v)
				memo.del(peekKey)
				memo.del(actKey)
				if v.exc != nil {
					return v.loc, v.exc.copyExc(), nil
				}
				return v.loc, v.result.pyCopyCopy(), nil
			}
			memo.del(peekKey)
			if prevPeek.exc != nil {
				return prevLoc, prevPeek.exc.copyExc(), nil
			}
			return prevLoc, prevPeek.result.pyCopyCopy(), nil
		}
		if doActions {
			l2, t2, err := e.enhanceParseImpl(st, loc, true)
			if err != nil {
				if pe, ok := asParseException(err); ok {
					v := &memoVal{loc: newLoc, exc: pe}
					memo.set(peekKey, v)
					memo.set(actKey, v)
				}
				return 0, nil, err
			}
			memo.set(actKey, &memoVal{loc: l2, result: t2.(*ParseResults)})
		}
		prevLoc, prevPeek = newLoc, newPeek
		memo.set(peekKey, newPeek)
	}
}

// copyExc is copy.copy(exc).
func (pe *ParseException) copyExc() *ParseException {
	c := *pe
	return &c
}

// ---------------------------------------------------------------- TokenConverter

func newTokenConverter(cls *pclass, expr any) *ParserElement {
	e := newEnhance(cls, expr, false)
	e.saveAsList = false
	return e
}

// Combine returns Combine(expr, join_string, adjacent).
func Combine(expr any, joinString string, adjacent bool) *ParserElement {
	e := newTokenConverter(clsCombine, expr)
	if adjacent {
		e.leaveWhitespace(true)
	}
	e.adjacent = adjacent
	e.skipWhitespace = true
	e.joinString = joinString
	e.callPreparse = true
	return e
}

func (e *ParserElement) combinePostParse(tokens any) (any, error) {
	tokenlist, ok := tokens.(*ParseResults)
	if !ok {
		var err error
		tokenlist, err = newPR(tokens, nil, true, true)
		if err != nil {
			return nil, err
		}
	}
	retToks := tokenlist.Copy()
	retToks.delAll()
	joined := strings.Join(tokenlist.asStringList(e.joinString), "")
	retToks.IAdd(mustPR([]any{joined}))
	if e.resultsName != nil && *e.resultsName != "" && retToks.HasKeys() {
		return []any{retToks}, nil
	}
	return retToks, nil
}

// Group returns Group(expr, aslist).
func Group(expr any, aslist bool) *ParserElement {
	e := newTokenConverter(clsGroup, expr)
	e.saveAsList = true
	e.asPythonList = aslist
	return e
}

// Dict returns Dict(expr, asdict).
func Dict(expr any, asdict bool) *ParserElement {
	e := newTokenConverter(clsDict, expr)
	e.saveAsList = true
	e.asPythonDict = asdict
	return e
}

// pyLen is len(v).
func pyLen(v any) (int, error) {
	switch x := v.(type) {
	case *ParseResults:
		return x.Len(), nil
	case string:
		return len([]rune(x)), nil
	case []any:
		return len(x), nil
	case prList:
		return len(x), nil
	case PyTuple:
		return len(x), nil
	case *PyDict:
		return x.Len(), nil
	}
	return 0, pyErr("TypeError", "object of type '"+pyTypeName(v)+"' has no len()")
}

// pyGetItem is v[i] for an int index.
func pyGetItem(v any, i int) (any, error) {
	switch x := v.(type) {
	case *ParseResults:
		return x.At(i)
	case string:
		rs := []rune(x)
		j := i
		if j < 0 {
			j += len(rs)
		}
		if j < 0 || j >= len(rs) {
			return nil, pyErr("IndexError", "string index out of range")
		}
		return string(rs[j]), nil
	case []any, prList, PyTuple:
		items, _ := pyIterItems(x)
		j := i
		if j < 0 {
			j += len(items)
		}
		if j < 0 || j >= len(items) {
			if _, ok := x.(PyTuple); ok {
				return nil, pyErr("IndexError", "tuple index out of range")
			}
			return nil, pyErr("IndexError", "list index out of range")
		}
		return items[j], nil
	case *PyDict:
		return nil, pyErr("KeyError", itoa(i))
	}
	return nil, pyErr("TypeError", "'"+pyTypeName(v)+"' object is not subscriptable")
}

func (e *ParserElement) dictPostParse(tokens any) (any, error) {
	tokenlist, ok := tokens.(*ParseResults)
	if !ok {
		var err error
		tokenlist, err = newPR(tokens, nil, true, true)
		if err != nil {
			return nil, err
		}
	}
	items := append([]any(nil), tokenlist.toks.items...)
	for i, tok := range items {
		n, err := pyLen(tok)
		if err != nil {
			return nil, err
		}
		if n == 0 {
			continue
		}
		ikey, err := pyGetItem(tok, 0)
		if err != nil {
			return nil, err
		}
		if _, isPR := ikey.(*ParseResults); isPR {
			continue
		}
		var key string
		if b, ok := ikey.(bool); ok {
			key = Repr(b)
		} else if bi, ok := toBigInt(ikey); ok {
			key = pyStrip(bi.String())
		} else {
			key = Str(ikey)
		}
		if n == 1 {
			tokenlist.setItemOffset(key, offsetEntry{"", i})
			continue
		}
		second, err := pyGetItem(tok, 1)
		if err != nil {
			return nil, err
		}
		if _, isPR := second.(*ParseResults); n == 2 && !isPR {
			tokenlist.setItemOffset(key, offsetEntry{second, i})
			continue
		}
		if lst, ok := tok.([]any); ok {
			dv := append([]any{}, lst[1:]...)
			if len(dv) != 1 {
				tokenlist.setItemOffset(key, offsetEntry{dv, i})
			} else {
				tokenlist.setItemOffset(key, offsetEntry{dv[0], i})
			}
			continue
		}
		if d, ok := tok.(*PyDict); ok {
			if _, has := d.Get("0"); !has {
				return nil, pyErr("KeyError", "0")
			}
		}
		tp, ok := tok.(*ParseResults)
		if !ok {
			return nil, pyErr("TypeError", "could not extract dict values from parsed results - Dict expression must contain Grouped expressions")
		}
		dictvalue := tp.Copy()
		if err := dictvalue.delIndex(0); err != nil {
			return nil, err
		}
		if dictvalue.Len() != 1 || dictvalue.HasKeys() {
			tokenlist.setItemOffset(key, offsetEntry{dictvalue, i})
		} else {
			v, _ := dictvalue.At(0)
			tokenlist.setItemOffset(key, offsetEntry{v, i})
		}
	}
	tokenlist.isDictContext = true
	named := e.resultsName != nil && *e.resultsName != ""
	if e.asPythonDict {
		if named {
			return []any{tokenlist.AsDict()}, nil
		}
		return tokenlist.AsDict(), nil
	}
	if named {
		return []any{tokenlist}, nil
	}
	return tokenlist, nil
}

// Suppress returns Suppress(expr); Suppress(Ellipsis) is the pending skip `Suppress(...)`.
func Suppress(expr any) *ParserElement {
	if _, ok := expr.(EllipsisType); ok {
		expr = newPendingSkip(NoMatch(), false)
	}
	return newTokenConverter(clsSuppress, expr)
}

func newPendingSkip(expr *ParserElement, mustSkip bool) *ParserElement {
	e := newElement(clsPendingSkip, false)
	e.anchor = expr
	e.mustSkip = mustSkip
	return e
}

// pendingSkipAdd is _PendingSkip.__add__.
func (e *ParserElement) pendingSkipAdd(other any) *ParserElement {
	skipper := SkipTo(other, false, nil, nil).SetName("...").Call("_skipped*")
	if e.mustSkip {
		mustSkip := func(s string, l int, t *ParseResults) (any, error) {
			sk := t.Attr("_skipped")
			if !pyTruth(sk) || isEmptyStrList(sk) {
				if err := t.delIndex(0); err != nil {
					return nil, err
				}
				if _, err := t.Pop("_skipped", nil); err != nil {
					return nil, err
				}
			}
			return nil, nil
		}
		anchor := e.anchor
		showSkip := func(s string, l int, t *ParseResults) (any, error) {
			sk := t.Attr("_skipped")
			pr, ok := sk.(*ParseResults)
			if !ok {
				return nil, pyErr("AttributeError", "'str' object has no attribute 'as_list'")
			}
			lst := pr.AsList(false)
			if len(lst) > 0 && pyEqual(lst[len(lst)-1], "") {
				if _, err := t.Pop("_skipped"); err != nil {
					return nil, err
				}
				t.setItemStr("_skipped", "missing <"+anchor.Repr()+">")
			}
			return nil, nil
		}
		left := e.anchor.Add(skipper.copy().addPA(mustSkip))
		alt := left.BitOr(skipper.copy().addPA(showSkip))
		return alt.Add(other)
	}
	return e.anchor.Add(skipper).Add(other)
}

func isEmptyStrList(v any) bool {
	pr, ok := v.(*ParseResults)
	if !ok {
		return false
	}
	l := pr.AsList(false)
	return len(l) == 1 && pyEqual(l[0], "")
}

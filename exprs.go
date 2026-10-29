// Port of pyparsing/core.py, ParseExpression and its subclasses (pyparsing 3.3.3+). MIT License, Copyright (c) 2003-2025  Paul McGuire; see LICENSE.

package pyparsing

import (
	"sort"
)

// exprArgs flattens the variadic expression arguments (a single []any is taken as the list).
func exprArgs(exprs []any) []any {
	if len(exprs) == 1 {
		switch x := exprs[0].(type) {
		case []any:
			return x
		case []*ParserElement:
			out := make([]any, len(x))
			for i, v := range x {
				out[i] = v
			}
			return out
		}
	}
	return exprs
}

// newParseExpression is ParseExpression.__init__.
func newParseExpression(cls *pclass, exprs []any, savelist bool) *ParserElement {
	e := newElement(cls, savelist)
	for _, x := range exprs {
		switch v := x.(type) {
		case string:
			e.exprs = append(e.exprs, literalOf(v))
		case *ParserElement:
			if v != nil {
				e.exprs = append(e.exprs, v)
			}
		}
	}
	e.callPreparse = false
	return e
}

// And returns And(exprs). A sequence ending in Ellipsis is an error.
func And(exprs ...any) (*ParserElement, error) {
	args := exprArgs(exprs)
	conv := make([]any, len(args))
	hasEllipsis := false
	for i, x := range args {
		if s, ok := x.(string); ok {
			conv[i] = literalOf(s)
		} else {
			conv[i] = x
		}
		if _, ok := x.(EllipsisType); ok {
			hasEllipsis = true
		}
	}
	if hasEllipsis {
		if _, ok := conv[len(conv)-1].(EllipsisType); ok {
			return nil, pyErr("Exception", "cannot construct And with sequence ending in ...")
		}
		tmp := make([]any, 0, len(conv))
		for i := 0; i+1 < len(conv); i++ {
			if _, ok := conv[i].(EllipsisType); ok {
				tmp = append(tmp, SkipTo(conv[i+1], false, nil, nil).Call("_skipped*"))
			} else {
				tmp = append(tmp, conv[i])
			}
		}
		conv = append(tmp, conv[len(conv)-1])
	}
	return newAnd(conv, true), nil
}

func newAnd(exprs []any, savelist bool) *ParserElement {
	e := newParseExpression(clsAnd, exprs, savelist)
	if len(e.exprs) > 0 {
		e.mayReturnEmpty = allExpr(e.exprs, func(x *ParserElement) bool { return x.mayReturnEmptyProp() })
		if !e.exprs[0].isa(clsWhite) {
			e.setWhitespaceSet(e.exprs[0].whiteChars, e.exprs[0].copyDefaultWhiteChars)
			e.skipWhitespace = e.exprs[0].skipWhitespace
		} else {
			e.skipWhitespace = false
		}
	} else {
		e.mayReturnEmpty = true
	}
	e.callPreparse = true
	return e
}

func andOf(exprs ...*ParserElement) *ParserElement {
	args := make([]any, len(exprs))
	for i, x := range exprs {
		args[i] = x
	}
	return newAnd(args, true)
}

func (e *ParserElement) andParseImpl(st *pstate, loc int, doActions bool) (int, any, error) {
	exprs := e.exprs
	if len(exprs) == 0 {
		exprs = []*ParserElement{Empty()}
	}
	loc, resultlist, err := exprs[0].parse(st, loc, doActions, false)
	if err != nil {
		return 0, nil, err
	}
	raiseImmediately := false
	for _, x := range exprs[1:] {
		if x.cls == clsErrorStop {
			raiseImmediately = true
			continue
		}
		var toks *ParseResults
		if raiseImmediately {
			loc, toks, err = x.parse(st, loc, doActions, true)
			if err != nil {
				if pe, ok := asBase(err); ok {
					if pe.Kind == KindParseSyntaxException {
						return 0, nil, err
					}
					return 0, nil, fromException(KindParseSyntaxException, pe)
				}
				if isIndexError(err) {
					return 0, nil, newPE(KindParseSyntaxException, st.str(), len(st.runes), e.errmsg, e)
				}
				return 0, nil, err
			}
		} else {
			loc, toks, err = x.parse(st, loc, doActions, true)
			if err != nil {
				return 0, nil, err
			}
		}
		resultlist.IAdd(toks)
	}
	return loc, resultlist, nil
}

// Or returns Or(exprs).
func Or(exprs ...any) *ParserElement {
	e := newParseExpression(clsOr, exprArgs(exprs), false)
	if len(e.exprs) > 0 {
		e.mayReturnEmpty = anyExpr(e.exprs, func(x *ParserElement) bool { return x.mayReturnEmptyProp() })
		e.skipWhitespace = allExpr(e.exprs, func(x *ParserElement) bool { return x.skipWhitespace })
	} else {
		e.mayReturnEmpty = true
	}
	return e
}

func sortFatals(fatals []*ParseException) {
	if len(fatals) > 1 {
		sort.SliceStable(fatals, func(i, j int) bool { return -fatals[i].Loc < -fatals[j].Loc })
		if fatals[0].Loc == fatals[1].Loc {
			sort.SliceStable(fatals, func(i, j int) bool {
				a, b := fatals[i], fatals[j]
				if a.Loc != b.Loc {
					return a.Loc > b.Loc
				}
				return runeLen(a.ParserElement.String()) > runeLen(b.ParserElement.String())
			})
		}
	}
}

func runeLen(s string) int { return len([]rune(s)) }

func (e *ParserElement) orParseImpl(st *pstate, loc int, doActions bool) (int, any, error) {
	maxExcLoc := -1
	var maxException *ParseException
	type match struct {
		loc  int
		expr *ParserElement
	}
	var matches []match
	var fatals []*ParseException
	if allExpr(e.exprs, func(x *ParserElement) bool { return x.callPreparse }) {
		var err error
		loc, err = e.preParse(st, loc)
		if err != nil {
			return 0, nil, err
		}
	}
	for _, x := range e.exprs {
		loc2, err := x.tryParse(st, loc, true, false)
		if err != nil {
			if pfe, ok := asFatal(err); ok {
				pfe.ParserElement = x
				fatals = append(fatals, pfe)
				maxException = nil
				maxExcLoc = -1
				continue
			}
			if pe, ok := asParseException(err); ok {
				if len(fatals) == 0 {
					if pe.Loc > maxExcLoc {
						maxException = pe
						maxExcLoc = pe.Loc
					}
				}
				continue
			}
			if isIndexError(err) {
				if len(st.runes) > maxExcLoc {
					maxException = newPE(KindParseException, st.str(), len(st.runes), x.errmsg, e)
					maxExcLoc = len(st.runes)
				}
				continue
			}
			return 0, nil, err
		}
		matches = append(matches, match{loc2, x})
	}
	if len(matches) > 0 {
		sort.SliceStable(matches, func(i, j int) bool { return matches[i].loc > matches[j].loc })
		if !doActions {
			l, t, err := matches[0].expr.parse(st, loc, doActions, true)
			if err != nil {
				return 0, nil, err
			}
			return l, t, nil
		}
		longestLoc := -1
		var longestToks *ParseResults
		for _, m := range matches {
			if m.loc <= longestLoc {
				return longestLoc, longestToks, nil
			}
			loc2, toks, err := m.expr.parse(st, loc, doActions, true)
			if err != nil {
				if pe, ok := asParseException(err); ok {
					if pe.Loc > maxExcLoc {
						maxException = pe
						maxExcLoc = pe.Loc
					}
					continue
				}
				return 0, nil, err
			}
			if loc2 >= m.loc {
				return loc2, toks, nil
			} else if loc2 > longestLoc {
				longestLoc, longestToks = loc2, toks
			}
		}
		if longestLoc != -1 || longestToks != nil {
			return longestLoc, longestToks, nil
		}
	}
	if len(fatals) > 0 {
		sortFatals(fatals)
		return 0, nil, fatals[0]
	}
	if maxException != nil {
		psl, err := e.preParse(st, loc)
		if err != nil {
			return 0, nil, err
		}
		if maxExcLoc == psl {
			maxException.Msg = e.errmsgOr()
		}
		return 0, nil, maxException
	}
	return 0, nil, newParseExc(st.str(), loc, "no defined alternatives to match", e)
}

// MatchFirst returns MatchFirst(exprs).
func MatchFirst(exprs ...any) *ParserElement {
	e := newParseExpression(clsMatchFirst, exprArgs(exprs), false)
	if len(e.exprs) > 0 {
		e.mayReturnEmpty = anyExpr(e.exprs, func(x *ParserElement) bool { return x.mayReturnEmptyProp() })
		e.skipWhitespace = allExpr(e.exprs, func(x *ParserElement) bool { return x.skipWhitespace })
	} else {
		e.mayReturnEmpty = true
	}
	return e
}

func (e *ParserElement) matchFirstParseImpl(st *pstate, loc int, doActions bool) (int, any, error) {
	maxExcLoc := -1
	var maxException *ParseException
	for _, x := range e.exprs {
		l, t, err := x.parse(st, loc, doActions, true)
		if err == nil {
			return l, t, nil
		}
		if pfe, ok := asFatal(err); ok {
			pfe.ParserElement = x
			return 0, nil, err
		}
		if pe, ok := asParseException(err); ok {
			if pe.Loc > maxExcLoc {
				maxException = pe
				maxExcLoc = pe.Loc
			}
			continue
		}
		if isIndexError(err) {
			if len(st.runes) > maxExcLoc {
				maxException = newPE(KindParseException, st.str(), len(st.runes), x.errmsg, e)
				maxExcLoc = len(st.runes)
			}
			continue
		}
		return 0, nil, err
	}
	if maxException != nil {
		psl, err := e.preParse(st, loc)
		if err != nil {
			return 0, nil, err
		}
		if maxExcLoc == psl {
			maxException.Msg = e.errmsgOr()
		}
		return 0, nil, maxException
	}
	return 0, nil, newParseExc(st.str(), loc, "no defined alternatives to match", e)
}

// Each returns Each(exprs).
func Each(exprs ...any) *ParserElement {
	e := newParseExpression(clsEach, exprArgs(exprs), true)
	if len(e.exprs) > 0 {
		e.mayReturnEmpty = allExpr(e.exprs, func(x *ParserElement) bool { return x.mayReturnEmptyProp() })
	} else {
		e.mayReturnEmpty = true
	}
	e.skipWhitespace = true
	e.saveAsList = true
	return e
}

func containsElem(xs []*ParserElement, x *ParserElement) int {
	for i, y := range xs {
		if y == x || elemEqual(y, x) {
			return i
		}
	}
	return -1
}

func (e *ParserElement) eachParseImpl(st *pstate, loc int, doActions bool) (int, any, error) {
	if e.each == nil {
		d := &eachData{opt1map: map[*ParserElement]*ParserElement{}}
		var opt1, opt2 []*ParserElement
		for _, x := range e.exprs {
			if x.isa(clsOpt) {
				d.opt1map[x.expr] = x
				opt1 = append(opt1, x.expr)
			}
		}
		for _, x := range e.exprs {
			if x.mayReturnEmptyProp() && !x.isa(clsOpt) && !x.isa(clsRegex) && !x.isa(clsZeroOrMore) {
				opt2 = append(opt2, x)
			}
		}
		d.optionals = append(opt1, opt2...)
		for _, x := range e.exprs {
			if x.isa(clsMultipleMatch) {
				d.multioptionals = append(d.multioptionals, x.expr.setResultsNameOpt(x.resultsName, true))
			}
		}
		for _, x := range e.exprs {
			if x.isa(clsOneOrMore) {
				d.multirequired = append(d.multirequired, x.expr.setResultsNameOpt(x.resultsName, true))
			}
		}
		for _, x := range e.exprs {
			if !x.isa(clsOpt) && !x.isa(clsZeroOrMore) && !x.isa(clsOneOrMore) {
				d.required = append(d.required, x)
			}
		}
		d.required = append(d.required, d.multirequired...)
		e.each = d
	}
	d := e.each
	tmpLoc := loc
	tmpReqd := append([]*ParserElement(nil), d.required...)
	tmpOpt := append([]*ParserElement(nil), d.optionals...)
	multis := append([]*ParserElement(nil), d.multioptionals...)
	var matchOrder []*ParserElement
	keepMatching := true
	var fatals []*ParseException
	for keepMatching {
		tmpExprs := append(append(append([]*ParserElement(nil), tmpReqd...), tmpOpt...), multis...)
		failed := 0
		fatals = fatals[:0]
		for _, x := range tmpExprs {
			l, err := x.tryParse(st, tmpLoc, true, false)
			if err != nil {
				if pfe, ok := asFatal(err); ok {
					pfe.ParserElement = x
					fatals = append(fatals, pfe)
					failed++
					continue
				}
				if _, ok := asParseException(err); ok {
					failed++
					continue
				}
				return 0, nil, err
			}
			tmpLoc = l
			if m, ok := d.opt1map[x]; ok {
				matchOrder = append(matchOrder, m)
			} else {
				matchOrder = append(matchOrder, x)
			}
			if i := containsElem(tmpReqd, x); i >= 0 {
				tmpReqd = append(tmpReqd[:i:i], tmpReqd[i+1:]...)
			} else if i := containsElem(tmpOpt, x); i >= 0 {
				tmpOpt = append(tmpOpt[:i:i], tmpOpt[i+1:]...)
			}
		}
		if failed == len(tmpExprs) {
			keepMatching = false
		}
	}
	if len(fatals) > 0 {
		sortFatals(fatals)
		return 0, nil, fatals[0]
	}
	if len(tmpReqd) > 0 {
		missing := joinNames(tmpReqd, ", ")
		return 0, nil, newParseExc(st.str(), loc, "Missing one or more required elements ("+missing+")", nil)
	}
	for _, x := range e.exprs {
		if x.isa(clsOpt) && containsElem(tmpOpt, x.expr) >= 0 {
			matchOrder = append(matchOrder, x)
		}
	}
	total := newEmptyPR()
	for _, x := range matchOrder {
		var res *ParseResults
		var err error
		loc, res, err = x.parse(st, loc, doActions, true)
		if err != nil {
			return 0, nil, err
		}
		total.IAdd(res)
	}
	return loc, total, nil
}

// setResultsNameOpt is _setResultsName with a possibly-None name.
func (e *ParserElement) setResultsNameOpt(name *string, listAll bool) *ParserElement {
	if name == nil {
		return e
	}
	return e.setResultsName(*name, listAll)
}

// ---------------------------------------------------------------- structural equality

// elemEqual approximates ParserElement.__eq__ (vars(self) == vars(other)) as used by `in`
// and list.remove in ignore() and Each.
func elemEqual(a, b *ParserElement) bool {
	if a == b {
		return true
	}
	if a == nil || b == nil || a.cls != b.cls {
		return false
	}
	if a.cls == clsForward || a.cls == clsTag || a.cls == clsPendingSkip {
		return false
	}
	if (a.cls.isSub(clsWord) && (a.useRe || b.useRe)) || (a.cls == clsRegex && (a.asGroupList || a.asMatch)) {
		return false
	}
	if len(a.parseAction) != len(b.parseAction) {
		return false
	}
	for i := range a.parseAction {
		if a.parseAction[i] != b.parseAction[i] {
			return false
		}
	}
	if !strpEq(a.customName, b.customName) || !strpEq(a.defaultName, b.defaultName) || !strpEq(a.resultsName, b.resultsName) || !strpEq(a.errmsg, b.errmsg) {
		return false
	}
	if a.saveAsList != b.saveAsList || a.skipWhitespace != b.skipWhitespace || !a.whiteChars.equal(b.whiteChars) ||
		a.copyDefaultWhiteChars != b.copyDefaultWhiteChars || a.mayReturnEmpty != b.mayReturnEmpty || a.keepTabs != b.keepTabs ||
		a.streamlined != b.streamlined || a.mayIndexError != b.mayIndexError || a.modalResults != b.modalResults ||
		a.callPreparse != b.callPreparse || a.callDuringTry != b.callDuringTry {
		return false
	}
	if !elemsEqual(a.ignoreExprs.items, b.ignoreExprs.items) || !elemsEqual(a.exprs, b.exprs) {
		return false
	}
	if (a.expr != nil || b.expr != nil) && !elemEqual(a.expr, b.expr) {
		return false
	}
	if string(a.match) != string(b.match) || a.returnString != b.returnString || a.caseless != b.caseless ||
		a.caselessmatch != b.caselessmatch || !a.identChars.equal(b.identChars) || a.maxMismatches != b.maxMismatches ||
		!a.initChars.equal(b.initChars) || !a.bodyChars.equal(b.bodyChars) || a.minLen != b.minLen || a.maxLen != b.maxLen ||
		a.asKeyword != b.asKeyword || a.reString != b.reString || a.pattern != b.pattern || a.flags != b.flags ||
		a.notChars != b.notChars || string(a.matchWhite) != string(b.matchWhite) || a.col != b.col ||
		!a.wordChars.equal(b.wordChars) || a.exact != b.exact || a.retreat != b.retreat ||
		a.includeMatch != b.includeMatch || a.adjacent != b.adjacent || a.joinString != b.joinString ||
		a.asPythonList != b.asPythonList || a.asPythonDict != b.asPythonDict || a.mustSkip != b.mustSkip {
		return false
	}
	if (a.qs == nil) != (b.qs == nil) || (a.qs != nil && !quotedEq(a.qs, b.qs)) {
		return false
	}
	if (a.maxCount == nil) != (b.maxCount == nil) || (a.maxCount != nil && *a.maxCount != *b.maxCount) {
		return false
	}
	if (a.notEnder != nil || b.notEnder != nil) && !elemEqual(a.notEnder, b.notEnder) {
		return false
	}
	if (a.failOn != nil || b.failOn != nil) && !elemEqual(a.failOn, b.failOn) {
		return false
	}
	if (a.skipper != nil || b.skipper != nil) && !elemEqual(a.skipper, b.skipper) {
		return false
	}
	if (a.ignorer != nil || b.ignorer != nil) && !elemEqual(a.ignorer, b.ignorer) {
		return false
	}
	if a.dl != b.dl || a.each != b.each {
		return false
	}
	if !pyEqual(a.defaultValue, b.defaultValue) || !pyEqual(a.tagValue, b.tagValue) {
		return false
	}
	return true
}

func quotedEq(a, b *quotedData) bool {
	return string(a.quoteChar) == string(b.quoteChar) && string(a.endQuoteChar) == string(b.endQuoteChar) &&
		a.escChar == b.escChar && a.escQuote == b.escQuote && a.unquoteResults == b.unquoteResults &&
		a.convertWS == b.convertWS && a.multiline == b.multiline
}

func elemsEqual(a, b []*ParserElement) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !elemEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}

func strpEq(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

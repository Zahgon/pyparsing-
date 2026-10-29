// Port of pyparsing/core.py, per-class method dispatch (pyparsing 3.3.3+). MIT License, Copyright (c) 2003-2025  Paul McGuire; see LICENSE.

package pyparsing

import (
	"strings"
)

// preParse is the class's preParse.
func (e *ParserElement) preParse(st *pstate, loc int) (int, error) {
	switch e.cls {
	case clsGoToColumn:
		return e.goToColumnPreParse(st, loc)
	case clsLineStart:
		return e.lineStartPreParse(st, loc)
	}
	if len(e.ignoreExprs.items) > 0 {
		var err error
		loc, err = e.skipIgnorables(st, loc)
		if err != nil {
			return 0, err
		}
	}
	if e.skipWhitespace {
		n := len(st.runes)
		for loc < n && e.whiteChars[st.runes[loc]] {
			loc++
		}
	}
	return loc, nil
}

// skipIgnorables is _skipIgnorables.
func (e *ParserElement) skipIgnorables(st *pstate, loc int) (int, error) {
	if len(e.ignoreExprs.items) == 0 {
		return loc, nil
	}
	fns := append([]*ParserElement(nil), e.ignoreExprs.items...)
	exprsFound := true
	lastLoc := loc
	for exprsFound {
		exprsFound = false
		for _, ig := range fns {
			for {
				l, _, err := ig.parse(st, loc, true, true)
				if err != nil {
					if _, ok := asParseException(err); ok {
						break
					}
					return 0, err
				}
				loc = l
				exprsFound = true
			}
		}
		if loc == lastLoc {
			break
		}
		lastLoc = loc
	}
	return loc, nil
}

// parseImpl is the class's parseImpl.
func (e *ParserElement) parseImpl(st *pstate, loc int, doActions bool) (int, any, error) {
	switch e.cls {
	case clsNoMatch:
		return 0, nil, e.raise(st, loc)
	case clsLiteral, clsSingleCharLit:
		return e.literalParseImpl(st, loc)
	case clsEmpty, clsErrorStop:
		return loc, []any{}, nil
	case clsCaselessLiteral:
		return e.caselessLiteralParseImpl(st, loc)
	case clsKeyword, clsCaselessKeyword:
		return e.keywordParseImpl(st, loc)
	case clsCloseMatch:
		return e.closeMatchParseImpl(st, loc)
	case clsWord, clsChar:
		return e.wordParseImpl(st, loc)
	case clsRegex:
		return e.regexParseImpl(st, loc)
	case clsQuotedString:
		return e.quotedParseImpl(st, loc)
	case clsCharsNotIn:
		return e.charsNotInParseImpl(st, loc)
	case clsWhite:
		return e.whiteParseImpl(st, loc)
	case clsGoToColumn:
		return e.goToColumnParseImpl(st, loc)
	case clsLineStart:
		if colRunes(loc, st.runes) == 1 {
			return loc, []any{}, nil
		}
		return 0, nil, e.raise(st, loc)
	case clsLineEnd:
		return e.lineEndParseImpl(st, loc)
	case clsStringStart:
		return e.stringStartParseImpl(st, loc)
	case clsStringEnd:
		return e.stringEndParseImpl(st, loc)
	case clsWordStart:
		return e.wordStartParseImpl(st, loc)
	case clsWordEnd:
		return e.wordEndParseImpl(st, loc)
	case clsTag, clsToken, clsPositionToken:
		return loc, []any{}, nil
	case clsPendingSkip:
		return 0, nil, pyErr("Exception", "use of `...` expression without following SkipTo target expression")
	case clsAnd:
		return e.andParseImpl(st, loc, doActions)
	case clsOr:
		return e.orParseImpl(st, loc, doActions)
	case clsMatchFirst:
		return e.matchFirstParseImpl(st, loc, doActions)
	case clsEach:
		return e.eachParseImpl(st, loc, doActions)
	case clsAtStringStart:
		if loc != 0 {
			return 0, nil, newParseExc(st.str(), loc, "not found at string start", nil)
		}
		return e.enhanceParseImpl(st, loc, doActions)
	case clsAtLineStart:
		if colRunes(loc, st.runes) != 1 {
			return 0, nil, newParseExc(st.str(), loc, "not found at line start", nil)
		}
		return e.enhanceParseImpl(st, loc, doActions)
	case clsFollowedBy:
		_, ret, err := e.expr.parse(st, loc, doActions, true)
		if err != nil {
			return 0, nil, err
		}
		ret.delAll()
		return loc, ret, nil
	case clsFB:
		if _, err := e.expr.tryParse(st, loc, false, false); err != nil {
			return 0, nil, err
		}
		return loc, []any{}, nil
	case clsPrecededBy:
		return e.precededByParseImpl(st, loc, doActions)
	case clsLocated:
		return e.locatedParseImpl(st, loc, doActions)
	case clsNotAny:
		ok, err := e.expr.canParseNext(st, loc, doActions)
		if err != nil {
			return 0, nil, err
		}
		if ok {
			return 0, nil, e.raise(st, loc)
		}
		return loc, []any{}, nil
	case clsOneOrMore:
		return e.multipleMatchParseImpl(st, loc, doActions)
	case clsZeroOrMore:
		l, t, err := e.multipleMatchParseImpl(st, loc, doActions)
		if err != nil {
			if _, ok := asParseException(err); ok || isIndexError(err) {
				r, err2 := newPR([]any{}, e.resultsName, true, true)
				if err2 != nil {
					return 0, nil, err2
				}
				return loc, r, nil
			}
			return 0, nil, err
		}
		return l, t, nil
	case clsOpt:
		return e.optParseImpl(st, loc, doActions)
	case clsSkipTo:
		return e.skipToParseImpl(st, loc, doActions)
	case clsForward:
		return e.forwardParseImpl(st, loc, doActions)
	}
	if e.cls.isSub(clsEnhance) {
		return e.enhanceParseImpl(st, loc, doActions)
	}
	return loc, []any{}, nil
}

// postParse is the class's postParse.
func (e *ParserElement) postParse(st *pstate, loc int, tokens any) (any, error) {
	switch e.cls {
	case clsCombine:
		return e.combinePostParse(tokens)
	case clsGroup:
		if e.asPythonList {
			if pr, ok := tokens.(*ParseResults); ok {
				return prList(pr.AsList(false)), nil
			}
			items, _ := pyIterItems(tokens)
			return prList(append([]any{}, items...)), nil
		}
		return []any{tokens}, nil
	case clsDict:
		return e.dictPostParse(tokens)
	case clsSuppress:
		return []any{}, nil
	}
	return tokens, nil
}

// generateDefaultName is the class's _generateDefaultName.
func (e *ParserElement) generateDefaultName() string {
	switch e.cls {
	case clsPendingSkip:
		return strings.ReplaceAll(e.anchor.Add(Empty()).String(), "Empty", "...")
	case clsLiteral, clsSingleCharLit, clsCaselessLiteral, clsKeyword, clsCaselessKeyword:
		return Repr(string(e.match))
	case clsEmpty:
		return "Empty"
	case clsErrorStop:
		return "-"
	case clsCloseMatch:
		return e.cls.name + ":" + Repr(string(e.match))
	case clsWord, clsChar:
		return e.wordDefaultName()
	case clsRegex:
		return "Re:(" + strings.ReplaceAll(Repr(e.pattern), `\\`, `\`) + ")"
	case clsQuotedString:
		return e.quotedDefaultName()
	case clsCharsNotIn:
		return e.charsNotInDefaultName()
	case clsWhite:
		return e.whiteDefaultName()
	case clsTag:
		return e.cls.name + ":" + e.tagName + "=" + Repr(e.tagValue)
	case clsAnd:
		return "{" + stripBraces(e.liveJoin(" ")) + "}"
	case clsOr:
		return "{" + e.liveJoin(" ^ ") + "}"
	case clsMatchFirst:
		return "{" + e.liveJoin(" | ") + "}"
	case clsEach:
		return "{" + e.liveJoin(" & ") + "}"
	case clsNotAny:
		return "~{" + e.expr.String() + "}"
	case clsOneOrMore:
		return "{" + e.expr.String() + "}..."
	case clsZeroOrMore:
		return "[" + e.expr.String() + "]..."
	case clsDelimitedList:
		content := e.dl.content.streamline()
		return content.String() + " [" + e.dl.rawDelim + " " + content.String() + "]..."
	case clsOpt:
		return "[" + stripBraces(e.expr.String()) + "]"
	case clsForward:
		save := e.defaultName
		e.defaultName = strp(": ...")
		var ret string
		if e.expr != nil {
			ret = string(pySlice([]rune(e.expr.String()), 0, 1000))
		} else {
			ret = "None"
		}
		e.defaultName = save
		return e.cls.name + ": " + ret
	}
	if e.cls.isSub(clsToken) {
		return e.cls.name
	}
	if e.cls.isSub(clsParseExpression) {
		parts := make([]string, len(e.exprs))
		for i, x := range e.exprs {
			parts[i] = x.String()
		}
		return e.cls.name + ":([" + strings.Join(parts, ", ") + "])"
	}
	if e.cls.isSub(clsEnhance) {
		return e.cls.name + ":(" + e.expr.String() + ")"
	}
	return e.cls.name
}

// liveJoin is sep.join(str(e) for e in self.exprs): the list is read again for every item,
// as Python's list iterator does, since computing a name may streamline (and so rewrite)
// this very expression.
func (e *ParserElement) liveJoin(sep string) string {
	var b strings.Builder
	for i := 0; i < len(e.exprs); i++ {
		if i > 0 {
			b.WriteString(sep)
		}
		b.WriteString(e.exprs[i].String())
	}
	return b.String()
}

func joinNames(xs []*ParserElement, sep string) string {
	parts := make([]string, len(xs))
	for i, x := range xs {
		parts[i] = x.String()
	}
	return strings.Join(parts, sep)
}

// stripBraces strips matched outer '{' '}' pairs as And/Opt names do.
func stripBraces(inner string) string {
	rs := []rune(inner)
	for len(rs) > 1 && rs[0] == '{' && rs[len(rs)-1] == '}' {
		rs = rs[1 : len(rs)-1]
	}
	return string(rs)
}

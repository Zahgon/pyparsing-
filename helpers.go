// Port of pyparsing/helpers.py (pyparsing 3.3.3+). MIT License, Copyright (c) 2003-2025  Paul McGuire; see LICENSE.

package pyparsing

import (
	"strings"

	"github.com/pyparsing/pyparsing-go/internal/pyuni"
	"github.com/pyparsing/pyparsing-go/internal/pyval"
)

// Srange is srange(s): the characters of a regex-style [...] character set ("" on error).
func Srange(s string) string {
	res, err := reBracketExpr.ParseString(s, false)
	if err != nil {
		return ""
	}
	body := res.Attr("body")
	items, ok := pyIterItems(body)
	if !ok {
		if str, ok := body.(string); ok {
			return str
		}
		return ""
	}
	var b strings.Builder
	for _, part := range items {
		if pr, ok := part.(*ParseResults); ok {
			a, err1 := pr.At(0)
			z, err2 := pr.At(1)
			if err1 != nil || err2 != nil {
				return ""
			}
			as, ok1 := a.(string)
			zs, ok2 := z.(string)
			ar, zr := []rune(as), []rune(zs)
			if !ok1 || !ok2 || len(ar) != 1 || len(zr) != 1 {
				return ""
			}
			for c := ar[0]; c <= zr[0]; c++ {
				b.WriteRune(c)
			}
		} else {
			b.WriteString(Str(part))
		}
	}
	return b.String()
}

func srangeHexAction(s string, l int, t *ParseResults) (any, error) {
	v, err := t.At(0)
	if err != nil {
		return nil, err
	}
	str := pyval.LStrip(v.(string), `\0x`)
	n, err := pyval.ParseInt(str, 16)
	if err != nil {
		return nil, fromInternal(err)
	}
	return pyChr(n)
}

func srangeOctAction(s string, l int, t *ParseResults) (any, error) {
	v, err := t.At(0)
	if err != nil {
		return nil, err
	}
	n, err := pyval.ParseInt(string([]rune(v.(string))[1:]), 8)
	if err != nil {
		return nil, fromInternal(err)
	}
	return pyChr(n)
}

// OneOf is one_of(strs, caseless, use_regex, as_keyword); strs is a str (split on
// whitespace) or a []string / []any of str.
func OneOf(strs any, caseless, useRegex, asKeyword bool) (*ParserElement, error) {
	var symbols []string
	switch v := strs.(type) {
	case string:
		symbols = pyval.Split(v)
	case []string:
		symbols = append(symbols, v...)
	case []any:
		for _, x := range v {
			s, ok := x.(string)
			if !ok {
				return nil, pyErr("TypeError", "Invalid argument to one_of, expected string or iterable")
			}
			symbols = append(symbols, s)
		}
	case PyTuple:
		for _, x := range v {
			s, ok := x.(string)
			if !ok {
				return nil, pyErr("TypeError", "Invalid argument to one_of, expected string or iterable")
			}
			symbols = append(symbols, s)
		}
	default:
		return nil, pyErr("TypeError", "Invalid argument to one_of, expected string or iterable")
	}
	isEqual := func(a, b string) bool { return a == b }
	masks := func(a, b string) bool { return strings.HasPrefix(b, a) }
	if caseless {
		isEqual = func(a, b string) bool { return pyuni.Upper(a) == pyuni.Upper(b) }
		masks = func(a, b string) bool { return strings.HasPrefix(pyuni.Upper(b), pyuni.Upper(a)) }
	}
	if len(symbols) == 0 {
		return NoMatch(), nil
	}
	i := 0
	for i < len(symbols)-1 {
		cur := symbols[i]
		changed := false
		for j, other := range symbols[i+1:] {
			if isEqual(other, cur) {
				symbols = append(symbols[:i+j+1:i+j+1], symbols[i+j+2:]...)
				changed = true
				break
			}
			if runeLen(other) > runeLen(cur) && masks(cur, other) {
				symbols = append(symbols[:i+j+1:i+j+1], symbols[i+j+2:]...)
				n := make([]string, 0, len(symbols)+1)
				n = append(n, symbols[:i]...)
				n = append(n, other)
				n = append(n, symbols[i:]...)
				symbols = n
				changed = true
				break
			}
		}
		if !changed {
			i++
		}
	}
	if useRegex {
		flags := 0
		if caseless {
			flags = IGNORECASE
		}
		allOne := true
		for _, s := range symbols {
			if runeLen(s) != 1 {
				allOne = false
			}
		}
		var patt string
		if allOne {
			var b strings.Builder
			for _, s := range symbols {
				b.WriteString(escapeRegexRangeChars(s))
			}
			patt = "[" + b.String() + "]"
		} else {
			esc := make([]string, len(symbols))
			for k, s := range symbols {
				esc[k] = reEscape(s)
			}
			patt = strings.Join(esc, "|")
		}
		if asKeyword {
			patt = `\b(?:` + patt + `)\b`
		}
		ret, err := Regex(patt, flags, false, false)
		if err == nil {
			reprs := make([]string, len(symbols))
			for k, s := range symbols {
				reprs[k] = Repr(s)
			}
			ret.SetName(strings.Join(reprs, " | "))
			if caseless {
				symbolMap := map[string]string{}
				for _, s := range symbols {
					symbolMap[pyuni.Lower(s)] = s
				}
				ret.addPA(func(s string, l int, t *ParseResults) (any, error) {
					v, err := t.At(0)
					if err != nil {
						return nil, err
					}
					str, ok := v.(string)
					if !ok {
						return nil, pyErr("AttributeError", "'"+pyTypeName(v)+"' object has no attribute 'lower'")
					}
					r, ok := symbolMap[pyuni.Lower(str)]
					if !ok {
						return nil, pyErr("KeyError", Repr(pyuni.Lower(str)))
					}
					return r, nil
				})
			}
			return ret, nil
		}
	}
	elems := make([]any, 0, len(symbols))
	for _, s := range symbols {
		var x *ParserElement
		var err error
		switch {
		case caseless && asKeyword:
			x, err = CaselessKeyword(s, nil)
		case caseless:
			x = CaselessLiteral(s)
		case asKeyword:
			x, err = Keyword(s, nil, false)
		default:
			x = Literal(s)
		}
		if err != nil {
			return nil, err
		}
		elems = append(elems, x)
	}
	return MatchFirst(elems...).SetName(strings.Join(symbols, " | ")), nil
}

// CountedArray is counted_array(expr, int_expr).
func CountedArray(expr any, intExpr any) *ParserElement {
	arrayExpr := Forward()
	countField := func(s string, l int, t *ParseResults) (any, error) {
		n, err := t.At(0)
		if err != nil {
			return nil, err
		}
		if pyTruth(n) {
			var x *ParserElement
			switch ex := expr.(type) {
			case string:
				k, ok := asInt(n)
				if !ok {
					return nil, pyErr("TypeError", "can't multiply sequence by non-int of type '"+pyTypeName(n)+"'")
				}
				if k < 0 {
					k = 0
				}
				x = literalOf(strings.Repeat(ex, k))
			case *ParserElement:
				x, err = ex.Mul(n)
				if err != nil {
					return nil, err
				}
			}
			arrayExpr.lshift(x)
		} else {
			arrayExpr.lshift(Empty())
		}
		t.delAll()
		return nil, nil
	}
	var ie *ParserElement
	if intExpr == nil {
		w, _ := Word(Nums, nil, 1, 0, 0, false, nil)
		ie = w.setPA(func(s string, l int, t *ParseResults) (any, error) {
			v, err := t.At(0)
			if err != nil {
				return nil, err
			}
			return pyInt(v)
		})
	} else {
		ie = toElement(intExpr).copy()
	}
	ie.SetName("arrayLen")
	ie.AddParseAction(true, countField)
	return ie.Add(arrayExpr).SetName("(len) " + Str(expr) + "...")
}

// MatchPreviousLiteral is match_previous_literal(expr).
func MatchPreviousLiteral(expr any) *ParserElement {
	e := toElement(expr)
	rep := Forward()
	e.AddParseAction(true, func(s string, l int, t *ParseResults) (any, error) {
		if !t.Truth() {
			rep.lshift(Empty())
			return nil, nil
		}
		if t.Len() == 1 {
			v, _ := t.At(0)
			if rep.lshift(v) == nil {
				return nil, pyErr("TypeError", "unsupported operand type(s) for <<: 'Forward' and '"+pyTypeName(v)+"'")
			}
			return nil, nil
		}
		tflat := flattenList(t.AsList(false))
		lits := make([]any, 0, len(tflat))
		for _, tt := range tflat {
			s, ok := tt.(string)
			if !ok {
				return nil, pyErr("TypeError", "object of type '"+pyTypeName(tt)+"' has no len()")
			}
			lits = append(lits, Literal(s))
		}
		rep.lshift(newAnd(lits, true))
		return nil, nil
	})
	rep.SetName("(prev) " + e.String())
	return rep
}

// MatchPreviousExpr is match_previous_expr(expr).
func MatchPreviousExpr(expr any) *ParserElement {
	e := toElement(expr)
	rep := Forward()
	e2 := e.copy()
	rep.lshift(e2)
	type entry struct {
		loc  int
		toks []any
	}
	var matchStack []entry
	var lastMatch *entry
	copyToken := func(s string, l int, t *ParseResults) (any, error) {
		for len(matchStack) > 0 && matchStack[len(matchStack)-1].loc >= l {
			matchStack = matchStack[:len(matchStack)-1]
		}
		matchStack = append(matchStack, entry{l, flattenList(t.AsList(false))})
		lastMatch = nil
		return nil, nil
	}
	mustMatch := func(s string, l int, t *ParseResults) (any, error) {
		these := flattenList(t.AsList(false))
		if lastMatch != nil && lastMatch.loc == l && pyEqual(lastMatch.toks, these) {
			return nil, nil
		}
		if len(matchStack) == 0 {
			return nil, newParseExc(s, l, "no previous expression to match", nil)
		}
		matchTokens := matchStack[len(matchStack)-1].toks
		if !pyEqual(these, matchTokens) {
			return nil, newParseExc(s, l, "Expected "+Repr(matchTokens)+", found"+Repr(these), nil)
		}
		matchStack = matchStack[:len(matchStack)-1]
		lastMatch = &entry{l, these}
		return nil, nil
	}
	e2.AddParseAction(true, mustMatch)
	e.AddParseAction(true, copyToken)
	rep.SetName("(prev) " + e.String())
	return rep
}

// addAnyAny is a + b for two `any` operands.
func addAnyAny(a, b any) *ParserElement {
	if ae, ok := a.(*ParserElement); ok {
		return ae.Add(b)
	}
	if be, ok := b.(*ParserElement); ok {
		return be.RAdd(a)
	}
	return literalOf(Str(a)).Add(b)
}

// DictOf is dict_of(key, value).
func DictOf(key, value any) *ParserElement {
	return Dict(oneOrMore(Group(addAnyAny(key, value), false)), false)
}

// OriginalTextFor is original_text_for(expr, as_string).
func OriginalTextFor(expr any, asString bool) *ParserElement {
	e := toElement(expr)
	locMarker := Empty().setPA(func(s string, loc int, t *ParseResults) (any, error) { return bigInt(loc), nil })
	endlocMarker := locMarker.copy()
	endlocMarker.callPreparse = false
	matchExpr := locMarker.Call("_original_start").Add(e).Add(endlocMarker.Call("_original_end"))
	if asString {
		matchExpr.setPA(func(s string, l int, t *ParseResults) (any, error) {
			rs := runesOf(s)
			a, err := sliceIndex(t.Attr("_original_start"), 0)
			if err != nil {
				return nil, err
			}
			b, err := sliceIndex(t.Attr("_original_end"), len(rs))
			if err != nil {
				return nil, err
			}
			return string(pySlice(rs, a, b)), nil
		})
	} else {
		matchExpr.setPA(func(s string, l int, t *ParseResults) (any, error) {
			av, err := t.Pop("_original_start")
			if err != nil {
				return nil, err
			}
			bv, err := t.Pop("_original_end")
			if err != nil {
				return nil, err
			}
			rs := runesOf(s)
			a, err := sliceIndex(av, 0)
			if err != nil {
				return nil, err
			}
			b, err := sliceIndex(bv, len(rs))
			if err != nil {
				return nil, err
			}
			t.toks.items = []any{string(pySlice(rs, a, b))}
			return nil, nil
		})
	}
	matchExpr.ignoreExprs = e.ignoreExprs
	return matchExpr
}

// sliceIndex converts a Python slice bound (an int, or None for the default).
func sliceIndex(v any, def int) (int, error) {
	if v == nil {
		return def, nil
	}
	if n, ok := asInt(v); ok {
		return n, nil
	}
	return 0, pyErr("TypeError", "slice indices must be integers or None or have an __index__ method")
}

var runesCacheS string
var runesCacheR []rune

// runesOf converts a str to code points, remembering the last conversion.
func runesOf(s string) []rune {
	if s == runesCacheS && runesCacheR != nil {
		return runesCacheR
	}
	runesCacheS, runesCacheR = s, []rune(s)
	return runesCacheR
}

// Ungroup is ungroup(expr).
func Ungroup(expr any) *ParserElement {
	return newTokenConverter(clsTokenConverter, expr).addPA(func(s string, l int, t *ParseResults) (any, error) {
		return t.At(0)
	})
}

func suppression(expr any) *ParserElement {
	if x, ok := expr.(*ParserElement); ok && x.isa(clsSuppress) {
		return x
	}
	return Suppress(expr)
}

// anyEq is opener == closer for str or element operands.
func anyEq(a, b any) bool {
	as, aStr := a.(string)
	bs, bStr := b.(string)
	ae, aEl := a.(*ParserElement)
	be, bEl := b.(*ParserElement)
	switch {
	case aStr && bStr:
		return as == bs
	case aEl && bEl:
		return elemEqual(ae, be)
	case aEl && bStr:
		ok, _ := ae.Matches(bs, true)
		return ok
	case aStr && bEl:
		ok, _ := be.Matches(as, true)
		return ok
	}
	return a == b
}

// NestedExpr is nested_expr(opener, closer, content, ignore_expr); omitting ignoreExpr uses
// the default (quoted_string()), nil means None.
func NestedExpr(opener, closer any, content any, ignoreExpr ...any) (*ParserElement, error) {
	var ign *ParserElement
	if len(ignoreExpr) == 0 {
		ign = quotedString.copy()
	} else {
		ign = toElement(ignoreExpr[0])
	}
	if anyEq(opener, closer) {
		return nil, pyErr("ValueError", "opening and closing strings cannot be the same")
	}
	var cont *ParserElement
	if content == nil {
		os, ok1 := opener.(string)
		cs, ok2 := closer.(string)
		if !ok1 || !ok2 {
			return nil, pyErr("ValueError", "opening and closing arguments must be strings if no content expression is given")
		}
		dw := defaultWhiteChars
		if runeLen(os) == 1 && runeLen(cs) == 1 {
			if ign != nil {
				cont = Combine(oneOrMore(ign.Invert().Add(mustCharsNotIn(os+cs+dw, 1))), "", true)
			} else {
				c, _ := CharsNotIn(os+cs+dw, 1, 0, 0)
				cont = Combine(Empty().Add(c), "", true)
			}
		} else if ign != nil {
			cont = Combine(oneOrMore(ign.Invert().Add(Literal(os).Invert()).Add(Literal(cs).Invert()).Add(mustCharsNotIn(dw, 1))), "", true)
		} else {
			cont = Combine(oneOrMore(Literal(os).Invert().Add(Literal(cs).Invert()).Add(mustCharsNotIn(dw, 1))), "", true)
		}
		if defaultWhiteChars != "" {
			cont.setPA(func(s string, l int, t *ParseResults) (any, error) {
				v, err := t.At(0)
				if err != nil {
					return nil, err
				}
				str, ok := v.(string)
				if !ok {
					return nil, pyErr("AttributeError", "'"+pyTypeName(v)+"' object has no attribute 'strip'")
				}
				return pyval.Strip(str, defaultWhiteChars), nil
			})
		}
	} else {
		cont = toElement(content)
	}
	ret := Forward()
	if ign != nil {
		ret.lshift(Group(suppression(opener).Add(zeroOrMore(ign.BitOr(ret).BitOr(cont))).Add(suppression(closer)), false))
	} else {
		ret.lshift(Group(suppression(opener).Add(zeroOrMore(ret.BitOr(cont))).Add(suppression(closer)), false))
	}
	ret.SetName("nested " + Str(opener) + Str(closer) + " expression")
	ret.errmsg = nil
	return ret, nil
}

var suppressLT, suppressGT, infixLpar, infixRpar *ParserElement

func titleJoin(resname string) string {
	return strings.Join(pyval.Split(pyval.Title(strings.ReplaceAll(resname, ":", " "))), "")
}

func makeTags(tagStr any, xml bool) (*ParserElement, *ParserElement) {
	var resname string
	var tag *ParserElement
	if s, ok := tagStr.(string); ok {
		resname = s
		k, err := Keyword(s, nil, !xml)
		if err != nil {
			return nil, nil
		}
		tag = k
	} else {
		tag = toElement(tagStr)
		resname = tag.name()
	}
	attrBody := Alphanums + "_-:"
	tagAttrName, _ := Word(Alphas, &attrBody, 1, 0, 0, false, nil)
	emptyFlag := func(s string, l int, t *ParseResults) (any, error) {
		v, err := t.At(0)
		if err != nil {
			return nil, err
		}
		return pyEqual(v, "/"), nil
	}
	var openTag *ParserElement
	if xml {
		tagAttrValue := dblQuotedString.copy().setPA(RemoveQuotes)
		openTag = suppressLT.Add(tag.Call("tag")).
			Add(Dict(zeroOrMore(Group(tagAttrName.Add(Suppress("=")).Add(tagAttrValue), false)), false)).
			Add(Opt("/", []any{false}).Call("empty").setPA(emptyFlag)).
			Add(suppressGT)
	} else {
		ex := ">"
		w, _ := Word(Printables, nil, 1, 0, 0, false, &ex)
		tagAttrValue := quotedString.copy().setPA(RemoveQuotes).BitOr(w)
		lowerName := tagAttrName.setPA(func(s string, l int, t *ParseResults) (any, error) {
			v, err := t.At(0)
			if err != nil {
				return nil, err
			}
			str, ok := v.(string)
			if !ok {
				return nil, pyErr("AttributeError", "'"+pyTypeName(v)+"' object has no attribute 'lower'")
			}
			return pyuni.Lower(str), nil
		})
		openTag = suppressLT.Add(tag.Call("tag")).
			Add(Dict(zeroOrMore(Group(lowerName.Add(Opt(Suppress("=").Add(tagAttrValue))), false)), false)).
			Add(Opt("/", []any{false}).Call("empty").setPA(emptyFlag)).
			Add(suppressGT)
	}
	closeTag := Combine(Literal("</").Add(tag).Add(">"), "", false)
	openTag.SetName("<" + resname + ">")
	startName := "start" + titleJoin(resname)
	openTag.addPA(func(s string, l int, t *ParseResults) (any, error) {
		t.setItemStr(startName, t.Copy())
		return nil, nil
	})
	closeTag = closeTag.Call("end" + titleJoin(resname)).SetName("</" + resname + ">")
	return openTag, closeTag
}

// MakeHTMLTags is make_html_tags(tag_str).
func MakeHTMLTags(tagStr any) (*ParserElement, *ParserElement) { return makeTags(tagStr, false) }

// MakeXMLTags is make_xml_tags(tag_str).
func MakeXMLTags(tagStr any) (*ParserElement, *ParserElement) { return makeTags(tagStr, true) }

// ReplaceHTMLEntity is replace_html_entity: the character(s) of the entity matched by
// common_html_entity.
func ReplaceHTMLEntity(s string, loc int, toks *ParseResults) (any, error) {
	k, ok := toks.Attr("entity").(string)
	if !ok {
		return nil, nil
	}
	if v, ok := htmlEntityMap[k]; ok {
		return v, nil
	}
	return nil, nil
}

var htmlEntityMap map[string]string

// OpAssoc is pyparsing's OpAssoc.
type OpAssoc int

// The associativities.
const (
	OpAssocLeft  OpAssoc = 1
	OpAssocRight OpAssoc = 2
)

// InfixOp is one entry of infix_notation's op_list.
type InfixOp struct {
	Expr   any // an expression, or a PyTuple of two for a ternary operator
	Arity  int
	Assoc  OpAssoc
	Action ParseAction // nil when the entry has no parse action
}

// InfixNotation is infix_notation(base_expr, op_list, lpar, rpar); nil lpar/rpar use the
// defaults Suppress("(") and Suppress(")").
func InfixNotation(baseExpr any, opList []InfixOp, lpar, rpar any) (*ParserElement, error) {
	base := toElement(baseExpr)
	if lpar == nil {
		lpar = infixLpar
	}
	if rpar == nil {
		rpar = infixRpar
	}
	ret := Forward()
	ret.SetName(base.name() + "_expression")
	if s, ok := lpar.(string); ok {
		lpar = Suppress(s)
	}
	if s, ok := rpar.(string); ok {
		rpar = Suppress(s)
	}
	lp, rp := toElement(lpar), toElement(rpar)
	nested := lp.Add(ret).Add(rp).SetName("nested_" + base.name() + "_expression")
	var lastExpr *ParserElement
	if !(lp.isa(clsSuppress) && rp.isa(clsSuppress)) {
		lastExpr = base.BitOr(Group(nested, false))
	} else {
		lastExpr = base.BitOr(nested)
	}
	gi := func(x *ParserElement, a, b any) *ParserElement {
		r, _ := x.GetItem(PyTuple{a, b})
		return r
	}
	for _, od := range opList {
		opExprAny := od.Expr
		if s, ok := opExprAny.(string); ok {
			opExprAny = literalOf(s)
		}
		arity := od.Arity
		var termName string
		var op1, op2 any
		if arity == 3 {
			t, ok := opExprAny.(PyTuple)
			if !ok {
				if l, ok2 := opExprAny.([]any); ok2 {
					t, ok = PyTuple(l), true
				}
			}
			if !ok || len(t) != 2 {
				return nil, pyErr("ValueError", "if numterms=3, opExpr must be a tuple or list of two expressions")
			}
			op1, op2 = t[0], t[1]
			termName = Str(op1) + Str(op2) + " operations"
		} else {
			termName = Str(opExprAny) + " operations"
		}
		if arity < 1 || arity > 3 {
			return nil, pyErr("ValueError", "operator must be unary (1), binary (2), or ternary (3)")
		}
		if od.Assoc != OpAssocLeft && od.Assoc != OpAssocRight {
			return nil, pyErr("ValueError", "operator must indicate right or left associativity")
		}
		thisExpr := Forward().SetName(termName)
		opExpr := toElement(opExprAny)
		var matchLookahead, matchExpr *ParserElement
		if od.Assoc == OpAssocLeft {
			switch arity {
			case 1:
				matchLookahead = newFB(lastExpr.Add(opExpr))
				matchExpr = Group(lastExpr.Add(gi(opExpr, 1, Ellipsis)), false)
			case 2:
				if opExpr != nil {
					matchLookahead = newFB(lastExpr.Add(opExpr).Add(lastExpr))
					matchExpr = Group(lastExpr.Add(gi(opExpr.Add(lastExpr), 1, Ellipsis)), false)
				} else {
					matchLookahead = newFB(lastExpr.Add(lastExpr))
					matchExpr = Group(gi(lastExpr, 2, Ellipsis), false)
				}
			case 3:
				matchLookahead = newFB(lastExpr.Add(op1).Add(lastExpr).Add(op2).Add(lastExpr))
				matchExpr = Group(lastExpr.Add(gi(addAnyAny(op1, lastExpr).Add(op2).Add(lastExpr), 1, Ellipsis)), false)
			}
		} else {
			switch arity {
			case 1:
				if !opExpr.isa(clsOpt) {
					opExpr = Opt(opExpr)
				}
				matchLookahead = newFB(opExpr.expr.Add(thisExpr))
				matchExpr = Group(opExpr.Add(thisExpr), false)
			case 2:
				if opExpr != nil {
					matchLookahead = newFB(lastExpr.Add(opExpr).Add(thisExpr))
					matchExpr = Group(lastExpr.Add(gi(opExpr.Add(thisExpr), 1, Ellipsis)), false)
				} else {
					matchLookahead = newFB(lastExpr.Add(thisExpr))
					matchExpr = Group(lastExpr.Add(gi(thisExpr, 1, Ellipsis)), false)
				}
			case 3:
				matchLookahead = newFB(lastExpr.Add(op1).Add(thisExpr).Add(op2).Add(thisExpr))
				matchExpr = Group(lastExpr.Add(op1).Add(thisExpr).Add(op2).Add(thisExpr), false)
			}
		}
		matchExpr = matchLookahead.Add(matchExpr)
		if od.Action != nil {
			matchExpr.setPA(od.Action)
		}
		thisExpr.lshift(matchExpr.BitOr(lastExpr).SetName(termName))
		lastExpr = thisExpr
	}
	ret.lshift(lastExpr)
	return ret, nil
}

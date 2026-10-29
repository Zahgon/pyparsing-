// Port of pyparsing (tests of the port; pyparsing 3.3.3+). MIT License, Copyright (c) 2003-2025  Paul McGuire; see LICENSE.

package pyparsing

import (
	"strings"
	"testing"
)

// Expected values were recorded with CPython 3.12 and the original pyparsing.

func mustW(t *testing.T, init string, body *string) *ParserElement {
	t.Helper()
	w, err := Word(init, body, 1, 0, 0, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func eq(t *testing.T, label, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("%s:\n got %q\nwant %q", label, got, want)
	}
}

func excText(err error) string {
	if pe, ok := err.(*ParseException); ok {
		return pe.ClassName() + " " + pe.Error() + " | " + pe.Explain(16) + " | " + pe.MarkInputLine()
	}
	if pe, ok := err.(*PyError); ok {
		return "ERR " + pe.Class + " " + pe.Msg
	}
	if err == nil {
		return "<nil>"
	}
	return err.Error()
}

func TestNames(t *testing.T) {
	b := Alphanums + "_"
	eq(t, "word", mustW(t, Alphas, &b).String(), "W:(A-Za-z, 0-9A-Z_a-z)")
	ex := ","
	w2, _ := Word(Printables, nil, 1, 0, 0, false, &ex)
	eq(t, "word exclude", w2.String(), "W:(!-+--~)")
	c, _ := Char("abc", false, nil)
	eq(t, "char", c.String(), "(a-c)")
	end := ">>"
	q, _ := QuotedString("<<", nil, nil, false, true, &end, true)
	eq(t, "qs", q.String(), "quoted string, starting with << ending with >>")
	r, _ := Regex(`\d+\\x`, 0, false, false)
	eq(t, "regex", r.String(), `Re:('\d+\\x')`)
	eq(t, "white", White(" \t", 1, 0, 0).String(), "<SP><TAB>")
	w := mustW(t, Alphas, nil)
	g, _ := w.GetItem(PyTuple{2, 3})
	eq(t, "getitem", g.String(), "{{W:(A-Za-z) W:(A-Za-z)} [W:(A-Za-z)]...}")
	eq(t, "pending", w.Add(Ellipsis).Add(mustW(t, Nums, nil)).String(), "{{W:(A-Za-z) ...} W:(0-9)}")
}

func TestParsing(t *testing.T) {
	w, n := mustW(t, Alphas, nil), mustW(t, Nums, nil)
	r, err := w.Add(n).ParseString("abc 123", false)
	if err != nil {
		t.Fatal(err)
	}
	eq(t, "and", r.String(), "['abc', '123']")
	r, _ = w.Call("name").Add(n.Call("num")).ParseString("abc 123", false)
	eq(t, "dump", r.Dump("", true, true), "['abc', '123']\n- name: 'abc'\n- num: '123'")
	nums, _ := n.GetItem(PyTuple{1, Ellipsis})
	r, _ = w.Call("name").Add(n.Call("num")).Add(nums.Call("nums")).ParseString("abc 1 2 3", false)
	eq(t, "asdict", Repr(r.AsDict()), "{'name': 'abc', 'num': '1', 'nums': ['2', '3']}")
	_, err = w.Add(n).ParseString("abc def", false)
	eq(t, "err", excText(err), "ParseException Expected W:(0-9), found 'def'  (at char 4), (line:1, col:5) | abc def\n    ^\nParseException: Expected W:(0-9), found 'def'  (at char 4), (line:1, col:5) | abc >!<def")
	_, err = w.BitOr(n).ParseString("  .", false)
	eq(t, "mf err", excText(err), "ParseException Expected {W:(A-Za-z) | W:(0-9)}, found '.'  (at char 2), (line:1, col:3) |   .\n  ^\nParseException: Expected {W:(A-Za-z) | W:(0-9)}, found '.'  (at char 2), (line:1, col:3) | >!<.")
	_, err = w.ParseString("abc def", true)
	eq(t, "parse_all", excText(err), "ParseException Expected end of text, found 'def'  (at char 4), (line:1, col:5) | abc def\n    ^\nParseException: Expected end of text, found 'def'  (at char 4), (line:1, col:5) | abc >!<def")
	_, err = w.Sub(n).ParseString("abc x", false)
	eq(t, "fatal", excText(err), "ParseSyntaxException Expected W:(0-9), found 'x'  (at char 4), (line:1, col:5) | abc x\n    ^\nParseSyntaxException: Expected W:(0-9), found 'x'  (at char 4), (line:1, col:5) | abc >!<x")
	k, _ := Keyword("if", nil, false)
	_, err = k.ParseString("ifx", false)
	eq(t, "keyword", excText(err), "ParseException Expected Keyword 'if', keyword was immediately followed by keyword character, found 'x'  (at char 2), (line:1, col:3) | ifx\n  ^\nParseException: Expected Keyword 'if', keyword was immediately followed by keyword character, found 'x'  (at char 2), (line:1, col:3) | if>!<x")
	ck, _ := CaselessKeyword("if", nil)
	r, _ = ck.ParseString("IF x", false)
	eq(t, "ckeyword", r.String(), "['if']")
	r, _ = CloseMatch("abcd", nil, false).ParseString("abxd", false)
	eq(t, "closematch", r.Repr(), "ParseResults(['abxd'], {'original': 'abcd', 'mismatches': [2]})")
	r, _ = Located(w).ParseString("  abc", false)
	eq(t, "located", r.Repr(), "ParseResults([2, ParseResults(['abc'], {}), 5], {'locn_start': 2, 'value': ['abc'], 'locn_end': 5})")
	esc := `\`
	q, _ := QuotedString(`"`, &esc, nil, false, true, nil, true)
	r, _ = q.ParseString(`"a\"b\n"`, false)
	eq(t, "qs", r.String(), `['a"b\n']`)
	r, _ = Combine(n.Add(".").Add(n), "", true).ParseString("3.14", false)
	eq(t, "combine", r.String(), "['3.14']")
	r, _ = SkipTo(n, false, nil, nil).Add(n).ParseString("abc def 12", false)
	eq(t, "skipto", r.String(), "['abc def ', '12']")
}

func TestEachAndDict(t *testing.T) {
	a, b := Literal("a"), Literal("b")
	r, _ := Each(a, b, Opt("c")).ParseString("b c a", false)
	eq(t, "each", r.String(), "['b', 'c', 'a']")
	_, err := Each(a, b).ParseString("b", false)
	eq(t, "each err", excText(err), "ParseException Missing one or more required elements ('a'), found 'b'  (at char 0), (line:1, col:1) | b\n^\nParseException: Missing one or more required elements ('a'), found 'b'  (at char 0), (line:1, col:1) | >!<b")
	w, n := mustW(t, Alphas, nil), mustW(t, Nums, nil)
	oom, _ := OneOrMore(Group(w.Add(n), false), nil, nil)
	r, _ = Dict(oom, false).ParseString("a 1 b 2", false)
	eq(t, "dict", r.Dump("", true, true), "[['a', '1'], ['b', '2']]\n- a: '1'\n- b: '2'\n[0]:\n  ['a', '1']\n[1]:\n  ['b', '2']")
}

func TestHelpersAndCommon(t *testing.T) {
	ops := []InfixOp{}
	md, _ := OneOf("* /", false, true, false)
	pm, _ := OneOf("+ -", false, true, false)
	ops = append(ops, InfixOp{md, 2, OpAssocLeft, nil}, InfixOp{pm, 2, OpAssocLeft, nil})
	inf, err := InfixNotation(Common.Integer, ops, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	r, _ := inf.ParseString("1+2*3", false)
	eq(t, "infix", r.String(), "[[1, '+', [2, '*', 3]]]")
	ne, _ := NestedExpr("(", ")", nil)
	r, _ = ne.ParseString("(a (b c) 'd)')", false)
	eq(t, "nested", r.String(), `[['a', ['b', 'c'], "'d)'"]]`)
	eq(t, "srange", Srange(`[a-dx\x41-\x43]`), "abcdxABC")
	eq(t, "alphas8bit", Alphas8bit, "ÀÁÂÃÄÅÆÇÈÉÊËÌÍÎÏÐÑÒÓÔÕÖØÙÚÛÜÝÞßàáâãäåæçèéêëìíîïðñòóôõöøùúûüýþÿ")
	if len([]rune(Identchars)) != 118 || len([]rune(Identbodychars)) != 129 {
		t.Errorf("identchars %d identbodychars %d", len([]rune(Identchars)), len([]rune(Identbodychars)))
	}
	n := mustW(t, Nums, nil)
	dbl := n.Copy().AddParseAction(false, func(s string, l int, t *ParseResults) (any, error) {
		v, _ := t.At(0)
		i, _ := pyInt(v)
		f, _ := pyAdd(i, i)
		return Str(f), nil
	})
	out, _ := dbl.TransformString("a 12 b 3")
	eq(t, "transform", out, "a 24 b 6")
	parts, _ := Literal(",").Split("a,b,,c", 1<<62, true)
	eq(t, "split", Repr(parts), "['a', ',', 'b', ',', '', ',', 'c']")
	sr, _ := n.SearchString("a1b22c333", 2)
	eq(t, "search", sr.String(), "[['1'], ['22']]")
	r, _ = Common.Number.ParseString("6.02e23", false)
	eq(t, "number", r.String(), "[6.02e+23]")
	r, _ = Common.MixedInteger.ParseString("1 1/2", false)
	eq(t, "mixed", r.String(), "[1.5]")
	r, _ = Common.Iso8601Datetime.Copy().AddParseAction(false, ConvertToDatetime("%Y-%m-%dT%H:%M:%S.%f")).ParseString("2024-01-02T03:04:05.123456", false)
	v, _ := r.At(0)
	eq(t, "datetime", Repr(v), "datetime.datetime(2024, 1, 2, 3, 4, 5, 123456)")
	_, err = Common.Iso8601Date.Copy().AddParseAction(false, ConvertToDate("%Y-%m-%d")).ParseString("2023-02-29", false)
	eq(t, "bad date", strings.SplitN(excText(err), " |", 2)[0], "ParseException day is out of range for month, found '2023'  (at char 0), (line:1, col:1)")
}

func TestActionsAndErrors(t *testing.T) {
	n := mustW(t, Nums, nil)
	_, err := n.Copy().AddParseAction(false, func(s string, l int, t *ParseResults) (any, error) { return t.At(5) }).ParseString("1", false)
	eq(t, "pa index", excText(err), "ERR IndexError list index out of range")
	_, err = n.Copy().AddCondition(nil, false, false, func(s string, l int, t *ParseResults) bool { return false }).ParseString("1", false)
	eq(t, "cond none", strings.SplitN(excText(err), " |", 2)[0], "ParseException None, found '1'  (at char 0), (line:1, col:1)")
	_, err = n.Mul(-1)
	eq(t, "mul", excText(err), "ERR ValueError cannot multiply ParserElement by negative value")
	_, err = Regex("(a", 0, false, false)
	eq(t, "regex", excText(err), "ERR ValueError invalid pattern ('(a') passed to Regex")
}

func TestResultsMethods(t *testing.T) {
	w, n := mustW(t, Alphas, nil), mustW(t, Nums, nil)
	r, _ := w.Call("x").Add(n).Add(n).ParseString("a 1 2", false)
	v, _ := r.Pop("x")
	eq(t, "pop name", Repr(v), "'a'")
	eq(t, "after", r.String(), "['a', '1', '2']")
	v, _ = r.Pop()
	eq(t, "pop", Repr(v), "'2'")
	eq(t, "after2", r.String(), "['a', '1']")
	r, _ = w.Call("x").Add(n.Call("y")).ParseString("a 1", false)
	r.Insert(0, "z")
	eq(t, "insert", r.Dump("", true, true), "['z', 'a', '1']\n- x: 'a'\n- y: '1'")
}

func TestPackratSharedNames(t *testing.T) {
	defer DisableMemoization()
	if err := EnablePackrat(nil, false); err != nil {
		t.Fatal(err)
	}
	w, n := mustW(t, Alphas, nil), mustW(t, Nums, nil)
	r, err := w.Call("x*").Add(n.Call("x*")).BitOr(w).ParseString("a 1", false)
	if err != nil {
		t.Fatal(err)
	}
	eq(t, "packrat", r.String(), "['a', '1']")
}

func TestLeftRecursion(t *testing.T) {
	defer DisableMemoization()
	if err := EnableLeftRecursion(nil, false); err != nil {
		t.Fatal(err)
	}
	if err := EnablePackrat(nil, false); err == nil || err.(*PyError).Msg != "Packrat and Bounded Recursion are not compatible" {
		t.Errorf("packrat after left recursion: %v", err)
	}
	n := mustW(t, Nums, nil)
	expr := Forward()
	expr.ILShift(expr.Add("+").Add(n).BitOr(n))
	r, err := expr.ParseString("1+2+3", false)
	if err != nil {
		t.Fatal(err)
	}
	eq(t, "lr", r.String(), "['1', '+', '2', '+', '3']")
	eq(t, "name", expr.String(), "Forward: {{{: ... '+'} W:(0-9)} | W:(0-9)}")
}

func TestQuotedStringEscapes(t *testing.T) {
	esc := `\`
	q, _ := QuotedString("`", &esc, nil, false, true, nil, true)
	// the original's rf-string makes these patterns \\[0-7]3, \\x[0-9a-fA-F]2 and \\u[0-9a-fA-F]4
	for in, want := range map[string]string{"`a\\u0041`": "['au0041']", "`a\\x412`": "['ax412']", "`a\\x42`": "['aB']", "`a\\t`": "['a\\t']"} {
		r, err := q.ParseString(in, false)
		if err != nil {
			t.Fatal(err)
		}
		eq(t, in, r.String(), want)
	}
}

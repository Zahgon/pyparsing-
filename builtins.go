// Port of the module-level constants and expressions of pyparsing/core.py and helpers.py (pyparsing 3.3.3+). MIT License, Copyright (c) 2003-2025  Paul McGuire; see LICENSE.

package pyparsing

// Module constants.
const (
	Alphas    = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	Nums      = "0123456789"
	Hexnums   = "0123456789ABCDEFabcdef"
	Alphanums = Alphas + Nums
)

// Module character-set variables (printables, alphas8bit, punc8bit, identchars, identbodychars).
var (
	Printables     string
	Alphas8bit     string
	Punc8bit       string
	Identchars     string
	Identbodychars string
)

// PredefinedExprs holds the module-level expressions of core.py and helpers.py.
type PredefinedExprs struct {
	Empty, LineStart, LineEnd, StringStart, StringEnd                                 *ParserElement
	DblQuotedString, SglQuotedString, QuotedString, PythonQuotedString, UnicodeString *ParserElement
	CStyleComment, HTMLComment, RestOfLine, DblSlashComment, CppStyleComment          *ParserElement
	JavaStyleComment, PythonStyleComment, AnyOpenTag, AnyCloseTag, CommonHTMLEntity   *ParserElement
}

// Exprs is the set of module-level expressions.
var Exprs PredefinedExprs

// Regex flags, as in Python's re module.
const (
	IGNORECASE = 2
	MULTILINE  = 8
	DOTALL     = 16
	UNICODE    = 32
	VERBOSE    = 64
	ASCII      = 256
)

var (
	reBracketExpr                 *ParserElement
	dblQuotedString, quotedString *ParserElement
	anyOpenTag, anyCloseTag       *ParserElement
	noIgnoreExprGiven             *ParserElement
)

func init() {
	Printables = "!\"#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_`abcdefghijklmnopqrstuvwxyz{|}~"
	Identchars = latin1Identchars()
	Identbodychars = latin1Identbodychars(Identchars)
	htmlEntityMap = make(map[string]string, len(htmlEntityKeys))
	for i, k := range htmlEntityKeys {
		htmlEntityMap[k] = htmlEntityValues[i]
	}

	var core []*ParserElement
	keep := func(e *ParserElement) *ParserElement { core = append(core, e); return e }
	x := &Exprs
	x.Empty = keep(Empty().SetName("empty"))
	x.LineStart = keep(LineStart().SetName("line_start"))
	x.LineEnd = keep(LineEnd().SetName("line_end"))
	x.StringStart = keep(StringStart().SetName("string_start"))
	x.StringEnd = keep(StringEnd().SetName("string_end"))
	escapedPunc := keep(mustRegex(patEscapedPunc, 0).setPA(func(s string, l int, t *ParseResults) (any, error) {
		v, err := t.At(0)
		if err != nil {
			return nil, err
		}
		return pyGetItem(v, 1)
	}))
	escapedHex := keep(mustRegex(patEscapedHexChar, 0).setPA(srangeHexAction))
	escapedOct := keep(mustRegex(patEscapedOctChar, 0).setPA(srangeOctAction))
	singleChar := keep(escapedPunc.BitOr(escapedHex).BitOr(escapedOct).BitOr(mustCharsNotIn(`\]`, 1)))
	charRange := keep(Group(singleChar.Add(Suppress("-")).Add(singleChar), false))
	reBracketExpr = keep(Literal("[").Add(Opt("^").SetResultsName("negate", false)).
		Add(Group(oneOrMore(charRange.BitOr(singleChar)), false).SetResultsName("body", false)).Add(Literal("]")))
	x.DblQuotedString = keep(mustRegex(patDblQuoted, 0).SetName("string enclosed in double quotes"))
	dblQuotedString = x.DblQuotedString
	x.SglQuotedString = keep(mustRegex(patSglQuoted, 0).SetName("string enclosed in single quotes"))
	x.QuotedString = keep(Combine(mustRegex(patQS1, 0).SetName("double quoted string").BitOr(
		mustRegex(patQS2, 0).SetName("single quoted string")), "", true).SetName("quoted string using single or double quotes"))
	quotedString = x.QuotedString
	x.PythonQuotedString = keep(Combine(mustRegex(patPyQS1, MULTILINE).SetName("multiline double quoted string").BitXor(
		mustRegex(patPyQS2, MULTILINE).SetName("multiline single quoted string")).BitXor(
		mustRegex(patPyQS3, 0).SetName("double quoted string")).BitXor(
		mustRegex(patPyQS4, 0).SetName("single quoted string")), "", true).SetName("Python quoted string"))
	x.UnicodeString = keep(Combine(quotedString.copy().RAdd("u"), "", true).SetName("unicode string literal"))
	Alphas8bit = Srange(`[\0xc0-\0xd6\0xd8-\0xf6\0xf8-\0xff]`)
	Punc8bit = Srange(`[\0xa1-\0xbf\0xd7\0xf7]`)

	// helpers.py
	var helpers []*ParserElement
	hkeep := func(e *ParserElement) *ParserElement { helpers = append(helpers, e); return e }
	suppressLT = Suppress("<")
	suppressGT = Suppress(">")
	infixLpar = Suppress("(")
	infixRpar = Suppress(")")
	tagBody := Alphanums + "_:"
	anyTag, _ := Word(Alphas, &tagBody, 1, 0, 0, false, nil)
	anyOpenTag, anyCloseTag = MakeHTMLTags(anyTag.SetName("any tag"))
	x.AnyOpenTag, x.AnyCloseTag = hkeep(anyOpenTag), hkeep(anyCloseTag)
	most := "nbsp|lt|gt|amp|quot|apos|cent|pound|euro|copy"
	compressed, _ := makeCompressedRe(htmlEntityKeys, 2, true, 1)
	x.CommonHTMLEntity = hkeep(mustRegex("&(?P<entity>"+most+"|"+compressed+");", 0).SetName("common HTML entity"))
	noIgnoreExprGiven = hkeep(NoMatch())
	x.CStyleComment = hkeep(mustRegex(patCStyle, 0).SetName("C style comment"))
	x.HTMLComment = hkeep(mustRegex(patHTMLComment, 0).SetName("HTML comment"))
	x.RestOfLine = hkeep(mustRegex(patRestOfLine, 0).leaveWhitespace(true).SetName("rest of line"))
	x.DblSlashComment = hkeep(mustRegex(patDblSlash, 0).SetName("// comment"))
	x.CppStyleComment = hkeep(mustRegex(patCppStyle, 0).SetName("C++ style comment"))
	x.JavaStyleComment = x.CppStyleComment
	x.PythonStyleComment = hkeep(mustRegex(patPyStyle, 0).SetName("Python style comment"))
	// helpers' vars() also holds core's public expressions (imported with *)
	helpers = append(helpers, x.Empty, x.LineStart, x.LineEnd, x.StringStart, x.StringEnd,
		x.DblQuotedString, x.SglQuotedString, x.QuotedString, x.PythonQuotedString, x.UnicodeString)

	common := initCommon()
	builtinExprs = append(append(core, common...), helpers...)
}

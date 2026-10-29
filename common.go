// Port of pyparsing/common.py, pyparsing_common (pyparsing 3.3.3+). MIT License, Copyright (c) 2003-2025  Paul McGuire; see LICENSE.

package pyparsing

import (
	"math"
	"math/big"

	"github.com/pyparsing/pyparsing-go/internal/pyuni"
	"github.com/pyparsing/pyparsing-go/internal/pyval"
)

// pyInt is Python's int(v).
func pyInt(v any) (any, error) {
	switch x := v.(type) {
	case string:
		n, err := pyval.ParseInt(x, 10)
		if err != nil {
			return nil, fromInternal(err)
		}
		return n, nil
	case bool:
		return boolInt(x), nil
	case float64:
		n, err := pyval.FloatToInt(x)
		if err != nil {
			return nil, fromInternal(err)
		}
		return n, nil
	}
	if b, ok := toBigInt(v); ok {
		return new(big.Int).Set(b), nil
	}
	return nil, pyErr("TypeError", "int() argument must be a string, a bytes-like object or a real number, not '"+pyTypeName(v)+"'")
}

// pyIntBase is int(v, base).
func pyIntBase(v any, base int) (any, error) {
	s, ok := v.(string)
	if !ok {
		return nil, pyErr("TypeError", "int() can't convert non-string with explicit base")
	}
	n, err := pyval.ParseInt(s, base)
	if err != nil {
		return nil, fromInternal(err)
	}
	return n, nil
}

// pyFloat is Python's float(v).
func pyFloat(v any) (any, error) {
	switch x := v.(type) {
	case string:
		f, err := pyval.ParseFloat(x)
		if err != nil {
			return nil, fromInternal(err)
		}
		return f, nil
	case float64:
		return x, nil
	case bool:
		if x {
			return 1.0, nil
		}
		return 0.0, nil
	}
	if b, ok := toBigInt(v); ok {
		f, _ := new(big.Float).SetInt(b).Float64()
		if math.IsInf(f, 0) {
			return nil, pyErr("OverflowError", "int too large to convert to float")
		}
		return f, nil
	}
	return nil, pyErr("TypeError", "float() argument must be a string or a real number, not '"+pyTypeName(v)+"'")
}

// pyChr is chr(n).
func pyChr(n *big.Int) (any, error) {
	s, _, err := pyval.Chr(n)
	if err != nil {
		return nil, fromInternal(err)
	}
	return s, nil
}

// ConvertToInteger is pyparsing_common.convert_to_integer: [int(tt) for tt in t].
func ConvertToInteger(s string, loc int, toks *ParseResults) (any, error) {
	out := make([]any, 0, toks.Len())
	for _, tt := range append([]any(nil), toks.toks.items...) {
		v, err := pyInt(tt)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// ConvertToFloat is pyparsing_common.convert_to_float: [float(tt) for tt in t].
func ConvertToFloat(s string, loc int, toks *ParseResults) (any, error) {
	out := make([]any, 0, toks.Len())
	for _, tt := range append([]any(nil), toks.toks.items...) {
		v, err := pyFloat(tt)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func strptimeArg(t *ParseResults) (string, error) {
	v, err := t.At(0)
	if err != nil {
		return "", err
	}
	str, ok := v.(string)
	if !ok {
		return "", pyErr("TypeError", "strptime() argument 1 must be str, not "+pyTypeName(v))
	}
	return str, nil
}

// ConvertToDate is pyparsing_common.convert_to_date(fmt).
func ConvertToDate(fmt string) ParseAction {
	return func(ss string, ll int, tt *ParseResults) (any, error) {
		arg, err := strptimeArg(tt)
		if err != nil {
			return nil, err
		}
		d, err := pyval.StrptimeDate(arg, fmt)
		if err != nil {
			if pe, ok := err.(*pyval.Error); ok && pe.Class == "ValueError" {
				return nil, newParseExc(ss, ll, pe.Msg, nil)
			}
			return nil, fromInternal(err)
		}
		return Date{d.Year, d.Month, d.Day}, nil
	}
}

// ConvertToDatetime is pyparsing_common.convert_to_datetime(fmt). An aware result (%z) is
// returned without its tzinfo.
func ConvertToDatetime(fmt string) ParseAction {
	return func(s string, l int, t *ParseResults) (any, error) {
		arg, err := strptimeArg(t)
		if err != nil {
			return nil, err
		}
		d, err := pyval.Strptime(arg, fmt)
		if err != nil {
			if pe, ok := err.(*pyval.Error); ok && pe.Class == "ValueError" {
				return nil, newParseExc(s, l, pe.Msg, nil)
			}
			return nil, fromInternal(err)
		}
		return DateTime{d.Year, d.Month, d.Day, d.Hour, d.Minute, d.Second, d.Microsecond}, nil
	}
}

// orValue is `v or default`.
func orValue(v any, def any) any {
	if pyTruth(v) {
		return v
	}
	return def
}

// AsDatetime is pyparsing_common.as_datetime (for the iso8601 expressions' named groups).
func AsDatetime(s string, loc int, toks *ParseResults) (any, error) {
	l, t := loc, toks
	yv := t.Attr("year")
	ys, ok := yv.(string)
	if !ok {
		return nil, pyErr("AttributeError", "'"+pyTypeName(yv)+"' object has no attribute 'lstrip'")
	}
	year, err := pyInt(orValue(pyval.LStrip(ys, "0"), bigInt(0)))
	if err != nil {
		return nil, err
	}
	month, err := pyInt(orValue(t.Attr("month"), bigInt(1)))
	if err != nil {
		return nil, err
	}
	day, err := pyInt(orValue(t.Attr("day"), bigInt(1)))
	if err != nil {
		return nil, err
	}
	hour, err := pyInt(orValue(t.Attr("hour"), bigInt(0)))
	if err != nil {
		return nil, err
	}
	minute, err := pyInt(orValue(t.Attr("minute"), bigInt(0)))
	if err != nil {
		return nil, err
	}
	sv, err := pyFloat(orValue(t.Attr("second"), bigInt(0)))
	if err != nil {
		return nil, err
	}
	second := sv.(float64)
	isec, err := pyval.FloatToInt(second)
	if err != nil {
		return nil, fromInternal(err)
	}
	dt, err := pyval.NewDateTimeBig(year.(*big.Int), month.(*big.Int), day.(*big.Int), hour.(*big.Int), minute.(*big.Int), isec, big.NewInt(0))
	if err != nil {
		if pe, ok := err.(*pyval.Error); ok && pe.Class == "ValueError" {
			return nil, newParseExc(s, l, "Invalid date/time: "+pe.Msg, nil)
		}
		return nil, fromInternal(err)
	}
	frac, err := pyval.FloatMod(second, 1)
	if err != nil {
		return nil, fromInternal(err)
	}
	us, err := pyval.Round(frac * 1000000)
	if err != nil {
		return nil, fromInternal(err)
	}
	r, err := pyval.AddMicroseconds(dt, us)
	if err != nil {
		if pe, ok := err.(*pyval.Error); ok && pe.Class == "ValueError" {
			return nil, newParseExc(s, l, "Invalid date/time: "+pe.Msg, nil)
		}
		return nil, fromInternal(err)
	}
	return DateTime{r.Year, r.Month, r.Day, r.Hour, r.Minute, r.Second, r.Microsecond}, nil
}

// StripHTMLTags is pyparsing_common.strip_html_tags.
func StripHTMLTags(s string, loc int, toks *ParseResults) (any, error) {
	v, err := toks.At(0)
	if err != nil {
		return nil, err
	}
	str, ok := v.(string)
	if !ok {
		return nil, pyErr("TypeError", "expected str, got "+pyTypeName(v))
	}
	return htmlStripper.transformString(str)
}

func strMethod(tt any, name string, f func(string) string) (any, error) {
	switch x := tt.(type) {
	case string:
		return f(x), nil
	case *ParseResults:
		return nil, pyErr("TypeError", "'str' object is not callable")
	}
	return nil, pyErr("AttributeError", "'"+pyTypeName(tt)+"' object has no attribute '"+name+"'")
}

// UpcaseTokens is pyparsing_common.upcase_tokens.
func UpcaseTokens(s string, loc int, toks *ParseResults) (any, error) {
	out := []any{}
	for _, tt := range toks.toks.items {
		v, err := strMethod(tt, "upper", pyuni.Upper)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// DowncaseTokens is pyparsing_common.downcase_tokens.
func DowncaseTokens(s string, loc int, toks *ParseResults) (any, error) {
	out := []any{}
	for _, tt := range toks.toks.items {
		v, err := strMethod(tt, "lower", pyuni.Lower)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// CommonExprs holds the expressions of pyparsing_common.
type CommonExprs struct {
	Integer, HexInteger, SignedInteger, Fraction, MixedInteger, Real, SciReal, Number, Fnumber *ParserElement
	IeeeFloat, Identifier, Ipv4Address, Ipv6Address, MacAddress, Iso8601Date, Iso8601Datetime  *ParserElement
	UUID, CommaSeparatedList, URL                                                              *ParserElement
}

// Common is pyparsing_common.
var Common CommonExprs

var htmlStripper *ParserElement

func sumAction(s string, l int, t *ParseResults) (any, error) {
	var acc any = bigInt(0)
	for _, v := range t.toks.items {
		r, err := pyAdd(acc, v)
		if err != nil {
			return nil, err
		}
		acc = r
	}
	return acc, nil
}

// pyAdd is a + b for numbers.
func pyAdd(a, b any) (any, error) {
	af, aIsF := a.(float64)
	bf, bIsF := b.(float64)
	ai, aIsI := toBigInt(a)
	bi, bIsI := toBigInt(b)
	if ab, ok := a.(bool); ok {
		ai, aIsI = boolInt(ab), true
	}
	if bb, ok := b.(bool); ok {
		bi, bIsI = boolInt(bb), true
	}
	switch {
	case aIsI && bIsI:
		return new(big.Int).Add(ai, bi), nil
	case aIsF && bIsF:
		return af + bf, nil
	case aIsF && bIsI:
		f, _ := new(big.Float).SetInt(bi).Float64()
		return af + f, nil
	case aIsI && bIsF:
		f, _ := new(big.Float).SetInt(ai).Float64()
		return f + bf, nil
	}
	return nil, pyErr("TypeError", "unsupported operand type(s) for +: '"+pyTypeName(a)+"' and '"+pyTypeName(b)+"'")
}

func initCommon() []*ParserElement {
	var all []*ParserElement
	keep := func(e *ParserElement) *ParserElement { all = append(all, e); return e }
	c := &Common
	w, _ := Word(Nums, nil, 1, 0, 0, false, nil)
	c.Integer = keep(w.SetName("integer").setPA(ConvertToInteger))
	hw, _ := Word(Hexnums, nil, 1, 0, 0, false, nil)
	c.HexInteger = keep(hw.SetName("hex integer").setPA(TokenMap(func(tok any, args ...any) (any, error) { return pyIntBase(tok, 16) })))
	c.SignedInteger = keep(mustRegex(patSignedInteger, 0).SetName("signed integer").setPA(ConvertToInteger))
	c.Fraction = keep(c.SignedInteger.copy().setPA(ConvertToFloat).Add("/").Add(c.SignedInteger.copy().setPA(ConvertToFloat)).SetName("fraction"))
	c.Fraction.addPA(func(s string, l int, tt *ParseResults) (any, error) {
		a, err := tt.At(0)
		if err != nil {
			return nil, err
		}
		b, err := tt.At(-1)
		if err != nil {
			return nil, err
		}
		af, _ := a.(float64)
		bf, _ := b.(float64)
		if bf == 0 {
			return nil, pyErr("ZeroDivisionError", "float division by zero")
		}
		return af / bf, nil
	})
	c.MixedInteger = keep(c.Fraction.BitOr(c.SignedInteger.Add(Opt(Opt("-").Suppress().Add(c.Fraction)))).SetName("fraction or mixed integer-fraction"))
	c.MixedInteger.parseAction = append(c.MixedInteger.parseAction, &action{sumAction})
	c.Real = keep(mustRegex(patReal, 0).SetName("real number").setPA(ConvertToFloat))
	c.SciReal = keep(mustRegex(patSciReal, 0).SetName("real number with scientific notation").setPA(ConvertToFloat))
	c.Number = keep(c.SciReal.BitOr(c.Real).BitOr(c.SignedInteger).SetName("number").streamline())
	c.Fnumber = keep(mustRegex(patFnumber, 0).SetName("fnumber").setPA(ConvertToFloat))
	c.IeeeFloat = keep(mustRegex(patIeeeFloat, 0).SetName("ieee_float").setPA(ConvertToFloat))
	ib := Identbodychars
	idw, _ := Word(Identchars, &ib, 1, 0, 0, false, nil)
	c.Identifier = keep(idw.SetName("identifier"))
	c.Ipv4Address = keep(mustRegex(patIPv4, 0).SetName("IPv4 address"))
	ipv6Part := keep(mustRegex(patIPv6Part, 0).SetName("hex_integer"))
	m7, _ := ipv6Part.RAdd(":").Mul(7)
	full := keep(ipv6Part.Add(m7).SetName("full IPv6 address"))
	m06a, _ := ipv6Part.RAdd(":").Mul(PyTuple{0, 6})
	m06b, _ := ipv6Part.RAdd(":").Mul(PyTuple{0, 6})
	short := keep(Opt(ipv6Part.Add(m06a)).Add("::").Add(Opt(ipv6Part.Add(m06b))).SetName("short IPv6 address"))
	short.AddCondition(nil, false, false, func(s string, l int, t *ParseResults) bool {
		n := 0
		for _, tt := range t.toks.items {
			str, ok := tt.(string)
			if !ok {
				continue
			}
			if ok, _ := ipv6Part.Matches(str, true); ok {
				n++
			}
		}
		return n < 8
	})
	// add_condition here passes no message: Python's str(kwargs.get("message")) is "None".
	mixed := keep(c.Ipv4Address.RAdd("::ffff:").SetName("mixed IPv6 address"))
	c.Ipv6Address = keep(Combine(full.BitOr(mixed).BitOr(short).SetName("IPv6 address"), "", true).SetName("IPv6 address"))
	c.MacAddress = keep(mustRegex(patMAC, 0).SetName("MAC address"))
	c.Iso8601Date = keep(mustRegex(patISODate, 0).SetName("ISO8601 date"))
	c.Iso8601Datetime = keep(mustRegex(patISODatetime, 0).SetName("ISO8601 datetime"))
	keep(c.Iso8601Date.copy().addPA(AsDatetime))
	keep(c.Iso8601Datetime.copy().addPA(AsDatetime))
	c.UUID = keep(mustRegex(patUUID, 0).SetName("UUID"))
	htmlStripper = keep(anyOpenTag.Suppress().BitOr(anyCloseTag.Suppress()))
	ex := ","
	pw, _ := Word(Printables, nil, 1, 0, 0, false, &ex)
	commasepitem := keep(Combine(oneOrMore(Literal(",").Invert().Add(LineEnd().Invert()).Add(pw).Add(
		Opt(White(" \t", 1, 0, 0).Add(FollowedBy(LineEnd().BitOr(",")).Invert())))), "", true).streamline().SetName("commaItem"))
	csl, _ := DelimitedList(Opt(quotedString.copy().BitOr(commasepitem), ""), ",", false, nil, nil, false)
	c.CommaSeparatedList = keep(csl.SetName("comma separated list"))
	c.URL = keep(mustRegex(patURL, 0).SetName("url"))
	return all
}

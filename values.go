// Port of pyparsing/results.py and pyparsing/core.py value handling (pyparsing 3.3.3+). MIT License, Copyright (c) 2003-2025  Paul McGuire; see LICENSE.

package pyparsing

import (
	"math/big"
	"strings"

	"github.com/pyparsing/pyparsing-go/internal/pyre"
	"github.com/pyparsing/pyparsing-go/internal/pyval"
)

// PyTuple is a Python tuple.
type PyTuple []any

// PyDict is a Python dict with str keys, iterated in insertion order.
type PyDict struct {
	keys []string
	vals map[string]any
}

// NewPyDict returns an empty dict.
func NewPyDict() *PyDict { return &PyDict{vals: map[string]any{}} }

// Set stores v under k, keeping k's original position when it is already present.
func (d *PyDict) Set(k string, v any) {
	if d.vals == nil {
		d.vals = map[string]any{}
	}
	if _, ok := d.vals[k]; !ok {
		d.keys = append(d.keys, k)
	}
	d.vals[k] = v
}

// Get returns the value stored under k.
func (d *PyDict) Get(k string) (any, bool) {
	if d == nil || d.vals == nil {
		return nil, false
	}
	v, ok := d.vals[k]
	return v, ok
}

// Keys returns the keys in insertion order.
func (d *PyDict) Keys() []string {
	if d == nil {
		return nil
	}
	return append([]string(nil), d.keys...)
}

// Len returns the number of keys.
func (d *PyDict) Len() int {
	if d == nil {
		return 0
	}
	return len(d.keys)
}

// Date is a Python datetime.date.
type Date struct{ Year, Month, Day int }

// DateTime is a naive Python datetime.datetime.
type DateTime struct{ Year, Month, Day, Hour, Minute, Second, Microsecond int }

// RegexMatch is a Python re.Match, the token Regex(as_match=True) produces.
type RegexMatch struct {
	m *pyre.Match
}

// Span returns match.span().
func (m *RegexMatch) Span() (int, int) {
	if m == nil || m.m == nil {
		return 0, 0
	}
	return m.m.Start(0), m.m.End(0)
}

// Group returns match.group(n); ok is false when the group did not participate.
func (m *RegexMatch) Group(n int) (s string, ok bool) {
	if m == nil || m.m == nil || n < 0 || n > m.m.Re().Groups() {
		return "", false
	}
	return m.m.Group(n)
}

// Groups returns match.groups(), nil standing for None.
func (m *RegexMatch) Groups() PyTuple {
	if m == nil || m.m == nil {
		return PyTuple{}
	}
	gs := m.m.Groups()
	out := make(PyTuple, len(gs))
	for i, g := range gs {
		if g == nil {
			out[i] = nil
		} else {
			out[i] = *g
		}
	}
	return out
}

// EllipsisType is the type of Ellipsis.
type EllipsisType struct{}

// Ellipsis is Python's `...`.
var Ellipsis = EllipsisType{}

// prList is ParseResults.List, the list subclass Group(aslist=True) returns.
type prList []any

// nullToken is Opt's private "not matched" sentinel (_NullToken).
type nullToken struct{}

// anyValueType is the type of with_attribute.ANY_VALUE.
type anyValueType struct{}

// Repr returns Python's repr() of a value the library produces or accepts.
func Repr(v any) string {
	switch x := v.(type) {
	case nil:
		return "None"
	case string:
		return pyval.StrRepr(x)
	case []any:
		return seqRepr("[", "]", x, false)
	case prList:
		return seqRepr("[", "]", x, false)
	case PyTuple:
		return seqRepr("(", ")", x, true)
	case *PyDict:
		if x == nil {
			return "None"
		}
		var b strings.Builder
		b.WriteByte('{')
		for i, k := range x.keys {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(pyval.StrRepr(k))
			b.WriteString(": ")
			b.WriteString(Repr(x.vals[k]))
		}
		b.WriteByte('}')
		return b.String()
	case Date:
		return pyval.DateRepr(x.Year, x.Month, x.Day)
	case *Date:
		return pyval.DateRepr(x.Year, x.Month, x.Day)
	case DateTime:
		return pyval.DateTimeRepr(x.Year, x.Month, x.Day, x.Hour, x.Minute, x.Second, x.Microsecond)
	case *DateTime:
		return pyval.DateTimeRepr(x.Year, x.Month, x.Day, x.Hour, x.Minute, x.Second, x.Microsecond)
	case *RegexMatch:
		if x == nil || x.m == nil {
			return "None"
		}
		return x.m.Repr()
	case *ParseResults:
		return x.Repr()
	case *ParserElement:
		return x.String()
	case EllipsisType:
		return "Ellipsis"
	case nullToken:
		return "<pyparsing.core._NullToken object>"
	case anyValueType:
		return "<object object>"
	case *ParseException:
		return x.ClassName() + "(" + pyval.StrRepr(x.Pstr) + ", " + itoa(x.Loc) + ", " + pyval.StrRepr(x.Msg) + ")"
	case *PyError:
		return x.Class + "(" + pyval.StrRepr(x.Msg) + ")"
	case *big.Int:
		return pyval.IntRepr(x)
	}
	return pyval.Repr(v)
}

func seqRepr(open, close string, items []any, tuple bool) string {
	var b strings.Builder
	b.WriteString(open)
	for i, it := range items {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(Repr(it))
	}
	if tuple && len(items) == 1 {
		b.WriteByte(',')
	}
	b.WriteString(close)
	return b.String()
}

// Str returns Python's str() of such a value.
func Str(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case Date:
		return pyval.DateStr(x.Year, x.Month, x.Day)
	case *Date:
		return pyval.DateStr(x.Year, x.Month, x.Day)
	case DateTime:
		return pyval.DateTimeStr(x.Year, x.Month, x.Day, x.Hour, x.Minute, x.Second, x.Microsecond)
	case *DateTime:
		return pyval.DateTimeStr(x.Year, x.Month, x.Day, x.Hour, x.Minute, x.Second, x.Microsecond)
	case *ParseResults:
		return x.String()
	case *ParseException:
		return x.Error()
	case *PyError:
		return x.Msg
	case nullToken:
		return ""
	}
	return Repr(v)
}

// PyError is a Python exception other than the parse exceptions, such as ValueError, TypeError,
// IndexError, KeyError, AttributeError, RuntimeError, NotImplementedError, Exception or re.error.
type PyError struct {
	Class string // the exception's class name; "re.error" for the re module's error
	Msg   string // str() of the exception
}

// Error returns "Class: Msg".
func (e *PyError) Error() string { return e.Class + ": " + e.Msg }

func pyErr(class, msg string) *PyError { return &PyError{Class: class, Msg: msg} }

// fromPyval converts an error of an internal package to *PyError.
func fromInternal(err error) error {
	if err == nil {
		return nil
	}
	switch e := err.(type) {
	case *pyval.Error:
		return &PyError{Class: e.Class, Msg: e.Msg}
	case *pyre.Error:
		return &PyError{Class: "re.error", Msg: e.Message()}
	case *pyre.PyError:
		return &PyError{Class: e.Class, Msg: e.Msg}
	}
	return err
}

func isPyErr(err error, class string) bool {
	if e, ok := err.(*PyError); ok {
		return e.Class == class
	}
	return false
}

// pyTypeName returns type(v).__name__ (the tp_name used in TypeError/AttributeError messages).
func pyTypeName(v any) string {
	switch v.(type) {
	case nil:
		return "NoneType"
	case string:
		return "str"
	case bool:
		return "bool"
	case *big.Int, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return "int"
	case float64, float32:
		return "float"
	case []any:
		return "list"
	case prList:
		return "List"
	case PyTuple:
		return "tuple"
	case *PyDict:
		return "dict"
	case Date, *Date:
		return "datetime.date"
	case DateTime, *DateTime:
		return "datetime.datetime"
	case *RegexMatch:
		return "re.Match"
	case *ParseResults:
		return "ParseResults"
	case EllipsisType:
		return "ellipsis"
	case *ParserElement:
		return v.(*ParserElement).cls.name
	case nullToken:
		return "_NullToken"
	}
	return "object"
}

// toBigInt converts Go integer kinds (and *big.Int) to *big.Int.
func toBigInt(v any) (*big.Int, bool) {
	switch x := v.(type) {
	case *big.Int:
		return x, true
	case int:
		return big.NewInt(int64(x)), true
	case int8:
		return big.NewInt(int64(x)), true
	case int16:
		return big.NewInt(int64(x)), true
	case int32:
		return big.NewInt(int64(x)), true
	case int64:
		return big.NewInt(x), true
	case uint:
		return new(big.Int).SetUint64(uint64(x)), true
	case uint8:
		return big.NewInt(int64(x)), true
	case uint16:
		return big.NewInt(int64(x)), true
	case uint32:
		return big.NewInt(int64(x)), true
	case uint64:
		return new(big.Int).SetUint64(x), true
	}
	return nil, false
}

func bigInt(n int) *big.Int { return big.NewInt(int64(n)) }

func itoa(n int) string { return big.NewInt(int64(n)).String() }

// pyTruth is Python's bool(v).
func pyTruth(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	case string:
		return x != ""
	case []any:
		return len(x) > 0
	case prList:
		return len(x) > 0
	case PyTuple:
		return len(x) > 0
	case *PyDict:
		return x != nil && len(x.keys) > 0
	case *ParseResults:
		return x.Truth()
	case float64:
		return x != 0
	case float32:
		return x != 0
	case nullToken:
		return false
	}
	if b, ok := toBigInt(v); ok {
		return b.Sign() != 0
	}
	return true
}

// pyEqual is Python's == for the values pyparsing compares (tokens and attribute values).
func pyEqual(a, b any) bool {
	switch x := a.(type) {
	case nil:
		return b == nil
	case string:
		y, ok := b.(string)
		return ok && x == y
	case bool:
		if y, ok := b.(bool); ok {
			return x == y
		}
		if bi, ok := toBigInt(b); ok {
			return bi.Cmp(boolInt(x)) == 0
		}
		if f, ok := b.(float64); ok {
			return float64(boolInt(x).Int64()) == f
		}
		return false
	case float64:
		switch y := b.(type) {
		case float64:
			return x == y
		case bool:
			return x == float64(boolInt(y).Int64())
		}
		if bi, ok := toBigInt(b); ok {
			f, acc := new(big.Float).SetInt(bi).Float64()
			return acc == big.Exact && f == x
		}
		return false
	case []any:
		return seqEqual(x, b)
	case prList:
		return seqEqual(x, b)
	case PyTuple:
		y, ok := b.(PyTuple)
		if !ok || len(x) != len(y) {
			return false
		}
		for i := range x {
			if !pyEqual(x[i], y[i]) {
				return false
			}
		}
		return true
	case *PyDict:
		y, ok := b.(*PyDict)
		if !ok || x.Len() != y.Len() {
			return false
		}
		for _, k := range x.keys {
			yv, ok := y.vals[k]
			if !ok || !pyEqual(x.vals[k], yv) {
				return false
			}
		}
		return true
	case *ParseResults:
		y, ok := b.(*ParseResults)
		if !ok {
			return false
		}
		return x.pyEq(y)
	case Date:
		y, ok := b.(Date)
		return ok && x == y
	case DateTime:
		y, ok := b.(DateTime)
		return ok && x == y
	case EllipsisType:
		_, ok := b.(EllipsisType)
		return ok
	}
	if xi, ok := toBigInt(a); ok {
		switch y := b.(type) {
		case bool:
			return xi.Cmp(boolInt(y)) == 0
		case float64:
			return pyEqual(y, a)
		}
		if yi, ok := toBigInt(b); ok {
			return xi.Cmp(yi) == 0
		}
		return false
	}
	return a == b
}

func seqEqual(x []any, b any) bool {
	var y []any
	switch t := b.(type) {
	case []any:
		y = t
	case prList:
		y = t
	default:
		return false
	}
	if len(x) != len(y) {
		return false
	}
	for i := range x {
		if !pyEqual(x[i], y[i]) {
			return false
		}
	}
	return true
}

func boolInt(b bool) *big.Int {
	if b {
		return big.NewInt(1)
	}
	return big.NewInt(0)
}

// isPyIterable reports whether a value is a Python Iterable that is not a str (for _flatten).
func pyIterItems(v any) ([]any, bool) {
	switch x := v.(type) {
	case []any:
		return x, true
	case prList:
		return x, true
	case PyTuple:
		return x, true
	case *ParseResults:
		return x.toks.items, true
	case *PyDict:
		out := make([]any, len(x.keys))
		for i, k := range x.keys {
			out[i] = k
		}
		return out, true
	}
	return nil, false
}

// flatten is util._flatten.
func flattenList(ll []any) []any {
	ret := []any{}
	for _, i := range ll {
		if s, ok := i.(string); ok {
			ret = append(ret, s)
		} else if items, ok := pyIterItems(i); ok {
			ret = append(ret, flattenList(items)...)
		} else {
			ret = append(ret, i)
		}
	}
	return ret
}

// Port of CPython 3.12's repr()/str() of builtin values: Objects/unicodeobject.c unicode_repr,
// Objects/listobject.c, Objects/tupleobject.c, Objects/dictobject.c, Objects/boolobject.c and
// Modules/_datetimemodule.c (PSF License Version 2; Copyright (c) 2001-2024 Python Software
// Foundation).

package pyval

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/pyparsing/pyparsing-go/internal/pyuni"
)

// Reprer is implemented by values (defined outside this package) that know their Python
// repr(), such as the root package's tuple, dict, date, datetime, match and results types.
type Reprer interface {
	PyRepr() string
}

// Strer is implemented by values whose Python str() differs from their repr(). A value that
// implements Reprer but not Strer has str() == repr(), as Python's object.__str__ does.
type Strer interface {
	PyStr() string
}

// Repr returns Python's repr(v). It handles nil (None), bool, string (str), *big.Int and Go
// integer types (int), float64 and float32 (float), []any (list), []string (a list of str),
// and any value implementing Reprer. Other values are rendered with fmt's %v.
//
// An int with more than 4300 decimal digits is rendered in full; Python raises ValueError
// for it (see IntReprChecked).
func Repr(v any) string {
	switch x := v.(type) {
	case nil:
		return "None"
	case Reprer:
		return x.PyRepr()
	case string:
		return StrRepr(x)
	case bool:
		if x {
			return "True"
		}
		return "False"
	case *big.Int:
		return IntRepr(x)
	case float64:
		return FloatRepr(x)
	case float32:
		return FloatRepr(float64(x))
	case int:
		return strconv.FormatInt(int64(x), 10)
	case int8:
		return strconv.FormatInt(int64(x), 10)
	case int16:
		return strconv.FormatInt(int64(x), 10)
	case int32:
		return strconv.FormatInt(int64(x), 10)
	case int64:
		return strconv.FormatInt(x, 10)
	case uint:
		return strconv.FormatUint(uint64(x), 10)
	case uint8:
		return strconv.FormatUint(uint64(x), 10)
	case uint16:
		return strconv.FormatUint(uint64(x), 10)
	case uint32:
		return strconv.FormatUint(uint64(x), 10)
	case uint64:
		return strconv.FormatUint(x, 10)
	case []any:
		return ListRepr(x)
	case []string:
		items := make([]any, len(x))
		for i, s := range x {
			items[i] = s
		}
		return ListRepr(items)
	}
	return fmt.Sprintf("%v", v)
}

// Str returns Python's str(v): a string is returned unchanged, a Strer gives PyStr(), and
// everything else gives Repr(v) (Python's str() of int, float, bool, None, list, tuple and
// dict equals their repr()).
func Str(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case Strer:
		return x.PyStr()
	}
	return Repr(v)
}

// StrRepr returns Python's repr() of a str: single quotes unless the string contains a single
// quote and no double quote; backslash, the chosen quote, \t, \n and \r escaped; other
// characters below U+0020, U+007F and every character that is not str.isprintable()
// escaped as \xhh, \uhhhh or \Uhhhhhhhh. Invalid UTF-8 bytes are rendered as \udcXX.
func StrRepr(s string) string {
	quote := byte('\'')
	if strings.IndexByte(s, '\'') >= 0 && strings.IndexByte(s, '"') < 0 {
		quote = '"'
	}
	var b strings.Builder
	b.Grow(len(s) + 2)
	b.WriteByte(quote)
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			// Invalid byte: the surrogateescape code point U+DC00+b.
			fmt.Fprintf(&b, "\\u%04x", 0xdc00+int(s[i]))
			i++
			continue
		}
		i += size
		switch {
		case r == rune(quote) || r == '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case r == '\t':
			b.WriteString(`\t`)
		case r == '\n':
			b.WriteString(`\n`)
		case r == '\r':
			b.WriteString(`\r`)
		case r < ' ' || r == 0x7f:
			fmt.Fprintf(&b, "\\x%02x", r)
		case r < 0x7f:
			b.WriteRune(r)
		case !pyuni.IsPrintable(r):
			switch {
			case r <= 0xff:
				fmt.Fprintf(&b, "\\x%02x", r)
			case r <= 0xffff:
				fmt.Fprintf(&b, "\\u%04x", r)
			default:
				fmt.Fprintf(&b, "\\U%08x", r)
			}
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte(quote)
	return b.String()
}

// BoolRepr returns "True" or "False".
func BoolRepr(v bool) string {
	if v {
		return "True"
	}
	return "False"
}

// ListRepr returns Python's repr() of a list with these items: "[a, b]".
func ListRepr(items []any) string {
	var b strings.Builder
	b.WriteByte('[')
	for i, it := range items {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(Repr(it))
	}
	b.WriteByte(']')
	return b.String()
}

// TupleRepr returns Python's repr() of a tuple with these items: "()", "(a,)", "(a, b)".
func TupleRepr(items []any) string {
	var b strings.Builder
	b.WriteByte('(')
	for i, it := range items {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(Repr(it))
	}
	if len(items) == 1 {
		b.WriteByte(',')
	}
	b.WriteByte(')')
	return b.String()
}

// DictRepr returns Python's repr() of a dict with str keys, in the given order:
// "{'a': 1, 'b': [2]}". Values beyond len(keys) are ignored; missing values are None.
func DictRepr(keys []string, values []any) string {
	var b strings.Builder
	b.WriteByte('{')
	for i, k := range keys {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(StrRepr(k))
		b.WriteString(": ")
		var v any
		if i < len(values) {
			v = values[i]
		}
		b.WriteString(Repr(v))
	}
	b.WriteByte('}')
	return b.String()
}

// DictReprAny is DictRepr for a dict whose keys are arbitrary Python values (rendered with
// Repr).
func DictReprAny(keys []any, values []any) string {
	var b strings.Builder
	b.WriteByte('{')
	for i, k := range keys {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(Repr(k))
		b.WriteString(": ")
		var v any
		if i < len(values) {
			v = values[i]
		}
		b.WriteString(Repr(v))
	}
	b.WriteByte('}')
	return b.String()
}

// DateRepr returns repr(datetime.date(y, m, d)): "datetime.date(1999, 12, 31)".
func DateRepr(y, m, d int) string {
	return fmt.Sprintf("datetime.date(%d, %d, %d)", y, m, d)
}

// DateStr returns str(datetime.date(y, m, d)), its isoformat(): "1999-12-31".
func DateStr(y, m, d int) string {
	return fmt.Sprintf("%04d-%02d-%02d", y, m, d)
}

// DateTimeRepr returns repr() of a naive datetime.datetime. Like CPython it always shows
// hour and minute, shows the second when second or microsecond is non-zero and the
// microsecond when it is non-zero: "datetime.datetime(1999, 12, 31, 23, 59, 59, 999000)",
// "datetime.datetime(2000, 1, 1, 0, 0)".
func DateTimeRepr(y, mo, d, h, mi, s, us int) string {
	switch {
	case us != 0:
		return fmt.Sprintf("datetime.datetime(%d, %d, %d, %d, %d, %d, %d)", y, mo, d, h, mi, s, us)
	case s != 0:
		return fmt.Sprintf("datetime.datetime(%d, %d, %d, %d, %d, %d)", y, mo, d, h, mi, s)
	}
	return fmt.Sprintf("datetime.datetime(%d, %d, %d, %d, %d)", y, mo, d, h, mi)
}

// DateTimeStr returns str() of a naive datetime.datetime, isoformat(sep=' '):
// "1999-12-31 23:59:59.999000", or "1999-12-31 23:59:59" when the microsecond is 0.
func DateTimeStr(y, mo, d, h, mi, s, us int) string {
	if us != 0 {
		return fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d.%06d", y, mo, d, h, mi, s, us)
	}
	return fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d", y, mo, d, h, mi, s)
}

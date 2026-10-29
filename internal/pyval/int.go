// Port of CPython 3.12's int semantics: Objects/longobject.c (PyLong_FromUnicodeObject,
// PyLong_FromString, long_from_string_base, long_to_decimal_string_internal) and
// Python/bltinmodule.c builtin_chr (PSF License Version 2; Copyright (c) 2001-2024 Python
// Software Foundation).

package pyval

import (
	"math"
	"math/big"
	"strconv"
	"strings"
	"unicode/utf8"
)

// MaxStrDigits is sys.get_int_max_str_digits() in a default CPython 3.12 interpreter: the
// largest number of decimal digits int() accepts and repr()/str() produce for an int
// (power-of-two bases are not limited).
const MaxStrDigits = 4300

// maxStrDigitsThreshold is _PY_LONG_MAX_STR_DIGITS_THRESHOLD.
const maxStrDigitsThreshold = 640

// IntRepr returns repr(z) (equal to str(z)), the decimal digits of z. A nil z gives "None".
// Unlike Python it does not raise for more than MaxStrDigits digits; see IntReprChecked.
func IntRepr(z *big.Int) string {
	if z == nil {
		return "None"
	}
	return z.String()
}

// IntReprChecked is repr(z) including CPython's limit: more than MaxStrDigits decimal digits
// gives ValueError "Exceeds the limit (4300 digits) for integer string conversion; use
// sys.set_int_max_str_digits() to increase the limit".
func IntReprChecked(z *big.Int) (string, error) {
	s := IntRepr(z)
	n := len(s)
	if z != nil && z.Sign() < 0 {
		n--
	}
	if n > MaxStrDigits {
		return "", valueError("Exceeds the limit (" + strconv.Itoa(MaxStrDigits) +
			" digits) for integer string conversion; use sys.set_int_max_str_digits() to increase the limit")
	}
	return s, nil
}

// digitValue is _PyLong_DigitValue: 0-9, a-z/A-Z = 10-35, everything else 37.
func digitValue(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'z':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'Z':
		return int(c-'A') + 10
	}
	return 37
}

// ParseInt is Python's int(s, base) for a str s (int(s) is base 10). base is 0 or 2..36;
// base 0 reads a 0x/0o/0b prefix and rejects other leading zeros of a non-zero value. It
// strips whitespace (Unicode whitespace, but of ASCII only " \t\n\v\f\r"), accepts Unicode
// decimal digits, a sign, a base prefix matching base (followed by at most one underscore)
// and single underscores between digits.
//
// Errors are ValueError "invalid literal for int() with base B: <repr(s) truncated to 200
// characters>", ValueError "Exceeds the limit (4300 digits) for integer string conversion:
// value has N digits; use sys.set_int_max_str_digits() to increase the limit" for long
// decimal input, and ValueError "int() base must be >= 2 and <= 36, or 0".
func ParseInt(s string, base int) (*big.Int, error) {
	if (base != 0 && base < 2) || base > 36 {
		return nil, valueError("int() base must be >= 2 and <= 36, or 0")
	}
	invalid := func() (*big.Int, error) {
		return nil, valueError("invalid literal for int() with base " + strconv.Itoa(base) +
			": " + truncateRunes(StrRepr(s), 200))
	}
	buf, ok := transformDecimalAndSpace(s)
	if !ok {
		// CPython writes '?' and stops; the '?' is a syntax error unless a NUL comes first.
		buf = append(buf, '?')
	}
	full := len(buf)
	// The C code reads a NUL-terminated buffer.
	if k := strings.IndexByte(string(buf), 0); k >= 0 {
		buf = buf[:k]
	}
	z, end, err := longFromString(buf, base)
	if err != nil {
		return nil, err
	}
	if z == nil || end != full {
		return invalid()
	}
	return z, nil
}

// longFromString is PyLong_FromString on a C string; it returns (nil, end, nil) for a syntax
// error and a non-nil error for the digit limit.
func longFromString(str []byte, base int) (*big.Int, int, error) {
	at := func(i int) byte {
		if i < len(str) {
			return str[i]
		}
		return 0
	}
	p := 0
	for p < len(str) && isPySpaceASCII(str[p]) {
		p++
	}
	neg := false
	if at(p) == '+' {
		p++
	} else if at(p) == '-' {
		p++
		neg = true
	}
	errorIfNonzero := false
	if base == 0 {
		switch {
		case at(p) != '0':
			base = 10
		case at(p+1) == 'x' || at(p+1) == 'X':
			base = 16
		case at(p+1) == 'o' || at(p+1) == 'O':
			base = 8
		case at(p+1) == 'b' || at(p+1) == 'B':
			base = 2
		default:
			errorIfNonzero = true
			base = 10
		}
	}
	if at(p) == '0' && ((base == 16 && (at(p+1) == 'x' || at(p+1) == 'X')) ||
		(base == 8 && (at(p+1) == 'o' || at(p+1) == 'O')) ||
		(base == 2 && (at(p+1) == 'b' || at(p+1) == 'B'))) {
		p += 2
		if at(p) == '_' {
			p++
		}
	}
	// long_from_string_base
	start := p
	if at(start) == '_' {
		return nil, start, nil
	}
	var prev byte
	digits := 0
	for digitValue(at(p)) < base || at(p) == '_' {
		if at(p) == '_' {
			if prev == '_' {
				return nil, p - 1, nil
			}
		} else {
			digits++
		}
		prev = at(p)
		p++
	}
	if prev == '_' {
		return nil, p - 1, nil
	}
	end := p
	if start == end {
		return nil, end, nil
	}
	for p < len(str) && isPySpaceASCII(str[p]) {
		p++
	}
	if p < len(str) {
		return nil, p, nil
	}
	if base&(base-1) != 0 && digits > maxStrDigitsThreshold && digits > MaxStrDigits {
		return nil, 0, valueError("Exceeds the limit (" + strconv.Itoa(MaxStrDigits) +
			" digits) for integer string conversion: value has " + strconv.Itoa(digits) +
			" digits; use sys.set_int_max_str_digits() to increase the limit")
	}
	clean := strings.ReplaceAll(string(str[start:end]), "_", "")
	z, ok := new(big.Int).SetString(clean, base)
	if !ok {
		return nil, end, nil
	}
	if errorIfNonzero && z.Sign() != 0 {
		return nil, p, nil
	}
	if neg {
		z.Neg(z)
	}
	return z, p, nil
}

// truncateRunes keeps the first n code points of s (PyUnicode_FromFormat's %.<n>R).
func truncateRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	i := 0
	for k := 0; k < n; k++ {
		_, size := utf8.DecodeRuneInString(s[i:])
		i += size
	}
	return s[:i]
}

// Chr is Python's chr(i). i outside range(0x110000) gives ValueError
// "chr() arg not in range(0x110000)", and i outside the C int range gives OverflowError
// "Python int too large to convert to C int". A surrogate code point (U+D800..U+DFFF), which a
// Go string cannot hold, is returned as U+FFFD with ok false.
func Chr(i *big.Int) (s string, ok bool, err error) {
	if i == nil || !i.IsInt64() || i.Int64() > math.MaxInt32 || i.Int64() < math.MinInt32 {
		return "", false, overflowError("Python int too large to convert to C int")
	}
	v := i.Int64()
	if v < 0 || v > 0x10ffff {
		return "", false, valueError("chr() arg not in range(0x110000)")
	}
	if v >= 0xd800 && v <= 0xdfff {
		return "\ufffd", false, nil
	}
	return string(rune(v)), true, nil
}

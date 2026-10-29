// Port of CPython 3.12's float semantics: Objects/floatobject.c (float_repr, float_rem,
// float_floor_div, float___round___impl, double_round, float___trunc___impl,
// PyFloat_FromString) and Python/pystrtod.c / Python/dtoa.c (PSF License Version 2;
// Copyright (c) 2001-2024 Python Software Foundation).

package pyval

import (
	"math"
	"math/big"
	"strconv"
	"strings"
)

// FloatRepr returns Python's repr(f) (equal to str(f)): the shortest string that round-trips,
// in positional notation when the decimal exponent is in [-4, 16) ("0.0001",
// "123456789012345.0", "1234567890123456.0") and in exponent notation otherwise ("1e-05",
// "1e+16", "1.2345678901234568e+16"); "-0.0", "inf", "-inf" and "nan" (the sign of a NaN is
// not shown).
func FloatRepr(f float64) string {
	switch {
	case math.IsNaN(f):
		return "nan"
	case math.IsInf(f, 1):
		return "inf"
	case math.IsInf(f, -1):
		return "-inf"
	}
	// Shortest round-trip digits; Go and David Gay's dtoa mode 0 both pick the digit string
	// closest to the exact value among the shortest ones.
	e := strconv.FormatFloat(f, 'e', -1, 64) // "-d.ddde±XX"
	neg := false
	if e[0] == '-' {
		neg = true
		e = e[1:]
	}
	epos := strings.IndexByte(e, 'e')
	mant, exps := e[:epos], e[epos+1:]
	exp, _ := strconv.Atoi(exps)
	digits := strings.Replace(mant, ".", "", 1)
	// Python's decpt: value = 0.DIGITS * 10**decpt.
	decpt := exp + 1
	if digits == "0" {
		decpt = 1
	}
	var b strings.Builder
	if neg {
		b.WriteByte('-')
	}
	if decpt <= -4 || decpt > 16 {
		b.WriteByte(digits[0])
		if len(digits) > 1 {
			b.WriteByte('.')
			b.WriteString(digits[1:])
		}
		x := decpt - 1
		b.WriteByte('e')
		if x < 0 {
			b.WriteByte('-')
			x = -x
		} else {
			b.WriteByte('+')
		}
		if x < 10 {
			b.WriteByte('0')
		}
		b.WriteString(strconv.Itoa(x))
		return b.String()
	}
	switch {
	case decpt <= 0:
		b.WriteString("0.")
		b.WriteString(strings.Repeat("0", -decpt))
		b.WriteString(digits)
	case decpt >= len(digits):
		b.WriteString(digits)
		b.WriteString(strings.Repeat("0", decpt-len(digits)))
		b.WriteString(".0")
	default:
		b.WriteString(digits[:decpt])
		b.WriteByte('.')
		b.WriteString(digits[decpt:])
	}
	return b.String()
}

// FloatStr returns Python's str(f), which equals repr(f).
func FloatStr(f float64) string { return FloatRepr(f) }

// isPySpaceASCII is Py_ISSPACE: the C-locale-free ASCII whitespace test used by the number
// parsers (it does not include \x1c-\x1f, unlike str.isspace).
func isPySpaceASCII(c byte) bool {
	switch c {
	case ' ', '\t', '\n', '\v', '\f', '\r':
		return true
	}
	return false
}

// ParseFloat is Python's float(s) for a str argument. It strips leading and trailing
// whitespace (Unicode whitespace, but of ASCII only " \t\n\v\f\r"), accepts Unicode decimal
// digits, single underscores between digits, an optional sign, "inf", "infinity" and "nan"
// in any case, and a decimal mantissa with an optional exponent. Overflow gives ±inf and
// underflow ±0.0, as in Python. Any other input gives ValueError
// "could not convert string to float: <repr(s)>".
func ParseFloat(s string) (float64, error) {
	fail := func() (float64, error) {
		return 0, valueError("could not convert string to float: " + StrRepr(s))
	}
	buf, ok := transformDecimalAndSpace(s)
	if !ok {
		return fail()
	}
	i, j := 0, len(buf)
	for i < j && isPySpaceASCII(buf[i]) {
		i++
	}
	if i == j {
		return fail()
	}
	for j-1 > i && isPySpaceASCII(buf[j-1]) {
		j--
	}
	t := buf[i:j]
	if strings.IndexByte(string(t), '_') >= 0 {
		var ok bool
		t, ok = removeUnderscores(t)
		if !ok {
			return fail()
		}
	}
	f, ok := parseFloatLiteral(string(t))
	if !ok {
		return fail()
	}
	return f, nil
}

// transformDecimalAndSpace is _PyUnicode_TransformDecimalAndSpaceToASCII: code points below
// 127 are kept, other whitespace becomes ' ', other decimal digits become their ASCII digit;
// any other code point makes the conversion fail (CPython writes '?', which no number parser
// accepts).
func transformDecimalAndSpace(s string) ([]byte, bool) {
	out := make([]byte, 0, len(s))
	for _, r := range s {
		switch {
		case r < 127:
			out = append(out, byte(r))
		case isSpaceRune(r):
			out = append(out, ' ')
		default:
			d, ok := decimalValue(r)
			if !ok {
				return out, false
			}
			out = append(out, byte('0'+d))
		}
	}
	return out, true
}

// removeUnderscores is _Py_string_to_number_with_underscores: an underscore is allowed only
// between two ASCII digits.
func removeUnderscores(s []byte) ([]byte, bool) {
	out := make([]byte, 0, len(s))
	var prev byte
	for _, c := range s {
		if c == '_' {
			if !(prev >= '0' && prev <= '9') {
				return nil, false
			}
		} else {
			out = append(out, c)
			if prev == '_' && !(c >= '0' && c <= '9') {
				return nil, false
			}
		}
		prev = c
	}
	if prev == '_' {
		return nil, false
	}
	return out, true
}

// parseFloatLiteral is PyOS_string_to_double followed by the whole-string check: _Py_dg_strtod
// syntax ([sign] digits [. digits] [e [sign] digits], at least one mantissa digit), falling
// back to _Py_parse_inf_or_nan when no number is recognised.
func parseFloatLiteral(s string) (float64, bool) {
	n := len(s)
	p := 0
	if p < n && (s[p] == '+' || s[p] == '-') {
		p++
	}
	start := p
	nd := 0
	for p < n && s[p] >= '0' && s[p] <= '9' {
		p++
		nd++
	}
	if p < n && s[p] == '.' {
		p++
		for p < n && s[p] >= '0' && s[p] <= '9' {
			p++
			nd++
		}
	}
	if nd == 0 {
		// _Py_dg_strtod consumed nothing; try inf/nan.
		q := start
		neg := s[0] == '-'
		rest := strings.ToLower(s[q:])
		var v float64
		switch {
		case rest == "inf" || rest == "infinity":
			v = math.Inf(1)
		case rest == "nan":
			v = math.NaN()
		default:
			return 0, false
		}
		if neg {
			v = math.Copysign(v, -1)
		}
		return v, true
	}
	if p < n && (s[p] == 'e' || s[p] == 'E') {
		q := p + 1
		if q < n && (s[q] == '+' || s[q] == '-') {
			q++
		}
		ds := q
		for q < n && s[q] >= '0' && s[q] <= '9' {
			q++
		}
		if q > ds {
			p = q
		}
	}
	if p != n {
		return 0, false
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		if ne, ok := err.(*strconv.NumError); !ok || ne.Err != strconv.ErrRange {
			return 0, false
		}
	}
	return f, true
}

// FloatMod is Python's x % y for floats: the result has the sign of y. y == 0 gives
// ZeroDivisionError "float modulo".
func FloatMod(x, y float64) (float64, error) {
	if y == 0 {
		return 0, &Error{Class: "ZeroDivisionError", Msg: "float modulo"}
	}
	mod := math.Mod(x, y)
	if mod != 0 {
		if (y < 0) != (mod < 0) {
			mod += y
		}
	} else {
		mod = math.Copysign(0, y)
	}
	return mod, nil
}

// FloatFloorDiv is Python's x // y for floats. y == 0 gives ZeroDivisionError
// "float floor division by zero".
func FloatFloorDiv(x, y float64) (float64, error) {
	if y == 0 {
		return 0, &Error{Class: "ZeroDivisionError", Msg: "float floor division by zero"}
	}
	floordiv, _ := floatDivMod(x, y)
	return floordiv, nil
}

// FloatDivMod is Python's divmod(x, y) for floats. y == 0 gives ZeroDivisionError
// "float divmod()".
func FloatDivMod(x, y float64) (float64, float64, error) {
	if y == 0 {
		return 0, 0, &Error{Class: "ZeroDivisionError", Msg: "float divmod()"}
	}
	d, m := floatDivMod(x, y)
	return d, m, nil
}

// floatDivMod is _float_div_mod.
func floatDivMod(vx, wx float64) (float64, float64) {
	mod := math.Mod(vx, wx)
	div := (vx - mod) / wx
	if mod != 0 {
		if (wx < 0) != (mod < 0) {
			mod += wx
			div -= 1.0
		}
	} else {
		mod = math.Copysign(0, wx)
	}
	var floordiv float64
	if div != 0 {
		floordiv = math.Floor(div)
		if div-floordiv > 0.5 {
			floordiv += 1.0
		}
	} else {
		floordiv = math.Copysign(0, vx/wx)
	}
	return floordiv, mod
}

// FloatToInt is Python's int(f) for a float: truncation toward zero. inf gives OverflowError
// "cannot convert float infinity to integer" and nan gives ValueError
// "cannot convert float NaN to integer".
func FloatToInt(f float64) (*big.Int, error) {
	if math.IsInf(f, 0) {
		return nil, overflowError("cannot convert float infinity to integer")
	}
	if math.IsNaN(f) {
		return nil, valueError("cannot convert float NaN to integer")
	}
	z, _ := big.NewFloat(math.Trunc(f)).Int(nil)
	return z, nil
}

// Round is Python's round(f) with no ndigits: the nearest int, ties to even. inf and nan
// raise as in FloatToInt.
func Round(f float64) (*big.Int, error) {
	if math.IsInf(f, 0) || math.IsNaN(f) {
		return FloatToInt(f)
	}
	return FloatToInt(math.RoundToEven(f))
}

// RoundN is Python's round(f, ndigits) for a float: the double nearest to f correctly rounded
// (ties to even on the exact binary value) to ndigits decimal places; ndigits may be
// negative. nan and inf are returned unchanged, a zero result keeps the sign of f, and a
// result too large for a float gives OverflowError "rounded value too large to represent".
func RoundN(f float64, ndigits int) (float64, error) {
	if math.IsNaN(f) || math.IsInf(f, 0) || f == 0 {
		return f, nil
	}
	const ndigitsMax = 323  // (DBL_MANT_DIG - DBL_MIN_EXP) * 0.30103
	const ndigitsMin = -308 // -(DBL_MAX_EXP + 1) * 0.30103
	if ndigits > ndigitsMax {
		return f, nil
	}
	if ndigits < ndigitsMin {
		return 0.0 * f, nil
	}
	x := new(big.Rat).SetFloat64(f)
	pow := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(abs(ndigits))), nil)
	scaled := new(big.Rat).Set(x)
	if ndigits >= 0 {
		scaled.Mul(scaled, new(big.Rat).SetInt(pow))
	} else {
		scaled.Quo(scaled, new(big.Rat).SetInt(pow))
	}
	q := roundRatHalfEven(scaled)
	res := new(big.Rat).SetInt(q)
	if ndigits >= 0 {
		res.Quo(res, new(big.Rat).SetInt(pow))
	} else {
		res.Mul(res, new(big.Rat).SetInt(pow))
	}
	r, _ := res.Float64()
	if math.IsInf(r, 0) {
		return 0, overflowError("rounded value too large to represent")
	}
	if r == 0 {
		return math.Copysign(0, f), nil
	}
	return r, nil
}

// roundRatHalfEven rounds a rational to the nearest integer, ties to even.
func roundRatHalfEven(x *big.Rat) *big.Int {
	num, den := x.Num(), x.Denom()
	q, m := new(big.Int).QuoRem(num, den, new(big.Int)) // truncated
	twice := new(big.Int).Mul(new(big.Int).Abs(m), big.NewInt(2))
	c := twice.Cmp(den)
	if c > 0 || (c == 0 && q.Bit(0) == 1) {
		if num.Sign() < 0 {
			q.Sub(q, big.NewInt(1))
		} else {
			q.Add(q, big.NewInt(1))
		}
	}
	return q
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

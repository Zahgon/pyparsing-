// Unit tests for internal/pyval; every expected value was printed by CPython 3.12.15 in the
// pinned python:3.12-bookworm image (PSF License Version 2; Copyright (c) 2001-2024 Python
// Software Foundation).

package pyval

import (
	"math"
	"math/big"
	"strings"
	"testing"
)

type tuple []any

func (t tuple) PyRepr() string { return TupleRepr(t) }

type dict struct {
	keys []string
	vals []any
}

func (d dict) PyRepr() string { return DictRepr(d.keys, d.vals) }

type date struct{ y, m, d int }

func (d date) PyRepr() string { return DateRepr(d.y, d.m, d.d) }
func (d date) PyStr() string  { return DateStr(d.y, d.m, d.d) }

func eq(t *testing.T, what, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got %q, want %q", what, got, want)
	}
}

func TestReprContainers(t *testing.T) {
	v := []any{big.NewInt(1), "a", 2.5, nil, true, []any{}, tuple{}, tuple{big.NewInt(1)},
		tuple{"x", []any{false}}}
	eq(t, "list", Repr(v), `[1, 'a', 2.5, None, True, [], (), (1,), ('x', [False])]`)
	d := dict{[]string{"a", "b", "it's"}, []any{big.NewInt(1), []any{2, "c"}, dict{[]string{"n"}, []any{nil}}}}
	eq(t, "dict", Repr(d), `{'a': 1, 'b': [2, 'c'], "it's": {'n': None}}`)
	eq(t, "empty dict", DictRepr(nil, nil), "{}")
	eq(t, "str of list", Str([]any{1.0, "a"}), "[1.0, 'a']")
	eq(t, "str of str", Str("it's"), "it's")
	eq(t, "str of date", Str(date{1, 2, 3}), "0001-02-03")
	eq(t, "repr of date", Repr(date{1999, 12, 31}), "datetime.date(1999, 12, 31)")
	eq(t, "str float", Str(1e16), "1e+16")
	eq(t, "str -0.0", Str(math.Copysign(0, -1)), "-0.0")
	eq(t, "nil big", Repr((*big.Int)(nil)), "None")
	eq(t, "[]string", Repr([]string{"a", "b'"}), `['a', "b'"]`)
}

func TestDateTimeRepr(t *testing.T) {
	eq(t, "dt", DateTimeRepr(2000, 1, 1, 0, 0, 0, 0), "datetime.datetime(2000, 1, 1, 0, 0)")
	eq(t, "dt s", DateTimeRepr(2000, 1, 1, 0, 0, 5, 0), "datetime.datetime(2000, 1, 1, 0, 0, 5)")
	eq(t, "dt us", DateTimeRepr(2000, 1, 1, 0, 0, 0, 7), "datetime.datetime(2000, 1, 1, 0, 0, 0, 7)")
	eq(t, "dt str", DateTimeStr(1999, 12, 31, 23, 59, 59, 999000), "1999-12-31 23:59:59.999000")
	eq(t, "dt str0", DateTimeStr(5, 1, 1, 0, 0, 0, 0), "0005-01-01 00:00:00")
}

func TestStrRepr(t *testing.T) {
	eq(t, "mixed", StrRepr("a\tb\n\r\x00\x7f\u0080 ­ \U0001f600\U000e0001\"'"),
		`'a\tb\n\r\x00\x7f\x80\xa0\xad\u2028`+"\U0001f600"+`\U000e0001"\''`)
	eq(t, "squote", StrRepr("'"), `"'"`)
	eq(t, "both", StrRepr(`"'`), `'"\''`)
	eq(t, "invalid utf8", StrRepr("a\xffb"), `'a\udcffb'`)
}

func TestFloatRepr(t *testing.T) {
	cases := map[float64]string{
		1e16: "1e+16", 1e-5: "1e-05", 0.0001: "0.0001", 123456789012345.0: "123456789012345.0",
		1.2345678901234568e+16: "1.2345678901234568e+16", 1e-7: "1e-07", 1e22: "1e+22",
		-1.5e-300: "-1.5e-300", 9007199254740993.0: "9007199254740992.0", 0.1: "0.1",
		math.Inf(1): "inf", math.Inf(-1): "-inf", 5e-324: "5e-324", 1234567890123456.0: "1234567890123456.0",
	}
	for f, want := range cases {
		eq(t, "repr", FloatRepr(f), want)
	}
	eq(t, "nan", FloatRepr(math.Copysign(math.NaN(), -1)), "nan")
}

func errIs(t *testing.T, what string, err error, want string) {
	t.Helper()
	if err == nil {
		t.Errorf("%s: no error, want %q", what, want)
		return
	}
	if errText(err) != want {
		t.Errorf("%s: got error %q, want %q", what, errText(err), want)
	}
}

func TestParseIntErrors(t *testing.T) {
	_, err := ParseInt(strings.Repeat("y", 300), 10)
	errIs(t, "truncated repr", err, "ValueError: invalid literal for int() with base 10: '"+strings.Repeat("y", 199))
	_, err = ParseInt(strings.Repeat("é", 300), 10)
	errIs(t, "truncated repr runes", err, "ValueError: invalid literal for int() with base 10: '"+strings.Repeat("é", 199))
	_, err = ParseInt("010", 0)
	errIs(t, "old octal", err, "ValueError: invalid literal for int() with base 0: '010'")
	_, err = ParseInt("1", 37)
	errIs(t, "base", err, "ValueError: int() base must be >= 2 and <= 36, or 0")
	_, err = ParseInt(strings.Repeat("1", 5000), 10)
	errIs(t, "limit", err, "ValueError: Exceeds the limit (4300 digits) for integer string conversion: value has 5000 digits; use sys.set_int_max_str_digits() to increase the limit")
	z, err := ParseInt(" ١٢\u0085", 10)
	if err != nil || z.Int64() != 12 {
		t.Errorf("unicode digits: %v %v", z, err)
	}
	_, err = ParseInt("1\x1c", 10)
	errIs(t, "x1c is not number whitespace", err, `ValueError: invalid literal for int() with base 10: '1\x1c'`)
	z, err = ParseInt("0x_1f", 0)
	if err != nil || z.Int64() != 31 {
		t.Errorf("0x_1f: %v %v", z, err)
	}
	big5000, _ := ParseInt(strings.Repeat("f", 5000), 16)
	_, err = IntReprChecked(big5000)
	errIs(t, "repr limit", err, "ValueError: Exceeds the limit (4300 digits) for integer string conversion; use sys.set_int_max_str_digits() to increase the limit")
}

func TestParseFloat(t *testing.T) {
	for s, want := range map[string]string{"1_0.5": "10.5", " 1.5\u0085": "1.5", "iNfInItY": "inf",
		"-nan": "nan", "1e400": "inf", "1e-400": "0.0", "١.٢": "1.2", "1.": "1.0", ".5": "0.5",
		"  -0.0  ": "-0.0"} {
		f, err := ParseFloat(s)
		if err != nil {
			t.Errorf("%q: %v", s, err)
			continue
		}
		eq(t, s, FloatRepr(f), want)
	}
	for _, s := range []string{"1\x1c", "1_e5", "1e_5", "inf_", "infx", "1e", ".", "0x1p3", "", " ", "1\x00", "+-1"} {
		_, err := ParseFloat(s)
		errIs(t, s, err, "ValueError: could not convert string to float: "+StrRepr(s))
	}
}

func TestFloatOps(t *testing.T) {
	r, _ := RoundN(2.675, 2)
	eq(t, "round 2.675", FloatRepr(r), "2.67")
	r, _ = RoundN(0.125, 2)
	eq(t, "round 0.125", FloatRepr(r), "0.12")
	r, _ = RoundN(12345.0, -2)
	eq(t, "round -2", FloatRepr(r), "12300.0")
	r, _ = RoundN(-0.04, 1)
	eq(t, "round -0.04", FloatRepr(r), "-0.0")
	_, err := RoundN(1.7976931348623157e308, -308)
	errIs(t, "round overflow", err, "OverflowError: rounded value too large to represent")
	for f, want := range map[float64]string{-0.4: "0", 2.5: "2", 3.5: "4"} {
		z, _ := Round(f)
		eq(t, "round", z.String(), want)
	}
	_, err = Round(math.NaN())
	errIs(t, "round nan", err, "ValueError: cannot convert float NaN to integer")
	_, err = FloatMod(1, 0)
	errIs(t, "mod 0", err, "ZeroDivisionError: float modulo")
	m, _ := FloatMod(-1, 3)
	eq(t, "mod sign", FloatRepr(m), "2.0")
	_, err = FloatFloorDiv(1, 0)
	errIs(t, "floordiv 0", err, "ZeroDivisionError: float floor division by zero")
}

func TestStrMethods(t *testing.T) {
	eq(t, "split", strings.Join(Split("  a\tb\x1c c　d\u0085  "), "|"), "a|b|c|d")
	eq(t, "splitlines", strings.Join(SplitLines("a\x1cb\x1dc\x1ed\u0085e f g\r\nh\ri\x0bj\x0ck\n", false), "|"),
		"a|b|c|d|e|f|g|h|i|j|k")
	eq(t, "splitlines keep", strings.Join(SplitLines("a\r\nb", true), "|"), "a\r\n|b")
	eq(t, "expandtabs", ExpandTabs("ab\tc\n\td\re\tf", 8), "ab      c\n        d\re       f")
	eq(t, "expandtabs 0", ExpandTabs("\t", 0), "")
	eq(t, "expandtabs runes", ExpandTabs("aé\tb", 4), "aé  b")
	eq(t, "title", Title("my-tag:hello ǆx ßa ﬁx ΣΑΣ Σ aΣ."), "My-Tag:Hello ǅx Ssa Fix Σας Σ Aς.")
	eq(t, "strip", StripSpace(" \x1c\x1fabc\u0085  "), "abc")
	eq(t, "strip chars", Strip("xxabcxx", "x"), "abc")
	eq(t, "strip empty chars", Strip(" abc ", ""), " abc ")
	none := (*string)(nil)
	eq(t, "strip None", StripOpt(" abc ", none), "abc")
	eq(t, "splitn", strings.Join(SplitSpaceN("a b  c ", 1), "|"), "a|b  c ")
	if Find("abc", "", 5, math.MaxInt) != -1 || RFind("abc", "", 1, 2) != 2 || Count("aaa", "", 0, math.MaxInt) != 4 ||
		Count("abc", "", 4, math.MaxInt) != 0 {
		t.Error("find/rfind/count empty-substring rules")
	}
	eq(t, "slice", Slice("héllo", -4, -1), "éll")
	if _, err := SplitSep("a", "", -1); err == nil || errText(err) != "ValueError: empty separator" {
		t.Errorf("empty separator: %v", err)
	}
	if !IsSpace("\x1c\u0085") || IsSpace("") || IsSpace("​") {
		t.Error("isspace")
	}
	if IsSpaceRune('​') || !IsSpaceRune('　') {
		t.Error("IsSpaceRune")
	}
}

func TestChr(t *testing.T) {
	s, ok, err := Chr(big.NewInt(0x41))
	if s != "A" || !ok || err != nil {
		t.Errorf("chr(0x41) = %q %v %v", s, ok, err)
	}
	_, ok, err = Chr(big.NewInt(0xd800))
	if ok || err != nil {
		t.Errorf("chr(0xd800) should report a surrogate: %v %v", ok, err)
	}
	_, _, err = Chr(big.NewInt(0x110000))
	errIs(t, "chr range", err, "ValueError: chr() arg not in range(0x110000)")
	_, _, err = Chr(new(big.Int).Lsh(big.NewInt(1), 63))
	errIs(t, "chr big", err, "OverflowError: Python int too large to convert to C int")
}

func TestDateTime(t *testing.T) {
	_, err := NewDateTime(0, 1, 1, 0, 0, 0, 0)
	errIs(t, "year 0", err, "ValueError: year 0 is out of range")
	_, err = NewDateTime(2001, 2, 29, 0, 0, 0, 0)
	errIs(t, "feb 29", err, "ValueError: day is out of range for month")
	_, err = NewDateTime(2000, 1, 1, 0, 0, 60, 0)
	errIs(t, "second", err, "ValueError: second must be in 0..59")
	_, err = NewDateTimeBig(new(big.Int).Lsh(big.NewInt(1), 31), big.NewInt(1), big.NewInt(1), new(big.Int), new(big.Int), new(big.Int), new(big.Int))
	errIs(t, "c int", err, "OverflowError: signed integer is greater than maximum")
	dt, _ := NewDateTime(9999, 12, 31, 23, 59, 59, 0)
	_, err = AddMicroseconds(dt, big.NewInt(1000000))
	errIs(t, "overflow", err, "OverflowError: date value out of range")
	dt, _ = NewDateTime(1999, 12, 31, 23, 59, 59, 999999)
	dt, err = AddMicroseconds(dt, big.NewInt(1))
	if err != nil {
		t.Fatal(err)
	}
	eq(t, "carry", dt.PyRepr(), "datetime.datetime(2000, 1, 1, 0, 0)")
	if Weekday(2000, 1, 1) != 5 || Weekday(2000, 13, 1) != -1 {
		t.Error("weekday")
	}
}

func TestStrptime(t *testing.T) {
	r, err := Strptime("1999-12-31T23:59:59.999", "%Y-%m-%dT%H:%M:%S.%f")
	if err != nil {
		t.Fatal(err)
	}
	eq(t, "default fmt", r.PyRepr(), "datetime.datetime(1999, 12, 31, 23, 59, 59, 999000)")
	d, err := StrptimeDate("1999-12-31", "%Y-%m-%d")
	if err != nil || d != (DateTimeFields{Year: 1999, Month: 12, Day: 31}) {
		t.Errorf("date: %v %v", d, err)
	}
	for _, c := range [][3]string{
		{"1999-02-30", "%Y-%m-%d", "ValueError: day is out of range for month"},
		{"1999-12-31x", "%Y-%m-%d", "ValueError: unconverted data remains: x"},
		{"1999/12/31", "%Y-%m-%d", "ValueError: time data '1999/12/31' does not match format '%Y-%m-%d'"},
		{"1", "%Q", "ValueError: 'Q' is a bad directive in format '%Q'"},
		{"1", "% ", "ValueError: '%' is a bad directive in format '% '"},
		{"5 5", "%d %d", "re.error: redefinition of group name 'd' as group 2; was group 1 at position 48"},
		{"Frİday", "%A", "ValueError: 'fri̇day' is not in list"},
	} {
		_, err := Strptime(c[0], c[1])
		errIs(t, c[0]+" "+c[1], err, c[2])
	}
	r, err = Strptime("ſtraſſe 5", "STRASSE %d")
	if err != nil || r.Day != 5 {
		t.Errorf("ignorecase extra cases: %v %v", r, err)
	}
	r, err = Strptime("UTC +0000", "%Z %z")
	if err != nil {
		t.Fatal(err)
	}
	eq(t, "aware", r.PyRepr(), "datetime.datetime(1900, 1, 1, 0, 0, tzinfo=datetime.timezone(datetime.timedelta(0), 'UTC'))")
}

// Port of CPython 3.12's unicodedata / str character predicates (PSF License Version 2;
// Copyright (c) 2001-2024 Python Software Foundation).

package pyuni

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

func bit(v bool) byte {
	if v {
		return '1'
	}
	return '0'
}

func esc(s []rune) string {
	parts := []string{}
	for _, r := range s {
		parts = append(parts, fmt.Sprintf("%x", r))
	}
	return strings.Join(parts, " ")
}

// dumpLine is gen/dump.py's line(): every predicate and mapping of one code point.
func dumpLine(c rune) string {
	preds := []bool{
		IsAlpha(c), IsDecimal(c), IsDigit(c), IsNumeric(c), IsAlnum(c), IsSpace(c), IsPrintable(c),
		IsLower(c), IsUpper(c), IsTitle(c), IsCased(c), IsCaseIgnorable(c), IsIdentifierStart(c),
		IsIdentifierContinue(c), IsLineBreak(c),
	}
	pb := make([]byte, len(preds))
	for i, p := range preds {
		pb[i] = bit(p)
	}
	dec := "-"
	if v, ok := IsDecimalValue(c); ok {
		dec = fmt.Sprint(v)
	}
	return fmt.Sprintf("%x %s %x %x %c [%s] [%s] [%s] %s [%s]\n", c, pb, SimpleLower(c), SimpleUpper(c),
		bit(IsSreCased(c)), esc(LowerRune(c)), esc(UpperRune(c)), esc(CaseFoldRune(c)), dec, esc(ExtraCases(c)))
}

// TestAllCodePoints compares every predicate and mapping of all 0x110000 code points with the
// dump CPython 3.12 produced (gen/dump.py), through its sha256. Set PYUNI_DUMP to the full dump
// to see the differing lines.
func TestAllCodePoints(t *testing.T) {
	want, err := os.ReadFile("testdata/codepoints.sha256")
	if err != nil {
		t.Fatal(err)
	}
	var ref *bufio.Scanner
	if p := os.Getenv("PYUNI_DUMP"); p != "" {
		f, err := os.Open(p)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		ref = bufio.NewScanner(f)
	}
	h := sha256.New()
	diffs := 0
	for c := rune(0); c <= MaxRune; c++ {
		ln := dumpLine(c)
		if c < 0xd800 || c > 0xdfff {
			s := string(c)
			if Lower(s) != string(LowerRune(c)) || Upper(s) != string(UpperRune(c)) || CaseFold(s) != string(CaseFoldRune(c)) {
				t.Errorf("string and rune mappings of %x disagree", c)
			}
		}
		h.Write([]byte(ln))
		if ref != nil {
			if !ref.Scan() {
				t.Fatalf("reference dump ends at %x", c)
			}
			if got := strings.TrimSuffix(ln, "\n"); got != ref.Text() {
				diffs++
				if diffs <= 20 {
					t.Errorf("code point %x:\n got  %s\n want %s", c, got, ref.Text())
				}
			}
		}
	}
	if ref != nil && diffs > 0 {
		t.Fatalf("%d code points differ", diffs)
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != strings.TrimSpace(string(want)) {
		t.Fatalf("dump sha256 %s, CPython 3.12 gives %s (rerun with PYUNI_DUMP=full dump)", got, want)
	}
}

// TestStrings checks str.lower/upper/casefold, including the final-sigma rule, on strings
// recorded from CPython 3.12.
func TestStrings(t *testing.T) {
	data, err := os.ReadFile("testdata/strings.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Unicode string      `json:"unicode"`
		Cases   [][4]string `json:"cases"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Unicode != UnicodeVersion {
		t.Fatalf("testdata is Unicode %s, tables are %s", doc.Unicode, UnicodeVersion)
	}
	if len(doc.Cases) < 1000 {
		t.Fatalf("only %d cases", len(doc.Cases))
	}
	for _, c := range doc.Cases {
		if got := Lower(c[0]); got != c[1] {
			t.Errorf("Lower(%+q) = %+q, want %+q", c[0], got, c[1])
		}
		if got := string(LowerRunes([]rune(c[0]))); got != c[1] {
			t.Errorf("LowerRunes(%+q) = %+q, want %+q", c[0], got, c[1])
		}
		if got := Upper(c[0]); got != c[2] {
			t.Errorf("Upper(%+q) = %+q, want %+q", c[0], got, c[2])
		}
		if got := CaseFold(c[0]); got != c[3] {
			t.Errorf("CaseFold(%+q) = %+q, want %+q", c[0], got, c[3])
		}
	}
}

// TestKnownValues pins a few values where CPython 3.12 (Unicode 15.0.0) differs from Go's
// unicode package or from the simple mappings.
func TestKnownValues(t *testing.T) {
	checks := []struct {
		name string
		got  any
		want any
	}{
		{"Lower(ΑΣ)", Lower("ΑΣ"), "ας"},
		{"Lower(Σ)", Lower("Σ"), "σ"},
		{"Lower(ΑΣα)", Lower("ΑΣα"), "ασα"},
		{"Lower(İ)", Lower("İ"), "i\u0307"},
		{"Upper(ß)", Upper("ß"), "SS"},
		{"CaseFold(ß)", CaseFold("ß"), "ss"},
		{"CaseFold(ﬃ)", CaseFold("ﬃ"), "ffi"},
		{"SimpleLower(İ)", SimpleLower('İ'), 'i'},
		{"SimpleUpper(ß)", SimpleUpper('ß'), 'S'},
		{"SimpleUpper(ſ)", SimpleUpper('ſ'), 'S'},
		{"SimpleLower(K)", SimpleLower('\u212a'), 'k'},
		{"IsSpace(0x1c)", IsSpace(0x1c), true},
		{"IsSpace(0x200b)", IsSpace(0x200b), false},
		{"IsAlpha(U+0C5C) (Unicode 16)", IsAlpha(0xc5c), false},
		{"IsPrintable(U+1FAE9) (Unicode 16)", IsPrintable(0x1fae9), false},
		{"IsPrintable(space)", IsPrintable(' '), true},
		{"IsPrintable(U+00A0)", IsPrintable(0xa0), false},
		{"IsDigit(²)", IsDigit('²'), true},
		{"IsDecimal(²)", IsDecimal('²'), false},
		{"IsNumeric(Ⅷ)", IsNumeric('Ⅷ'), true},
		{"IsAlnum(Ⅷ)", IsAlnum('Ⅷ'), true},
		{"IsIdentifierStart(_)", IsIdentifierStart('_'), true},
		{"IsIdentifierStart(1)", IsIdentifierStart('1'), false},
		{"IsIdentifierContinue(1)", IsIdentifierContinue('1'), true},
		{"IsIdentifier(a·b)", IsIdentifier("a·b"), true},
		{"IsTitle(ǅ)", IsTitle('ǅ'), true},
		{"IsCased(ʰ)", IsCased('ʰ'), true},
		{"IsCaseIgnorable(ʰ)", IsCaseIgnorable('ʰ'), true},
		{"IsSreCased(ß)", IsSreCased('ß'), true},
		{"ExtraCases(s)", string(ExtraCases('s')), "ſ"},
		{"IsAlpha(-1)", IsAlpha(-1), false},
		{"SimpleLower(0x110000)", SimpleLower(0x110000), rune(0x110000)},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
	if v, ok := IsDecimalValue('٣'); !ok || v != 3 {
		t.Errorf("IsDecimalValue(٣) = %d, %v", v, ok)
	}
	if _, ok := IsDecimalValue('²'); ok {
		t.Errorf("IsDecimalValue(²) reported a decimal")
	}
}

// TestNames compares Name for every code point with CPython 3.12's unicodedata.name (through
// the sha256 gen/gen_names.py recorded) and checks that Lookup inverts it.
func TestNames(t *testing.T) {
	want, err := os.ReadFile("testdata/names.sha256")
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.New()
	for c := rune(0); c <= MaxRune; c++ {
		n, ok := Name(c)
		if !ok {
			n = "-"
		} else {
			if got, ok := Lookup(n); !ok || got != c {
				t.Errorf("Lookup(%q) = %x, %v; want %x", n, got, ok, c)
			}
			if !strings.HasPrefix(n, "HANGUL SYLLABLE ") && !strings.HasPrefix(n, "CJK UNIFIED IDEOGRAPH-") {
				if got, ok := Lookup(strings.ToLower(n)); !ok || got != c {
					t.Errorf("Lookup(%q) = %x, %v; want %x", strings.ToLower(n), got, ok, c)
				}
			}
		}
		fmt.Fprintf(h, "%x %s\n", c, n)
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != strings.TrimSpace(string(want)) {
		t.Fatalf("name dump sha256 %s, CPython 3.12 gives %s", got, want)
	}
	for _, c := range []struct {
		name string
		r    rune
		ok   bool
	}{
		{"LATIN CAPITAL LETTER GHA", 0x1a2, true}, {"latin capital letter gha", 0x1a2, true},
		{"BYTE ORDER MARK", 0xfeff, true}, {"LINE FEED", 0xa, true}, {"END OF LINE", 0xa, true},
		{"NULL", 0, true}, {"BEL", 7, true}, {"CJK UNIFIED IDEOGRAPH-04E00", 0x4e00, true},
		{"CJK UNIFIED IDEOGRAPH-4e00", 0, false}, {"cjk unified ideograph-4e00", 0, false},
		{"CJK UNIFIED IDEOGRAPH-2B73A", 0, false}, {"HANGUL SYLLABLE GAG", 0xac01, true},
		{"hangul syllable ga", 0, false}, {"HANGUL SYLLABLE ", 0, false}, {"TANGUT IDEOGRAPH-17000", 0, false},
		{"cjk compatibility ideograph-f900", 0xf900, true}, {"LATIN SMALL LETTER A ", 0, false},
		{"LATIN CAPITAL LETTER A WITH MACRON AND GRAVE", 0, false}, {"KEYCAP NUMBER SIGN", 0, false},
	} {
		r, ok := Lookup(c.name)
		if r != c.r || ok != c.ok {
			t.Errorf("Lookup(%q) = %x, %v; want %x, %v", c.name, r, ok, c.r, c.ok)
		}
	}
}

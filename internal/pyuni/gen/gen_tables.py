#!/usr/bin/env python3
"""Write internal/pyuni/tables.go from CPython 3.12's own str methods.

Every table is what CPython 3.12 (Unicode 15.0.0) answers for each of the 0x110000 code points,
probed through str methods and _sre, so the Go side reproduces the interpreter rather than a
reading of the Unicode data files.  Go's unicode package (Unicode 17.0.0) is never consulted.

Run inside the pinned image (see ../../../../DESIGN.md):

    docker run --rm --network none -v $PWD:/w -w /w \
        python:3.12-bookworm@sha256:14a4933e6cef421424f7a66d89a3f4eafcbd0d90e0a822251f298e05132dc735 \
        python3 gen/gen_tables.py tables.go

from gold/internal/pyuni.
"""
import sys
import unicodedata

import _sre
from re import _casefix

MAX = 0x110000

# Property bits, in the order of the Go constants in pyuni.go.
PROPS = [
    ("propAlpha", str.isalpha),
    ("propDecimal", str.isdecimal),
    ("propDigit", str.isdigit),
    ("propNumeric", str.isnumeric),
    ("propSpace", str.isspace),
    ("propPrintable", str.isprintable),
    ("propLower", str.islower),
    ("propUpper", str.isupper),
    ("propTitle", str.istitle),
    ("propCased", None),
    ("propCaseIgnorable", None),
    ("propIDStart", str.isidentifier),
    ("propIDContinue", None),
    ("propLineBreak", None),
]


def cased(ch):
    # str.title lower-cases a letter that follows a cased character (_PyUnicode_IsCased).
    return (ch + "A").title()[-1] == "a"


def case_ignorable(ch):
    # The final-sigma rule skips case-ignorable characters going backwards.
    if not cased(ch):
        return ("A" + ch + "Σ").lower()[-1] == "ς"
    # A cased character that is also case-ignorable is skipped, leaving nothing cased before Σ.
    return (ch + "Σ").lower()[-1] != "ς"


def id_continue(ch):
    return ("a" + ch).isidentifier()


def line_break(ch):
    return len(("x" + ch + "y").splitlines()) == 2


def props_of(c):
    ch = chr(c)
    bits = 0
    for i, (name, fn) in enumerate(PROPS):
        if name == "propCased":
            v = cased(ch)
        elif name == "propCaseIgnorable":
            v = case_ignorable(ch)
        elif name == "propIDContinue":
            v = id_continue(ch)
        elif name == "propLineBreak":
            v = line_break(ch)
        else:
            v = fn(ch)
        if v:
            bits |= 1 << i
    return bits


def mapping(fn):
    """(runs of single-rune deltas, multi-rune specials) for c -> fn(c) where it differs from c."""
    singles, specials = [], {}
    for c in range(MAX):
        m = fn(c)
        if m == chr(c):
            continue
        if len(m) == 1:
            singles.append((c, ord(m) - c))
        else:
            specials[c] = m
    runs = []  # (lo, hi, step, delta); step 0 while a run has one member
    for c, d in singles:
        if runs:
            lo, hi, step, delta = runs[-1]
            if delta == d and ((step == 0 and c - hi in (1, 2)) or (step and c - hi == step)):
                runs[-1] = (lo, c, c - hi, delta)
                continue
        runs.append((c, c, 0, d))
    return [(lo, hi, step or 1, d) for lo, hi, step, d in runs], specials


def go_str(s):
    return '"' + "".join(f"\\u{ord(ch):04x}" if ord(ch) <= 0xFFFF else f"\\U{ord(ch):08x}" for ch in s) + '"'


def emit_mapping(name, doc, m, simple=False):
    runs, specials = m
    lines = [f"// {name}Runs: {doc}", f"var {name}Runs = []caseRun{{"]
    flat = [f"{{0x{lo:x}, 0x{hi:x}, {st}, {d}}}," for lo, hi, st, d in runs]
    for i in range(0, len(flat), 4):
        lines.append("\t" + " ".join(flat[i:i + 4]))
    lines.append("}")
    if simple:
        assert not specials, name
        return "\n".join(lines)
    lines.append("")
    lines.append(f"// {name}Special holds the code points whose {name} mapping has several code points.")
    lines.append(f"var {name}Special = map[rune]string{{")
    for c in sorted(specials):
        lines.append(f"\t0x{c:05x}: {go_str(specials[c])},")
    lines.append("}")
    return "\n".join(lines)


def main():
    out = sys.argv[1]
    # property segments: start code point and the property bits that hold up to the next start
    starts, bits = [], []
    prev = None
    for c in range(MAX):
        b = props_of(c)
        if b != prev:
            starts.append(c)
            bits.append(b)
            prev = b
    # decimal values
    decimal = []
    for c in range(MAX):
        ch = chr(c)
        if ch.isdecimal():
            base = c - int(ch)
            assert unicodedata.decimal(ch) == int(ch)
            if decimal and decimal[-1][2] == base and decimal[-1][1] == c - 1:
                decimal[-1] = (decimal[-1][0], c, base)
            else:
                decimal.append((c, c, base))
    simple_lower = mapping(lambda c: chr(_sre.unicode_tolower(c)))
    simple_upper = mapping(lambda c: chr(c).upper()[0])
    full_lower = mapping(lambda c: chr(c).lower())
    full_upper = mapping(lambda c: chr(c).upper())
    full_fold = mapping(lambda c: chr(c).casefold())
    for c in range(MAX):
        lo = _sre.unicode_tolower(c)
        up = ord(chr(c).upper()[0])
        assert lo == ord(chr(c).lower()[0]), hex(c)
        assert _sre.unicode_iscased(c) == (c != lo or c != up), hex(c)

    parts = [
        "// Code generated by gen/gen_tables.py from CPython "
        f"{sys.version.split()[0]} (Unicode {unicodedata.unidata_version}). DO NOT EDIT.",
        "",
        "// Port of CPython 3.12's unicodedata / str character predicates (PSF License Version 2;",
        "// Copyright (c) 2001-2024 Python Software Foundation).",
        "",
        "package pyuni",
        "",
        "// UnicodeVersion is the Unicode database the tables describe, as CPython 3.12 uses it.",
        f'const UnicodeVersion = "{unicodedata.unidata_version}"',
        "",
        "// PythonVersion is the CPython release the tables were probed from.",
        f'const PythonVersion = "{sys.version.split()[0]}"',
        "",
        "// propStarts[i] is the first code point of a segment whose properties are propBits[i].",
        "var propStarts = []uint32{",
    ]
    flat = [f"0x{s:x}," for s in starts]
    for i in range(0, len(flat), 12):
        parts.append("\t" + " ".join(flat[i:i + 12]))
    parts.append("}")
    parts.append("")
    parts.append("// propBits are the property bits of the segments starting at propStarts.")
    parts.append("var propBits = []uint16{")
    flat = [f"0x{b:x}," for b in bits]
    for i in range(0, len(flat), 14):
        parts.append("\t" + " ".join(flat[i:i + 14]))
    parts.append("}")
    parts.append("")
    parts.append("// decimalRuns are the str.isdecimal ranges; a code point c in a run has the digit value c - base.")
    parts.append("var decimalRuns = []decimalRun{")
    flat = [f"{{0x{a:x}, 0x{b:x}, 0x{base:x}}}," for a, b, base in decimal]
    for i in range(0, len(flat), 4):
        parts.append("\t" + " ".join(flat[i:i + 4]))
    parts.append("}")
    parts += [
        "",
        emit_mapping("simpleLower", "_PyUnicode_ToLowercase (what _sre.unicode_tolower returns).", simple_lower, True),
        "",
        emit_mapping("simpleUpper", "_PyUnicode_ToUppercase (what sre's upper-casing of a lowered character uses).", simple_upper, True),
        "",
        emit_mapping("fullLower", "the full lowercase mapping of str.lower, except the contextual final sigma.", full_lower),
        "",
        emit_mapping("fullUpper", "the full uppercase mapping of str.upper.", full_upper),
        "",
        emit_mapping("fullFold", "the full case folding of str.casefold.", full_fold),
        "",
        "// extraCases is re._casefix._EXTRA_CASES: a lowercased code point maps to the other lowercased",
        "// code points that share its uppercase form.",
        "var extraCases = map[rune][]rune{",
    ]
    for k in sorted(_casefix._EXTRA_CASES):
        vals = ", ".join(f"0x{v:x}" for v in _casefix._EXTRA_CASES[k])
        parts.append(f"\t0x{k:05x}: {{{vals}}},")
    parts.append("}")
    with open(out, "w", encoding="utf-8") as f:
        f.write("\n".join(parts) + "\n")
    print(f"wrote {out}: {len(starts)} segments, {len(decimal)} decimal runs, "
          f"lower {len(full_lower[0])}+{len(full_lower[1])} upper {len(full_upper[0])}+{len(full_upper[1])} "
          f"fold {len(full_fold[0])}+{len(full_fold[1])}")


if __name__ == "__main__":
    main()

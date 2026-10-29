#!/usr/bin/env python3
"""Reference dump for the exhaustive pyuni test, computed straight from CPython 3.12's str methods.

Writes testdata/codepoints.sha256 (the sha256 of one line per code point, see line()) and
testdata/strings.json (str.lower/upper/casefold of random strings, for the final-sigma rule).
With a third argument it also writes the full per-code-point dump (about 60 MB) for diffing.

    docker run --rm --network none -e PYTHONHASHSEED=0 -v $PWD:/w -w /w <pinned python:3.12 image> \
        python3 gen/dump.py testdata/codepoints.sha256 testdata/strings.json [full-dump.txt]
"""
import hashlib
import json
import random
import sys
import unicodedata

import _sre
from re import _casefix

MAX = 0x110000


def b(v):
    return "1" if v else "0"


def esc(s):
    return " ".join(f"{ord(c):x}" for c in s)


def line(c):
    ch = chr(c)
    preds = [
        ch.isalpha(), ch.isdecimal(), ch.isdigit(), ch.isnumeric(), ch.isalnum(), ch.isspace(),
        ch.isprintable(), ch.islower(), ch.isupper(), ch.istitle(),
        (ch + "A").title()[-1] == "a",  # cased
        ("A" + ch + "Σ").lower()[-1] == "ς" if (ch + "A").title()[-1] != "a"
        else (ch + "Σ").lower()[-1] != "ς",  # case-ignorable
        ch.isidentifier(), ("a" + ch).isidentifier(),
        len(("x" + ch + "y").splitlines()) == 2,
    ]
    dec = str(int(ch)) if ch.isdecimal() else "-"
    extra = esc("".join(map(chr, _casefix._EXTRA_CASES.get(c, ()))))
    return (f"{c:x} {''.join(map(b, preds))} {_sre.unicode_tolower(c):x} {ord(ch.upper()[0]):x} "
            f"{b(_sre.unicode_iscased(c))} [{esc(ch.lower())}] [{esc(ch.upper())}] [{esc(ch.casefold())}] "
            f"{dec} [{extra}]\n")


def main():
    h = hashlib.sha256()
    full = open(sys.argv[3], "w", encoding="ascii") if len(sys.argv) > 3 else None
    for c in range(MAX):
        ln = line(c)
        h.update(ln.encode("ascii"))
        if full:
            full.write(ln)
    with open(sys.argv[1], "w") as f:
        f.write(h.hexdigest() + "\n")
    rnd = random.Random(20261030)
    pool = ("ΣσςaAżͅ'.:­ʰᴬİıßſK"
            "ẞﬀŉǅ Ⅰ\U00010400\U0001e900̀‍")
    cases = []
    for _ in range(2000):
        n = rnd.randint(0, 8)
        s = "".join(rnd.choice(pool) if rnd.random() < 0.8 else chr(rnd.randrange(0x110000)) for _ in range(n))
        if "\ud800" <= s[:1] <= "\udfff":
            continue
        s = "".join(ch for ch in s if not ("\ud800" <= ch <= "\udfff"))
        cases.append([s, s.lower(), s.upper(), s.casefold()])
    with open(sys.argv[2], "w", encoding="utf-8") as f:
        json.dump({"python": sys.version.split()[0], "unicode": unicodedata.unidata_version, "cases": cases}, f,
                  ensure_ascii=True, indent=0)
    print("ok", h.hexdigest())


if __name__ == "__main__":
    main()

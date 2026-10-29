# Differential test vectors for internal/pyval, produced by CPython 3.12 itself.
#
# Run inside the pinned image (see DESIGN.md), e.g.
#   python3 gen_vectors.py <kind> <count> <seed> > out.jsonl
# kinds: floatrepr floatparse intparse strrepr strmeth strptime floatops asdatetime
# Each output line is a JSON object {"k": kind, "in": [...], "out": ..., "err": "Class: msg"}.
# Large runs (seed 1; strptime seed 2; floatrepr 250000, strptime 80000, the rest 30000-60000)
# go to a scratch dir and are read by TestGoldenLarge through PYVAL_GOLDEN_DIR; ../testdata/
# holds every 100th-400th line of those runs (plus the first 100 strptime lines).
import datetime, json, math, random, re, struct, sys, _strptime

assert sys.version_info[:2] == (3, 12), sys.version

kind, count, seed = sys.argv[1], int(sys.argv[2]), int(sys.argv[3])
rnd = random.Random(seed)


def emit(inp, f):
    rec = {"k": kind, "in": inp}
    try:
        rec["out"] = f()
    except Exception as e:  # noqa: BLE001
        rec["err"] = "%s: %s" % ("re.error" if isinstance(e, re.error) else type(e).__name__, e)
    print(json.dumps(rec, ensure_ascii=True))


def bits(f):
    return struct.pack(">d", f).hex()


def from_bits(h):
    return struct.unpack(">d", bytes.fromhex(h))[0]


def rand_cp(pool):
    while True:
        c = rnd.choice(pool)
        if isinstance(c, tuple):
            c = rnd.randint(*c)
        if not (0xD800 <= c <= 0xDFFF):
            return chr(c)


UNI_DIGITS = [c for c in range(0x110000) if not (0xD800 <= c <= 0xDFFF) and chr(c).isdecimal() and c > 127]
SPACES = [c for c in range(0x110000) if not (0xD800 <= c <= 0xDFFF) and chr(c).isspace()]


def mutate_number(s):
    ops = rnd.randint(0, 3)
    for _ in range(ops):
        r = rnd.random()
        if r < 0.2 and s:
            i = rnd.randrange(len(s))
            if s[i].isdigit():
                d = int(s[i])
                cands = [c for c in UNI_DIGITS if int(chr(c)) == d]
                s = s[:i] + chr(rnd.choice(cands)) + s[i + 1:]
        elif r < 0.35:
            i = rnd.randrange(len(s) + 1)
            s = s[:i] + "_" + s[i:]
        elif r < 0.5:
            sp = chr(rnd.choice(SPACES + [32] * 10))
            s = (sp + s) if rnd.random() < 0.5 else (s + sp)
        elif r < 0.6 and s:
            i = rnd.randrange(len(s))
            s = s[:i] + s[i + 1:]
        elif r < 0.7:
            i = rnd.randrange(len(s) + 1)
            s = s[:i] + rnd.choice("+-.eE0xXoObB_ \x00\x1c\x85é?a9") + s[i:]
        elif r < 0.8:
            s = rnd.choice(["+", "-", " ", ""]) + s
    return s


def rand_number_text():
    r = rnd.random()
    if r < 0.15:
        return rnd.choice(["inf", "infinity", "nan", "INF", "Infinity", "NaN", "iNfInItY", "infinit", "nana", "in"])
    digits = "".join(rnd.choice("0123456789") for _ in range(rnd.choice([0, 1, 1, 2, 3, 5, 8, 17, 20, 40, 400])))
    s = digits
    if rnd.random() < 0.5:
        s += "." + "".join(rnd.choice("0123456789") for _ in range(rnd.choice([0, 1, 2, 5, 17, 30])))
    if rnd.random() < 0.4:
        s += rnd.choice("eE") + rnd.choice(["", "+", "-"]) + str(rnd.choice([0, 1, 5, 22, 23, 308, 309, 324, 325, 400, 99999]))
    if rnd.random() < 0.3:
        s = rnd.choice("+-") + s
    return s


def gen_floatrepr():
    edge = [0.0, -0.0, math.inf, -math.inf, math.nan, 5e-324, -5e-324, 2.2250738585072014e-308,
            2.225073858507201e-308, 1.7976931348623157e308, 1e16, 1e15, 9999999999999998.0,
            1234567890123456.0, 12345678901234567.0, 1e-4, 1e-5, 9.999999999999999e-05, 0.0001,
            0.1, 0.2, 0.3, 1 / 3, 2 / 3, 123456789012345.0, 1.2345678901234568e+16, 2.0 ** 53,
            2.0 ** 53 + 2, 2.0 ** 63, 2.0 ** 64, 1e22, 1e23, 5e-5, 0.5, 1.5, 100.0, 1e100]
    for k in range(-325, 310):
        for v in (10.0 ** k if -308 <= k <= 308 else float("1e%d" % k),):
            edge += [v, math.nextafter(v, 0), math.nextafter(v, math.inf), -v]
    for f in edge:
        emit([bits(f)], lambda: repr(f))
    for _ in range(count):
        r = rnd.random()
        if r < 0.7:
            f = from_bits("%016x" % rnd.getrandbits(64))
        elif r < 0.85:
            f = rnd.uniform(-1e17, 1e17) * 10.0 ** rnd.randint(-20, 0)
        else:
            f = float(round(rnd.uniform(-1e6, 1e6), rnd.randint(0, 8)))
        emit([bits(f)], lambda: repr(f))


def gen_floatparse():
    fixed = ["1\x1c", " 1.5\x85", "1_0.5", "1_e5", "1e_5", "inf_", "-nan", "+inf", "infx", "1e", ".", "1.",
             ".5", "0x1p3", "1e400", "1e-400", "١.٢", "x" * 300, "١_٢", "", " ", "\x00",
             "1\x00", "1 2", "--1", "+-1", "1__0", "_1", "1_", "1._5", "1_.5", "1e1_0", "  -0.0  ", "　-3　",
             "4" * 800, "0." + "0" * 400 + "1", "2.4703282292062327e-324", "2.4703282292062328e-324",
             "1.7976931348623158e308", "1.7976931348623159e308", "nan(123)", "infinity1", " " + "7"]
    for s in fixed:
        emit([s], lambda: bits(float(s)) if not math.isnan(float(s)) else "nan")
    for _ in range(count):
        s = mutate_number(rand_number_text())
        emit([s], lambda: bits(float(s)) if not math.isnan(float(s)) else "nan")


def gen_intparse():
    fixed = [("1\x1c", 10), (" 1\x85", 10), ("١٢", 10), ("1_000", 0), ("_1", 10), ("1__0", 16),
             ("0x1f", 10), ("0x1f", 0), ("0x1f", 16), ("0_7", 0), ("0_7", 10), ("00", 0), ("010", 0), ("0b", 16),
             ("0b", 0), ("0x_1f", 0), ("0x__1f", 16), ("x" * 300, 10), ("1" * 5000, 10), ("f" * 5000, 16),
             ("1" * 4300, 10), ("1" * 4301, 10), ("1" * 5000 + "\x00", 10), ("1" * 5000 + "\x00é", 10),
             ("1", 1), ("1", 37), ("z", 36), ("Z", 36), ("0o17", 0), ("0O17", 8), ("0B101", 2), ("-0x10", 0),
             ("+ 1", 10), ("1 ", 10), (" ", 10), ("", 0), ("\x00", 10), ("12é", 10), ("1" * 641 + "_", 10),
             ("0" * 5000, 10), ("-" + "9" * 700, 10), ("é" * 250, 10)]
    for s, b in fixed:
        emit([s, b], lambda: repr_int(int(s, b)))
    for _ in range(count):
        b = rnd.choice([10, 10, 10, 0, 0, 16, 16, 2, 8, 36, 3, 7])
        if b in (16, 36):
            digits = "".join(rnd.choice("0123456789abcdefABCDEF") for _ in range(rnd.choice([1, 2, 3, 8, 20])))
        else:
            digits = "".join(rnd.choice("0123456789") for _ in range(rnd.choice([0, 1, 2, 3, 8, 20, 50])))
        if rnd.random() < 0.3:
            digits = rnd.choice(["0x", "0o", "0b", "0X", "0", "00"]) + digits
        s = mutate_number(digits)
        emit([s, b], lambda: repr_int(int(s, b)))


def repr_int(z):
    try:
        return repr(z)
    except ValueError as e:
        return "REPR-ERR " + str(e)


def rand_text(n, pool):
    return "".join(rand_cp(pool) for _ in range(n))


TEXT_POOL = [(0, 0x7f)] * 8 + [(0x80, 0xff)] * 3 + [(0x100, 0x2fff), (0x3000, 0xffff), (0x10000, 0x10ffff),
                                                     0x1c, 0x1d, 0x1e, 0x1f, 0x85, 0xa0, 0x2028, 0x2029, 0x3000,
                                                     0x3a3, 0x130, 0x131, 0x17f, 0xdf, 0x1c5, 0xfb00, 0x345, 0x307,
                                                     ord("'"), ord('"'), ord("\\"), 9, 10, 13]


def gen_strrepr():
    # every code point, in chunks
    cps = [c for c in range(0x110000) if not (0xD800 <= c <= 0xDFFF)]
    for i in range(0, len(cps), 61):
        s = "".join(map(chr, cps[i:i + 61]))
        emit([s], lambda: repr(s))
    for s in ["", "'", '"', "'\"", "a'b", 'a"b', "\\", "\x7f\x80\x9f\xa0\xad", "\U0010ffff\U000e0001"]:
        emit([s], lambda: repr(s))
    for _ in range(count):
        s = rand_text(rnd.randint(0, 12), TEXT_POOL)
        emit([s], lambda: repr(s))


WS_POOL = [32, 9, 10, 13, 11, 12, 0x1c, 0x1d, 0x1e, 0x1f, 0x85, 0xa0, 0x2028, 0x3000, 0x200b, 0xfeff] + \
          [(ord("a"), ord("e"))] * 6 + [(ord("A"), ord("E"))] * 3 + [0x3a3, 0x3c3, 0x130, 0xdf, 0x1c5, 0x1c4,
                                                                      0x345, 0x307, ord("'"), ord("-"), ord(":"),
                                                                      0xfb01, 0x1f80, (0x370, 0x3ff), (0x10400, 0x1044f)]


def gen_strmeth():
    for _ in range(count):
        s = rand_text(rnd.randint(0, 14), WS_POOL)
        op = rnd.choice(["expandtabs", "strip", "lstrip", "rstrip", "stripc", "split", "splitn", "splitlines",
                         "title", "upper", "lower", "casefold", "isspace", "find", "rfind", "count", "slice"])
        if op == "expandtabs":
            n = rnd.choice([8, 8, 4, 1, 0, -1, 3])
            emit([op, s, n], lambda: s.expandtabs(n))
        elif op in ("strip", "lstrip", "rstrip"):
            emit([op, s], lambda: getattr(s, op)())
        elif op == "stripc":
            c = rand_text(rnd.randint(0, 3), WS_POOL)
            emit([op, s, c], lambda: s.strip(c))
        elif op == "split":
            emit([op, s], lambda: s.split())
        elif op == "splitn":
            n = rnd.randint(-1, 3)
            emit([op, s, n], lambda: s.split(None, n))
        elif op == "splitlines":
            k = rnd.random() < 0.5
            emit([op, s, k], lambda: s.splitlines(k))
        elif op in ("title", "upper", "lower", "casefold", "isspace"):
            emit([op, s], lambda: getattr(s, op)())
        elif op in ("find", "rfind", "count"):
            sub = s[rnd.randint(0, len(s)):][:rnd.randint(0, 2)] if rnd.random() < 0.7 else rand_text(1, WS_POOL)
            a, b = rnd.randint(-20, 20), rnd.randint(-20, 20)
            emit([op, s, sub, a, b], lambda: getattr(s, op)(sub, a, b))
        else:
            a, b = rnd.randint(-20, 20), rnd.randint(-20, 20)
            emit([op, s, a, b], lambda: s[a:b])


FORMATS = ["%Y-%m-%d", "%Y-%m-%dT%H:%M:%S.%f", "%d/%m/%y", "%b %d %Y", "%B %d, %Y", "%a %b %d %H:%M:%S %Y",
           "%c", "%x", "%X", "%I:%M %p", "%j %Y", "%Y %j", "%U %w %Y", "%W %a %Y", "%G-W%V-%u", "%Y%m%d",
           "%H%M%S", "%d%m%Y", "%Od %Om %Oy", "%Y-%m-%d %H:%M:%S%z", "%Z %Y", "%%Y %Y", "%y", "%m", "%d",
           "it's %Y", "%Y ' %m", "T%H", "%A, %B %d", "%e", "%", "%Y %Y", "%Y\t%m", "%H:%M:%S.%f", "%Y-%m-%d %I %p",
           "%c%x", "% %Y", "%O", "%OH:%OM:%OS", "%Ow %U %Y", "%u %V %G", "%V %Y %a", "%G %j", "%G %V", "%z", "%Z",
           "%j", "%U %Y", "%W %Y %w", "%y%m%d%H%M", "%Y-%m-%d %H:%M", "%d.%m.%Y", "(%Y)", "[%m]|%d", "%Y^$*+?",
           "été %Y", "STRASSE %d", "Kelvin %Y", "%B", "%b", "%A", "%a", "%p %I", "%I", "%f", "%Y%",
           "%Y-%m-%d\n%H", "%Y　%m", "%Y %Z %z", "%y %Y", "%Y %y", "%E", "%Oz", "%S", "%M", "%H", "%Y-%j"]


def rand_dt():
    y = rnd.choice([rnd.randint(1, 9999), rnd.randint(1900, 2100), 1, 9999, 2000, 1904, 1900])
    start = datetime.datetime(y, 1, 1)
    days = 365 + (1 if (y % 4 == 0 and (y % 100 != 0 or y % 400 == 0)) else 0)
    d = start + datetime.timedelta(days=rnd.randrange(days), seconds=rnd.randrange(86400),
                                   microseconds=rnd.choice([0, rnd.randrange(1000000)]))
    return d


def strftime_safe(d, fmt):
    try:
        return d.strftime(fmt)
    except Exception:  # noqa: BLE001
        return fmt


def mutate_date(s):
    for _ in range(rnd.choice([0, 0, 0, 1, 1, 2])):
        r = rnd.random()
        if r < 0.12:
            s = s.upper()
        elif r < 0.2:
            s = s.swapcase()
        elif r < 0.32 and s:
            i = rnd.randrange(len(s))
            if s[i].isdigit():
                s = s[:i] + chr(0x660 + int(s[i])) + s[i + 1:]
        elif r < 0.42:
            s = s.replace(" ", rnd.choice(["  ", "\t", "\x1c", "　", " ", ""]), 1)
        elif r < 0.52 and s:
            i = rnd.randrange(len(s))
            s = s[:i] + s[i + 1:]
        elif r < 0.6:
            s = s + rnd.choice(["x", "0", " ", "Z", "1"])
        elif r < 0.7 and s:
            i = rnd.randrange(len(s))
            s = s[:i] + rnd.choice("0123456789") + s[i + 1:]
        elif r < 0.75:
            s = s.replace("s", "ſ").replace("i", "ı")
        elif r < 0.8:
            s = s.replace("I", "İ").replace("K", "K")
        elif r < 0.85:
            s = " " + s
        elif r < 0.9:
            s = s.replace("'", "ʼ")
        else:
            s = rnd.choice(["+05:30", "-0000", "Z", "z", "+01:30:45.123", "+01:3045", "+0130:45", "+99:59",
                            "-23:59:59.999999", "+24:00", "UTC", "gmt", "Gmt "]) + s
    return s


def strptime_out(s, f):
    d = datetime.datetime.strptime(s, f)
    tt = _strptime._strptime(s, f)[0]
    return [repr(d), str(d), tt[6], tt[7], tt[8], tt[9]]


STRPTIME_FIXED = [
    ("2001-02-29", "%Y-%m-%d"), ("02-29", "%m-%d"), ("1999-04-31", "%Y-%m-%d"), ("0000-01-01", "%Y-%m-%d"),
    ("12:00:60", "%H:%M:%S"), ("12:00:61", "%H:%M:%S"), ("99:99:99", "%OH:%OM:%OS"), ("366 1999", "%j %Y"),
    ("366 9999", "%j %Y"), ("1 0 0001", "%w %U %Y"), ("0 0 0001", "%w %W %Y"), ("53 1 9999", "%U %w %Y"),
    ("2020-W53-1", "%G-W%V-%u"), ("2021-W53-1", "%G-W%V-%u"), ("0000-W01-1", "%G-W%V-%u"),
    ("9999-W52-7", "%G-W%V-%u"), ("+24:00", "%z"), ("+99:59", "%z"), ("-23:59:59.999999", "%z"),
    ("+01:3045", "%z"), ("+0130:45", "%z"), ("Z", "%z"), ("z", "%z"), ("UTC +0000", "%Z %z"), ("Mon", "%a"),
    ("2000 Mon 00", "%Y %a %W"), (" 5", "%d"), ("5", "%Od"), (" 5", "%Od"), ("12 AM", "%I %p"),
    ("12 pm", "%I %p"), ("1 PM", "%I %p"), ("13 pm", "%I %p"), ("%", "%"), ("%Y", "%%Y"),
    ("1999-12-31T23:59:59.999", "%Y-%m-%dT%H:%M:%S.%f"), ("1999-12-31T23:59:59.9999999", "%Y-%m-%dT%H:%M:%S.%f"),
    ("\uff11\uff19\uff19\uff19", "%Y"), ("it's 1999", "it's %Y"), ("it\u02bcs 1999", "it's %Y"),
    ("1999\x1c12", "%Y %m"), ("1999  12", "%Y %m"), ("199912", "%Y %m"), ("2000-01-01", "%Y-%m-%d%"),
    ("\u017ftra\u017f\u017fe 5", "STRASSE %d"), ("\u212aelvin 2000", "Kelvin %Y"), ("\u0130T'S 1999", "it's %Y"),
    ("Fr\u0130day", "%A"), ("02/29/00", "%x"), ("Tue Feb 29 00:00:00 2000", "%c"), ("1999-12-31", "%Y-%m-%d %%"),
    ("29 feb", "%d %b"), ("31 JUNE 2000", "%d %B %Y"), ("2000 2 31", "%Y %m %d"), ("00", "%y"), ("68", "%y"),
    ("69", "%y"), ("99", "%y"), ("0", "%d"), ("00", "%d"), ("1 1 1", "%d %m %y"), ("2000 61", "%Y %S"),
    ("13", "%I"), ("0", "%I"), ("1999 000", "%Y %j"), ("1999 0", "%Y %j"), ("1999 365", "%Y %j"),
    ("\u0661\u0669\u0669\u0669-\u0661\u0662-\u0663\u0661", "%Y-%m-%d"), ("1999-12-31", "%Y\u00a0%m"),
    ("1999\u00a012", "%Y %m"), ("1999\u200b12", "%Y %m"), ("", ""), ("x", ""), ("", "%Y"), ("a", "A"),
    ("1999", "%Y%Y"), ("1999 12", "%Y %m %d %m"), ("9999-12-31 23:59:59.999999", "%Y-%m-%d %H:%M:%S.%f"),
    ("Sunday 2000 00", "%A %Y %U"), ("Saturday 2000 00", "%A %Y %U"), ("Sunday 2000 53", "%A %Y %U"),
    ("Sunday 9999 53", "%A %Y %W"), ("1 0001 00", "%u %Y %W"),
]


def gen_strptime():
    for s, f in STRPTIME_FIXED:
        emit([s, f], lambda: strptime_out(s, f))
    for _ in range(count // 10):
        y = rnd.choice(["0000", "0001", "0004", "1900", "2000", "2001", "9999", "1904", "2100"])
        m = "%02d" % rnd.randint(0, 13)
        d = "%02d" % rnd.choice([0, 1, 28, 29, 30, 31, 32])
        f = rnd.choice(["%Y-%m-%d", "%m-%d", "%d %b %Y", "%Y %j", "%Y-%m-%d %H:%M:%S"])
        s = {"%Y-%m-%d": "%s-%s-%s" % (y, m, d), "%m-%d": "%s-%s" % (m, d),
             "%d %b %Y": "%s %s %s" % (d, rnd.choice(["Feb", "apr", "JUN", "Sep", "nov", "Jan"]), y),
             "%Y %j": "%s %03d" % (y, rnd.choice([0, 1, 59, 60, 365, 366, 367])),
             "%Y-%m-%d %H:%M:%S": "%s-%s-%s %02d:%02d:%02d" % (y, m, d, rnd.randint(0, 25), rnd.randint(0, 61), rnd.randint(0, 62))}[f]
        emit([s, f], lambda: strptime_out(s, f))
    for _ in range(count):
        f = rnd.choice(FORMATS)
        d = rand_dt()
        if rnd.random() < 0.3:
            d = d.replace(tzinfo=datetime.timezone(datetime.timedelta(minutes=rnd.randint(-1439, 1439))))
        if rnd.random() < 0.85:
            s = mutate_date(strftime_safe(d, f))
        else:
            s = "".join(rnd.choice("0123456789 -:/.TZ+apmAPMJanFebDecMon%'") for _ in range(rnd.randint(0, 20)))
        emit([s, f], lambda: strptime_out(s, f))


def gen_floatops():
    for _ in range(count):
        r = rnd.random()
        if r < 0.3:
            x = from_bits("%016x" % rnd.getrandbits(64))
        elif r < 0.6:
            x = rnd.uniform(-1000, 1000)
        else:
            x = rnd.randint(-10 ** 6, 10 ** 6) / rnd.choice([1, 2, 4, 8, 10, 100, 1000, 1024, 3])
        y = rnd.choice([1.0, -1.0, 0.5, 3.0, -2.5, 0.0, 1e-300, 1e300, math.inf, rnd.uniform(-10, 10)])
        n = rnd.choice([0, 1, 2, 3, -1, -2, 5, 10, 17, 300, 400, -310, -400, -308])
        op = rnd.choice(["mod", "floordiv", "divmod", "round", "roundn", "int"])
        if op == "mod":
            emit([op, bits(x), bits(y)], lambda: fbits(x % y))
        elif op == "floordiv":
            emit([op, bits(x), bits(y)], lambda: fbits(x // y))
        elif op == "divmod":
            emit([op, bits(x), bits(y)], lambda: [fbits(v) for v in divmod(x, y)])
        elif op == "round":
            emit([op, bits(x)], lambda: str(round(x)))
        elif op == "roundn":
            emit([op, bits(x), n], lambda: fbits(round(x, n)))
        else:
            emit([op, bits(x)], lambda: str(int(x)))


def fbits(f):
    return "nan" if math.isnan(f) else bits(f)


def as_datetime(y, mo, d, h, mi, sec):
    return datetime.datetime(y, mo, d, h, mi, int(sec)) + datetime.timedelta(microseconds=round((sec % 1) * 1_000_000))


def gen_asdatetime():
    for _ in range(count):
        y = rnd.choice([rnd.randint(0, 10000), 2000, 9999, 1, 0])
        mo = rnd.choice([rnd.randint(0, 13), 2, 12])
        d = rnd.choice([rnd.randint(0, 32), 28, 29, 30, 31])
        h = rnd.choice([rnd.randint(0, 24), 23])
        mi = rnd.choice([rnd.randint(0, 60), 59])
        sec = rnd.choice([rnd.uniform(0, 61), 59.9999995, 59.9999994, 59.99999949999, rnd.randint(0, 60) + rnd.randint(0, 10 ** 7) / 10 ** 7, 0.0])
        emit([str(y), str(mo), str(d), str(h), str(mi), bits(sec)], lambda: repr(as_datetime(y, mo, d, h, mi, sec)))
    for args in [(2**31, 1, 1, 0, 0, 0.0), (2**63, 1, 1, 0, 0, 0.0), (-2**31 - 1, 1, 1, 0, 0, 0.0)]:
        emit([str(a) for a in args[:5]] + [bits(args[5])], lambda: repr(as_datetime(*args)))


{"floatrepr": gen_floatrepr, "floatparse": gen_floatparse, "intparse": gen_intparse, "strrepr": gen_strrepr,
 "strmeth": gen_strmeth, "strptime": gen_strptime, "floatops": gen_floatops, "asdatetime": gen_asdatetime}[kind]()

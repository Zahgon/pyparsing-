// Port of CPython 3.12's Lib/_strptime.py (TimeRE, _strptime, _strptime_datetime) for the
// C/POSIX locale with TZ=UTC, with the small part of the re module its regexes need
// (IGNORECASE matching by sre's rules, named groups, backtracking alternation) (PSF License
// Version 2; Copyright (c) 2001-2024 Python Software Foundation).

package pyval

import (
	"strconv"
	"strings"

	"github.com/pyparsing/pyparsing-go/internal/pyuni"
)

// C locale data, as LocaleTime computes it (lowercased).
var (
	localeAWeekday = []string{"mon", "tue", "wed", "thu", "fri", "sat", "sun"}
	localeFWeekday = []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"}
	localeAMonth   = []string{"", "jan", "feb", "mar", "apr", "may", "jun", "jul", "aug", "sep", "oct", "nov", "dec"}
	localeFMonth   = []string{"", "january", "february", "march", "april", "may", "june", "july", "august",
		"september", "october", "november", "december"}
	localeAmPm = []string{"am", "pm"}
)

// timeRE is TimeRE() for the C locale with time.tzname == ('UTC', 'UTC') and
// time.daylight == 0.
var timeRE = func() map[string]string {
	m := map[string]string{
		"d": `(?P<d>3[0-1]|[1-2]\d|0[1-9]|[1-9]| [1-9])`,
		"f": `(?P<f>[0-9]{1,6})`,
		"H": `(?P<H>2[0-3]|[0-1]\d|\d)`,
		"I": `(?P<I>1[0-2]|0[1-9]|[1-9]| [1-9])`,
		"G": `(?P<G>\d\d\d\d)`,
		"j": `(?P<j>36[0-6]|3[0-5]\d|[1-2]\d\d|0[1-9]\d|00[1-9]|[1-9]\d|0[1-9]|[1-9])`,
		"m": `(?P<m>1[0-2]|0[1-9]|[1-9])`,
		"M": `(?P<M>[0-5]\d|\d)`,
		"S": `(?P<S>6[0-1]|[0-5]\d|\d)`,
		"U": `(?P<U>5[0-3]|[0-4]\d|\d)`,
		"w": `(?P<w>[0-6])`,
		"u": `(?P<u>[1-7])`,
		"V": `(?P<V>5[0-3]|0[1-9]|[1-4]\d|\d)`,
		"y": `(?P<y>\d\d)`,
		"Y": `(?P<Y>\d\d\d\d)`,
		"z": `(?P<z>[+-]\d\d:?[0-5]\d(:?[0-5]\d(\.\d{1,6})?)?|(?-i:Z))`,
		"A": `(?P<A>wednesday|thursday|saturday|tuesday|monday|friday|sunday)`,
		"a": `(?P<a>mon|tue|wed|thu|fri|sat|sun)`,
		"B": `(?P<B>september|february|november|december|january|october|august|march|april|june|july|may)`,
		"b": `(?P<b>jan|feb|mar|apr|may|jun|jul|aug|sep|oct|nov|dec)`,
		"p": `(?P<p>am|pm)`,
		"Z": `(?P<Z>utc|gmt)`,
		"%": `%`,
	}
	for _, d := range "dmyHIMS" {
		m["O"+string(d)] = `(?P<` + string(d) + `>\d\d|\d| \d)`
	}
	m["Ow"] = `(?P<w>\d)`
	m["W"] = strings.ReplaceAll(m["U"], "U", "W")
	m["X"], _ = strptimePattern(m, "%H:%M:%S")
	m["x"], _ = strptimePattern(m, "%m/%d/%y")
	m["c"], _ = strptimePattern(m, "%a %b %d %H:%M:%S %Y")
	return m
}()

// strptimePattern is TimeRE.pattern: escape regex syntax, turn whitespace runs into \s+,
// "'" into ['\u02bc], then substitute %(O?.) directives. A missing directive returns its
// key as a KeyError would.
func strptimePattern(m map[string]string, format string) (string, *string) {
	var b strings.Builder
	rs := []rune(format)
	for i := 0; i < len(rs); {
		r := rs[i]
		switch {
		case strings.ContainsRune(`\.^$*+?(){}[]|`, r):
			b.WriteByte('\\')
			b.WriteRune(r)
			i++
		case isSpaceRune(r):
			for i < len(rs) && isSpaceRune(rs[i]) {
				i++
			}
			b.WriteString(`\s+`)
		case r == '\'':
			b.WriteString("['\u02bc]")
			i++
		default:
			b.WriteRune(r)
			i++
		}
	}
	rs = []rune(b.String())
	var out strings.Builder
	for i := 0; i < len(rs); {
		if rs[i] == '%' && i+1 < len(rs) && rs[i+1] != '\n' {
			key := string(rs[i+1])
			n := 2
			if rs[i+1] == 'O' && i+2 < len(rs) && rs[i+2] != '\n' {
				key = "O" + string(rs[i+2])
				n = 3
			}
			v, ok := m[key]
			if !ok {
				return "", &key
			}
			out.WriteString(v)
			i += n
			continue
		}
		out.WriteRune(rs[i])
		i++
	}
	return out.String(), nil
}

// Timedelta is a normalized datetime.timedelta (0 <= Seconds < 86400,
// 0 <= Microseconds < 1000000).
type Timedelta struct{ Days, Seconds, Microseconds int }

// PyRepr returns repr() of the timedelta: "datetime.timedelta(0)",
// "datetime.timedelta(days=-1, seconds=68400)".
func (t Timedelta) PyRepr() string {
	var parts []string
	if t.Days != 0 {
		parts = append(parts, "days="+strconv.Itoa(t.Days))
	}
	if t.Seconds != 0 {
		parts = append(parts, "seconds="+strconv.Itoa(t.Seconds))
	}
	if t.Microseconds != 0 {
		parts = append(parts, "microseconds="+strconv.Itoa(t.Microseconds))
	}
	if len(parts) == 0 {
		return "datetime.timedelta(0)"
	}
	return "datetime.timedelta(" + strings.Join(parts, ", ") + ")"
}

func newTimedelta(seconds, microseconds int) Timedelta {
	total := int64(seconds)*1000000 + int64(microseconds)
	const day = 86400 * 1000000
	d := total / day
	r := total % day
	if r < 0 {
		r += day
		d--
	}
	return Timedelta{Days: int(d), Seconds: int(r / 1000000), Microseconds: int(r % 1000000)}
}

// StrptimeResult is what datetime.datetime.strptime builds, plus the struct_time extras of
// _strptime.
type StrptimeResult struct {
	DateTimeFields
	// Weekday (Monday = 0) and Julian (day of the year, 1-based) are tm_wday and tm_yday.
	Weekday, Julian int
	// TZ is tm_isdst: -1, or 0 when %Z matched "utc" or "gmt".
	TZ int
	// TZName is the text %Z matched, nil when the format has no %Z.
	TZName *string
	// Offset is the UTC offset from %z, nil for a naive datetime. When it is set the
	// datetime is aware: datetime.timezone(Offset[, TZName]).
	Offset *Timedelta
}

// TZInfoRepr returns repr() of the tzinfo of an aware result ("datetime.timezone.utc",
// "datetime.timezone(datetime.timedelta(seconds=3600))",
// "datetime.timezone(datetime.timedelta(0), 'UTC')"), or "None" for a naive one.
func (r StrptimeResult) TZInfoRepr() string {
	if r.Offset == nil {
		return "None"
	}
	if r.TZName == nil || *r.TZName == "" {
		if *r.Offset == (Timedelta{}) {
			return "datetime.timezone.utc"
		}
		return "datetime.timezone(" + r.Offset.PyRepr() + ")"
	}
	return "datetime.timezone(" + r.Offset.PyRepr() + ", " + StrRepr(*r.TZName) + ")"
}

// PyRepr returns repr() of the datetime.datetime strptime returns (with ", tzinfo=..." for an
// aware one).
func (r StrptimeResult) PyRepr() string {
	s := r.DateTimeFields.PyRepr()
	if r.Offset == nil {
		return s
	}
	return s[:len(s)-1] + ", tzinfo=" + r.TZInfoRepr() + ")"
}

// PyStr returns str() of the datetime.datetime strptime returns (with the +HH:MM[:SS[.ffffff]]
// suffix for an aware one).
func (r StrptimeResult) PyStr() string {
	s := r.DateTimeFields.PyStr()
	if r.Offset == nil {
		return s
	}
	o := *r.Offset
	sign := "+"
	if o.Days < 0 {
		sign = "-"
		o = newTimedelta(-(o.Days*86400 + o.Seconds), -o.Microseconds)
	}
	secs := o.Days*86400 + o.Seconds
	h, mi, sec := secs/3600, secs/60%60, secs%60
	s += sign + pad2(h) + ":" + pad2(mi)
	if sec != 0 || o.Microseconds != 0 {
		s += ":" + pad2(sec)
		if o.Microseconds != 0 {
			s += "." + strconv.Itoa(1000000 + o.Microseconds)[1:]
		}
	}
	return s
}

func pad2(n int) string {
	if n < 10 {
		return "0" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}

// Strptime is datetime.datetime.strptime(s, format) for the C locale (TZ=UTC), following
// CPython 3.12's Lib/_strptime.py: every directive of TimeRE (%a %A %b %B %c %d %f %G %H %I
// %j %m %M %p %S %u %U %V %w %W %x %X %y %Y %z %Z %%, and %O with d m y H I M S w), literal
// text matched case-insensitively, whitespace in the format matching one or more whitespace
// characters, "'" matching "'" or U+02BC. Errors are *Error values with CPython's messages:
// "time data 's' does not match format 'f'", "unconverted data remains: ...",
// "'%q' is a bad directive in format '...'", the date/time range errors of the datetime
// constructor ("day is out of range for month", "year 0 is out of range",
// "second must be in 0..59", ...), and re.error "redefinition of group name ..." when a
// directive's field is repeated.
func Strptime(s, format string) (StrptimeResult, error) {
	pat, bad := strptimePattern(timeRE, format)
	if bad != nil {
		k := *bad
		if k == `\` {
			k = "%"
		}
		return StrptimeResult{}, valueError("'" + k + "' is a bad directive in format '" + format + "'")
	}
	re, err := compileMiniRE(pat)
	if err != nil {
		return StrptimeResult{}, err
	}
	text := []rune(s)
	end, caps, ok := re.match(text)
	if !ok {
		return StrptimeResult{}, valueError("time data " + StrRepr(s) + " does not match format " + StrRepr(format))
	}
	if end != len(text) {
		return StrptimeResult{}, valueError("unconverted data remains: " + string(text[end:]))
	}
	found := map[string]string{}
	var order []string
	for _, g := range re.named {
		c := caps[g.index]
		if c[0] < 0 {
			continue // an unmatched group is None; none of TimeRE's named groups can be
		}
		found[g.name] = string(text[c[0]:c[1]])
		order = append(order, g.name)
	}
	return strptimeFields(found, order)
}

// pyInt is int(s) for a str the regex guarantees to be a decimal literal.
func pyInt(s string) (int, error) {
	z, err := ParseInt(s, 10)
	if err != nil {
		return 0, err
	}
	if !z.IsInt64() {
		return 0, overflowError("Python int too large to convert to C long")
	}
	return int(z.Int64()), nil
}

func listIndex(list []string, v string) (int, error) {
	for i, x := range list {
		if x == v {
			return i, nil
		}
	}
	return 0, valueError(StrRepr(v) + " is not in list")
}

// strptimeFields is the body of _strptime after the match, then _strptime_datetime.
func strptimeFields(found map[string]string, order []string) (StrptimeResult, error) {
	var res StrptimeResult
	var isoYear, year, isoWeek, weekOfYear, weekday, julian *int
	set := func(p **int, v int) { *p = &v }
	month, day := 1, 1
	hour, minute, second, fraction := 0, 0, 0, 0
	tz := -1
	var gmtoff *int
	gmtoffFraction := 0
	weekOfYearStart := 0
	var err error
	var v int
	for _, key := range order {
		val := found[key]
		switch key {
		case "y":
			if v, err = pyInt(val); err != nil {
				return res, err
			}
			if v <= 68 {
				v += 2000
			} else {
				v += 1900
			}
			set(&year, v)
		case "Y":
			if v, err = pyInt(val); err != nil {
				return res, err
			}
			set(&year, v)
		case "G":
			if v, err = pyInt(val); err != nil {
				return res, err
			}
			set(&isoYear, v)
		case "m":
			if month, err = pyInt(val); err != nil {
				return res, err
			}
		case "B":
			if month, err = listIndex(localeFMonth, pyuni.Lower(val)); err != nil {
				return res, err
			}
		case "b":
			if month, err = listIndex(localeAMonth, pyuni.Lower(val)); err != nil {
				return res, err
			}
		case "d":
			if day, err = pyInt(val); err != nil {
				return res, err
			}
		case "H":
			if hour, err = pyInt(val); err != nil {
				return res, err
			}
		case "I":
			if hour, err = pyInt(val); err != nil {
				return res, err
			}
			ampm := pyuni.Lower(found["p"])
			if ampm == "" || ampm == localeAmPm[0] {
				if hour == 12 {
					hour = 0
				}
			} else if ampm == localeAmPm[1] {
				if hour != 12 {
					hour += 12
				}
			}
		case "M":
			if minute, err = pyInt(val); err != nil {
				return res, err
			}
		case "S":
			if second, err = pyInt(val); err != nil {
				return res, err
			}
		case "f":
			if fraction, err = pyInt(val + strings.Repeat("0", 6-Len(val))); err != nil {
				return res, err
			}
		case "A":
			if v, err = listIndex(localeFWeekday, pyuni.Lower(val)); err != nil {
				return res, err
			}
			set(&weekday, v)
		case "a":
			if v, err = listIndex(localeAWeekday, pyuni.Lower(val)); err != nil {
				return res, err
			}
			set(&weekday, v)
		case "w":
			if v, err = pyInt(val); err != nil {
				return res, err
			}
			if v == 0 {
				v = 6
			} else {
				v--
			}
			set(&weekday, v)
		case "u":
			if v, err = pyInt(val); err != nil {
				return res, err
			}
			set(&weekday, v-1)
		case "j":
			if v, err = pyInt(val); err != nil {
				return res, err
			}
			set(&julian, v)
		case "U", "W":
			if v, err = pyInt(val); err != nil {
				return res, err
			}
			set(&weekOfYear, v)
			if key == "U" {
				weekOfYearStart = 6
			} else {
				weekOfYearStart = 0
			}
		case "V":
			if v, err = pyInt(val); err != nil {
				return res, err
			}
			set(&isoWeek, v)
		case "z":
			off, frac, err := parseZ(val)
			if err != nil {
				return res, err
			}
			gmtoff = &off
			gmtoffFraction = frac
		case "Z":
			z := pyuni.Lower(val)
			if z == "utc" || z == "gmt" {
				tz = 0
			}
		}
	}

	if isoYear != nil {
		if julian != nil {
			return res, valueError("Day of the year directive '%j' is not compatible with ISO year directive '%G'. Use '%Y' instead.")
		} else if isoWeek == nil || weekday == nil {
			return res, valueError("ISO year directive '%G' must be used with the ISO week directive '%V' and a weekday directive ('%A', '%a', '%w', or '%u').")
		}
	} else if isoWeek != nil {
		if year == nil || weekday == nil {
			return res, valueError("ISO week directive '%V' must be used with the ISO year directive '%G' and a weekday directive ('%A', '%a', '%w', or '%u').")
		}
		return res, valueError("ISO week directive '%V' is incompatible with the year directive '%Y'. Use the ISO year '%G' instead.")
	}

	leapYearFix := false
	if year == nil {
		if month == 2 && day == 29 {
			set(&year, 1904)
			leapYearFix = true
		} else {
			set(&year, 1900)
		}
	}
	y := *year

	if julian == nil && weekday != nil {
		if weekOfYear != nil {
			j, err := calcJulianFromUorW(y, *weekOfYear, *weekday, weekOfYearStart == 0)
			if err != nil {
				return res, err
			}
			set(&julian, j)
		} else if isoYear != nil && isoWeek != nil {
			d, err := FromISOCalendar(*isoYear, *isoWeek, *weekday+1)
			if err != nil {
				return res, err
			}
			y, month, day = d.Year, d.Month, d.Day
		}
		if julian != nil && *julian <= 0 {
			y--
			yday := 365
			if IsLeap(y) {
				yday = 366
			}
			set(&julian, *julian+yday)
		}
	}

	if julian == nil {
		if err := CheckDate(y, month, day); err != nil {
			return res, err
		}
		set(&julian, ymdToOrd(y, month, day)-ymdToOrd(y, 1, 1)+1)
	} else {
		if err := CheckDate(y, 1, 1); err != nil {
			return res, err
		}
		d, err := FromOrdinal((*julian - 1) + ymdToOrd(y, 1, 1))
		if err != nil {
			return res, err
		}
		y, month, day = d.Year, d.Month, d.Day
	}
	if weekday == nil {
		if err := CheckDate(y, month, day); err != nil {
			return res, err
		}
		set(&weekday, (ymdToOrd(y, month, day)+6)%7)
	}
	if name, ok := found["Z"]; ok {
		res.TZName = &name
	}
	if leapYearFix {
		y = 1900
	}

	// _strptime_datetime: cls(*args)
	dt, err := NewDateTime(y, month, day, hour, minute, second, fraction)
	if err != nil {
		return res, err
	}
	res.DateTimeFields = dt
	res.Weekday = *weekday
	res.Julian = *julian
	res.TZ = tz
	if gmtoff != nil {
		off := newTimedelta(*gmtoff, gmtoffFraction)
		if (off.Days == -1 && off.Seconds == 0 && off.Microseconds < 1) || off.Days < -1 || off.Days >= 1 {
			return StrptimeResult{}, valueError("offset must be a timedelta strictly between -timedelta(hours=24) and timedelta(hours=24), not " + off.PyRepr() + ".")
		}
		res.Offset = &off
	}
	return res, nil
}

// parseZ is the %z branch of _strptime: it returns gmtoff (seconds) and gmtoff_fraction.
func parseZ(val string) (int, int, error) {
	if val == "Z" {
		return 0, 0, nil
	}
	z := []rune(val)
	if z[3] == ':' {
		z = append(z[:3:3], z[4:]...)
		if len(z) > 5 {
			if z[5] != ':' {
				return 0, 0, valueError("Inconsistent use of : in " + val)
			}
			z = append(z[:5:5], z[6:]...)
		}
	}
	hours, err := pyInt(string(z[1:3]))
	if err != nil {
		return 0, 0, err
	}
	minutes, err := pyInt(string(z[3:5]))
	if err != nil {
		return 0, 0, err
	}
	seconds := 0
	if len(z) > 5 {
		hi := 7
		if len(z) < 7 {
			hi = len(z)
		}
		if seconds, err = pyInt(string(z[5:hi])); err != nil {
			return 0, 0, err
		}
	}
	gmtoff := hours*3600 + minutes*60 + seconds
	rem := ""
	if len(z) > 8 {
		rem = string(z[8:])
	}
	frac, err := pyInt(rem + strings.Repeat("0", 6-Len(rem)))
	if err != nil {
		return 0, 0, err
	}
	if z[0] == '-' {
		gmtoff, frac = -gmtoff, -frac
	}
	return gmtoff, frac, nil
}

// calcJulianFromUorW is _calc_julian_from_U_or_W.
func calcJulianFromUorW(year, weekOfYear, dayOfWeek int, weekStartsMon bool) (int, error) {
	if err := CheckDate(year, 1, 1); err != nil {
		return 0, err
	}
	firstWeekday := (ymdToOrd(year, 1, 1) + 6) % 7
	if !weekStartsMon {
		firstWeekday = (firstWeekday + 1) % 7
		dayOfWeek = (dayOfWeek + 1) % 7
	}
	week0Length := (7 - firstWeekday) % 7
	if weekOfYear == 0 {
		return 1 + dayOfWeek - firstWeekday, nil
	}
	daysToWeek := week0Length + 7*(weekOfYear-1)
	return 1 + daysToWeek + dayOfWeek, nil
}

// ---- the small regex engine for TimeRE patterns ----

type rxNode interface{}

type rxLit struct {
	r  rune
	ci bool
}

type rxRange struct{ lo, hi rune }

type rxClass struct{ ranges []rxRange }

type rxCat struct{ space bool } // \s when space, else \d

type rxSeq []rxNode

type rxAlt []rxNode

type rxGroup struct {
	index int // 0 for a non-capturing group
	body  rxNode
}

type rxRepeat struct {
	min, max int
	body     rxNode
}

type rxNamed struct {
	name  string
	index int
}

type miniRE struct {
	root   rxNode
	groups int
	named  []rxNamed
}

type rxParser struct {
	p     []rune
	i     int
	ci    bool
	re    *miniRE
	names map[string]int
}

// compileMiniRE parses a TimeRE pattern (compiled with re.IGNORECASE).
func compileMiniRE(pattern string) (*miniRE, error) {
	ps := &rxParser{p: []rune(pattern), ci: true, re: &miniRE{}, names: map[string]int{}}
	n, err := ps.alt()
	if err != nil {
		return nil, err
	}
	if ps.i != len(ps.p) {
		return nil, &Error{Class: "re.error", Msg: "unbalanced parenthesis at position " + strconv.Itoa(ps.i)}
	}
	ps.re.root = n
	return ps.re, nil
}

func (ps *rxParser) alt() (rxNode, error) {
	var alts rxAlt
	for {
		s, err := ps.seq()
		if err != nil {
			return nil, err
		}
		alts = append(alts, s)
		if ps.i < len(ps.p) && ps.p[ps.i] == '|' {
			ps.i++
			continue
		}
		break
	}
	if len(alts) == 1 {
		return alts[0], nil
	}
	return alts, nil
}

func (ps *rxParser) seq() (rxNode, error) {
	var items rxSeq
	for ps.i < len(ps.p) && ps.p[ps.i] != '|' && ps.p[ps.i] != ')' {
		a, err := ps.atom()
		if err != nil {
			return nil, err
		}
		if ps.i < len(ps.p) {
			switch ps.p[ps.i] {
			case '?':
				ps.i++
				a = rxRepeat{0, 1, a}
			case '+':
				ps.i++
				a = rxRepeat{1, -1, a}
			case '*':
				ps.i++
				a = rxRepeat{0, -1, a}
			case '{':
				j := ps.i + 1
				for j < len(ps.p) && ps.p[j] != '}' {
					j++
				}
				lo, hi, ok := strings.Cut(string(ps.p[ps.i+1:j]), ",")
				mn, e1 := strconv.Atoi(lo)
				mx, e2 := strconv.Atoi(hi)
				if !ok || e1 != nil || e2 != nil || j == len(ps.p) {
					return nil, &Error{Class: "re.error", Msg: "unsupported repeat at position " + strconv.Itoa(ps.i)}
				}
				ps.i = j + 1
				a = rxRepeat{mn, mx, a}
			}
		}
		items = append(items, a)
	}
	return items, nil
}

func (ps *rxParser) atom() (rxNode, error) {
	c := ps.p[ps.i]
	switch c {
	case '(':
		ps.i++
		rest := string(ps.p[ps.i:])
		saveCI := ps.ci
		g := rxGroup{}
		switch {
		case strings.HasPrefix(rest, "?P<"):
			ps.i += 3
			start := ps.i
			for ps.i < len(ps.p) && ps.p[ps.i] != '>' {
				ps.i++
			}
			name := string(ps.p[start:ps.i])
			ps.i++
			ps.re.groups++
			g.index = ps.re.groups
			if old, dup := ps.names[name]; dup {
				return nil, &Error{Class: "re.error", Msg: "redefinition of group name " + StrRepr(name) +
					" as group " + strconv.Itoa(g.index) + "; was group " + strconv.Itoa(old) +
					" at position " + strconv.Itoa(start)}
			}
			ps.names[name] = g.index
			ps.re.named = append(ps.re.named, rxNamed{name, g.index})
		case strings.HasPrefix(rest, "?-i:"):
			ps.i += 4
			ps.ci = false
		case strings.HasPrefix(rest, "?:"):
			ps.i += 2
		default:
			ps.re.groups++
			g.index = ps.re.groups
		}
		body, err := ps.alt()
		if err != nil {
			return nil, err
		}
		ps.ci = saveCI
		if ps.i >= len(ps.p) || ps.p[ps.i] != ')' {
			return nil, &Error{Class: "re.error", Msg: "missing ), unterminated subpattern"}
		}
		ps.i++
		g.body = body
		return g, nil
	case '[':
		ps.i++
		var cl rxClass
		for ps.i < len(ps.p) && ps.p[ps.i] != ']' {
			lo := ps.classChar()
			if ps.i+1 < len(ps.p) && ps.p[ps.i] == '-' && ps.p[ps.i+1] != ']' {
				ps.i++
				hi := ps.classChar()
				cl.ranges = append(cl.ranges, rxRange{lo, hi})
			} else {
				cl.ranges = append(cl.ranges, rxRange{lo, lo})
			}
		}
		ps.i++
		return cl, nil
	case '\\':
		ps.i++
		if ps.i >= len(ps.p) {
			return nil, &Error{Class: "re.error", Msg: "bad escape (end of pattern)"}
		}
		e := ps.p[ps.i]
		ps.i++
		switch e {
		case 'd':
			return rxCat{space: false}, nil
		case 's':
			return rxCat{space: true}, nil
		}
		return rxLit{e, ps.ci}, nil
	}
	ps.i++
	return rxLit{c, ps.ci}, nil
}

func (ps *rxParser) classChar() rune {
	c := ps.p[ps.i]
	ps.i++
	if c == '\\' && ps.i < len(ps.p) {
		c = ps.p[ps.i]
		ps.i++
	}
	return c
}

// litMatch is sre's LITERAL / LITERAL_UNI_IGNORE / IN_UNI_IGNORE test for a literal of a
// str pattern.
func litMatch(l rxLit, c rune) bool {
	if !l.ci || !pyuni.IsSreCased(l.r) {
		return c == l.r
	}
	lo := pyuni.SimpleLower(l.r)
	lc := pyuni.SimpleLower(c)
	if lc == lo {
		return true
	}
	for _, x := range pyuni.ExtraCases(lo) {
		if lc == x {
			return true
		}
	}
	return false
}

type rxMatcher struct {
	text []rune
	caps [][2]int
}

func (re *miniRE) match(text []rune) (int, [][2]int, bool) {
	m := &rxMatcher{text: text, caps: make([][2]int, re.groups+1)}
	for i := range m.caps {
		m.caps[i] = [2]int{-1, -1}
	}
	end := -1
	ok := m.run(re.root, 0, func(p int) bool { end = p; return true })
	return end, m.caps, ok
}

// run matches n at pos and calls k with each possible end, in sre's priority order, until k
// accepts.
func (m *rxMatcher) run(n rxNode, pos int, k func(int) bool) bool {
	switch x := n.(type) {
	case rxLit:
		return pos < len(m.text) && litMatch(x, m.text[pos]) && k(pos+1)
	case rxCat:
		if pos >= len(m.text) {
			return false
		}
		c := m.text[pos]
		if x.space {
			if !isSpaceRune(c) {
				return false
			}
		} else if !pyuni.IsDecimal(c) {
			return false
		}
		return k(pos + 1)
	case rxClass:
		if pos >= len(m.text) {
			return false
		}
		c := m.text[pos]
		for _, r := range x.ranges {
			// TimeRE's classes hold no cased characters, so IGNORECASE does not widen them.
			if c >= r.lo && c <= r.hi {
				return k(pos + 1)
			}
		}
		return false
	case rxSeq:
		return m.seq(x, 0, pos, k)
	case rxAlt:
		for _, a := range x {
			if m.run(a, pos, k) {
				return true
			}
		}
		return false
	case rxGroup:
		if x.index == 0 {
			return m.run(x.body, pos, k)
		}
		old := m.caps[x.index]
		if m.run(x.body, pos, func(e int) bool {
			prev := m.caps[x.index]
			m.caps[x.index] = [2]int{pos, e}
			if k(e) {
				return true
			}
			m.caps[x.index] = prev
			return false
		}) {
			return true
		}
		m.caps[x.index] = old
		return false
	case rxRepeat:
		var rep func(count, p int) bool
		rep = func(count, p int) bool {
			if x.max < 0 || count < x.max {
				if m.run(x.body, p, func(e int) bool {
					if e == p {
						return false
					}
					return rep(count+1, e)
				}) {
					return true
				}
			}
			return count >= x.min && k(p)
		}
		return rep(0, pos)
	}
	return false
}

func (m *rxMatcher) seq(items rxSeq, i, pos int, k func(int) bool) bool {
	if i == len(items) {
		return k(pos)
	}
	return m.run(items[i], pos, func(e int) bool { return m.seq(items, i+1, e, k) })
}

// StrptimeDate is datetime.datetime.strptime(s, format).date(): the date fields of the
// result (the time fields are zeroed).
func StrptimeDate(s, format string) (DateTimeFields, error) {
	r, err := Strptime(s, format)
	if err != nil {
		return DateTimeFields{}, err
	}
	return DateTimeFields{Year: r.Year, Month: r.Month, Day: r.Day}, nil
}

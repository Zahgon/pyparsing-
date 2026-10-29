// Port of CPython 3.12's Modules/_datetimemodule.c: proleptic Gregorian ordinals, the
// date/datetime constructor checks, date.fromordinal, date.fromisocalendar and
// datetime + timedelta (PSF License Version 2; Copyright (c) 2001-2024 Python Software
// Foundation).

package pyval

import (
	"math"
	"math/big"
	"strconv"
)

// DateTimeFields is a naive datetime.datetime (or, ignoring the time fields, a
// datetime.date) as plain ints.
type DateTimeFields struct {
	Year, Month, Day, Hour, Minute, Second, Microsecond int
}

// PyRepr returns repr() of the value as a datetime.datetime.
func (d DateTimeFields) PyRepr() string {
	return DateTimeRepr(d.Year, d.Month, d.Day, d.Hour, d.Minute, d.Second, d.Microsecond)
}

// PyStr returns str() of the value as a datetime.datetime.
func (d DateTimeFields) PyStr() string {
	return DateTimeStr(d.Year, d.Month, d.Day, d.Hour, d.Minute, d.Second, d.Microsecond)
}

// MinYear and MaxYear are datetime.MINYEAR and datetime.MAXYEAR.
const (
	MinYear = 1
	MaxYear = 9999
)

var daysInMonthTable = [13]int{0, 31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
var daysBeforeMonthTable = [13]int{0, 0, 31, 59, 90, 120, 151, 181, 212, 243, 273, 304, 334}

// IsLeap is calendar.isleap(year).
func IsLeap(year int) bool { return year%4 == 0 && (year%100 != 0 || year%400 == 0) }

// DaysInMonth is the number of days in the month (1..12) of year; 0 for an invalid month.
func DaysInMonth(year, month int) int {
	if month < 1 || month > 12 {
		return 0
	}
	if month == 2 && IsLeap(year) {
		return 29
	}
	return daysInMonthTable[month]
}

func daysBeforeYear(year int) int {
	y := year - 1
	return y*365 + y/4 - y/100 + y/400
}

func daysBeforeMonth(year, month int) int {
	d := daysBeforeMonthTable[month]
	if month > 2 && IsLeap(year) {
		d++
	}
	return d
}

// ymdToOrd is ymd_to_ord for a valid date: 0001-01-01 is day 1.
func ymdToOrd(y, m, d int) int { return daysBeforeYear(y) + daysBeforeMonth(y, m) + d }

// ordToYmd is ord_to_ymd.
func ordToYmd(ordinal int) (int, int, int) {
	const di400y = 146097
	const di100y = 36524
	const di4y = 1461
	n := ordinal - 1
	n400 := n / di400y
	n = n % di400y
	year := n400*400 + 1
	n100 := n / di100y
	n = n % di100y
	n4 := n / di4y
	n = n % di4y
	n1 := n / 365
	n = n % 365
	year += n100*100 + n4*4 + n1
	if n1 == 4 || n100 == 4 {
		return year - 1, 12, 31
	}
	leap := n1 == 3 && (n4 != 24 || n100 == 3)
	month := (n + 50) >> 5
	preceding := daysBeforeMonthTable[month]
	if month > 2 && leap {
		preceding++
	}
	if preceding > n {
		month--
		preceding -= daysInMonthTable[month]
		if month == 2 && leap {
			preceding--
		}
	}
	n -= preceding
	return year, month, n + 1
}

// CheckDate is the datetime.date constructor's validation: ValueError "year Y is out of
// range", "month must be in 1..12" or "day is out of range for month".
func CheckDate(year, month, day int) error {
	if year < MinYear || year > MaxYear {
		return valueError("year " + strconv.Itoa(year) + " is out of range")
	}
	if month < 1 || month > 12 {
		return valueError("month must be in 1..12")
	}
	if day < 1 || day > DaysInMonth(year, month) {
		return valueError("day is out of range for month")
	}
	return nil
}

// CheckTime is the time-field validation of the datetime.datetime constructor: ValueError
// "hour must be in 0..23", "minute must be in 0..59", "second must be in 0..59" or
// "microsecond must be in 0..999999".
func CheckTime(hour, minute, second, microsecond int) error {
	switch {
	case hour < 0 || hour > 23:
		return valueError("hour must be in 0..23")
	case minute < 0 || minute > 59:
		return valueError("minute must be in 0..59")
	case second < 0 || second > 59:
		return valueError("second must be in 0..59")
	case microsecond < 0 || microsecond > 999999:
		return valueError("microsecond must be in 0..999999")
	}
	return nil
}

// NewDateTime is datetime.datetime(year, month, day, hour, minute, second, microsecond) with
// int arguments: CheckDate, then CheckTime.
func NewDateTime(year, month, day, hour, minute, second, microsecond int) (DateTimeFields, error) {
	if err := CheckDate(year, month, day); err != nil {
		return DateTimeFields{}, err
	}
	if err := CheckTime(hour, minute, second, microsecond); err != nil {
		return DateTimeFields{}, err
	}
	return DateTimeFields{year, month, day, hour, minute, second, microsecond}, nil
}

// cInt is the "i" argument converter: OverflowError "signed integer is greater than maximum"
// / "signed integer is less than minimum" outside the C int range, and "Python int too large
// to convert to C long" outside the C long (64-bit) range.
func cInt(z *big.Int) (int, error) {
	if z == nil {
		return 0, &Error{Class: "TypeError", Msg: "'NoneType' object cannot be interpreted as an integer"}
	}
	if !z.IsInt64() {
		return 0, overflowError("Python int too large to convert to C long")
	}
	v := z.Int64()
	if v > math.MaxInt32 {
		return 0, overflowError("signed integer is greater than maximum")
	}
	if v < math.MinInt32 {
		return 0, overflowError("signed integer is less than minimum")
	}
	return int(v), nil
}

// NewDateTimeBig is datetime.datetime(year, month, day, hour, minute, second, microsecond)
// with Python int arguments, including the C int conversion errors of the arguments (checked
// in argument order before the range checks).
func NewDateTimeBig(year, month, day, hour, minute, second, microsecond *big.Int) (DateTimeFields, error) {
	var v [7]int
	for i, z := range []*big.Int{year, month, day, hour, minute, second, microsecond} {
		n, err := cInt(z)
		if err != nil {
			return DateTimeFields{}, err
		}
		v[i] = n
	}
	return NewDateTime(v[0], v[1], v[2], v[3], v[4], v[5], v[6])
}

// Toordinal is date(year, month, day).toordinal(); it returns 0 for an invalid date.
func Toordinal(year, month, day int) int {
	if CheckDate(year, month, day) != nil {
		return 0
	}
	return ymdToOrd(year, month, day)
}

// Weekday is date(year, month, day).weekday(): Monday is 0, Sunday is 6; -1 for an invalid
// date.
func Weekday(year, month, day int) int {
	if CheckDate(year, month, day) != nil {
		return -1
	}
	return (ymdToOrd(year, month, day) + 6) % 7
}

// FromOrdinal is date.fromordinal(n): ValueError "ordinal must be >= 1" or
// "year 10000 is out of range".
func FromOrdinal(n int) (DateTimeFields, error) {
	if n < 1 {
		return DateTimeFields{}, valueError("ordinal must be >= 1")
	}
	y, m, d := ordToYmd(n)
	if err := CheckDate(y, m, d); err != nil {
		return DateTimeFields{}, err
	}
	return DateTimeFields{Year: y, Month: m, Day: d}, nil
}

// FromISOCalendar is date.fromisocalendar(year, week, day).
func FromISOCalendar(year, week, day int) (DateTimeFields, error) {
	if year < MinYear || year > MaxYear {
		return DateTimeFields{}, valueError("Year is out of range: " + strconv.Itoa(year))
	}
	if week <= 0 || week >= 53 {
		ok := false
		if week == 53 {
			firstWeekday := (ymdToOrd(year, 1, 1) + 6) % 7
			if firstWeekday == 3 || (firstWeekday == 2 && IsLeap(year)) {
				ok = true
			}
		}
		if !ok {
			return DateTimeFields{}, valueError("Invalid week: " + strconv.Itoa(week))
		}
	}
	if day <= 0 || day >= 8 {
		return DateTimeFields{}, valueError("Invalid day: " + strconv.Itoa(day) + " (range is [1, 7])")
	}
	// iso_week1_monday
	first := ymdToOrd(year, 1, 1)
	firstWeekday := (first + 6) % 7
	week1Monday := first - firstWeekday
	if firstWeekday > 3 {
		week1Monday += 7
	}
	ord := week1Monday + (week-1)*7 + (day - 1)
	y, m, d := ordToYmd(ord)
	if err := CheckDate(y, m, d); err != nil {
		return DateTimeFields{}, err
	}
	return DateTimeFields{Year: y, Month: m, Day: d}, nil
}

var usPerDay = big.NewInt(86400 * 1000000)

// AddMicroseconds is dt + datetime.timedelta(microseconds=us). The timedelta itself raises
// OverflowError "Python int too large to convert to C int" or "days=N; must have magnitude
// <= 999999999" for huge us; a result outside years 1..9999 gives OverflowError
// "date value out of range".
func AddMicroseconds(dt DateTimeFields, us *big.Int) (DateTimeFields, error) {
	if us == nil {
		us = new(big.Int)
	}
	if _, err := NewDateTime(dt.Year, dt.Month, dt.Day, dt.Hour, dt.Minute, dt.Second, dt.Microsecond); err != nil {
		return DateTimeFields{}, err
	}
	days, rem := new(big.Int).DivMod(us, usPerDay, new(big.Int)) // floor division
	if !days.IsInt64() || days.Int64() > math.MaxInt32 || days.Int64() < math.MinInt32 {
		return DateTimeFields{}, overflowError("Python int too large to convert to C int")
	}
	dd := int(days.Int64())
	if dd > 999999999 || dd < -999999999 {
		return DateTimeFields{}, overflowError("days=" + strconv.Itoa(dd) + "; must have magnitude <= 999999999")
	}
	r := rem.Int64() // 0 <= r < 86400e6
	tod := int64(((dt.Hour*60+dt.Minute)*60+dt.Second))*1000000 + int64(dt.Microsecond) + r
	ord := ymdToOrd(dt.Year, dt.Month, dt.Day) + dd + int(tod/(86400*1000000))
	tod %= 86400 * 1000000
	if ord < 1 || ord > ymdToOrd(MaxYear, 12, 31) {
		return DateTimeFields{}, overflowError("date value out of range")
	}
	y, m, d := ordToYmd(ord)
	sec := tod / 1000000
	return DateTimeFields{
		Year: y, Month: m, Day: d,
		Hour: int(sec / 3600), Minute: int(sec / 60 % 60), Second: int(sec % 60),
		Microsecond: int(tod % 1000000),
	}, nil
}

// Port of CPython 3.12 value semantics: Objects/unicodeobject.c, Objects/floatobject.c,
// Objects/longobject.c, Python/pystrtod.c, Python/dtoa.c, Modules/_datetimemodule.c and
// Lib/_strptime.py (PSF License Version 2; Copyright (c) 2001-2024 Python Software Foundation).

// Package pyval implements the Python 3.12 value semantics the pyparsing port needs:
// repr()/str() of str, int, float, bool, None, list, tuple, dict, datetime.date and
// datetime.datetime; int() and float() of a str; round(), % and // on floats; the str
// methods pyparsing calls (with code point positions and Python's Unicode 15.0 whitespace
// and case rules from internal/pyuni); and datetime.strptime with the C locale.
//
// Python values map to Go values as follows: str is string (UTF-8; every length and index
// counts code points), int is *big.Int, float is float64, bool is bool, None is nil, list is
// []any. Tuples, dicts, dates and other values defined by the root package implement Reprer
// (and optionally Strer) and are rendered through the helpers TupleRepr, DictRepr, DateRepr,
// DateTimeRepr, DateStr and DateTimeStr.
//
// Python exceptions are returned as *Error values carrying the exception class name and
// str(exc). No exported function panics.
//
// Invalid UTF-8: a Python str cannot hold invalid UTF-8 and pyparsing never produces it.
// StrRepr renders each invalid byte b as the lone surrogate escape '\udcXX' (XX = b), the
// value Python's surrogateescape decoding would give it; every other function in this package
// treats each invalid byte as one code point U+FFFD (Go's []rune conversion).
package pyval

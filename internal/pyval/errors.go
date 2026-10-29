// Port of CPython 3.12 exception reporting conventions (PSF License Version 2;
// Copyright (c) 2001-2024 Python Software Foundation).

package pyval

// Error is a Python exception raised by one of this package's functions.
type Error struct {
	// Class is type(exc).__name__ ("ValueError", "OverflowError", "ZeroDivisionError",
	// "TypeError"), or "re.error" for an error raised by the re module.
	Class string
	// Msg is str(exc).
	Msg string
}

// Error returns "Class: Msg".
func (e *Error) Error() string { return e.Class + ": " + e.Msg }

func valueError(msg string) *Error    { return &Error{Class: "ValueError", Msg: msg} }
func overflowError(msg string) *Error { return &Error{Class: "OverflowError", Msg: msg} }

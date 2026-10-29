// Port of the CPython 3.12 Unicode predicates used by str methods and number parsing
// (PSF License Version 2; Copyright (c) 2001-2024 Python Software Foundation). The data
// comes from internal/pyuni (Unicode 15.0.0), never from Go's unicode package.

package pyval

import "github.com/pyparsing/pyparsing-go/internal/pyuni"

// isSpaceRune is Py_UNICODE_ISSPACE (str.isspace for one character).
func isSpaceRune(r rune) bool { return pyuni.IsSpace(r) }

// decimalValue is Py_UNICODE_TODECIMAL.
func decimalValue(r rune) (int, bool) { return pyuni.IsDecimalValue(r) }

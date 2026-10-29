// Port of pyparsing/exceptions.py (pyparsing 3.3.3+). MIT License, Copyright (c) 2003-2025  Paul McGuire; see LICENSE.

package pyparsing

import (
	"strings"

	"github.com/pyparsing/pyparsing-go/internal/pyre"
	"github.com/pyparsing/pyparsing-go/internal/pyval"
)

// ExceptionKind is the class of a ParseException.
type ExceptionKind int

// The parse exception classes.
const (
	KindParseException ExceptionKind = iota
	KindParseFatalException
	KindParseSyntaxException
)

// ParseException is pyparsing's ParseException, ParseFatalException or ParseSyntaxException.
type ParseException struct {
	Kind          ExceptionKind
	Pstr          string
	Loc           int
	Msg           string
	ParserElement *ParserElement

	// args is the exception's args tuple (pstr, loc, msg) as passed to the constructor.
	argsPstr string
	argsLoc  int
	argsMsg  string

	// cached properties: line, lineno, col, column, found
	cLine, cLineno, cCol, cColumn, cFound bool
	vLine, vFound                         string
	vLineno, vCol, vColumn                int
}

// newPE is ParseBaseException.__init__(pstr, loc, msg, elem), msg nil standing for None.
func newPE(kind ExceptionKind, pstr string, loc int, msg *string, elem *ParserElement) *ParseException {
	var m string
	if msg == nil {
		m, pstr = pstr, ""
	} else {
		m = *msg
	}
	return &ParseException{Kind: kind, Pstr: pstr, Loc: loc, Msg: m, ParserElement: elem,
		argsPstr: pstr, argsLoc: loc, argsMsg: m}
}

func newParseExc(pstr string, loc int, msg string, elem *ParserElement) *ParseException {
	return newPE(KindParseException, pstr, loc, &msg, elem)
}

// NewParseException returns ParseException(pstr, loc, msg, elem).
func NewParseException(pstr string, loc int, msg string, elem *ParserElement) *ParseException {
	return newParseExc(pstr, loc, msg, elem)
}

// NewParseFatalException returns ParseFatalException(pstr, loc, msg, elem).
func NewParseFatalException(pstr string, loc int, msg string, elem *ParserElement) *ParseException {
	return newPE(KindParseFatalException, pstr, loc, &msg, elem)
}

// fromArgs is pe.__class__(*pe.args).
func (e *ParseException) fromArgs() *ParseException {
	m := e.argsMsg
	return newPE(e.Kind, e.argsPstr, e.argsLoc, &m, nil)
}

// fromException is cls._from_exception(pe).
func fromException(kind ExceptionKind, pe *ParseException) *ParseException {
	m := pe.Msg
	return newPE(kind, pe.Pstr, pe.Loc, &m, pe.ParserElement)
}

func (e *ParseException) isFatal() bool { return e.Kind != KindParseException }

// Error returns str(exc), the formatted message.
func (e *ParseException) Error() string { return e.str() }

// ClassName returns the Python class name.
func (e *ParseException) ClassName() string {
	switch e.Kind {
	case KindParseFatalException:
		return "ParseFatalException"
	case KindParseSyntaxException:
		return "ParseSyntaxException"
	}
	return "ParseException"
}

func (e *ParseException) str() string { return e.FormattedMessage() }

// Repr returns repr(exc).
func (e *ParseException) Repr() string { return e.str() }

// Line returns exc.line.
func (e *ParseException) Line() string {
	if !e.cLine {
		e.vLine = lineRunes(e.Loc, []rune(e.Pstr))
		e.cLine = true
	}
	return e.vLine
}

// Lineno returns exc.lineno.
func (e *ParseException) Lineno() int {
	if !e.cLineno {
		e.vLineno = linenoRunes(e.Loc, []rune(e.Pstr))
		e.cLineno = true
	}
	return e.vLineno
}

// Col returns exc.col.
func (e *ParseException) Col() int {
	if !e.cCol {
		e.vCol = colRunes(e.Loc, []rune(e.Pstr))
		e.cCol = true
	}
	return e.vCol
}

// Column returns exc.column.
func (e *ParseException) Column() int {
	if !e.cColumn {
		e.vColumn = colRunes(e.Loc, []rune(e.Pstr))
		e.cColumn = true
	}
	return e.vColumn
}

// Found returns exc.found.
func (e *ParseException) Found() string {
	if !e.cFound {
		e.vFound = e.computeFound()
		e.cFound = true
	}
	return e.vFound
}

func (e *ParseException) computeFound() string {
	if e.Pstr == "" {
		return ""
	}
	rs := []rune(e.Pstr)
	if e.Loc >= len(rs) {
		return "end of text"
	}
	var foundText string
	pos := e.Loc
	if pos < 0 {
		pos = 0
	}
	if m := exceptionWordExtractor().Match(rs, pos); m != nil {
		foundText, _ = m.Group(0)
	} else {
		foundText = string(pySlice(rs, e.Loc, e.Loc+1))
	}
	return strings.ReplaceAll(pyval.StrRepr(foundText), `\\`, `\`)
}

// FormattedMessage returns exc.formatted_message().
func (e *ParseException) FormattedMessage() string {
	found := e.Found()
	foundPhrase := ""
	if found != "" {
		foundPhrase = ", found " + found
	}
	return e.Msg + foundPhrase + "  (at char " + itoa(e.Loc) + "), (line:" + itoa(e.Lineno()) + ", col:" + itoa(e.Column()) + ")"
}

// MarkInputLine returns exc.mark_input_line(); with an argument, mark_input_line(marker).
func (e *ParseException) MarkInputLine(marker ...string) string {
	markerString := ">!<"
	if len(marker) > 0 {
		markerString = marker[0]
	}
	lineStr := e.Line()
	lineColumn := e.Column() - 1
	if markerString != "" {
		rs := []rune(lineStr)
		lineStr = string(pySlice(rs, 0, lineColumn)) + markerString + string(pySlice(rs, lineColumn, len(rs)))
	}
	return pyStrip(lineStr)
}

// Explain returns exc.explain(depth). The traceback of an exception raised by the parse entry
// points holds only pyparsing frames that explain() skips (the entry points re-raise with
// with_traceback(None)), so the result is the failing line, a caret and the exception string.
func (e *ParseException) Explain(depth int) string {
	ret := []string{e.Line()}
	col := e.Column()
	if col < 1 {
		col = 1
	}
	ret = append(ret, strings.Repeat(" ", col-1)+"^")
	ret = append(ret, e.ClassName()+": "+e.str())
	return strings.Join(ret, "\n")
}

var exceptionWordRe *pyre.Pattern

func exceptionWordExtractor() *pyre.Pattern {
	if exceptionWordRe == nil {
		// _ExceptionWordUnicodeSet(Latin1, LatinA, LatinB, Greek, Cyrillic).alphanums
		chars := charsForRanges(latin1Ranges, latinARanges, latinBRanges, greekRanges, cyrillicRanges)
		set := runeSet{}
		for _, c := range chars {
			if isPyAlpha(c) {
				set[c] = true
			}
		}
		for _, c := range chars {
			if isPyDigit(c) {
				set[c] = true
			}
		}
		p, err := pyre.Compile("(["+collapseStringToRanges(set, true)+"]{1,16})|.", 0)
		if err != nil {
			panic("pyparsing: cannot compile the exception word extractor: " + err.Error())
		}
		exceptionWordRe = p
	}
	return exceptionWordRe
}

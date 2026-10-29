// Port of pyparsing/core.py, class ParserElement (pyparsing 3.3.3+). MIT License, Copyright (c) 2003-2025  Paul McGuire; see LICENSE.

package pyparsing

import (
	"github.com/pyparsing/pyparsing-go/internal/pyre"
)

// pclass is a Python class of the element hierarchy (for isinstance checks and names).
type pclass struct {
	name   string
	parent *pclass
}

var (
	clsParserElement    = &pclass{"ParserElement", nil}
	clsPendingSkip      = &pclass{"_PendingSkip", clsParserElement}
	clsToken            = &pclass{"Token", clsParserElement}
	clsNoMatch          = &pclass{"NoMatch", clsToken}
	clsLiteral          = &pclass{"Literal", clsToken}
	clsEmpty            = &pclass{"Empty", clsLiteral}
	clsErrorStop        = &pclass{"_ErrorStop", clsEmpty}
	clsSingleCharLit    = &pclass{"_SingleCharLiteral", clsLiteral}
	clsCaselessLiteral  = &pclass{"CaselessLiteral", clsLiteral}
	clsKeyword          = &pclass{"Keyword", clsToken}
	clsCaselessKeyword  = &pclass{"CaselessKeyword", clsKeyword}
	clsCloseMatch       = &pclass{"CloseMatch", clsToken}
	clsWord             = &pclass{"Word", clsToken}
	clsChar             = &pclass{"Char", clsWord}
	clsRegex            = &pclass{"Regex", clsToken}
	clsQuotedString     = &pclass{"QuotedString", clsToken}
	clsCharsNotIn       = &pclass{"CharsNotIn", clsToken}
	clsWhite            = &pclass{"White", clsToken}
	clsPositionToken    = &pclass{"PositionToken", clsToken}
	clsGoToColumn       = &pclass{"GoToColumn", clsPositionToken}
	clsLineStart        = &pclass{"LineStart", clsPositionToken}
	clsLineEnd          = &pclass{"LineEnd", clsPositionToken}
	clsStringStart      = &pclass{"StringStart", clsPositionToken}
	clsStringEnd        = &pclass{"StringEnd", clsPositionToken}
	clsWordStart        = &pclass{"WordStart", clsPositionToken}
	clsWordEnd          = &pclass{"WordEnd", clsPositionToken}
	clsTag              = &pclass{"Tag", clsToken}
	clsParseExpression  = &pclass{"ParseExpression", clsParserElement}
	clsAnd              = &pclass{"And", clsParseExpression}
	clsOr               = &pclass{"Or", clsParseExpression}
	clsMatchFirst       = &pclass{"MatchFirst", clsParseExpression}
	clsEach             = &pclass{"Each", clsParseExpression}
	clsEnhance          = &pclass{"ParseElementEnhance", clsParserElement}
	clsAtStringStart    = &pclass{"AtStringStart", clsEnhance}
	clsAtLineStart      = &pclass{"AtLineStart", clsEnhance}
	clsFollowedBy       = &pclass{"FollowedBy", clsEnhance}
	clsFB               = &pclass{"FollowedBy>", clsFollowedBy}
	clsPrecededBy       = &pclass{"PrecededBy", clsEnhance}
	clsLocated          = &pclass{"Located", clsEnhance}
	clsNotAny           = &pclass{"NotAny", clsEnhance}
	clsMultipleMatch    = &pclass{"_MultipleMatch", clsEnhance}
	clsOneOrMore        = &pclass{"OneOrMore", clsMultipleMatch}
	clsZeroOrMore       = &pclass{"ZeroOrMore", clsMultipleMatch}
	clsDelimitedList    = &pclass{"DelimitedList", clsEnhance}
	clsOpt              = &pclass{"Opt", clsEnhance}
	clsSkipTo           = &pclass{"SkipTo", clsEnhance}
	clsForward          = &pclass{"Forward", clsEnhance}
	clsTokenConverter   = &pclass{"TokenConverter", clsEnhance}
	clsCombine          = &pclass{"Combine", clsTokenConverter}
	clsGroup            = &pclass{"Group", clsTokenConverter}
	clsDict             = &pclass{"Dict", clsTokenConverter}
	clsSuppress         = &pclass{"Suppress", clsTokenConverter}
	clsIndentedBlockStd = &pclass{"IndentedBlock", clsEnhance}
)

func (c *pclass) isSub(of *pclass) bool {
	for x := c; x != nil; x = x.parent {
		if x == of {
			return true
		}
	}
	return false
}

// elemList is a Python list of elements that may be shared between elements.
type elemList struct{ items []*ParserElement }

// ParserElement is any pyparsing element.
type ParserElement struct {
	cls *pclass

	parseAction           []*action
	customName            *string
	defaultName           *string
	resultsName           *string
	saveAsList            bool
	skipWhitespace        bool
	whiteChars            runeSet
	copyDefaultWhiteChars bool
	mayReturnEmpty        bool
	keepTabs              bool
	ignoreExprs           *elemList
	streamlined           bool
	mayIndexError         bool
	errmsg                *string
	modalResults          bool
	callPreparse          bool
	callDuringTry         bool

	exprs []*ParserElement // ParseExpression
	expr  *ParserElement   // ParseElementEnhance

	// Literal / Keyword / CaselessLiteral / CloseMatch
	match          []rune
	matchLen       int
	returnString   string
	caseless       bool
	caselessmatch  string
	identChars     runeSet
	maxMismatches  int
	initChars      runeSet
	bodyChars      runeSet
	initCharsOrig  string
	bodyCharsOrig  string
	minLen, maxLen int
	asKeyword      bool
	reString       string
	re             *pyre.Pattern
	useRe          bool

	// Regex
	pattern     string
	flags       int
	asGroupList bool
	asMatch     bool

	// QuotedString
	qs *quotedData

	// CharsNotIn
	notChars    string
	notCharsSet runeSet
	// White
	matchWhite []rune
	// GoToColumn
	col int
	// LineStart
	origWhiteChars runeSet
	skipper        *ParserElement
	// WordStart / WordEnd
	wordChars runeSet
	// Tag
	tagName  string
	tagValue any
	// Each
	each *eachData
	// PrecededBy
	exact   bool
	retreat int
	// _MultipleMatch
	maxCount *int
	notEnder *ParserElement
	// DelimitedList
	dl *delimData
	// Opt
	defaultValue any
	// SkipTo
	ignoreExpr   any
	includeMatch bool
	failOn       *ParserElement
	ignorer      *ParserElement
	// Combine / Group / Dict
	adjacent     bool
	joinString   string
	asPythonList bool
	asPythonDict bool
	// _PendingSkip
	anchor   *ParserElement
	mustSkip bool
}

type eachData struct {
	opt1map        map[*ParserElement]*ParserElement
	optionals      []*ParserElement
	multioptionals []*ParserElement
	multirequired  []*ParserElement
	required       []*ParserElement
}

type delimData struct {
	content            *ParserElement
	rawDelim           string
	delim              any
	combine            bool
	min                int
	max                *int
	allowTrailingDelim bool
}

// action is one entry of parseAction (a pointer, so that copies share it and identity
// comparisons work as in Python).
type action struct{ fn ParseAction }

// ParseAction is a parse action.
type ParseAction func(s string, loc int, toks *ParseResults) (any, error)

// Condition is a condition for AddCondition.
type Condition func(s string, loc int, toks *ParseResults) bool

// TokenFunc is the function TokenMap applies to each token.
type TokenFunc func(tok any, args ...any) (any, error)

// ScanMatch is one item scan_string yields.
type ScanMatch struct {
	Tokens     *ParseResults
	Start, End int
}

// ---------------------------------------------------------------- class-level state

var (
	defaultWhiteChars  = " \n\t\r"
	literalStringClass func(s string) *ParserElement
	literalClassCls    = clsLiteral
	builtinExprs       []*ParserElement
	packratEnabled     bool
	leftRecursion      bool
)

func (e *ParserElement) isa(c *pclass) bool { return e != nil && e.cls.isSub(c) }

// newElement is ParserElement.__init__(savelist).
func newElement(cls *pclass, savelist bool) *ParserElement {
	return &ParserElement{
		cls:                   cls,
		saveAsList:            savelist,
		skipWhitespace:        true,
		whiteChars:            newRuneSet(defaultWhiteChars),
		copyDefaultWhiteChars: true,
		ignoreExprs:           &elemList{},
		mayIndexError:         true,
		errmsg:                strp(""),
		modalResults:          true,
		callPreparse:          true,
	}
}

func (e *ParserElement) setErrmsg(s string) { e.errmsg = strp(s) }

func (e *ParserElement) errmsgOr() string {
	if e.errmsg == nil {
		return ""
	}
	return *e.errmsg
}

// literal converts a str argument as ParserElement._literalStringClass does.
func literalOf(s string) *ParserElement {
	if literalStringClass == nil {
		return Literal(s)
	}
	return literalStringClass(s)
}

// toElement converts an `any` expression argument (string, element) to an element; nil stays nil.
func toElement(v any) *ParserElement {
	switch x := v.(type) {
	case nil:
		return nil
	case *ParserElement:
		return x
	case string:
		return literalOf(x)
	}
	return nil
}

// ---------------------------------------------------------------- copy, names

// Copy returns expr.copy().
func (e *ParserElement) Copy() *ParserElement {
	if e == nil {
		return nil
	}
	return e.copy()
}

func (e *ParserElement) copy() *ParserElement {
	if e.cls == clsForward && e.expr == nil {
		ret := Forward()
		ret.lshift(e)
		return ret
	}
	cpy := new(ParserElement)
	*cpy = *e
	cpy.parseAction = append([]*action(nil), e.parseAction...)
	cpy.ignoreExprs = &elemList{items: append([]*ParserElement(nil), e.ignoreExprs.items...)}
	if e.copyDefaultWhiteChars {
		cpy.whiteChars = newRuneSet(defaultWhiteChars)
	}
	if e.cls.isSub(clsParseExpression) {
		cpy.exprs = make([]*ParserElement, len(e.exprs))
		for i, x := range e.exprs {
			cpy.exprs[i] = x.copy()
		}
	}
	return cpy
}

// SetResultsName returns a copy of expr with a results name.
func (e *ParserElement) SetResultsName(name string, listAllMatches bool) *ParserElement {
	if e == nil {
		return nil
	}
	return e.setResultsName(name, listAllMatches)
}

func (e *ParserElement) setResultsName(name string, listAllMatches bool) *ParserElement {
	newself := e.copy()
	if len(name) > 0 && name[len(name)-1] == '*' {
		name = name[:len(name)-1]
		listAllMatches = true
	}
	newself.resultsName = strp(name)
	newself.modalResults = !listAllMatches
	return newself
}

// Call is expr(name).
func (e *ParserElement) Call(name string) *ParserElement { return e.setResultsName(name, false) }

func (e *ParserElement) defaultNameStr() string {
	if e.defaultName == nil {
		n := e.generateDefaultName()
		e.defaultName = &n
	}
	return *e.defaultName
}

func (e *ParserElement) name() string {
	if e.customName != nil {
		return *e.customName
	}
	return e.defaultNameStr()
}

// SetName sets the element's name.
func (e *ParserElement) SetName(name string) *ParserElement {
	if e == nil {
		return nil
	}
	e.customName = strp(name)
	e.setErrmsg("Expected " + e.String())
	return e
}

// String returns str(expr), the element's name.
func (e *ParserElement) String() string {
	if e == nil {
		return "None"
	}
	return e.name()
}

// Repr returns repr(expr).
func (e *ParserElement) Repr() string {
	if e.cls == clsPendingSkip {
		return e.defaultNameStr()
	}
	return e.String()
}

// ---------------------------------------------------------------- parse actions

// trimArity is _trim_arity: IndexError raised before the action has once returned normally
// becomes _ParseActionIndexError.
func trimArity(fn ParseAction) ParseAction {
	found := false
	return func(s string, loc int, toks *ParseResults) (any, error) {
		if found {
			return fn(s, loc, toks)
		}
		ret, err := fn(s, loc, toks)
		if err != nil {
			if pe, ok := err.(*PyError); ok && pe.Class == "IndexError" {
				return nil, &paIndexError{exc: pe}
			}
			return nil, err
		}
		found = true
		return ret, nil
	}
}

// paIndexError is _ParseActionIndexError.
type paIndexError struct{ exc *PyError }

// Error returns str() of the _ParseActionIndexError.
func (p *paIndexError) Error() string {
	return "('IndexError raised in parse action', " + Repr(p.exc) + ")"
}

// SetParseAction replaces the parse actions (a single nil function clears them).
func (e *ParserElement) SetParseAction(callDuringTry bool, fns ...ParseAction) *ParserElement {
	if e == nil {
		return nil
	}
	if len(fns) == 1 && fns[0] == nil {
		e.parseAction = e.parseAction[:0:0]
		return e
	}
	acts := make([]*action, 0, len(fns))
	for _, f := range fns {
		if f == nil {
			continue
		}
		acts = append(acts, &action{trimArity(f)})
	}
	e.parseAction = acts
	e.callDuringTry = e.callDuringTry || callDuringTry
	return e
}

// AddParseAction appends parse actions.
func (e *ParserElement) AddParseAction(callDuringTry bool, fns ...ParseAction) *ParserElement {
	if e == nil {
		return nil
	}
	for _, f := range fns {
		if f == nil {
			continue
		}
		e.parseAction = append(e.parseAction, &action{trimArity(f)})
	}
	e.callDuringTry = e.callDuringTry || callDuringTry
	return e
}

// setPA / addPA are the internal forms (no arity trimming difference in Go).
func (e *ParserElement) setPA(fns ...ParseAction) *ParserElement {
	return e.SetParseAction(false, fns...)
}

func (e *ParserElement) addPA(fns ...ParseAction) *ParserElement {
	return e.AddParseAction(false, fns...)
}

// conditionAsParseAction is condition_as_parse_action.
func conditionAsParseAction(fn Condition, message *string, fatal bool) ParseAction {
	msg := "failed user-defined condition"
	if message != nil {
		msg = *message
	}
	kind := KindParseException
	if fatal {
		kind = KindParseFatalException
	}
	return func(s string, l int, t *ParseResults) (any, error) {
		if !fn(s, l, t) {
			m := msg
			return nil, newPE(kind, s, l, &m, nil)
		}
		return nil, nil
	}
}

// AddCondition adds conditions (add_condition(*fns, message=message, fatal=fatal,
// call_during_try=callDuringTry)). Note that, as in the original, a nil message is
// converted with str() and so reads "None".
func (e *ParserElement) AddCondition(message *string, fatal, callDuringTry bool, fns ...Condition) *ParserElement {
	if e == nil {
		return nil
	}
	m := "None"
	if message != nil {
		m = *message
	}
	for _, fn := range fns {
		if fn == nil {
			continue
		}
		e.parseAction = append(e.parseAction, &action{conditionAsParseAction(fn, &m, fatal)})
	}
	e.callDuringTry = e.callDuringTry || callDuringTry
	return e
}

// ---------------------------------------------------------------- whitespace / ignore

// LeaveWhitespace disables whitespace skipping (recursively copying contained expressions).
func (e *ParserElement) LeaveWhitespace(recursive bool) *ParserElement {
	if e == nil {
		return nil
	}
	return e.leaveWhitespace(recursive)
}

func (e *ParserElement) leaveWhitespace(recursive bool) *ParserElement {
	switch {
	case e.cls == clsForward:
		e.skipWhitespace = false
		return e
	case e.cls.isSub(clsParseExpression):
		e.skipWhitespace = false
		if recursive {
			n := make([]*ParserElement, len(e.exprs))
			for i, x := range e.exprs {
				n[i] = x.copy()
			}
			e.exprs = n
			for _, x := range e.exprs {
				x.leaveWhitespace(recursive)
			}
		}
		return e
	case e.cls.isSub(clsEnhance):
		e.skipWhitespace = false
		if recursive && e.expr != nil {
			e.expr = e.expr.copy()
			e.expr.leaveWhitespace(recursive)
		}
		return e
	}
	e.skipWhitespace = false
	return e
}

// IgnoreWhitespace enables whitespace skipping (recursively copying contained expressions).
func (e *ParserElement) IgnoreWhitespace(recursive bool) *ParserElement {
	if e == nil {
		return nil
	}
	switch {
	case e.cls == clsForward:
		e.skipWhitespace = true
		return e
	case e.cls.isSub(clsParseExpression):
		e.skipWhitespace = true
		if recursive {
			n := make([]*ParserElement, len(e.exprs))
			for i, x := range e.exprs {
				n[i] = x.copy()
			}
			e.exprs = n
			for _, x := range e.exprs {
				x.IgnoreWhitespace(recursive)
			}
		}
		return e
	case e.cls.isSub(clsEnhance):
		e.skipWhitespace = true
		if recursive && e.expr != nil {
			e.expr = e.expr.copy()
			e.expr.IgnoreWhitespace(recursive)
		}
		return e
	}
	e.skipWhitespace = true
	return e
}

// SetWhitespaceChars sets the whitespace characters.
func (e *ParserElement) SetWhitespaceChars(chars string, copyDefaults bool) *ParserElement {
	if e == nil {
		return nil
	}
	return e.setWhitespaceSet(newRuneSet(chars), copyDefaults)
}

func (e *ParserElement) setWhitespaceSet(chars runeSet, copyDefaults bool) *ParserElement {
	e.skipWhitespace = true
	e.whiteChars = chars.copySet()
	e.copyDefaultWhiteChars = copyDefaults
	return e
}

// ParseWithTabs keeps tabs in the input string.
func (e *ParserElement) ParseWithTabs() *ParserElement {
	if e == nil {
		return nil
	}
	e.keepTabs = true
	return e
}

// Ignore adds an expression to skip (a str becomes Suppress(str)) and returns the element;
// on a SkipTo it returns nil, as the original's SkipTo.ignore returns None.
func (e *ParserElement) Ignore(other any) *ParserElement {
	if e == nil {
		return nil
	}
	switch x := other.(type) {
	case string:
		e.ignoreArg(Suppress(x), true)
	case *ParserElement:
		if x != nil {
			e.ignoreArg(x, false)
		}
	}
	if e.cls == clsSkipTo {
		return nil
	}
	return e
}

func ignoreArg(other any) *ParserElement {
	switch x := other.(type) {
	case string:
		return Suppress(x)
	case *ParserElement:
		return x
	}
	return nil
}

func (e *ParserElement) containsIgnore(o *ParserElement) bool {
	for _, x := range e.ignoreExprs.items {
		if elemEqual(x, o) {
			return true
		}
	}
	return false
}

// baseIgnore is ParserElement.ignore (other already converted from str when it was one).
func (e *ParserElement) baseIgnore(other *ParserElement) {
	if other.isa(clsSuppress) {
		if !e.containsIgnore(other) {
			e.ignoreExprs.items = append(e.ignoreExprs.items, other)
		}
	} else {
		e.ignoreExprs.items = append(e.ignoreExprs.items, Suppress(other.copy()))
	}
}

func (e *ParserElement) lastIgnore() *ParserElement {
	return e.ignoreExprs.items[len(e.ignoreExprs.items)-1]
}

func (e *ParserElement) ignore(other *ParserElement) { e.ignoreArg(other, false) }

// ignoreArg is the class's ignore(); fromStr tells that the argument was a str (which the
// overrides test with isinstance(other, Suppress) before the base converts it).
func (e *ParserElement) ignoreArg(other *ParserElement, fromStr bool) {
	isSup := other.isa(clsSuppress) && !fromStr
	switch {
	case e.cls == clsCombine && e.adjacent:
		e.baseIgnore(other)
	case e.cls == clsSkipTo:
		e.enhanceIgnore(other, isSup)
		e.updateIgnorer()
	case e.cls.isSub(clsParseExpression):
		if isSup {
			if !e.containsIgnore(other) {
				e.baseIgnore(other)
				for _, x := range e.exprs {
					x.ignore(e.lastIgnore())
				}
			}
		} else {
			e.baseIgnore(other)
			for _, x := range e.exprs {
				x.ignore(e.lastIgnore())
			}
		}
	case e.cls.isSub(clsEnhance):
		e.enhanceIgnore(other, isSup)
	default:
		e.baseIgnore(other)
	}
}

func (e *ParserElement) enhanceIgnore(other *ParserElement, isSup bool) {
	if !isSup || !e.containsIgnore(other) {
		e.baseIgnore(other)
		if e.expr != nil {
			e.expr.ignore(e.lastIgnore())
		}
	}
}

// Suppress returns Suppress(expr) (a Suppress returns itself).
func (e *ParserElement) Suppress() *ParserElement {
	if e == nil {
		return nil
	}
	if e.cls == clsSuppress {
		return e
	}
	return Suppress(e)
}

// ---------------------------------------------------------------- recursion helpers

func (e *ParserElement) recurse() []*ParserElement {
	switch {
	case e.cls.isSub(clsParseExpression):
		return append([]*ParserElement(nil), e.exprs...)
	case e.cls.isSub(clsEnhance):
		if e.expr != nil {
			return []*ParserElement{e.expr}
		}
	}
	return nil
}

// Streamline returns expr.streamline().
func (e *ParserElement) Streamline() *ParserElement { return e.streamline() }

func (e *ParserElement) baseStreamline() {
	e.streamlined = true
	e.defaultName = nil
}

func (e *ParserElement) streamline() *ParserElement {
	switch {
	case e.cls == clsForward:
		if !e.streamlined {
			e.streamlined = true
			if e.expr != nil {
				e.expr.streamline()
			}
		}
		return e
	case e.cls == clsAnd:
		return e.andStreamline()
	case e.cls == clsOr:
		e.peStreamline()
		if len(e.exprs) > 0 {
			e.mayReturnEmpty = anyExpr(e.exprs, func(x *ParserElement) bool { return x.mayReturnEmptyProp() })
			e.saveAsList = anyExpr(e.exprs, func(x *ParserElement) bool { return x.saveAsList })
			e.skipWhitespace = allExpr(e.exprs, func(x *ParserElement) bool { return x.skipWhitespace && !x.isa(clsWhite) })
		} else {
			e.saveAsList = false
		}
		return e
	case e.cls == clsMatchFirst:
		if e.streamlined {
			return e
		}
		e.peStreamline()
		if len(e.exprs) > 0 {
			e.saveAsList = anyExpr(e.exprs, func(x *ParserElement) bool { return x.saveAsList })
			e.mayReturnEmpty = anyExpr(e.exprs, func(x *ParserElement) bool { return x.mayReturnEmptyProp() })
			e.skipWhitespace = allExpr(e.exprs, func(x *ParserElement) bool { return x.skipWhitespace && !x.isa(clsWhite) })
		} else {
			e.saveAsList = false
			e.mayReturnEmpty = true
		}
		return e
	case e.cls == clsEach:
		e.peStreamline()
		if len(e.exprs) > 0 {
			e.mayReturnEmpty = allExpr(e.exprs, func(x *ParserElement) bool { return x.mayReturnEmptyProp() })
		} else {
			e.mayReturnEmpty = true
		}
		return e
	case e.cls.isSub(clsParseExpression):
		e.peStreamline()
		return e
	case e.cls.isSub(clsEnhance):
		e.baseStreamline()
		if e.expr != nil {
			e.expr.streamline()
		}
		return e
	}
	e.baseStreamline()
	return e
}

func (e *ParserElement) mayReturnEmptyProp() bool { return e.mayReturnEmpty }

func anyExpr(xs []*ParserElement, f func(*ParserElement) bool) bool {
	for _, x := range xs {
		if f(x) {
			return true
		}
	}
	return false
}

func allExpr(xs []*ParserElement, f func(*ParserElement) bool) bool {
	for _, x := range xs {
		if !f(x) {
			return false
		}
	}
	return true
}

// peStreamline is ParseExpression.streamline.
func (e *ParserElement) peStreamline() {
	if e.streamlined {
		return
	}
	e.baseStreamline()
	for _, x := range e.exprs {
		x.streamline()
	}
	if len(e.exprs) == 2 {
		first, second := e.exprs[0], e.exprs[1]
		if first.cls.isSub(e.cls) && len(first.parseAction) == 0 && first.resultsName == nil {
			n := append(append([]*ParserElement(nil), first.exprs...), second)
			e.exprs = n
			e.defaultName = nil
			e.mayReturnEmpty = e.mayReturnEmpty || first.mayReturnEmptyProp()
			e.mayIndexError = e.mayIndexError || first.mayIndexError
		}
		last := e.exprs[len(e.exprs)-1]
		if last.cls.isSub(e.cls) && len(last.parseAction) == 0 && last.resultsName == nil {
			n := append(append([]*ParserElement(nil), e.exprs[:len(e.exprs)-1]...), last.exprs...)
			e.exprs = n
			e.defaultName = nil
			e.mayReturnEmpty = e.mayReturnEmpty || last.mayReturnEmptyProp()
			e.mayIndexError = e.mayIndexError || last.mayIndexError
		}
	}
	e.setErrmsg("Expected " + e.String())
}

func (e *ParserElement) andStreamline() *ParserElement {
	if len(e.exprs) > 0 {
		has := false
		for _, x := range e.exprs[:len(e.exprs)-1] {
			if x.cls.isSub(clsParseExpression) && len(x.exprs) > 0 && x.exprs[len(x.exprs)-1].cls == clsPendingSkip {
				has = true
				break
			}
		}
		if has {
			marker := NoMatch()
			snapshot := append([]*ParserElement(nil), e.exprs[:len(e.exprs)-1]...)
			for i, x := range snapshot {
				if x == marker {
					continue
				}
				if x.cls.isSub(clsParseExpression) && len(x.exprs) > 0 && x.exprs[len(x.exprs)-1].cls == clsPendingSkip {
					x.exprs[len(x.exprs)-1] = x.exprs[len(x.exprs)-1].Add(e.exprs[i+1])
					e.exprs[i+1] = marker
				}
			}
			var n []*ParserElement
			for _, x := range e.exprs {
				if x != marker {
					n = append(n, x)
				}
			}
			e.exprs = n
		}
	}
	e.peStreamline()
	e.mayReturnEmpty = allExpr(e.exprs, func(x *ParserElement) bool { return x.mayReturnEmptyProp() })
	return e
}

// ---------------------------------------------------------------- class-level settings

// SetDefaultWhitespaceChars is ParserElement.set_default_whitespace_chars.
func SetDefaultWhitespaceChars(chars string) {
	defaultWhiteChars = chars
	for _, x := range builtinExprs {
		if x.copyDefaultWhiteChars {
			x.whiteChars = newRuneSet(chars)
		}
	}
}

// InlineLiteralsUsing is ParserElement.inline_literals_using.
func InlineLiteralsUsing(cls func(s string) *ParserElement) {
	if cls == nil {
		literalStringClass = nil
		literalClassCls = clsLiteral
		return
	}
	save := literalStringClass
	literalStringClass = nil
	probe := cls("x")
	literalStringClass = save
	literalStringClass = cls
	if probe != nil {
		literalClassCls = probe.cls
		if literalClassCls == clsSingleCharLit || literalClassCls == clsEmpty {
			literalClassCls = clsLiteral
		}
	} else {
		literalClassCls = clsLiteral
	}
}

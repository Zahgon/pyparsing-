// Port of pyparsing/actions.py (pyparsing 3.3.3+). MIT License, Copyright (c) 2003-2025  Paul McGuire; see LICENSE.

package pyparsing

// OnlyOnce is pyparsing's OnlyOnce wrapper.
type OnlyOnce struct {
	called   bool
	callable ParseAction
}

// NewOnlyOnce returns OnlyOnce(fn).
func NewOnlyOnce(fn ParseAction) *OnlyOnce { return &OnlyOnce{callable: trimArity(fn)} }

// Call runs the wrapped action the first time; later calls raise ParseException.
func (o *OnlyOnce) Call(s string, loc int, toks *ParseResults) (any, error) {
	if !o.called {
		r, err := o.callable(s, loc, toks)
		if err != nil {
			return nil, err
		}
		o.called = true
		return r, nil
	}
	return nil, newParseExc(s, loc, "OnlyOnce obj called multiple times w/out reset", nil)
}

// Reset allows the wrapped action to run again.
func (o *OnlyOnce) Reset() { o.called = false }

// MatchOnlyAtCol is match_only_at_col(n).
func MatchOnlyAtCol(n int) ParseAction {
	return func(s string, loc int, toks *ParseResults) (any, error) {
		if colRunes(loc, runesOf(s)) != n {
			return nil, newParseExc(s, loc, "matched token not at column "+itoa(n), nil)
		}
		return nil, nil
	}
}

// ReplaceWith is replace_with(repl).
func ReplaceWith(repl any) ParseAction {
	return func(s string, loc int, toks *ParseResults) (any, error) { return []any{repl}, nil }
}

// RemoveQuotes is remove_quotes: t[0][1:-1].
func RemoveQuotes(s string, loc int, toks *ParseResults) (any, error) {
	v, err := toks.At(0)
	if err != nil {
		return nil, err
	}
	switch x := v.(type) {
	case string:
		rs := []rune(x)
		return string(pySlice(rs, 1, len(rs)-1)), nil
	case *ParseResults:
		items := x.toks.items
		return append([]any{}, items[pyIndex(1, len(items)):pyIndex(max(len(items)-1, pyIndex(1, len(items))), len(items))]...), nil
	case []any:
		return append([]any{}, x[pyIndex(1, len(x)):pyIndex(max(len(x)-1, pyIndex(1, len(x))), len(x))]...), nil
	}
	return nil, pyErr("TypeError", "'"+pyTypeName(v)+"' object is not subscriptable")
}

// AttrSpec is one attribute name and value for WithAttribute.
type AttrSpec struct {
	Name  string
	Value any // a string, or WithAttributeAnyValue
}

// WithAttributeAnyValue is with_attribute.ANY_VALUE.
var WithAttributeAnyValue any = anyValueType{}

// WithAttribute is with_attribute(*attrs).
func WithAttribute(attrs ...AttrSpec) ParseAction {
	list := append([]AttrSpec(nil), attrs...)
	return func(s string, l int, tokens *ParseResults) (any, error) {
		for _, a := range list {
			if !tokens.Contains(a.Name) {
				return nil, newParseExc(s, l, "no matching attribute "+Repr(a.Name), nil)
			}
			if _, isAny := a.Value.(anyValueType); isAny {
				continue
			}
			v, _ := tokens.getItemStr(a.Name)
			if !pyEqual(v, a.Value) {
				return nil, newParseExc(s, l, "attribute "+Repr(a.Name)+" has value "+Repr(v)+", must be "+Repr(a.Value), nil)
			}
		}
		return nil, nil
	}
}

// WithClass is with_class(classname, namespace).
func WithClass(classname string, namespace string) ParseAction {
	attr := "class"
	if namespace != "" {
		attr = namespace + ":class"
	}
	return WithAttribute(AttrSpec{attr, classname})
}

// TokenMap is token_map(func, *args).
func TokenMap(fn TokenFunc, args ...any) ParseAction {
	return func(s string, l int, t *ParseResults) (any, error) {
		out := make([]any, 0, t.Len())
		for _, tok := range append([]any(nil), t.toks.items...) {
			r, err := fn(tok, args...)
			if err != nil {
				return nil, err
			}
			out = append(out, r)
		}
		return out, nil
	}
}

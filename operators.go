// Port of pyparsing/core.py, ParserElement operators (pyparsing 3.3.3+). MIT License, Copyright (c) 2003-2025  Paul McGuire; see LICENSE.

package pyparsing

import (
	"math"
	"math/big"

	"github.com/pyparsing/pyparsing-go/internal/pyval"
)

func pyStrip(s string) string { return pyval.StripSpace(s) }

// operand converts the right operand of a binary operator: Ellipsis is returned as is, a str
// becomes a literal, an element stays; anything else gives nil (NotImplemented).
func operand(other any) (*ParserElement, bool) {
	switch v := other.(type) {
	case *ParserElement:
		return v, v != nil
	case string:
		return literalOf(v), true
	}
	return nil, false
}

// Add is expr + other (other: element, str or Ellipsis).
func (e *ParserElement) Add(other any) *ParserElement {
	if e == nil {
		return nil
	}
	if e.cls == clsPendingSkip {
		return e.pendingSkipAdd(other)
	}
	if e.cls == clsSuppress && e.expr != nil && e.expr.cls == clsPendingSkip {
		return Suppress(SkipTo(other, false, nil, nil)).Add(other)
	}
	if _, ok := other.(EllipsisType); ok {
		return newPendingSkip(e, false)
	}
	o, ok := operand(other)
	if !ok {
		return nil
	}
	return andOf(e, o)
}

// iadd is And.__iadd__ (append in place).
func (e *ParserElement) iadd(other *ParserElement) *ParserElement {
	if e.cls == clsAnd {
		e.exprs = append(e.exprs, other)
		e.defaultName = nil
		return e
	}
	return e.Add(other)
}

// RAdd is other + expr.
func (e *ParserElement) RAdd(other any) *ParserElement {
	if e == nil {
		return nil
	}
	if _, ok := other.(EllipsisType); ok {
		return SkipTo(e, false, nil, nil).Call("_skipped*").Add(e)
	}
	o, ok := operand(other)
	if !ok {
		return nil
	}
	return o.Add(e)
}

// Sub is expr - other.
func (e *ParserElement) Sub(other any) *ParserElement {
	if e == nil {
		return nil
	}
	if e.cls == clsSuppress && e.expr != nil && e.expr.cls == clsPendingSkip {
		s := Suppress(SkipTo(other, false, nil, nil))
		return s.Sub(other)
	}
	o, ok := operand(other)
	if !ok {
		return nil
	}
	return e.Add(errorStop()).Add(o)
}

// RSub is other - expr.
func (e *ParserElement) RSub(other any) *ParserElement {
	if e == nil {
		return nil
	}
	o, ok := operand(other)
	if !ok {
		return nil
	}
	return o.Sub(e)
}

// BitOr is expr | other.
func (e *ParserElement) BitOr(other any) *ParserElement {
	if e == nil {
		return nil
	}
	if _, ok := other.(EllipsisType); ok {
		return newPendingSkip(e, true)
	}
	if s, ok := other.(string); ok && s == "" {
		return Opt(e)
	}
	o, ok := operand(other)
	if !ok {
		return nil
	}
	return MatchFirst(e, o)
}

// RBitOr is other | expr.
func (e *ParserElement) RBitOr(other any) *ParserElement {
	if e == nil {
		return nil
	}
	o, ok := operand(other)
	if !ok {
		return nil
	}
	return o.BitOr(e)
}

// BitXor is expr ^ other.
func (e *ParserElement) BitXor(other any) *ParserElement {
	if e == nil {
		return nil
	}
	o, ok := operand(other)
	if !ok {
		return nil
	}
	return Or(e, o)
}

// RBitXor is other ^ expr.
func (e *ParserElement) RBitXor(other any) *ParserElement {
	if e == nil {
		return nil
	}
	o, ok := operand(other)
	if !ok {
		return nil
	}
	return o.BitXor(e)
}

// BitAnd is expr & other.
func (e *ParserElement) BitAnd(other any) *ParserElement {
	if e == nil {
		return nil
	}
	o, ok := operand(other)
	if !ok {
		return nil
	}
	return Each(e, o)
}

// RBitAnd is other & expr.
func (e *ParserElement) RBitAnd(other any) *ParserElement {
	if e == nil {
		return nil
	}
	o, ok := operand(other)
	if !ok {
		return nil
	}
	return o.BitAnd(e)
}

// Invert is ~expr.
func (e *ParserElement) Invert() *ParserElement { return NotAny(e) }

func isSeqType(v any) bool {
	switch v.(type) {
	case string, []any, PyTuple, prList:
		return true
	}
	return false
}

func mulTypeError(left, right any) error {
	return pyErr("TypeError", "unsupported operand type(s) for *: '"+pyTypeName(left)+"' and '"+pyTypeName(right)+"'")
}

// Mul is expr * other (an int, Ellipsis, or a PyTuple of two items each an int, nil or Ellipsis).
func (e *ParserElement) Mul(other any) (*ParserElement, error) {
	if e == nil {
		return nil, pyErr("AttributeError", "'NoneType' object has no attribute '__mul__'")
	}
	r, ok, err := e.mul(other)
	if err != nil {
		return nil, err
	}
	if !ok {
		if isSeqType(other) {
			return nil, pyErr("TypeError", "can't multiply sequence by non-int of type '"+e.cls.name+"'")
		}
		return nil, mulTypeError(e, other)
	}
	return r, nil
}

// RMul is other * expr.
func (e *ParserElement) RMul(other any) (*ParserElement, error) {
	if e == nil {
		return nil, pyErr("AttributeError", "'NoneType' object has no attribute '__rmul__'")
	}
	r, ok, err := e.mul(other)
	if err != nil {
		return nil, err
	}
	if !ok {
		if isSeqType(other) {
			return nil, pyErr("TypeError", "can't multiply sequence by non-int of type '"+e.cls.name+"'")
		}
		return nil, mulTypeError(other, e)
	}
	return r, nil
}

func isNoneOrEllipsis(v any) bool {
	if v == nil {
		return true
	}
	_, ok := v.(EllipsisType)
	return ok
}

func asInt(v any) (int, bool) {
	switch x := v.(type) {
	case int:
		return x, true
	case bool:
		if x {
			return 1, true
		}
		return 0, true
	}
	if b, ok := toBigInt(v); ok && b.IsInt64() {
		return int(b.Int64()), true
	}
	return 0, false
}

// asBig converts a Python int (Go integer, *big.Int or bool) to *big.Int.
func asBig(v any) (*big.Int, bool) {
	if b, ok := v.(bool); ok {
		return boolInt(b), true
	}
	return toBigInt(v)
}

func (e *ParserElement) mul(other any) (*ParserElement, bool, error) {
	if _, ok := other.(EllipsisType); ok {
		other = PyTuple{0, nil}
	} else if t, ok := other.(PyTuple); ok && len(t) > 0 {
		if _, ok := t[0].(EllipsisType); ok {
			n := PyTuple{0}
			if len(t) > 1 {
				n = append(n, t[1])
			}
			n = append(n, nil)
			other = n[:2]
		}
	}
	var minE, optE *big.Int
	if n, ok := asBig(other); ok {
		minE, optE = n, big.NewInt(0)
	} else if t, ok := other.(PyTuple); ok {
		tt := make(PyTuple, 0, len(t)+2)
		for _, o := range t {
			if isNoneOrEllipsis(o) {
				tt = append(tt, nil)
			} else {
				tt = append(tt, o)
			}
		}
		tt = append(tt, nil, nil)[:2]
		if tt[0] == nil {
			tt[0] = 0
		}
		a, aInt := asBig(tt[0])
		if aInt && tt[1] == nil {
			switch {
			case a.Sign() == 0:
				return zeroOrMore(e), true, nil
			case a.Cmp(big.NewInt(1)) == 0:
				return oneOrMore(e), true, nil
			}
			first, err := e.Mul(a)
			if err != nil {
				return nil, false, err
			}
			return first.Add(zeroOrMore(e)), true, nil
		}
		b, bInt := asBig(tt[1])
		if aInt && bInt {
			minE, optE = a, new(big.Int).Sub(b, a)
		} else {
			return nil, false, nil
		}
	} else {
		return nil, false, nil
	}
	if minE.Sign() < 0 {
		return nil, false, pyErr("ValueError", "cannot multiply ParserElement by negative value")
	}
	if optE.Sign() < 0 {
		return nil, false, pyErr("ValueError", "second tuple value must be greater or equal to first tuple value")
	}
	if minE.Sign() == 0 && optE.Sign() == 0 {
		return newAnd(nil, true), true, nil
	}
	repeat := func(n *big.Int) (*ParserElement, error) {
		if !n.IsInt64() {
			return nil, pyErr("OverflowError", "cannot fit 'int' into an index-sized integer")
		}
		if n.Int64() > 1<<26 {
			return nil, pyErr("MemoryError", "")
		}
		args := make([]any, n.Int64())
		for i := range args {
			args[i] = e
		}
		return newAnd(args, true), nil
	}
	one := big.NewInt(1)
	if optE.Sign() > 0 {
		m := math.MaxInt
		if optE.IsInt64() {
			m = int(optE.Int64())
		}
		tail, _ := ZeroOrMore(e, nil, &m)
		if minE.Sign() > 0 {
			if minE.Cmp(one) == 0 {
				return e.Add(tail), true, nil
			}
			r, err := repeat(minE)
			if err != nil {
				return nil, false, err
			}
			return r.Add(tail), true, nil
		}
		return tail, true, nil
	}
	if minE.Cmp(one) == 0 {
		return e, true, nil
	}
	r, err := repeat(minE)
	if err != nil {
		return nil, false, err
	}
	return r, true, nil
}

// GetItem is expr[key]: an int n is expr * (n, n); a PyTuple is expr * key; Ellipsis is
// expr[...] (ZeroOrMore).
func (e *ParserElement) GetItem(key any) (*ParserElement, error) {
	if e == nil {
		return nil, pyErr("AttributeError", "'NoneType' object has no attribute '__getitem__'")
	}
	var k PyTuple
	switch v := key.(type) {
	case string:
		k = PyTuple{v}
	case PyTuple:
		k = v
	case []any:
		k = PyTuple(v)
	default:
		k = PyTuple{key, key}
	}
	if len(k) > 2 {
		head := k
		if len(head) > 5 {
			head = head[:5]
		}
		extra := ""
		if len(k) > 5 {
			extra = "... [" + itoa(len(k)) + "]"
		}
		return nil, pyErr("TypeError", "only 1 or 2 index arguments supported ("+Repr(head)+extra+")")
	}
	return e.Mul(k)
}

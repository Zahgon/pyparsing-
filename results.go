// Port of pyparsing/results.py (pyparsing 3.3.3+). MIT License, Copyright (c) 2003-2025  Paul McGuire; see LICENSE.

package pyparsing

import (
	"sort"
	"strings"
)

// offsetEntry is _ParseResultsWithOffset.
type offsetEntry struct {
	result any
	offset int
}

// occList is one Python list of _ParseResultsWithOffset (shared by shallow copies, as in Python).
type occList struct{ items []offsetEntry }

// anyList is a Python list object (ParseResults._toklist), shared where Python shares it.
type anyList struct{ items []any }

// tokDict is ParseResults._tokdict: an insertion-ordered dict of str -> list.
type tokDict struct {
	keys []string
	m    map[string]*occList
}

func newTokDict() *tokDict { return &tokDict{m: map[string]*occList{}} }

func (d *tokDict) get(k string) (*occList, bool) {
	v, ok := d.m[k]
	return v, ok
}

func (d *tokDict) set(k string, v *occList) {
	if _, ok := d.m[k]; !ok {
		d.keys = append(d.keys, k)
	}
	d.m[k] = v
}

func (d *tokDict) del(k string) bool {
	if _, ok := d.m[k]; !ok {
		return false
	}
	delete(d.m, k)
	for i, kk := range d.keys {
		if kk == k {
			d.keys = append(d.keys[:i:i], d.keys[i+1:]...)
			break
		}
	}
	return true
}

func (d *tokDict) clear() {
	d.keys = nil
	d.m = map[string]*occList{}
}

func (d *tokDict) shallowCopy() *tokDict {
	n := &tokDict{keys: append([]string(nil), d.keys...), m: make(map[string]*occList, len(d.m))}
	for k, v := range d.m {
		n.m[k] = v
	}
	return n
}

func (d *tokDict) length() int { return len(d.keys) }

// KeyValue is one item of ParseResults.items().
type KeyValue struct {
	Key   string
	Value any
}

// ParseResults is pyparsing's ParseResults.
type ParseResults struct {
	name          *string
	parent        *ParseResults
	allNames      map[string]bool
	toks          *anyList
	tokdict       *tokDict
	isDictContext bool
}

func strp(s string) *string { return &s }

// newPR is ParseResults(toklist, name, aslist, modal): __new__ followed by __init__.
func newPR(toklist any, name *string, asList, modal bool) (*ParseResults, error) {
	var self *ParseResults
	if pr, ok := toklist.(*ParseResults); ok {
		self = pr
	} else {
		self = &ParseResults{allNames: map[string]bool{}, tokdict: newTokDict()}
		switch t := toklist.(type) {
		case nil:
			self.toks = &anyList{items: []any{}}
		case []any:
			self.toks = &anyList{items: append([]any{}, t...)}
		case prList:
			self.toks = &anyList{items: []any{append([]any{}, t...)}}
		default:
			self.toks = &anyList{items: []any{toklist}}
		}
	}
	if name == nil || *name == "" {
		return self, nil
	}
	n := *name
	if !modal {
		self.allNames = map[string]bool{n: true}
	}
	self.name = strp(n)
	if isNullValue(toklist) {
		return self, nil
	}
	if s, ok := toklist.(string); ok {
		toklist = []any{s}
	}
	if asList {
		var sub *ParseResults
		if pr, ok := toklist.(*ParseResults); ok {
			sub, _ = newPR(append([]any{}, pr.toks.items...), nil, true, true)
		} else {
			first, err := pyGetItem0(toklist)
			if err != nil {
				return nil, err
			}
			sub, _ = newPR(first, nil, true, true)
		}
		self.setItemOffset(n, offsetEntry{sub, 0})
		if v, err := self.getItemStr(n); err == nil {
			if vp, ok := v.(*ParseResults); ok {
				vp.name = strp(n)
			}
		}
		return self, nil
	}
	first, err := pyGetItem0(toklist)
	if err == nil {
		self.setItemStr(n, first)
	} else if e, ok := err.(*PyError); ok && (e.Class == "KeyError" || e.Class == "TypeError" || e.Class == "IndexError") {
		if pr, ok := toklist.(*ParseResults); !ok || pr != self {
			self.setItemStr(n, toklist)
		} else {
			self.name = strp(n)
		}
	} else {
		return nil, err
	}
	return self, nil
}

// mustPR builds a ParseResults from a list that cannot fail.
func mustPR(toklist any) *ParseResults {
	r, _ := newPR(toklist, nil, true, true)
	return r
}

// isNullValue is `toklist in (None, [], ())`.
func isNullValue(v any) bool {
	switch x := v.(type) {
	case nil:
		return true
	case []any:
		return len(x) == 0
	case prList:
		return len(x) == 0
	case PyTuple:
		return len(x) == 0
	}
	return false
}

// pyGetItem0 is v[0] for an arbitrary Python value.
func pyGetItem0(v any) (any, error) {
	switch x := v.(type) {
	case []any:
		if len(x) == 0 {
			return nil, pyErr("IndexError", "list index out of range")
		}
		return x[0], nil
	case prList:
		if len(x) == 0 {
			return nil, pyErr("IndexError", "list index out of range")
		}
		return x[0], nil
	case PyTuple:
		if len(x) == 0 {
			return nil, pyErr("IndexError", "tuple index out of range")
		}
		return x[0], nil
	case string:
		for _, r := range x {
			return string(r), nil
		}
		return nil, pyErr("IndexError", "string index out of range")
	case *ParseResults:
		return x.At(0)
	case *PyDict:
		return nil, pyErr("KeyError", "0")
	case *RegexMatch:
		s, _ := x.Group(0)
		return s, nil
	}
	return nil, pyErr("TypeError", "'"+pyTypeName(v)+"' object is not subscriptable")
}

// NewParseResults returns ParseResults(list(toks)).
func NewParseResults(toks ...any) *ParseResults {
	return mustPR(append([]any{}, toks...))
}

// newEmptyPR returns ParseResults([]).
func newEmptyPR() *ParseResults { return mustPR([]any{}) }

func (r *ParseResults) pyEq(o *ParseResults) bool {
	if r == o {
		return true
	}
	if (r.name == nil) != (o.name == nil) || (r.name != nil && *r.name != *o.name) {
		return false
	}
	return pyEqual(r.AsList(false), o.AsList(false)) && pyEqual(r.AsDict(), o.AsDict())
}

// Len returns len(results).
func (r *ParseResults) Len() int { return len(r.toks.items) }

// Truth returns bool(results).
func (r *ParseResults) Truth() bool { return len(r.toks.items) > 0 || r.tokdict.length() > 0 }

// At returns results[i] for an int index (negative indexes count from the end).
func (r *ParseResults) At(i int) (any, error) {
	n := len(r.toks.items)
	if i < 0 {
		i += n
	}
	if i < 0 || i >= n {
		return nil, pyErr("IndexError", "list index out of range")
	}
	return r.toks.items[i], nil
}

// Item returns results[name].
func (r *ParseResults) Item(name string) (any, error) { return r.getItemStr(name) }

func (r *ParseResults) getItemStr(name string) (any, error) {
	occ, ok := r.tokdict.get(name)
	if !ok {
		return nil, pyErr("KeyError", Repr(name))
	}
	if !r.allNames[name] {
		return occ.items[len(occ.items)-1].result, nil
	}
	vals := make([]any, len(occ.items))
	for i, e := range occ.items {
		vals[i] = e.result
	}
	return mustPR(vals), nil
}

// Attr returns results.name: results[name], or "" when there is no such name.
func (r *ParseResults) Attr(name string) any {
	v, err := r.getItemStr(name)
	if err != nil {
		return ""
	}
	return v
}

// Get returns results.get(key, default).
func (r *ParseResults) Get(key string, def any) any {
	if r.Contains(key) {
		v, _ := r.getItemStr(key)
		return v
	}
	return def
}

// Contains returns `name in results`.
func (r *ParseResults) Contains(name string) bool {
	_, ok := r.tokdict.get(name)
	return ok
}

// Keys returns results.keys().
func (r *ParseResults) Keys() []string { return append([]string(nil), r.tokdict.keys...) }

// Values returns results.values().
func (r *ParseResults) Values() []any {
	out := []any{}
	for _, k := range r.Keys() {
		v, _ := r.getItemStr(k)
		out = append(out, v)
	}
	return out
}

// Items returns results.items().
func (r *ParseResults) Items() []KeyValue {
	out := []KeyValue{}
	for _, k := range r.Keys() {
		v, _ := r.getItemStr(k)
		out = append(out, KeyValue{k, v})
	}
	return out
}

// HasKeys returns results.haskeys().
func (r *ParseResults) HasKeys() bool { return r.tokdict.length() > 0 }

// setItemOffset is __setitem__ with a _ParseResultsWithOffset value.
func (r *ParseResults) setItemOffset(k string, v offsetEntry) {
	if cur, ok := r.tokdict.get(k); ok {
		cur.items = append(cur.items, v)
	} else {
		r.tokdict.set(k, &occList{items: []offsetEntry{v}})
	}
	if sub, ok := v.result.(*ParseResults); ok {
		sub.parent = r
	}
}

// setItemStr is __setitem__ with a str key and a plain value.
func (r *ParseResults) setItemStr(k string, v any) {
	if cur, ok := r.tokdict.get(k); ok {
		cur.items = append(cur.items, offsetEntry{v, 0})
	} else {
		r.tokdict.set(k, &occList{items: []offsetEntry{{v, 0}}})
	}
	if sub, ok := v.(*ParseResults); ok {
		sub.parent = r
	}
}

// SetItem is results[key] = v for an int or str key.
func (r *ParseResults) SetItem(key any, v any) error {
	switch k := key.(type) {
	case string:
		r.setItemStr(k, v)
		return nil
	case int:
		n := len(r.toks.items)
		i := k
		if i < 0 {
			i += n
		}
		if i < 0 || i >= n {
			return pyErr("IndexError", "list assignment index out of range")
		}
		r.toks.items[i] = v
		if sub, ok := v.(*ParseResults); ok {
			sub.parent = r
		}
		return nil
	}
	return pyErr("TypeError", "unhashable type: '"+pyTypeName(key)+"'")
}

// DelItem is del results[key] for an int or str key.
func (r *ParseResults) DelItem(key any) error {
	switch k := key.(type) {
	case string:
		if !r.tokdict.del(k) {
			return pyErr("KeyError", Repr(k))
		}
		return nil
	case int:
		return r.delIndex(k)
	}
	return pyErr("TypeError", "unhashable type: '"+pyTypeName(key)+"'")
}

// delAll is del results[:] (only the list is cleared).
func (r *ParseResults) delAll() { r.toks.items = r.toks.items[:0:0] }

func (r *ParseResults) delIndex(i int) error {
	mylen := len(r.toks.items)
	j := i
	if j < 0 {
		j += mylen
	}
	if j < 0 || j >= mylen {
		return pyErr("IndexError", "list assignment index out of range")
	}
	items := r.toks.items
	r.toks.items = append(items[:j:j], items[j+1:]...)
	removed := []int{j}
	for _, name := range r.tokdict.keys {
		occ := append([]offsetEntry{}, r.tokdict.m[name].items...)
		for _, jj := range removed {
			for k := 0; k < len(occ); k++ {
				idx := len(occ) - 1 - k
				if occ[idx].offset > jj {
					occ[idx] = offsetEntry{occ[idx].result, occ[idx].offset - 1}
				} else {
					break
				}
			}
		}
		r.tokdict.m[name] = &occList{items: occ}
	}
	return nil
}

// Pop is results.pop(): no argument pops the last item; an int pops that index; a str pops
// that name, returning the second argument (the default) when the name is absent.
func (r *ParseResults) Pop(args ...any) (any, error) {
	if len(args) == 0 {
		args = []any{-1}
	}
	_, isInt := args[0].(int)
	key, isStr := args[0].(string)
	if isInt || len(args) == 1 || (isStr && r.Contains(key)) {
		switch k := args[0].(type) {
		case int:
			ret, err := r.At(k)
			if err != nil {
				return nil, err
			}
			if err := r.delIndex(k); err != nil {
				return nil, err
			}
			return ret, nil
		case string:
			ret, err := r.getItemStr(k)
			if err != nil {
				return nil, err
			}
			r.tokdict.del(k)
			return ret, nil
		}
		return nil, pyErr("TypeError", "list indices must be integers or slices, not "+pyTypeName(args[0]))
	}
	return args[1], nil
}

// Insert is results.insert(index, v).
func (r *ParseResults) Insert(index int, v any) {
	n := len(r.toks.items)
	i := index
	if i < 0 {
		i += n
		if i < 0 {
			i = 0
		}
	}
	if i > n {
		i = n
	}
	items := make([]any, 0, n+1)
	items = append(items, r.toks.items[:i]...)
	items = append(items, v)
	items = append(items, r.toks.items[i:]...)
	r.toks.items = items
	for _, name := range r.tokdict.keys {
		occ := append([]offsetEntry{}, r.tokdict.m[name].items...)
		for k := range occ {
			if occ[k].offset > index {
				occ[k] = offsetEntry{occ[k].result, occ[k].offset + 1}
			}
		}
		r.tokdict.m[name] = &occList{items: occ}
	}
}

// Append is results.append(v).
func (r *ParseResults) Append(v any) { r.toks.items = append(r.toks.items, v) }

// Extend is results.extend(v): a ParseResults is added with +=, any other iterable is
// appended item by item (a str by character). A non-iterable value is ignored.
func (r *ParseResults) Extend(v any) {
	if pr, ok := v.(*ParseResults); ok {
		r.IAdd(pr)
		return
	}
	if s, ok := v.(string); ok {
		for _, c := range s {
			r.toks.items = append(r.toks.items, string(c))
		}
		return
	}
	if items, ok := pyIterItems(v); ok {
		r.toks.items = append(r.toks.items, items...)
	}
}

// Clear is results.clear().
func (r *ParseResults) Clear() {
	r.toks.items = r.toks.items[:0:0]
	r.tokdict.clear()
}

// Add returns results + other.
func (r *ParseResults) Add(other *ParseResults) *ParseResults {
	if other == nil || !other.Truth() {
		return r
	}
	ret := r.Copy()
	ret.IAdd(other)
	return ret
}

// IAdd is results += other; it returns results.
func (r *ParseResults) IAdd(other *ParseResults) *ParseResults {
	if other == nil || !other.Truth() {
		return r
	}
	if other.tokdict.length() > 0 {
		offset := len(r.toks.items)
		type kv struct {
			k string
			v offsetEntry
		}
		var items []kv
		for _, k := range other.tokdict.keys {
			for _, v := range other.tokdict.m[k].items {
				off := v.offset + offset
				if v.offset < 0 {
					off = offset
				}
				items = append(items, kv{k, offsetEntry{v.result, off}})
			}
		}
		for _, it := range items {
			r.setItemOffset(it.k, it.v)
			if sub, ok := it.v.result.(*ParseResults); ok {
				sub.parent = r
			}
		}
	}
	r.toks.items = append(r.toks.items, other.toks.items...)
	for k := range other.allNames {
		r.allNames[k] = true
	}
	return r
}

// Repr returns repr(results).
func (r *ParseResults) Repr() string {
	return "ParseResults(" + Repr(r.toks.items) + ", " + Repr(r.AsDict()) + ")"
}

// String returns str(results).
func (r *ParseResults) String() string {
	var b strings.Builder
	b.WriteByte('[')
	for i, it := range r.toks.items {
		if i > 0 {
			b.WriteString(", ")
		}
		if pr, ok := it.(*ParseResults); ok {
			b.WriteString(pr.String())
		} else {
			b.WriteString(Repr(it))
		}
	}
	b.WriteByte(']')
	return b.String()
}

func (r *ParseResults) asStringList(sep string) []string {
	out := []string{}
	for _, item := range r.toks.items {
		if len(out) > 0 && sep != "" {
			out = append(out, sep)
		}
		if pr, ok := item.(*ParseResults); ok {
			out = append(out, pr.asStringList("")...)
		} else {
			out = append(out, Str(item))
		}
	}
	return out
}

// AsList returns results.as_list(flatten=flatten).
func (r *ParseResults) AsList(flatten bool) []any {
	if flatten {
		return flattenList(r.toks.items)
	}
	out := make([]any, len(r.toks.items))
	for i, res := range r.toks.items {
		if pr, ok := res.(*ParseResults); ok {
			out[i] = pr.AsList(false)
		} else {
			out[i] = res
		}
	}
	return out
}

// AsDict returns results.as_dict().
func (r *ParseResults) AsDict() *PyDict {
	var toItem func(obj any) any
	toItem = func(obj any) any {
		if pr, ok := obj.(*ParseResults); ok {
			if pr.HasKeys() || pr.isDictContext {
				return pr.AsDict()
			}
			out := make([]any, len(pr.toks.items))
			for i, v := range pr.toks.items {
				out[i] = toItem(v)
			}
			return out
		}
		return obj
	}
	d := NewPyDict()
	for _, kv := range r.Items() {
		d.Set(kv.Key, toItem(kv.Value))
	}
	return d
}

// Copy returns results.copy().
func (r *ParseResults) Copy() *ParseResults {
	ret := &ParseResults{
		toks:     &anyList{items: append([]any{}, r.toks.items...)},
		tokdict:  r.tokdict.shallowCopy(),
		parent:   r.parent,
		allNames: map[string]bool{},
		name:     r.name,
	}
	for k := range r.allNames {
		ret.allNames[k] = true
	}
	return ret
}

// pyCopyCopy is copy.copy(results) (via __getnewargs__/__getstate__/__setstate__): the new
// object shares the token list and gets a shallow copy of the names dict.
func (r *ParseResults) pyCopyCopy() *ParseResults {
	ret := &ParseResults{
		toks:          r.toks,
		tokdict:       r.tokdict.shallowCopy(),
		allNames:      map[string]bool{},
		name:          r.name,
		isDictContext: r.isDictContext,
	}
	for k := range r.allNames {
		ret.allNames[k] = true
	}
	return ret
}

// DeepCopy returns results.deepcopy().
func (r *ParseResults) DeepCopy() *ParseResults {
	ret := r.Copy()
	type memoEnt struct {
		key any
		val any
	}
	var memo []memoEnt
	for i, obj := range r.toks.items {
		var nv any
		switch o := obj.(type) {
		case *ParseResults:
			nv = o.DeepCopy()
		case string:
			continue
		case *PyDict:
			d := NewPyDict()
			for _, k := range o.keys {
				v := o.vals[k]
				if pr, ok := v.(*ParseResults); ok {
					d.Set(k, pr.DeepCopy())
				} else {
					d.Set(k, v)
				}
			}
			nv = d
		case []any:
			nv = deepSeq(o)
		case prList:
			nv = prList(deepSeq(o))
		case PyTuple:
			nv = PyTuple(deepSeq(o))
		default:
			continue
		}
		ret.toks.items[i] = nv
		memo = append(memo, memoEnt{obj, nv})
	}
	nd := newTokDict()
	for _, name := range r.tokdict.keys {
		occ := r.tokdict.m[name].items
		n := make([]offsetEntry, len(occ))
		for i, e := range occ {
			v := e.result
			for k := len(memo) - 1; k >= 0; k-- {
				if sameObject(memo[k].key, v) {
					v = memo[k].val
					break
				}
			}
			n[i] = offsetEntry{v, e.offset}
		}
		nd.set(name, &occList{items: n})
	}
	ret.tokdict = nd
	return ret
}

func deepSeq(s []any) []any {
	out := make([]any, len(s))
	for i, v := range s {
		if pr, ok := v.(*ParseResults); ok {
			out[i] = pr.DeepCopy()
		} else {
			out[i] = v
		}
	}
	return out
}

// sameObject approximates Python's `is` for values stored in results.
func sameObject(a, b any) bool {
	switch x := a.(type) {
	case *ParseResults:
		y, ok := b.(*ParseResults)
		return ok && x == y
	case *PyDict:
		y, ok := b.(*PyDict)
		return ok && x == y
	case []any:
		y, ok := b.([]any)
		return ok && len(x) == len(y) && (len(x) == 0 || &x[0] == &y[0])
	case prList:
		y, ok := b.(prList)
		return ok && len(x) == len(y) && (len(x) == 0 || &x[0] == &y[0])
	case PyTuple:
		y, ok := b.(PyTuple)
		return ok && len(x) == len(y) && (len(x) == 0 || &x[0] == &y[0])
	}
	return false
}

// GetName returns results.get_name() (nil for None).
func (r *ParseResults) GetName() *string {
	if r.name != nil && *r.name != "" {
		return strp(*r.name)
	}
	if r.parent != nil && r.parent.Truth() {
		par := r.parent
		for _, k := range par.tokdict.keys {
			for _, v := range par.tokdict.m[k].items {
				if pr, ok := v.result.(*ParseResults); ok && pr == r {
					return strp(k)
				}
			}
		}
		return nil
	}
	if len(r.toks.items) == 1 && r.tokdict.length() == 1 {
		k := r.tokdict.keys[0]
		if r.tokdict.m[k].items[0].offset <= 0 {
			return strp(k)
		}
	}
	return nil
}

// Dump returns results.dump(indent, full, include_list).
func (r *ParseResults) Dump(indent string, full, includeList bool) string {
	return r.dump(indent, full, includeList, 0)
}

func (r *ParseResults) dump(indent string, full, includeList bool, depth int) string {
	var out []string
	const NL = "\n"
	if includeList {
		out = append(out, indent+Str(r.AsList(false)))
	} else {
		out = append(out, "")
	}
	if !full {
		return strings.Join(out, "")
	}
	if r.HasKeys() {
		items := r.Items()
		sort.SliceStable(items, func(i, j int) bool { return items[i].Key < items[j].Key })
		for _, kv := range items {
			if len(out) > 0 {
				out = append(out, NL)
			}
			out = append(out, indent+strings.Repeat("  ", depth)+"- "+kv.Key+": ")
			v, ok := kv.Value.(*ParseResults)
			if !ok {
				out = append(out, Repr(kv.Value))
				continue
			}
			if !v.Truth() {
				out = append(out, v.String())
				continue
			}
			out = append(out, v.dump(indent, full, includeList, depth+1))
		}
	}
	anyPR := false
	for _, vv := range r.toks.items {
		if _, ok := vv.(*ParseResults); ok {
			anyPR = true
			break
		}
	}
	if !anyPR {
		return strings.Join(out, "")
	}
	incr := "  "
	for i, vv := range r.toks.items {
		if pr, ok := vv.(*ParseResults); ok {
			vvDump := pr.dump(indent, full, includeList, depth+1)
			out = append(out, NL+indent+strings.Repeat(incr, depth)+"["+itoa(i)+"]:"+NL+indent+strings.Repeat(incr, depth+1)+vvDump)
		} else {
			out = append(out, NL+indent+strings.Repeat(incr, depth)+"["+itoa(i)+"]:"+NL+indent+strings.Repeat(incr, depth+1)+Str(vv))
		}
	}
	return strings.Join(out, "")
}

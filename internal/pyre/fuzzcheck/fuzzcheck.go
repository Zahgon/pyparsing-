// Port of CPython 3.12's re module (Lib/re/_parser.py, _compiler.py, Modules/_sre/sre.c); PSF License Version 2, Copyright (c) 2001-2024 Python Software Foundation.

// Package fuzzcheck replays one vector recorded by regex-fuzz/record.py against pyre and
// reports the differences from CPython 3.12.
package fuzzcheck

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/pyparsing/pyparsing-go/internal/pyre"
)

// Vector is one recorded test vector.
type Vector = vector

// Parse decodes one JSON line of record.py output.
func Parse(line []byte) (*Vector, error) {
	var v vector
	if err := json.Unmarshal(line, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// Check replays v; it returns nil when pyre agrees with CPython on everything recorded.
func Check(v *Vector) []string { return check(v) }

type op struct {
	Op       string          `json:"op"`
	T        int             `json:"t"`
	Pos      int             `json:"pos"`
	EndPos   *int            `json:"endpos"`
	MaxSplit int             `json:"maxsplit"`
	Repl     string          `json:"repl"`
	Count    int             `json:"count"`
	Res      json.RawMessage `json:"res"`
}

type vector struct {
	ID      string          `json:"id"`
	Pattern string          `json:"pattern"`
	Flags   int             `json:"flags"`
	Texts   []string        `json:"texts"`
	Ops     []op            `json:"ops"`
	Compile json.RawMessage `json:"compile"`
	Timeout bool            `json:"timeout"`
}

func errValue(err error) any {
	var re *pyre.Error
	var pe *pyre.PyError
	switch {
	case errors.As(err, &re):
		return map[string]any{"err": "error", "msg": re.Message()}
	case errors.As(err, &pe):
		return map[string]any{"err": pe.Class, "msg": pe.Msg}
	}
	return map[string]any{"err": "go", "msg": err.Error()}
}

func spans(m *pyre.Match, groups int) []any {
	out := make([]any, groups+1)
	for g := 0; g <= groups; g++ {
		out[g] = []any{m.Start(g), m.End(g)}
	}
	return out
}

func strOrNil(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}

func encMatch(m *pyre.Match, groups int) any {
	if m == nil {
		return nil
	}
	var li, lg any
	if m.LastIndex() >= 0 {
		li = m.LastIndex()
	}
	if name, ok := m.LastGroup(); ok {
		lg = name
	}
	gs := []any{}
	for _, g := range m.Groups() {
		gs = append(gs, strOrNil(g))
	}
	gd := []any{}
	for _, ng := range m.GroupDict() {
		gd = append(gd, []any{ng.Name, strOrNil(ng.Value)})
	}
	return map[string]any{"spans": spans(m, groups), "li": li, "lg": lg, "groups": gs, "gd": gd,
		"repr": m.Repr(), "pos": m.Pos(), "endpos": m.EndPos()}
}

func runOp(p *pyre.Pattern, texts [][]rune, o op) any {
	t := texts[o.T]
	end := len(t)
	if o.EndPos != nil {
		end = *o.EndPos
	}
	switch o.Op {
	case "match":
		return encMatch(p.MatchRange(t, o.Pos, end), p.Groups())
	case "search":
		return encMatch(p.SearchRange(t, o.Pos, end), p.Groups())
	case "fullmatch":
		return encMatch(p.FullMatch(t, o.Pos, end), p.Groups())
	case "finditer":
		out := []any{}
		for _, m := range p.FindIter(t, 0, len(t)) {
			out = append(out, spans(m, p.Groups()))
		}
		return out
	case "split":
		out := []any{}
		for _, s := range p.Split(t, o.MaxSplit) {
			out = append(out, strOrNil(s))
		}
		return out
	case "sub":
		s, n, err := p.Subn(o.Repl, t, o.Count)
		if err != nil {
			return errValue(err)
		}
		return map[string]any{"ok": []any{s, n}}
	case "expand":
		m := p.Search(t, 0)
		if m == nil {
			return nil
		}
		s, err := m.Expand(o.Repl)
		if err != nil {
			return errValue(err)
		}
		return map[string]any{"ok": s}
	}
	return "unknown op " + o.Op
}

func norm(v any) any {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		panic(err)
	}
	return out
}

func decode(raw json.RawMessage) any {
	var out any
	if len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		panic(err)
	}
	return out
}

func show(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// check replays one vector and returns its mismatch descriptions.
func check(v *vector) []string {
	var diffs []string
	p, err := pyre.Compile(v.Pattern, v.Flags)
	var got any
	if err != nil {
		got = errValue(err)
	} else {
		code := []any{}
		for _, c := range p.Code() {
			code = append(code, c)
		}
		gi := []any{}
		for _, g := range p.GroupIndex() {
			gi = append(gi, []any{g.Name, g.Index})
		}
		got = map[string]any{"groups": p.Groups(), "groupindex": gi, "flags": p.Flags(), "code": code}
	}
	want := decode(v.Compile)
	if g := norm(got); !reflect.DeepEqual(g, want) {
		diffs = append(diffs, fmt.Sprintf("compile:\n   got  %s\n   want %s", show(g), show(want)))
		return diffs
	}
	if err != nil {
		return nil
	}
	texts := make([][]rune, len(v.Texts))
	for i, s := range v.Texts {
		texts[i] = []rune(s)
	}
	for _, o := range v.Ops {
		g := norm(runOp(p, texts, o))
		w := decode(o.Res)
		if !reflect.DeepEqual(g, w) {
			diffs = append(diffs, fmt.Sprintf("%s t=%d %+q pos=%d endpos=%v repl=%+q count=%d:\n   got  %s\n   want %s",
				o.Op, o.T, v.Texts[o.T], o.Pos, show(o.EndPos), o.Repl, o.Count, show(g), show(w)))
		}
	}
	return diffs
}

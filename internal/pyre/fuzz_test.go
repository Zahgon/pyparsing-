// Port of CPython 3.12's re module (Lib/re/_parser.py, _compiler.py, Modules/_sre/sre.c); PSF License Version 2, Copyright (c) 2001-2024 Python Software Foundation.

package pyre_test

import (
	"bufio"
	"os"
	"testing"

	"github.com/pyparsing/pyparsing-go/internal/pyre/fuzzcheck"
)

// TestRecordedVectors replays a sample of the differential vectors (regex-fuzz) recorded from
// CPython 3.12: compile result and compiled code, match/search/fullmatch spans, groups,
// lastindex, repr, finditer, split, sub and expand.
func TestRecordedVectors(t *testing.T) {
	f, err := os.Open("testdata/vectors.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	n := 0
	for sc.Scan() {
		v, err := fuzzcheck.Parse(sc.Bytes())
		if err != nil {
			t.Fatal(err)
		}
		n++
		for _, d := range fuzzcheck.Check(v) {
			t.Errorf("%s %+q flags=%d: %s", v.ID, v.Pattern, v.Flags, d)
		}
	}
	if n < 1000 {
		t.Fatalf("only %d vectors", n)
	}
}

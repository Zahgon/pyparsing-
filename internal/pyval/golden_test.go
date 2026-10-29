// Differential tests against vectors recorded from CPython 3.12 by gen/gen_vectors.py
// (PSF License Version 2; Copyright (c) 2001-2024 Python Software Foundation).

package pyval

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
)

type vector struct {
	K   string            `json:"k"`
	In  []json.RawMessage `json:"in"`
	Out json.RawMessage   `json:"out"`
	Err *string           `json:"err"`
}

func (v *vector) str(i int) string {
	var s string
	if err := json.Unmarshal(v.In[i], &s); err != nil {
		panic(err)
	}
	return s
}

func (v *vector) int(i int) int {
	var n int
	if err := json.Unmarshal(v.In[i], &n); err != nil {
		panic(err)
	}
	return n
}

func (v *vector) bool(i int) bool {
	var b bool
	if err := json.Unmarshal(v.In[i], &b); err != nil {
		panic(err)
	}
	return b
}

func (v *vector) float(i int) float64 { return bitsFloat(v.str(i)) }

func bitsFloat(h string) float64 {
	b, err := hex.DecodeString(h)
	if err != nil || len(b) != 8 {
		panic("bad float bits " + h)
	}
	var u uint64
	for _, c := range b {
		u = u<<8 | uint64(c)
	}
	return math.Float64frombits(u)
}

func floatBits(f float64) string {
	if math.IsNaN(f) {
		return "nan"
	}
	return fmt.Sprintf("%016x", math.Float64bits(f))
}

func errText(err error) string {
	if e, ok := err.(*Error); ok {
		return e.Class + ": " + e.Msg
	}
	return "go: " + err.Error()
}

// expect compares an (output, error) pair with the recorded result.
func (v *vector) expect(got any, err error) (bool, string) {
	if v.Err != nil {
		if err == nil {
			return false, fmt.Sprintf("want error %q, got %#v", *v.Err, got)
		}
		if errText(err) != *v.Err {
			return false, fmt.Sprintf("want error %q, got %q", *v.Err, errText(err))
		}
		return true, ""
	}
	if err != nil {
		return false, fmt.Sprintf("want %s, got error %q", v.Out, errText(err))
	}
	want := reflect.New(reflect.TypeOf(got)).Interface()
	if e := json.Unmarshal(v.Out, want); e != nil {
		return false, fmt.Sprintf("cannot decode want %s as %T", v.Out, got)
	}
	w := reflect.ValueOf(want).Elem().Interface()
	if !reflect.DeepEqual(w, got) {
		return false, fmt.Sprintf("want %#v, got %#v", w, got)
	}
	return true, ""
}

func runVector(v *vector) (bool, string) {
	switch v.K {
	case "floatrepr":
		return v.expect(FloatRepr(v.float(0)), nil)
	case "floatparse":
		f, err := ParseFloat(v.str(0))
		return v.expect(floatBits(f), err)
	case "intparse":
		z, err := ParseInt(v.str(0), v.int(1))
		if err != nil {
			return v.expect("", err)
		}
		s, err := IntReprChecked(z)
		if err != nil {
			s = "REPR-ERR " + err.(*Error).Msg
		}
		return v.expect(s, nil)
	case "strrepr":
		return v.expect(StrRepr(v.str(0)), nil)
	case "strmeth":
		return runStrMeth(v)
	case "strptime":
		r, err := Strptime(v.str(0), v.str(1))
		if err != nil {
			return v.expect([]any{}, err)
		}
		tzn := any(nil)
		if r.TZName != nil {
			tzn = *r.TZName
		}
		return v.expect([]any{r.PyRepr(), r.PyStr(), float64(r.Weekday), float64(r.Julian), float64(r.TZ), tzn}, nil)
	case "floatops":
		return runFloatOps(v)
	case "asdatetime":
		return runAsDatetime(v)
	}
	return false, "unknown kind " + v.K
}

func runStrMeth(v *vector) (bool, string) {
	op := v.str(0)
	s := v.str(1)
	switch op {
	case "expandtabs":
		return v.expect(ExpandTabs(s, v.int(2)), nil)
	case "strip":
		return v.expect(StripSpace(s), nil)
	case "lstrip":
		return v.expect(LStripSpace(s), nil)
	case "rstrip":
		return v.expect(RStripSpace(s), nil)
	case "stripc":
		return v.expect(Strip(s, v.str(2)), nil)
	case "split":
		return v.expect(Split(s), nil)
	case "splitn":
		return v.expect(SplitSpaceN(s, v.int(2)), nil)
	case "splitlines":
		return v.expect(SplitLines(s, v.bool(2)), nil)
	case "title":
		return v.expect(Title(s), nil)
	case "upper":
		return v.expect(Upper(s), nil)
	case "lower":
		return v.expect(Lower(s), nil)
	case "casefold":
		return v.expect(CaseFold(s), nil)
	case "isspace":
		return v.expect(IsSpace(s), nil)
	case "find":
		return v.expect(Find(s, v.str(2), v.int(3), v.int(4)), nil)
	case "rfind":
		return v.expect(RFind(s, v.str(2), v.int(3), v.int(4)), nil)
	case "count":
		return v.expect(Count(s, v.str(2), v.int(3), v.int(4)), nil)
	case "slice":
		return v.expect(Slice(s, v.int(2), v.int(3)), nil)
	}
	return false, "unknown strmeth op " + op
}

func runFloatOps(v *vector) (bool, string) {
	op := v.str(0)
	x := v.float(1)
	switch op {
	case "mod":
		r, err := FloatMod(x, v.float(2))
		return v.expect(floatBits(r), err)
	case "floordiv":
		r, err := FloatFloorDiv(x, v.float(2))
		return v.expect(floatBits(r), err)
	case "divmod":
		d, m, err := FloatDivMod(x, v.float(2))
		return v.expect([]string{floatBits(d), floatBits(m)}, err)
	case "round":
		z, err := Round(x)
		if err != nil {
			return v.expect("", err)
		}
		return v.expect(z.String(), nil)
	case "roundn":
		r, err := RoundN(x, v.int(2))
		return v.expect(floatBits(r), err)
	case "int":
		z, err := FloatToInt(x)
		if err != nil {
			return v.expect("", err)
		}
		return v.expect(z.String(), nil)
	}
	return false, "unknown floatops op " + op
}

// runAsDatetime mirrors pyparsing_common.as_datetime:
// datetime(y, mo, d, h, mi, int(sec)) + timedelta(microseconds=round((sec % 1) * 1_000_000)).
func runAsDatetime(v *vector) (bool, string) {
	var args [5]*big.Int
	for i := range args {
		z, ok := new(big.Int).SetString(v.str(i), 10)
		if !ok {
			return false, "bad int"
		}
		args[i] = z
	}
	sec := v.float(5)
	isec, err := FloatToInt(sec)
	if err != nil {
		return v.expect("", err)
	}
	dt, err := NewDateTimeBig(args[0], args[1], args[2], args[3], args[4], isec, new(big.Int))
	if err != nil {
		return v.expect("", err)
	}
	frac, err := FloatMod(sec, 1)
	if err != nil {
		return v.expect("", err)
	}
	us, err := Round(frac * 1_000_000)
	if err != nil {
		return v.expect("", err)
	}
	dt, err = AddMicroseconds(dt, us)
	if err != nil {
		return v.expect("", err)
	}
	return v.expect(dt.PyRepr(), nil)
}

func runGoldenFile(t *testing.T, path string) {
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<26)
	n, bad := 0, 0
	for sc.Scan() {
		var v vector
		if err := json.Unmarshal(sc.Bytes(), &v); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		n++
		ok, msg := runVector(&v)
		if !ok {
			bad++
			if bad <= 15 {
				in, _ := json.Marshal(v.In)
				t.Errorf("%s line %d: in=%s: %s", filepath.Base(path), n, truncate(string(in), 300), truncate(msg, 400))
			}
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if n == 0 {
		t.Errorf("%s: no vectors", path)
	}
	t.Logf("%s: %d vectors, %d mismatches", filepath.Base(path), n, bad)
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n] + "..." + strconv.Itoa(len(s)-n) + " more"
	}
	return s
}

// TestGolden runs the small committed vector files.
func TestGolden(t *testing.T) {
	files, _ := filepath.Glob("testdata/*.jsonl")
	if len(files) == 0 {
		t.Fatal("no testdata/*.jsonl")
	}
	for _, p := range files {
		t.Run(filepath.Base(p), func(t *testing.T) { runGoldenFile(t, p) })
	}
}

// TestGoldenLarge runs the large vector files in $PYVAL_GOLDEN_DIR (generated with
// gen/gen_vectors.py; not committed).
func TestGoldenLarge(t *testing.T) {
	dir := os.Getenv("PYVAL_GOLDEN_DIR")
	if dir == "" {
		t.Skip("PYVAL_GOLDEN_DIR not set")
	}
	files, _ := filepath.Glob(filepath.Join(dir, "*.jsonl"))
	for _, p := range files {
		t.Run(filepath.Base(p), func(t *testing.T) { runGoldenFile(t, p) })
	}
}

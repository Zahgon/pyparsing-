// Port of CPython 3.12's re module (Lib/re/_parser.py, _compiler.py, Modules/_sre/sre.c); PSF License Version 2, Copyright (c) 2001-2024 Python Software Foundation.

package pyre

import (
	"strings"
	"testing"
)

func BenchmarkWordMatch(b *testing.B) {
	p := MustCompile(`[A-Za-z_][A-Za-z0-9_]*`, 0)
	text := []rune(strings.Repeat("hello_world 123 ", 100))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Match(text, (i%100)*16)
	}
}

func BenchmarkQuotedString(b *testing.B) {
	p := MustCompile(`"(?:[^"\n\r\\]|(?:"")|(?:\\(?:[^x]|x[0-9a-fA-F]+)))*"`, 0)
	text := []rune(`"` + strings.Repeat(`abc \" def `, 20) + `"`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Match(text, 0)
	}
}

func BenchmarkSearchFail(b *testing.B) {
	p := MustCompile(`(a|b)+c`, 0)
	text := []rune(strings.Repeat("x", 1000))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Search(text, 0)
	}
}

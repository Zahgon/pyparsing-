// Port of CPython 3.12's re module (Lib/re/_parser.py, _compiler.py, Modules/_sre/sre.c); PSF License Version 2, Copyright (c) 2001-2024 Python Software Foundation.

// Command replay compares internal/pyre with results recorded from CPython 3.12 by
// regex-fuzz/record.py.
//
//	go run ./internal/pyre/replay [-v] [-max N] [-timeout D] FILE.jsonl[.gz]...
//
// It exits non-zero when any vector differs. A vector taking longer than -timeout is reported
// as a hang (CPython finished it within record.py's timeout).
package main

import (
	"bufio"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/pyparsing/pyparsing-go/internal/pyre/fuzzcheck"
)

func main() {
	verbose := flag.Bool("v", false, "print every mismatch")
	maxShow := flag.Int("max", 20, "mismatching vectors to print")
	timeout := flag.Duration("timeout", 20*time.Second, "time limit per vector")
	flag.Parse()
	total, skipped, bad, ops := 0, 0, 0, 0
	for _, path := range flag.Args() {
		f, err := os.Open(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		var r io.Reader = f
		if strings.HasSuffix(path, ".gz") {
			zr, err := gzip.NewReader(f)
			if err != nil {
				fmt.Fprintln(os.Stderr, path, err)
				os.Exit(2)
			}
			r = zr
		}
		sc := bufio.NewScanner(r)
		sc.Buffer(make([]byte, 1<<20), 1<<28)
		for sc.Scan() {
			v, err := fuzzcheck.Parse(sc.Bytes())
			if err != nil {
				fmt.Fprintln(os.Stderr, path, err)
				os.Exit(2)
			}
			if v.Timeout {
				skipped++
				continue
			}
			total++
			ops += len(v.Ops)
			done := make(chan []string, 1)
			go func() {
				defer func() {
					if r := recover(); r != nil {
						done <- []string{fmt.Sprintf("panic: %v", r)}
					}
				}()
				done <- fuzzcheck.Check(v)
			}()
			var diffs []string
			select {
			case diffs = <-done:
			case <-time.After(*timeout):
				fmt.Printf("HANG %s %+q flags=%d\n", v.ID, v.Pattern, v.Flags)
				os.Exit(1)
			}
			if len(diffs) > 0 {
				bad++
				if *verbose || bad <= *maxShow {
					fmt.Printf("MISMATCH %s pattern=%+q flags=%d\n", v.ID, v.Pattern, v.Flags)
					for i, d := range diffs {
						if i >= 3 && !*verbose {
							fmt.Printf("  ... %d more\n", len(diffs)-i)
							break
						}
						fmt.Println("  " + d)
					}
				}
			}
		}
		f.Close()
	}
	fmt.Printf("vectors %d (ops %d), skipped (timeout) %d, mismatching %d\n", total, ops, skipped, bad)
	if bad > 0 {
		os.Exit(1)
	}
}

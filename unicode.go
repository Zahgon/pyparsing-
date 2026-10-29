// Port of the character sets of pyparsing/unicode.py (pyparsing 3.3.3+). MIT License, Copyright (c) 2003-2025  Paul McGuire; see LICENSE.

package pyparsing

import (
	"sort"

	"github.com/pyparsing/pyparsing-go/internal/pyuni"
)

type uniRange [2]rune

var (
	latin1Ranges   = []uniRange{{32, 126}, {160, 255}}
	latinARanges   = []uniRange{{256, 383}}
	latinBRanges   = []uniRange{{384, 591}}
	greekRanges    = []uniRange{{834, 837}, {880, 887}, {890, 895}, {900, 906}, {908, 908}, {910, 929}, {931, 993}, {1008, 1023}, {7462, 7466}, {7518, 7518}, {7520, 7520}, {7526, 7530}, {7936, 7957}, {7960, 7965}, {7968, 8005}, {8008, 8013}, {8016, 8023}, {8025, 8025}, {8027, 8027}, {8029, 8029}, {8031, 8061}, {8064, 8116}, {8118, 8132}, {8134, 8147}, {8150, 8155}, {8157, 8175}, {8178, 8180}, {8182, 8190}, {8489, 8489}, {10009, 10010}, {43877, 43877}, {65856, 65933}, {65952, 65952}, {119296, 119365}, {128929, 128935}}
	cyrillicRanges = []uniRange{{1024, 1327}, {7296, 7304}, {7467, 7467}, {7544, 7544}, {11744, 11775}, {42560, 42610}, {42612, 42655}, {65070, 65071}}
)

// charsForRanges is unicode_set._chars_for_ranges: the sorted distinct characters of the ranges.
func charsForRanges(lists ...[]uniRange) []rune {
	set := map[rune]bool{}
	for _, l := range lists {
		for _, rr := range l {
			for c := rr[0]; c <= rr[1]; c++ {
				set[c] = true
			}
		}
	}
	out := make([]rune, 0, len(set))
	for c := range set {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func isPyAlpha(r rune) bool { return pyuni.IsAlpha(r) }
func isPyDigit(r rune) bool { return pyuni.IsDigit(r) }
func isPySpace(r rune) bool { return pyuni.IsSpace(r) }

// latin1Identchars is pyparsing_unicode.Latin1.identchars.
func latin1Identchars() string {
	set := runeSet{}
	for _, c := range charsForRanges(latin1Ranges) {
		if pyuni.IsIdentifier(string(c)) {
			set[c] = true
		}
	}
	for _, c := range "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyzªµºÀÁÂÃÄÅÆÇÈÉÊËÌÍÎÏÐÑÒÓÔÕÖØÙÚÛÜÝÞßàáâãäåæçèéêëìíîïðñòóôõöøùúûüýþÿ_" {
		set[c] = true
	}
	return string(set.sorted())
}

// latin1Identbodychars is pyparsing_unicode.Latin1.identbodychars.
func latin1Identbodychars(identchars string) string {
	set := runeSet{}
	for _, c := range charsForRanges(latin1Ranges) {
		if pyuni.IsIdentifier("_" + string(c)) {
			set[c] = true
		}
	}
	for _, c := range identchars {
		set[c] = true
	}
	for _, c := range "0123456789·" {
		set[c] = true
	}
	return string(set.sorted())
}

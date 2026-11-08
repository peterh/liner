package liner

import (
	"github.com/clipperhouse/uax29/v2/graphemes"
	"github.com/mattn/go-runewidth"
)

// glyphs splits s into the sequences of runes a terminal draws as one
// character each, by the extended grapheme cluster rules of UAX #29.
//
// A glyph is often one rune and need not be. A letter followed by a combining
// accent is one, and so are the emoji built out of several runes: a flag is a
// pair of regional indicators, a thumb carries its skin tone in a second rune,
// and a family is four people joined by zero-width joiners.
func glyphs(s []rune) [][]rune {
	var out [][]rune
	for g := graphemes.FromString(string(s)); g.Next(); {
		out = append(out, []rune(g.Value()))
	}
	return out
}

// countGlyphs considers zero-width characters to be zero glyphs wide,
// and members of Chinese, Japanese, and Korean scripts to be 2 glyphs wide.
func countGlyphs(s []rune) int {
	n := 0
	for g := graphemes.FromString(string(s)); g.Next(); {
		glyph := g.Value()

		// speed up the common case
		if len(glyph) == 1 && glyph[0] < 127 {
			n++
			continue
		}

		n += runewidth.StringWidth(glyph)
	}
	return n
}

func countMultiLineGlyphs(s []rune, columns int, start int) int {
	n := start
	for g := graphemes.FromString(string(s)); g.Next(); {
		glyph := g.Value()
		if len(glyph) == 1 && glyph[0] < 127 {
			n++
			continue
		}
		switch runewidth.StringWidth(glyph) {
		case 0:
		case 1:
			n++
		case 2:
			n += 2
			// no room for a 2-glyphs-wide char in the ending
			// so skip a column and display it at the beginning
			if n%columns == 1 {
				n++
			}
		}
	}
	return n
}

func getPrefixGlyphs(s []rune, num int) []rune {
	g := glyphs(s)
	p := 0
	for n := 0; n < num && n < len(g); n++ {
		p += len(g[n])
	}
	return s[:p]
}

func getSuffixGlyphs(s []rune, num int) []rune {
	g := glyphs(s)
	p := len(s)
	for n := 0; n < num && n < len(g); n++ {
		p -= len(g[len(g)-1-n])
	}
	return s[p:]
}

package liner

import (
	"strconv"
	"testing"
)

func accent(in []rune) []rune {
	var out []rune
	for _, r := range in {
		out = append(out, r)
		out = append(out, '\u0301')
	}
	return out
}

type testCase struct {
	s      []rune
	glyphs int
}

var testCases = []testCase{
	{[]rune("query"), 5},
	{[]rune("私"), 2},
	{[]rune("hello『世界』"), 13},
}

func TestCountGlyphs(t *testing.T) {
	for _, testCase := range testCases {
		count := countGlyphs(testCase.s)
		if count != testCase.glyphs {
			t.Errorf("ASCII count incorrect. %d != %d", count, testCase.glyphs)
		}
		count = countGlyphs(accent(testCase.s))
		if count != testCase.glyphs {
			t.Errorf("Accent count incorrect. %d != %d", count, testCase.glyphs)
		}
	}
}

func compare(a, b []rune, name string, t *testing.T) {
	if len(a) != len(b) {
		t.Errorf(`"%s" != "%s" in %s"`, string(a), string(b), name)
		return
	}
	for i := range a {
		if a[i] != b[i] {
			t.Errorf(`"%s" != "%s" in %s"`, string(a), string(b), name)
			return
		}
	}
}

func TestPrefixGlyphs(t *testing.T) {
	for _, testCase := range testCases {
		for i := 0; i <= len(testCase.s); i++ {
			iter := strconv.Itoa(i)
			out := getPrefixGlyphs(testCase.s, i)
			compare(out, testCase.s[:i], "ascii prefix "+iter, t)
			out = getPrefixGlyphs(accent(testCase.s), i)
			compare(out, accent(testCase.s[:i]), "accent prefix "+iter, t)
		}
		out := getPrefixGlyphs(testCase.s, 999)
		compare(out, testCase.s, "ascii prefix overflow", t)
		out = getPrefixGlyphs(accent(testCase.s), 999)
		compare(out, accent(testCase.s), "accent prefix overflow", t)

		out = getPrefixGlyphs(testCase.s, -3)
		if len(out) != 0 {
			t.Error("ascii prefix negative")
		}
		out = getPrefixGlyphs(accent(testCase.s), -3)
		if len(out) != 0 {
			t.Error("accent prefix negative")
		}
	}
}

func TestSuffixGlyphs(t *testing.T) {
	for _, testCase := range testCases {
		for i := 0; i <= len(testCase.s); i++ {
			iter := strconv.Itoa(i)
			out := getSuffixGlyphs(testCase.s, i)
			compare(out, testCase.s[len(testCase.s)-i:], "ascii suffix "+iter, t)
			out = getSuffixGlyphs(accent(testCase.s), i)
			compare(out, accent(testCase.s[len(testCase.s)-i:]), "accent suffix "+iter, t)
		}
		out := getSuffixGlyphs(testCase.s, 999)
		compare(out, testCase.s, "ascii suffix overflow", t)
		out = getSuffixGlyphs(accent(testCase.s), 999)
		compare(out, accent(testCase.s), "accent suffix overflow", t)

		out = getSuffixGlyphs(testCase.s, -3)
		if len(out) != 0 {
			t.Error("ascii suffix negative")
		}
		out = getSuffixGlyphs(accent(testCase.s), -3)
		if len(out) != 0 {
			t.Error("accent suffix negative")
		}
	}
}

// Glyphs a terminal draws as one character out of several runes. An accent is
// the case the tests above already cover; these are the rest of them.
var multiRuneCases = []testCase{
	{[]rune("\U0001F468\u200d\U0001F469\u200d\U0001F467\u200d\U0001F466"), 2}, // family
	{[]rune("\U0001F3F3\ufe0f\u200d\U0001F308"), 2},                      // rainbow flag
	{[]rune("\U0001F44D\U0001F3FD"), 2},                        // thumb with a skin tone
	{[]rune("\U0001F1EF\U0001F1F5"), 2},                        // flag, a pair of regional indicators
}

func TestCountGlyphsMultiRune(t *testing.T) {
	for _, testCase := range multiRuneCases {
		count := countGlyphs(testCase.s)
		if count != testCase.glyphs {
			t.Errorf("%q counted as %d glyphs, occupies %d",
				string(testCase.s), count, testCase.glyphs)
		}
	}
}

// A line of one-glyph-wide characters must not be thought wider than it is,
// or the prompt wraps where it does not need to.
func TestCountMultiLineGlyphsMultiRune(t *testing.T) {
	for _, testCase := range multiRuneCases {
		var line []rune
		for i := 0; i < 40; i++ {
			line = append(line, testCase.s...)
		}
		want := 40 * testCase.glyphs
		if got := countMultiLineGlyphs(line, 80, 0); got != want {
			t.Errorf("40 copies of %q wrapped to %d columns, occupies %d",
				string(testCase.s), got, want)
		}
	}
}

// Moving over one glyph must move over all of it. Stopping partway leaves the
// cursor inside the character, and the runes either side of it get edited
// separately.
func TestGlyphMovementKeepsGlyphsWhole(t *testing.T) {
	for _, testCase := range multiRuneCases {
		out := getPrefixGlyphs(testCase.s, 1)
		if len(out) != len(testCase.s) {
			t.Errorf("prefix of one glyph of %q took %d of its %d runes",
				string(testCase.s), len(out), len(testCase.s))
		}
		out = getSuffixGlyphs(testCase.s, 1)
		if len(out) != len(testCase.s) {
			t.Errorf("suffix of one glyph of %q took %d of its %d runes",
				string(testCase.s), len(out), len(testCase.s))
		}

		// And one glyph of "a<glyph>b" is just the "a".
		padded := append(append([]rune("a"), testCase.s...), 'b')
		if out = getPrefixGlyphs(padded, 1); len(out) != 1 {
			t.Errorf("prefix of one glyph of %q took %d runes, wanted 1",
				string(padded), len(out))
		}
		if out = getSuffixGlyphs(padded, 1); len(out) != 1 {
			t.Errorf("suffix of one glyph of %q took %d runes, wanted 1",
				string(padded), len(out))
		}
	}
}

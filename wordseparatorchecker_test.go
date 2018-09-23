package liner

import (
	"testing"
)

func TestSeparators(t *testing.T) {
	// test spaces
	spaces := []rune{' ', '	'}
	for _, r := range spaces {
		if !SpaceWordSeparatorChecker(r) {
			t.Errorf("'%s' was not recognized by the space separator", string(r))
		}
	}

	// test punctuation character
	puncts := []rune{'(', ')', '{', '}'} // etc
	for _, r := range puncts {
		if !PunctWordSeparatorChecker(r) {
			t.Errorf(`'%s' was not recognized by the "punctuation" separator`, string(r))
		}
	}

	// test combination of them
	combinedChecker := CombineWordSeparatorChecker(
		SpaceWordSeparatorChecker,
		PunctWordSeparatorChecker,
	)
	for _, r := range append(puncts, spaces...) {
		if !combinedChecker(r) {
			t.Errorf("'%s' was not recognized by the combined separator", string(r))
		}
	}

	// test some letters
	justLetters := []rune{'a', 'b', 'c'}
	for _, r := range justLetters {
		if combinedChecker(r) {
			t.Errorf("'%s' was recognized as a space or a punctuation char", string(r))
		}
	}
}

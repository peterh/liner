package liner

import "unicode"

// SpaceWordSeparatorChecker (default) returns true if r is a unicode-space
func SpaceWordSeparatorChecker(r rune) bool {
	return unicode.IsSpace(r)
}

// PunctWordSeparatorChecker returns true if r is a unicode punctuation character
func PunctWordSeparatorChecker(r rune) bool {
	return unicode.IsPunct(r)
}

// CombineWordSeparatorChecker combines checkers
// eg. line.SetWordSeparatorChecker(liner.CombineWordSeparatorChecker(
//       liner.SpaceWordSeparatorChecker,
//       liner.PunctWordSeparatorChecker,
//     ))
func CombineWordSeparatorChecker(checkers ...WordSeparatorChecker) WordSeparatorChecker {
	return func(r rune) bool {
		for _, isSeparator := range checkers {
			if isSeparator(r) {
				return true
			}
		}

		return false
	}
}

package liner

import "unicode"

// isWordSeparator is used by alt-{F, B, D}, ctrl-{W} functions to determinate where to stop
func isWordSeparator(r rune) bool {
	return unicode.IsSpace(r) || unicode.IsPunct(r)
}

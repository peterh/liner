package liner

type (
	// WordController provides interface for word related actions
	WordController interface {
		// an effect for ctrlW
		EraseWordBack(line []rune, pos int) (effect, error)
		// an effect for alt-d
		DeleteNextWord(line []rune, pos int) (effect, error)

		// an effect for alt-b
		WordLeft(line []rune, pos int) (effect, error)
		// an affect for alt-f
		WordRight(line []rune, pos int) (effect, error)
	}

	// DefaultWordController describes default behavior for the word related functions
	DefaultWordController struct {
		wordSeparatorChecker WordSeparatorChecker
	}
)

// NewDefaultWordController returns default word controller with default settings
func NewDefaultWordController() DefaultWordController {
	return DefaultWordController{
		wordSeparatorChecker: SpaceWordSeparatorChecker,
	}
}

// EraseWordBack returns an effect for ctrlW
func (bc DefaultWordController) EraseWordBack(line []rune, originalPos int) (effect, error) {
	pos := originalPos
	if pos == 0 {
		return effect{newPosition: pos, beep: true}, nil
	}

	// Remove word separators to the left
	for {
		if pos == 0 || !bc.isWordSeparator(line[pos-1]) {
			break
		}
		pos--
	}

	// Remove non-word separators to the left
	for {
		if pos == 0 || bc.isWordSeparator(line[pos-1]) {
			break
		}
		pos--
	}

	return effect{
		toDelete: &deleteEffect{
			from: pos,
			to:   originalPos,
		},
		newPosition: pos,
	}, nil
}

// WordLeft returns an effect for alt-b
func (bc DefaultWordController) WordLeft(line []rune, pos int) (effect, error) {
	if pos == 0 {
		return effect{newPosition: pos, beep: true}, nil
	}

	var atWordSeparator, wordSeparatorLeft, leftKnown bool
	for {
		pos--
		if pos == 0 {
			break
		}
		if leftKnown {
			atWordSeparator = wordSeparatorLeft
		} else {
			atWordSeparator = bc.isWordSeparator(line[pos])
		}

		wordSeparatorLeft = bc.isWordSeparator(line[pos-1])
		leftKnown = true

		if !atWordSeparator && wordSeparatorLeft {
			break
		}
	}

	return effect{newPosition: pos}, nil
}

// WordRight returns an effect for alt-f
func (bc DefaultWordController) WordRight(line []rune, pos int) (effect, error) {
	if pos >= len(line) {
		return effect{beep: true, newPosition: pos}, nil
	}

	var atWordSeparator, wordSeparatorLeft, hereKnown bool
	for {
		pos++
		if pos == len(line) {
			break
		}
		if hereKnown {
			wordSeparatorLeft = atWordSeparator
		} else {
			wordSeparatorLeft = bc.isWordSeparator(line[pos-1])
		}

		atWordSeparator = bc.isWordSeparator(line[pos])
		hereKnown = true

		if atWordSeparator && !wordSeparatorLeft {
			break
		}
	}

	return effect{newPosition: pos}, nil
}

// DeleteNextWord return an effect for alt-d
func (bc DefaultWordController) DeleteNextWord(line []rune, originalPos int) (effect, error) {
	if originalPos == len(line) {
		return effect{beep: true}, nil

	}

	virtualPosition := originalPos
	// Remove word separators to the right
	for {
		if virtualPosition == len(line) || !bc.isWordSeparator(line[virtualPosition]) {
			break
		}
		virtualPosition++
	}

	// Remove non-word separators to the right
	for {
		if virtualPosition == len(line) || bc.isWordSeparator(line[virtualPosition]) {
			break
		}
		virtualPosition++
	}

	return effect{
		toDelete: &deleteEffect{
			from: originalPos,
			to:   virtualPosition,
		},
		newPosition: originalPos,
	}, nil
}

// SetWordSeparatorChecker sets word separator strategy
func (bc *DefaultWordController) SetWordSeparatorChecker(ws WordSeparatorChecker) {
	bc.wordSeparatorChecker = ws
}

func (bc *DefaultWordController) isWordSeparator(r rune) bool {
	return bc.wordSeparatorChecker(r)
}

var _ WordController = (*DefaultWordController)(nil)

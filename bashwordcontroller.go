package liner

type (
	// BashWordController describes bash-like behavior for the word related functions
	BashWordController struct {
		DefaultWordController
	}
)

// NewBashWordController returns default word controller with default settings
func NewBashWordController() BashWordController {
	return BashWordController{
		DefaultWordController: DefaultWordController{
			wordSeparatorChecker: CombineWordSeparatorChecker(
				SpaceWordSeparatorChecker,
				PunctWordSeparatorChecker,
			),
		},
	}
}

// EraseWordBack returns an effect for ctrlW
func (bc BashWordController) EraseWordBack(line []rune, originalPos int) (effect, error) {
	pos := originalPos
	if pos == 0 {
		return effect{newPosition: pos, beep: true}, nil
	}

	// Remove word separators to the left
	for {
		if pos == 0 || !bc.isSpace(line[pos-1]) {
			break
		}
		pos--
	}

	// Remove non-word separators to the left
	for {
		if pos == 0 || bc.isSpace(line[pos-1]) {
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

func (bc BashWordController) isSpace(r rune) bool {
	return SpaceWordSeparatorChecker(r)
}

var _ WordController = (*BashWordController)(nil)

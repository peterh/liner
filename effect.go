package liner

type (
	deleteEffect struct {
		from, to int
	}

	effect struct {
		beep        bool
		toDelete    *deleteEffect
		newPosition int // position after modification
	}
)

// +build !windows

package liner

import (
	"fmt"
)

func (s *State) cursorPos(x int) {
	fmt.Printf("\x1b[%dG", x+1)
}

func (s *State) eraseLine() {
	fmt.Print("\x1b[0K")
}

func (s *State) eraseScreen() {
	fmt.Print("\x1b[H\x1b[2J")
}

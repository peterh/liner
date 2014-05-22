// +build !windows,!linux,!darwin,!openbsd,!freebsd,!netbsd

package liner

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
)

// State represents an open terminal
type State struct {
	commonState
	r *bufio.Reader
}

// Prompt displays p, and then waits for user input. Prompt does not support
// line editing on this operating system.
func (s *State) Prompt(p string) (string, error) {
	fmt.Print(p)
	linebuf, _, err := s.r.ReadLine()
	if err != nil {
		return "", err
	}
	return string(bytes.TrimSpace(linebuf)), nil
}

// NewLiner initializes a new *State
//
// Note that this operating system uses a fallback mode without line
// editing. Patches welcome.
func NewLiner() *State {
	var s State
	s.r = bufio.NewReader(os.Stdin)
	return &s
}

// Close returns the terminal to its previous mode
func (s *State) Close() error {
	return nil
}

// TerminalSupported returns false because line editing is not
// supported on this platform.
func TerminalSupported() bool {
	return false
}

type noopMode struct{}

func (n noopMode) ApplyMode() error {
	return nil
}

// TerminalMode returns a noop InputModeSetter on this platform.
func TerminalMode() (ModeApplier, error) {
	return noopMode{}, nil
}

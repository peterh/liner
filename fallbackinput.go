// +build !windows,!linux,!darwin,!openbsd,!freebsd,!netbsd

package liner

import (
	"errors"
)

// State represents an open terminal
type State struct {
	commonState
}

// Prompt displays p, and then waits for user input. Prompt does not support
// line editing on this operating system.
func (s *State) Prompt(p string) (string, error) {
	return s.promptUnsupported(p)
}

// PasswordPrompt is not supported in this OS.
func (s *State) PasswordPrompt(p string) (string, error) {
	return "", errors.New("liner: function not supported in this terminal")
}

func (s *State) init() {
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

func (n noopMode) ApplyMode(uintptr) error {
	return nil
}

// TerminalMode returns a noop InputModeSetter on this platform.
func (s *State) TerminalMode() (ModeApplier, error) {
	return noopMode{}, nil
}

const cursorColumn = true

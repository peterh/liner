package liner

import (
	"golang.org/x/sys/unix"
)

func (mode *termios) ApplyMode() error {
	return unix.IoctlSetTermio(unix.Stdin, setTermios, (*unix.Termio)(mode))
}

// TerminalMode returns the current terminal input mode as an InputModeSetter.
//
// This function is provided for convenience, and should
// not be necessary for most users of liner.
func TerminalMode() (ModeApplier, error) {
	return getMode(unix.Stdin)
}

func getMode(handle int) (*termios, error) {
	tos, err := unix.IoctlGetTermio(handle, getTermios)
	return (*termios)(tos), err
}

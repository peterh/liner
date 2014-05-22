// +build linux darwin freebsd openbsd netbsd

package liner

import (
	"syscall"
	"unsafe"
)

func (mode *termios) ApplyMode() error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(syscall.Stdin), setTermios, uintptr(unsafe.Pointer(mode)))

	if errno != 0 {
		return errno
	}
	return nil
}

// TerminalMode returns the current terminal input mode as an InputModeSetter.
//
// This function is provided for convenience, and should
// not be necessary for most users of liner.
func TerminalMode() (ModeApplier, error) {
	var mode termios
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(syscall.Stdin), getTermios, uintptr(unsafe.Pointer(&mode)))

	if errno != 0 {
		return nil, errno
	}
	return &mode, nil
}

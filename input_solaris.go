// +build solaris

package liner

import (
	"syscall"

	"golang.org/x/sys/unix"
)

const (
	getTermios = unix.TCGETS
	setTermios = unix.TCSETS
)

const (
	icrnl  = syscall.ICRNL
	inpck  = syscall.INPCK
	istrip = syscall.ISTRIP
	ixon   = syscall.IXON
	opost  = syscall.OPOST
	cs8    = syscall.CS8
	isig   = syscall.ISIG
	icanon = syscall.ICANON
	iexten = syscall.IEXTEN
)

type termios struct {
	syscall.Termios
}

const cursorColumn = false

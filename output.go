// +build !windows

package liner

import (
	"fmt"
	"syscall"
	"unsafe"
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

type winSize struct {
	row, col       uint16
	xpixel, ypixel uint16
}

func (s *State) getColumns() {
	var ws winSize
	ok, _, _ := syscall.Syscall(syscall.SYS_IOCTL, uintptr(syscall.Stdout),
		syscall.TIOCGWINSZ, uintptr(unsafe.Pointer(&ws)))
	if ok < 0 {
		s.columns = 80
	}
	s.columns = int(ws.col)
}

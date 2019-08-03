// +build linux darwin openbsd freebsd netbsd

package liner

import (
	"os"
	"strings"
	"syscall"
	"unsafe"
)

func (s *State) cursorPos(x int) {
	if s.useCHA {
		// 'G' is "Cursor Character Absolute (CHA)"
		s.printf("\x1b[%dG", x+1)
	} else {
		// 'C' is "Cursor Forward (CUF)"
		s.print("\r")
		if x > 0 {
			s.printf("\x1b[%dC", x)
		}
	}
}

func (s *State) eraseLine() {
	s.print("\x1b[0K")
}

func (s *State) eraseScreen() {
	s.print("\x1b[H\x1b[2J")
}

func (s *State) moveUp(lines int) {
	s.printf("\x1b[%dA", lines)
}

func (s *State) moveDown(lines int) {
	s.printf("\x1b[%dB", lines)
}

func (s *State) emitNewLine() {
	s.print("\n")
}

type winSize struct {
	row, col       uint16
	xpixel, ypixel uint16
}

func (s *State) getColumns() bool {
	var ws winSize
	ok, _, _ := syscall.Syscall(syscall.SYS_IOCTL, uintptr(syscall.Stdout),
		syscall.TIOCGWINSZ, uintptr(unsafe.Pointer(&ws)))
	if int(ok) < 0 {
		return false
	}
	s.columns = int(ws.col)
	return true
}

func (s *State) checkOutput() {
	// xterm is known to support CHA
	if strings.Contains(strings.ToLower(os.Getenv("TERM")), "xterm") {
		s.useCHA = true
		return
	}

	// The test for functional ANSI CHA is unreliable (eg the Windows
	// telnet command does not support reading the cursor position with
	// an ANSI DSR request, despite setting TERM=ansi)

	// Assume CHA isn't supported (which should be safe, although it
	// does result in occasional visible cursor jitter)
	s.useCHA = false
}

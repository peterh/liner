// +build linux darwin openbsd freebsd netbsd

package liner

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

func (s *State) cursorPos(x int) {
	if s.useCHA {
		// 'G' is "Cursor Character Absolute (CHA)"
		fmt.Printf("\x1b[%dG", x+1)
	} else {
		// 'C' is "Cursor Forward (CUF)"
		fmt.Print("\r")
		if x > 0 {
			fmt.Printf("\x1b[%dC", x)
		}
	}
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

func (s *State) getPos() (x, y int64, err error) {
	fmt.Print("\x1b[6n")
	var reply []rune
	for {
		var r rune
		r, _, err = s.r.ReadRune()
		if err != nil {
			return
		}
		if r == 'R' {
			break
		}
		if len(reply) == 0 && r != 0x1b {
			return 0, 0, io.ErrUnexpectedEOF
		}
		if len(reply) == 1 && r != '[' {
			return 0, 0, io.ErrUnexpectedEOF
		}
		reply = append(reply, r)
	}

	if len(reply) < 3 {
		return 0, 0, io.ErrUnexpectedEOF
	}
	num := strings.Split(string(reply[2:]), ";")
	if len(num) != 2 {
		return 0, 0, io.ErrUnexpectedEOF
	}
	y, err = strconv.ParseInt(num[0], 10, 32)
	if err != nil {
		return
	}
	x, err = strconv.ParseInt(num[1], 10, 32)
	return
}

func (s *State) checkOutput() {
	// xterm is known to support CHA
	if strings.Contains(strings.ToLower(os.Getenv("TERM")), "xterm") {
		s.useCHA = true
		return
	}

	// test for functional ANSI CHA
	xOrig, _, err := s.getPos()
	if err != nil {
		return
	}

	// Move using CHA
	fmt.Printf("\x1b[%dG", xOrig+1%2)
	x, _, err := s.getPos()
	if err != nil {
		return
	}
	if x == xOrig {
		return
	}

	// X moved, CHA is functional
	s.useCHA = true
	s.cursorPos(int(xOrig - 1))
}

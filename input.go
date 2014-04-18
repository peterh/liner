// +build linux darwin

package liner

import (
	"errors"
	"io"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"
	"unsafe"
)

type nexter struct {
	r   rune
	err error
}

// State represents an open terminal
type State struct {
	commonState
	r        *readRune
	editMode termios
	origMode termios
	next     <-chan nexter
	winch    chan os.Signal
	pending  []rune
}

// The readRune structure and methods copied from src/pkg/fmt/scan.go:
// readRune is a structure to enable reading UTF-8 encoded code points
// from an io.Reader.  It is used if the Reader given to the scanner does
// not already implement io.RuneReader.
type readRune struct {
	reader  io.Reader
  	buf	[utf8.UTFMax]byte // used only inside ReadRune
	pending int               // number of bytes in pendBuf; only >0 for bad UTF-8
	pendBuf [utf8.UTFMax]byte // bytes left over
}

// readByte returns the next byte from the input, which may be
// left over from a previous read if the UTF-8 was ill-formed.
func (r *readRune) readByte() (b byte, err error) {
	if r.pending > 0 {
		b = r.pendBuf[0]
		copy(r.pendBuf[0:], r.pendBuf[1:])
		r.pending--
		return
	}
	n, err := io.ReadFull(r.reader, r.pendBuf[0:1])
	if n != 1 {
		return 0, err
	}
	return r.pendBuf[0], err
}

// unread saves the bytes for the next read.
func (r *readRune) unread(buf []byte) {
	copy(r.pendBuf[r.pending:], buf)
	r.pending += len(buf)
}

// ReadRune returns the next UTF-8 encoded code point from the
// io.Reader inside r.
func (r *readRune) ReadRune() (rr rune, size int, err error) {
	r.buf[0], err = r.readByte()
	if err != nil {
		return 0, 0, err
	}
	if r.buf[0] < utf8.RuneSelf { // fast check for common ASCII case
		rr = rune(r.buf[0])
		return
	}
	var n int
	for n = 1; !utf8.FullRune(r.buf[0:n]); n++ {
		r.buf[n], err = r.readByte()
		if err != nil {
			if err == io.EOF {
				err = nil
				break
			}
			return
		}
	}
	rr, size = utf8.DecodeRune(r.buf[0:n])
	if size < n { // an error
		r.unread(r.buf[size:n])
	}
	return
}

func UnsupportedTerminal() bool {
	term := os.Getenv("TERM")
	bad := map[string]bool{"": true, "dumb": true, "cons25": true}
	if bad[strings.ToLower(term)] {
		return true
	}
	return false
}

// NewLiner initializes a new *State, and saves the previous terminal mode.
// To restore the terminal to its previous state, call State.Close().
//
// Note if you are still using Go 1.0: NewLiner handles SIGWINCH, so it will
// leak a channel every time you call it. Therefore, it is recommened that you
// upgrade to a newer release of Go, or ensure that NewLiner is only called
// once.
func NewLiner() *State {
	var s State

	s.r = &readRune{reader: os.Stdin}

	if UnsupportedTerminal() {
		panic("liner: unsupported terminal type")
	}

	syscall.Syscall(syscall.SYS_IOCTL, uintptr(syscall.Stdin),
			getTermios, uintptr(unsafe.Pointer(&s.origMode)))

	s.editMode = s.origMode
	s.editMode.Iflag &^= icrnl | inpck | istrip | ixon
	s.editMode.Cflag |= cs8
	s.editMode.Lflag &^= syscall.ECHO | icanon | iexten

	s.winch = make(chan os.Signal, 1)
	signal.Notify(s.winch, syscall.SIGWINCH)

	s.getColumns()
	s.terminalOutput = s.columns > 0

	return &s
}

var errTimedOut = errors.New("timeout")

func (s *State) startPrompt() {
	next := make(chan nexter)
	go func() {
		for {
			var n nexter
			n.r, _, n.err = s.r.ReadRune()
			next <- n
			// Shut down nexter loop when an end condition has been reached
			// with the exception that this does not detect ^D on an empty line
			if n.err != nil || n.r == '\n' || n.r == '\r' {
				close(next)
				return
			}
		}
	}()
	s.next = next
}

func (s *State) nextPending(timeout <-chan time.Time) (rune, error) {
	select {
	case thing, ok := <-s.next:
		if !ok {
			return 0, errors.New("liner: internal error")
		}
		if thing.err != nil {
			return 0, thing.err
		}
		s.pending = append(s.pending, thing.r)
		return thing.r, nil
	case <-timeout:
		rv := s.pending[0]
		s.pending = s.pending[1:]
		return rv, errTimedOut
	}
	// not reached
	return 0, nil
}

func (s *State) readNext() (interface{}, error) {
	if len(s.pending) > 0 {
		rv := s.pending[0]
		s.pending = s.pending[1:]
		return rv, nil
	}
	var r rune
	select {
	case thing, ok := <-s.next:
		if !ok {
			return 0, errors.New("liner: internal error")
		}
		if thing.err != nil {
			return nil, thing.err
		}
		r = thing.r
	case <-s.winch:
		s.getColumns()
		return winch, nil
	}
	if r == tabKey {
		return tab, nil
	}
	if r != escKey {
		return r, nil
	}
	s.pending = append(s.pending, r)

	// Wait at most 50 ms for the rest of the escape sequence
	// If nothing else arrives, it was an actual press of the esc key
	timeout := time.After(50 * time.Millisecond)
	flag, err := s.nextPending(timeout)
	if err != nil {
		if err == errTimedOut {
			return esc, nil
		}
		return unknown, err
	}

	switch flag {
	case '[':
		code, err := s.nextPending(timeout)
		if err != nil {
			if err == errTimedOut {
				return code, nil
			}
			return unknown, err
		}
		switch code {
		case 'A':
			s.pending = s.pending[:0] // escape code complete
			return up, nil
		case 'B':
			s.pending = s.pending[:0] // escape code complete
			return down, nil
		case 'C':
			s.pending = s.pending[:0] // escape code complete
			return right, nil
		case 'D':
			s.pending = s.pending[:0] // escape code complete
			return left, nil
		case 'Z':
			s.pending = s.pending[:0] // escape code complete
			return shiftTab, nil
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			num := []rune{code}
			for {
				code, err := s.nextPending(timeout)
				if err != nil {
					if err == errTimedOut {
						return code, nil
					}
					return nil, err
				}
				switch code {
				case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
					num = append(num, code)
				case ';':
					// Modifier code to follow
					// This only supports Ctrl-left and Ctrl-right for now
					x, _ := strconv.ParseInt(string(num), 10, 32)
					if x != 1 {
						// Can't be left or right
						rv := s.pending[0]
						s.pending = s.pending[1:]
						return rv, nil
					}
					num = num[:0]
					for {
						code, err = s.nextPending(timeout)
						if err != nil {
							if err == errTimedOut {
								rv := s.pending[0]
								s.pending = s.pending[1:]
								return rv, nil
							}
							return nil, err
						}
						switch code {
						case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
							num = append(num, code)
						case 'C', 'D':
							// right, left
							mod, _ := strconv.ParseInt(string(num), 10, 32)
							if mod != 5 {
								// Not bare Ctrl
								rv := s.pending[0]
								s.pending = s.pending[1:]
								return rv, nil
							}
							s.pending = s.pending[:0] // escape code complete
							if code == 'C' {
								return wordRight, nil
							}
							return wordLeft, nil
						default:
							// Not left or right
							rv := s.pending[0]
							s.pending = s.pending[1:]
							return rv, nil
						}
					}
				case '~':
					s.pending = s.pending[:0] // escape code complete
					x, _ := strconv.ParseInt(string(num), 10, 32)
					switch x {
					case 2:
						return insert, nil
					case 3:
						return del, nil
					case 5:
						return pageUp, nil
					case 6:
						return pageDown, nil
					case 15:
						return f5, nil
					case 17:
						return f6, nil
					case 18:
						return f7, nil
					case 19:
						return f8, nil
					case 20:
						return f9, nil
					case 21:
						return f10, nil
					case 23:
						return f11, nil
					case 24:
						return f12, nil
					default:
						return unknown, nil
					}
				default:
					// unrecognized escape code
					rv := s.pending[0]
					s.pending = s.pending[1:]
					return rv, nil
				}
			}
		}

	case 'O':
		code, err := s.nextPending(timeout)
		if err != nil {
			if err == errTimedOut {
				return code, nil
			}
			return nil, err
		}
		s.pending = s.pending[:0] // escape code complete
		switch code {
		case 'H':
			return home, nil
		case 'F':
			return end, nil
		case 'P':
			return f1, nil
		case 'Q':
			return f2, nil
		case 'R':
			return f3, nil
		case 'S':
			return f4, nil
		default:
			return unknown, nil
		}
	default:
		rv := s.pending[0]
		s.pending = s.pending[1:]
		return rv, nil
	}

	// not reached
	return r, nil
}

func (s *State) Close() error {
	if s != nil {
		stopSignal(s.winch)
		s.restoreTerminalMode()
	}

	return nil
}

// Return the terminal to its original mode.
func (s *State) restoreTerminalMode() {
	if s == nil {
		return
	}

	syscall.Syscall(syscall.SYS_IOCTL, uintptr(syscall.Stdin),
			setTermios, uintptr(unsafe.Pointer(&s.origMode)))
}

// Put the terminal into the mode required for line editing.
func (s *State) lineEditingMode() {
	if s == nil {
		return
	}

	syscall.Syscall(syscall.SYS_IOCTL, uintptr(syscall.Stdin),
			setTermios, uintptr(unsafe.Pointer(&s.editMode)))
}


// +build linux darwin

package liner

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

type nexter struct {
	r   rune
	err error
}

// State represents an open terminal
type State struct {
	commonState
	r        *bufio.Reader
	origMode termios
	next     <-chan nexter
	winch    chan os.Signal
	pending  []rune
}

// NewLiner initializes a new *State, and sets the terminal into raw mode. To
// restore the terminal to its previous state, call State.Close().
//
// Note if you are still using Go 1.0: NewLiner handles SIGWINCH, so it will
// leak a channel every time you call it. Therefore, it is recommened that you
// upgrade to a newer release of Go, or ensure that NewLiner is only called
// once.
func NewLiner() *State {
	bad := map[string]bool{"": true, "dumb": true, "cons25": true}
	var s State
	s.r = bufio.NewReader(os.Stdin)

	s.terminalSupported = !bad[strings.ToLower(os.Getenv("TERM"))]
	if s.terminalSupported {
		syscall.Syscall(syscall.SYS_IOCTL, uintptr(syscall.Stdin), getTermios, uintptr(unsafe.Pointer(&s.origMode)))
		mode := s.origMode
		mode.Iflag &^= icrnl | inpck | istrip | ixon
		mode.Cflag |= cs8
		mode.Lflag &^= syscall.ECHO | icanon | iexten
		syscall.Syscall(syscall.SYS_IOCTL, uintptr(syscall.Stdin), setTermios, uintptr(unsafe.Pointer(&mode)))

		winch := make(chan os.Signal, 1)
		signal.Notify(winch, syscall.SIGWINCH)
		s.winch = winch

		next := make(chan nexter)
		go func() {
			for {
				var n nexter
				n.r, _, n.err = s.r.ReadRune()
				next <- n
			}
		}()
		s.next = next
	}

	s.getColumns()
	s.terminalOutput = s.columns > 0

	return &s
}

var errTimedOut = errors.New("timeout")

func (s *State) nextPending(timeout <-chan time.Time) (rune, error) {
	select {
	case thing := <-s.next:
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
	case thing := <-s.next:
		if thing.err != nil {
			return nil, thing.err
		}
		r = thing.r
	case <-s.winch:
		s.getColumns()
		return winch, nil
	}
	if r != esc {
		return r, nil
	}
	s.pending = append(s.pending, r)

	// Wait at most 50 ms for the rest of the escape sequence
	// If nothing else arrives, it was an actual press of the esc key
	timeout := time.After(50 * time.Millisecond)
	flag, err := s.nextPending(timeout)
	if err != nil {
		if err == errTimedOut {
			return flag, nil
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

func (s *State) promptUnsupported(p string) (string, error) {
	fmt.Print(p)
	linebuf, _, err := s.r.ReadLine()
	if err != nil {
		return "", err
	}
	return string(bytes.TrimSpace(linebuf)), nil
}

// Close returns the terminal to its previous mode
func (s *State) Close() error {
	stopSignal(s.winch)
	if s.terminalSupported {
		syscall.Syscall(syscall.SYS_IOCTL, uintptr(syscall.Stdin), setTermios, uintptr(unsafe.Pointer(&s.origMode)))
	}
	return nil
}

// +build !windows

package liner

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type State struct {
	commonState
	r *bufio.Reader
}

func NewLiner() *State {
	bad := map[string]bool{"": true, "dumb": true, "cons25": true}
	var s State
	s.r = bufio.NewReader(os.Stdin)
	s.supported = !bad[strings.ToLower(os.Getenv("TERM"))]
	return &s
}

const esc = 27

func (s *State) readNext() (interface{}, error) {
	r, _, err := s.r.ReadRune()
	if err != nil {
		return nil, err
	}
	if r != esc {
		return r, nil
	}
	flag, _, err := s.r.ReadRune()
	if err != nil {
		return nil, err
	}
	switch flag {
	case '[':
		code, _, err := s.r.ReadRune()
		if err != nil {
			return nil, err
		}
		switch code {
		case 'A':
			return up, nil
		case 'B':
			return down, nil
		case 'C':
			return right, nil
		case 'D':
			return left, nil
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			num := []rune{code}
			for {
				code, _, err := s.r.ReadRune()
				if err != nil {
					return nil, err
				}
				if code < '0' || code > '9' {
					if code != '~' {
						s.r.UnreadRune()
						return unknown, nil
					}
					break
				}
				num = append(num, code)
			}
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
		}

	case 'O':
		code, _, err := s.r.ReadRune()
		if err != nil {
			return nil, err
		}
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
		s.r.UnreadRune()
		return r, nil
	}

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

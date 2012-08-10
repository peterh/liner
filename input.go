// +build !windows

package line

import (
	"bufio"
	"os"
	"strconv"
)

type State struct {
	r *bufio.Reader
}

func NewSimpleLine() *State {
	var s State
	s.r = bufio.NewReader(os.Stdin)
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
			return Up, nil
		case 'B':
			return Down, nil
		case 'C':
			return Right, nil
		case 'D':
			return Left, nil
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
						return Unknown, nil
					}
					break
				}
				num = append(num, code)
			}
			x, _ := strconv.ParseInt(string(num), 10, 32)
			switch x {
			case 2:
				return Insert, nil
			case 3:
				return Delete, nil
			case 5:
				return PageUp, nil
			case 6:
				return PageDown, nil
			case 15:
				return F5, nil
			case 17:
				return F6, nil
			case 18:
				return F7, nil
			case 19:
				return F8, nil
			case 20:
				return F9, nil
			case 21:
				return F10, nil
			case 23:
				return F11, nil
			case 24:
				return F12, nil
			default:
				return Unknown, nil
			}
		}

	case 'O':
		code, _, err := s.r.ReadRune()
		if err != nil {
			return nil, err
		}
		switch code {
		case 'H':
			return Home, nil
		case 'F':
			return End, nil
		case 'P':
			return F1, nil
		case 'Q':
			return F2, nil
		case 'R':
			return F3, nil
		case 'S':
			return F4, nil
		default:
			return Unknown, nil
		}
	default:
		s.r.UnreadRune()
		return r, nil
	}

	return r, nil
}

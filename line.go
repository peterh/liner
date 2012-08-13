package liner

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"unicode/utf8"
)

type action int

const (
	left action = iota
	right
	up
	down
	home
	end
	insert
	del
	pageUp
	pageDown
	f1
	f2
	f3
	f4
	f5
	f6
	f7
	f8
	f9
	f10
	f11
	f12
	unknown
)

type commonState struct {
	history   []string
	supported bool
}

func (s *State) ReadHistory(r io.Reader) (num int, err error) {
	in := bufio.NewReader(r)
	num = 0
	for {
		line, part, err := in.ReadLine()
		if err == io.EOF {
			break
		}
		if err != nil {
			return num, err
		}
		if part {
			return num, errors.New("Line too long")
		}
		if !utf8.Valid(line) {
			return num, errors.New("Invalid string")
		}
		num++
		s.history = append(s.history, string(line))
	}
	return num, nil
}

func (s *State) WriteHistory(w io.Writer) (num int, err error) {
	for _, item := range s.history {
		_, err := fmt.Fprintln(w, item)
		if err != nil {
			return num, err
		}
		num++
	}
	return num, nil
}

const (
	ctrlA = 1
	ctrlB = 2
	ctrlC = 3
	ctrlD = 4
	ctrlE = 5
	ctrlF = 6
	ctrlH = 8
	lf    = 10
	ctrlK = 11
	ctrlL = 12
	cr    = 13
	ctrlN = 14
	ctrlP = 16
	ctrlU = 21
	esc   = 27
	bs    = 127
)

func (s *State) Prompt(p string) (string, error) {
	if !s.supported {
		return s.promptUnsupported(p)
	}

	fmt.Print(p)
	line := make([]rune, 0)
mainLoop:
	for {
		next, err := s.readNext()
		if err != nil {
			return "", err
		}
		switch v := next.(type) {
		case rune:
			switch v {
			case cr, lf:
				break mainLoop
			default:
				line = append(line, v)
				fmt.Printf("%c", v)
			}
		case action:
		}
	}
	return string(line), nil
}

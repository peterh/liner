package line

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"unicode/utf8"
)

type Action int

const (
	Left Action = iota
	Right
	Up
	Down
	Home
	End
	Insert
	Delete
	PageUp
	PageDown
	F1
	F2
	F3
	F4
	F5
	F6
	F7
	F8
	F9
	F10
	F11
	F12
	Unknown
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

func (s *State) Prompt(p string) (string, error) {
	if !s.supported {
		return s.promptUnsupported(p)
	}
	return s.promptUnsupported(p)
}

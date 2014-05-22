/*
Package liner implements a simple command line editor, inspired by linenoise
(https://github.com/antirez/linenoise/). This package supports WIN32 in
addition to the xterm codes supported by everything else.
*/
package liner

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

type commonState struct {
	terminalSupported bool
	terminalOutput    bool
	history           []string
	completer         WordCompleter
	columns           int
}

var errNotTerminalOutput = errors.New("standard output is not a terminal")

// HistoryLimit is the maximum number of entries saved in the scrollback history.
const HistoryLimit = 1000

// ReadHistory reads scrollback history from r. Returns the number of lines
// read, and any read error (except io.EOF).
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
			return num, fmt.Errorf("line %d is too long", num+1)
		}
		if !utf8.Valid(line) {
			return num, fmt.Errorf("invalid string at line %d", num+1)
		}
		num++
		s.history = append(s.history, string(line))
		if len(s.history) > HistoryLimit {
			s.history = s.history[1:]
		}
	}
	return num, nil
}

// WriteHistory writes scrollback history to w. Returns the number of lines
// successfully written, and any write error.
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

// AppendHistory appends an entry to the scrollback history. AppendHistory
// should be called iff Prompt returns a valid command.
func (s *State) AppendHistory(item string) {
	if len(s.history) > 0 {
		if item == s.history[len(s.history)-1] {
			return
		}
	}
	s.history = append(s.history, item)
	if len(s.history) > HistoryLimit {
		s.history = s.history[1:]
	}
}

// Returns the history lines starting with prefix
func (s *State) getHistoryByPrefix(prefix string) (ph []string) {
	for _, h := range s.history {
		if strings.HasPrefix(h, prefix) {
			ph = append(ph, h)
		}
	}
	return
}

// Completer takes the currently edited line content at the left of the cursor
// and returns a list of completion candidates.
// If the line is "Hello, wo!!!" and the cursor is before the first '!', "Hello, wo" is passed
// to the completer which may return {"Hello, world", "Hello, Word"} to have "Hello, world!!!".
type Completer func(line string) []string

// WordCompleter takes the currently edited line with the cursor position and
// returns the completion candidates for the partial word to be completed.
// If the line is "Hello, wo!!!" and the cursor is before the first '!', ("Hello, wo!!!", 9) is passed
// to the completer which may returns ("Hello, ", {"world", "Word"}, "!!!") to have "Hello, world!!!".
type WordCompleter func(line string, pos int) (head string, completions []string, tail string)

// SetCompleter sets the completion function that Liner will call to
// fetch completion candidates when the user presses tab.
func (s *State) SetCompleter(f Completer) {
	if f == nil {
		s.completer = nil
		return
	}
	s.completer = func(line string, pos int) (string, []string, string) {
		return "", f(line[:pos]), line[pos:]
	}
}

// SetWordCompleter sets the completion function that Liner will call to
// fetch completion candidates when the user presses tab.
func (s *State) SetWordCompleter(f WordCompleter) {
	s.completer = f
}

// ModeApplier is the interface that wraps a representation of the terminal
// mode. ApplyMode sets the terminal to this mode.
type ModeApplier interface {
	ApplyMode() error
}

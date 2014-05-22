// +build windows linux darwin openbsd freebsd netbsd

package liner

import (
	"fmt"
	"io"
	"unicode"
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
	shiftTab
	wordLeft
	wordRight
	winch
	unknown
)

const (
	ctrlA = 1
	ctrlB = 2
	ctrlC = 3
	ctrlD = 4
	ctrlE = 5
	ctrlF = 6
	ctrlH = 8
	tab   = 9
	lf    = 10
	ctrlK = 11
	ctrlL = 12
	cr    = 13
	ctrlN = 14
	ctrlP = 16
	ctrlT = 20
	ctrlU = 21
	ctrlW = 23
	esc   = 27
	bs    = 127
)

const (
	beep = "\a"
)

func (s *State) refresh(prompt string, buf string, pos int) error {
	s.cursorPos(0)
	_, err := fmt.Print(prompt)
	if err != nil {
		return err
	}

	pLen := utf8.RuneCountInString(prompt)
	bLen := utf8.RuneCountInString(buf)
	if pLen+bLen <= s.columns {
		_, err = fmt.Print(buf)
		s.eraseLine()
		s.cursorPos(pLen + pos)
	} else {
		// Find space available
		space := s.columns - pLen
		space-- // space for cursor
		start := pos - space/2
		end := start + space
		if end > bLen {
			end = bLen
			start = end - space
		}
		if start < 0 {
			start = 0
			end = space
		}
		pos -= start

		// Leave space for markers
		if start > 0 {
			start++
		}
		if end < bLen {
			end--
		}
		line := []rune(buf)
		line = line[start:end]

		// Output
		if start > 0 {
			fmt.Print("{")
		}
		fmt.Print(string(line))
		if end < bLen {
			fmt.Print("}")
		}

		// Set cursor position
		s.eraseLine()
		s.cursorPos(pLen + pos)
	}
	return err
}

func (s *State) tabComplete(p string, line []rune, pos int) ([]rune, int, interface{}, error) {
	if s.completer == nil {
		return line, pos, rune(tab), nil
	}
	head, list, tail := s.completer(string(line), pos)
	if len(list) <= 0 {
		return line, pos, rune(tab), nil
	}
	listEntry := 0
	hl := utf8.RuneCountInString(head)
	for {
		pick := list[listEntry]
		s.refresh(p, head+pick+tail, hl+utf8.RuneCountInString(pick))

		next, err := s.readNext()
		if err != nil {
			return line, pos, rune(tab), err
		}
		if key, ok := next.(rune); ok {
			if key == tab {
				if listEntry < len(list)-1 {
					listEntry++
				} else {
					listEntry = 0
				}
				continue
			}
			if key == esc {
				return line, pos, rune(esc), nil
			}
		}
		if a, ok := next.(action); ok && a == shiftTab {
			if listEntry > 0 {
				listEntry--
			} else {
				listEntry = len(list) - 1
			}
			continue
		}
		return []rune(head + pick + tail), hl + utf8.RuneCountInString(pick), next, nil
	}
	// Not reached
	return line, pos, rune(tab), nil
}

// Prompt displays p, and then waits for user input. Prompt allows line editing
// if the terminal supports it.
func (s *State) Prompt(p string) (string, error) {
	if !s.terminalOutput {
		return "", errNotTerminalOutput
	}
	if !s.terminalSupported {
		return s.promptUnsupported(p)
	}

	s.startPrompt()
	s.getColumns()

	fmt.Print(p)
	var line []rune
	pos := 0
	var historyEnd string
	prefixHistory := s.getHistoryByPrefix(string(line))
	historyPos := len(prefixHistory)
	var historyAction bool // used to mark history related actions
mainLoop:
	for {
		historyAction = false
		next, err := s.readNext()
		if err != nil {
			return "", err
		}

		if key, ok := next.(rune); ok && key == tab {
			line, pos, next, err = s.tabComplete(p, line, pos)
			if err != nil {
				return "", err
			}
			s.refresh(p, string(line), pos)
		}

		switch v := next.(type) {
		case rune:
			switch v {
			case cr, lf:
				fmt.Println()
				break mainLoop
			case ctrlA: // Start of line
				pos = 0
				s.refresh(p, string(line), pos)
			case ctrlE: // End of line
				pos = len(line)
				s.refresh(p, string(line), pos)
			case ctrlB: // left
				if pos > 0 {
					pos--
					s.refresh(p, string(line), pos)
				} else {
					fmt.Print(beep)
				}
			case ctrlF: // right
				if pos < len(line) {
					pos++
					s.refresh(p, string(line), pos)
				} else {
					fmt.Print(beep)
				}
			case ctrlD: // del
				if pos == 0 && len(line) == 0 {
					// exit
					return "", io.EOF
				}
				if pos >= len(line) {
					fmt.Print(beep)
				} else {
					line = append(line[:pos], line[pos+1:]...)
					s.refresh(p, string(line), pos)
				}
			case ctrlK: // delete remainder of line
				if pos >= len(line) {
					fmt.Print(beep)
				} else {
					line = line[:pos]
					s.refresh(p, string(line), pos)
				}
			case ctrlP: // up
				historyAction = true
				if historyPos > 0 {
					if historyPos == len(prefixHistory) {
						historyEnd = string(line)
					}
					historyPos--
					line = []rune(prefixHistory[historyPos])
					pos = len(line)
					s.refresh(p, string(line), pos)
				} else {
					fmt.Print(beep)
				}
			case ctrlN: // down
				historyAction = true
				if historyPos < len(prefixHistory) {
					historyPos++
					if historyPos == len(prefixHistory) {
						line = []rune(historyEnd)
					} else {
						line = []rune(prefixHistory[historyPos])
					}
					pos = len(line)
					s.refresh(p, string(line), pos)
				} else {
					fmt.Print(beep)
				}
			case ctrlT: // transpose prev rune with rune under cursor
				if len(line) < 2 || pos < 1 {
					fmt.Print(beep)
				} else {
					if pos == len(line) {
						pos--
					}
					line[pos-1], line[pos] = line[pos], line[pos-1]
					pos++
					s.refresh(p, string(line), pos)
				}
			case ctrlL: // clear screen
				s.eraseScreen()
				s.refresh(p, string(line), pos)
			case ctrlH, bs: // Backspace
				if pos <= 0 {
					fmt.Print(beep)
				} else {
					line = append(line[:pos-1], line[pos:]...)
					pos--
					s.refresh(p, string(line), pos)
				}
			case ctrlU: // Erase line before cursor
				line = line[pos:]
				pos = 0
				s.refresh(p, string(line), pos)
			case ctrlW: // Erase word
				if pos == 0 {
					fmt.Print(beep)
					break
				}
				// Remove whitespace to the left
				for {
					if pos == 0 || !unicode.IsSpace(line[pos-1]) {
						break
					}
					line = append(line[:pos-1], line[pos:]...)
					pos--
				}
				// Remove non-whitespace to the left
				for {
					if pos == 0 || unicode.IsSpace(line[pos-1]) {
						break
					}
					line = append(line[:pos-1], line[pos:]...)
					pos--
				}
				s.refresh(p, string(line), pos)
			// Catch unhandled control codes (anything <= 31)
			case 0, 3, 7, 9, 15:
				fallthrough
			case 17, 18, 19, 22, 24, 25, 26, 27, 28, 29, 30, 31:
				fmt.Print(beep)
			default:
				if pos == len(line) && len(p)+len(line) < s.columns {
					line = append(line, v)
					fmt.Printf("%c", v)
					pos++
				} else {
					line = append(line[:pos], append([]rune{v}, line[pos:]...)...)
					pos++
					s.refresh(p, string(line), pos)
				}
			}
		case action:
			switch v {
			case del:
				if pos >= len(line) {
					fmt.Print(beep)
				} else {
					line = append(line[:pos], line[pos+1:]...)
				}
			case left:
				if pos > 0 {
					pos--
				} else {
					fmt.Print(beep)
				}
			case wordLeft:
				if pos > 0 {
					for {
						pos--
						if pos == 0 || unicode.IsSpace(line[pos-1]) {
							break
						}
					}
				} else {
					fmt.Print(beep)
				}
			case right:
				if pos < len(line) {
					pos++
				} else {
					fmt.Print(beep)
				}
			case wordRight:
				if pos < len(line) {
					for {
						pos++
						if pos == len(line) || unicode.IsSpace(line[pos]) {
							break
						}
					}
				} else {
					fmt.Print(beep)
				}
			case up:
				historyAction = true
				if historyPos > 0 {
					if historyPos == len(prefixHistory) {
						historyEnd = string(line)
					}
					historyPos--
					line = []rune(prefixHistory[historyPos])
					pos = len(line)
				} else {
					fmt.Print(beep)
				}
			case down:
				historyAction = true
				if historyPos < len(prefixHistory) {
					historyPos++
					if historyPos == len(prefixHistory) {
						line = []rune(historyEnd)
					} else {
						line = []rune(prefixHistory[historyPos])
					}
					pos = len(line)
				} else {
					fmt.Print(beep)
				}
			case home: // Start of line
				pos = 0
			case end: // End of line
				pos = len(line)
			}
			s.refresh(p, string(line), pos)
		}
		if !historyAction {
			prefixHistory = s.getHistoryByPrefix(string(line))
			historyPos = len(prefixHistory)
		}
	}
	return string(line), nil
}

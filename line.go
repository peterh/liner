// +build windows linux darwin openbsd freebsd netbsd

package liner

import (
	"container/ring"
	"errors"
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
	altY
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
	ctrlG = 7
	ctrlH = 8
	tab   = 9
	lf    = 10
	ctrlK = 11
	ctrlL = 12
	cr    = 13
	ctrlN = 14
	ctrlO = 15
	ctrlP = 16
	ctrlQ = 17
	ctrlR = 18
	ctrlS = 19
	ctrlT = 20
	ctrlU = 21
	ctrlV = 22
	ctrlW = 23
	ctrlX = 24
	ctrlY = 25
	ctrlZ = 26
	esc   = 27
	bs    = 127
)

const (
	beep = "\a"
)

func (s *State) refresh(prompt []rune, buf []rune, pos int) error {
	s.cursorPos(0)
	_, err := fmt.Print(string(prompt))
	if err != nil {
		return err
	}

	pLen := countGlyphs(prompt)
	bLen := countGlyphs(buf)
	pos = countGlyphs(buf[:pos])
	if pLen+bLen < s.columns {
		_, err = fmt.Print(string(buf))
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
		startRune := len(getPrefixGlyphs(buf, start))
		line := getPrefixGlyphs(buf[startRune:], end-start)

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

func (s *State) tabComplete(p []rune, line []rune, pos int) ([]rune, int, interface{}, error) {
	if s.completer == nil {
		return line, pos, rune(esc), nil
	}
	head, list, tail := s.completer(string(line), pos)
	if len(list) <= 0 {
		return line, pos, rune(esc), nil
	}
	listEntry := 0
	hl := utf8.RuneCountInString(head)
	for {
		pick := list[listEntry]
		s.refresh(p, []rune(head+pick+tail), hl+utf8.RuneCountInString(pick))

		next, err := s.readNext()
		if err != nil {
			return line, pos, rune(esc), err
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
	return line, pos, rune(esc), nil
}

// reverse intelligent search, implements a bash-like history search.
func (s *State) reverseISearch(origLine []rune, origPos int) ([]rune, int, interface{}, error) {
	p := "(reverse-i-search)`': "
	s.refresh([]rune(p), origLine, origPos)

	line := []rune{}
	pos := 0
	foundLine := string(origLine)
	foundPos := origPos

	getLine := func() ([]rune, []rune, int) {
		search := string(line)
		prompt := "(reverse-i-search)`%s': "
		return []rune(fmt.Sprintf(prompt, search)), []rune(foundLine), foundPos
	}

	history, positions := s.getHistoryByPattern(string(line))
	historyPos := len(history) - 1

	for {
		next, err := s.readNext()
		if err != nil {
			return []rune(foundLine), foundPos, rune(esc), err
		}

		switch v := next.(type) {
		case rune:
			switch v {
			case ctrlR: // Search backwards
				if historyPos > 0 && historyPos < len(history) {
					historyPos--
					foundLine = history[historyPos]
					foundPos = positions[historyPos]
				} else {
					fmt.Print(beep)
				}
			case ctrlS: // Search forward
				if historyPos < len(history)-1 && historyPos >= 0 {
					historyPos++
					foundLine = history[historyPos]
					foundPos = positions[historyPos]
				} else {
					fmt.Print(beep)
				}
			case ctrlH, bs: // Backspace
				if pos <= 0 {
					fmt.Print(beep)
				} else {
					n := len(getSuffixGlyphs(line[:pos], 1))
					line = append(line[:pos-n], line[pos:]...)
					pos -= n

					// For each char deleted, display the last matching line of history
					history, positions := s.getHistoryByPattern(string(line))
					historyPos = len(history) - 1
					if len(history) > 0 {
						foundLine = history[historyPos]
						foundPos = positions[historyPos]
					} else {
						foundLine = ""
						foundPos = 0
					}
				}
			case ctrlG: // Cancel
				return origLine, origPos, rune(esc), err

			case tab, cr, lf, ctrlA, ctrlB, ctrlD, ctrlE, ctrlF, ctrlK,
				ctrlL, ctrlN, ctrlO, ctrlP, ctrlQ, ctrlT, ctrlU, ctrlV, ctrlW, ctrlX, ctrlY, ctrlZ:
				fallthrough
			case 0, ctrlC, esc, 28, 29, 30, 31:
				return []rune(foundLine), foundPos, next, err
			default:
				line = append(line[:pos], append([]rune{v}, line[pos:]...)...)
				pos++

				// For each keystroke typed, display the last matching line of history
				history, positions = s.getHistoryByPattern(string(line))
				historyPos = len(history) - 1
				if len(history) > 0 {
					foundLine = history[historyPos]
					foundPos = positions[historyPos]
				} else {
					foundLine = ""
					foundPos = 0
				}
			}
		case action:
			return []rune(foundLine), foundPos, next, err
		}
		s.refresh(getLine())
	}
}

// addToKillRing adds some text to the kill ring. If mode is 0 it adds it to a
// new node in the end of the kill ring, and move the current pointer to the new
// node. If mode is 1 or 2 it appends or prepends the text to the current entry
// of the killRing.
func (s *State) addToKillRing(text []rune, mode int) {
	// Don't use the same underlying array as text
	killLine := make([]rune, len(text))
	copy(killLine, text)

	// Point killRing to a newNode, procedure depends on the killring state and
	// append mode.
	if mode == 0 { // Add new node to killRing
		if s.killRing == nil { // if killring is empty, create a new one
			s.killRing = ring.New(1)
		} else if s.killRing.Len() >= KillRingMax { // if killring is "full"
			s.killRing = s.killRing.Next()
		} else { // Normal case
			s.killRing.Link(ring.New(1))
			s.killRing = s.killRing.Next()
		}
	} else {
		if s.killRing == nil { // if killring is empty, create a new one
			s.killRing = ring.New(1)
			s.killRing.Value = []rune{}
		}
		if mode == 1 { // Append to last entry
			killLine = append(s.killRing.Value.([]rune), killLine...)
		} else if mode == 2 { // Prepend to last entry
			killLine = append(killLine, s.killRing.Value.([]rune)...)
		}
	}

	// Save text in the current killring node
	s.killRing.Value = killLine
}

func (s *State) yank(p []rune, text []rune, pos int) ([]rune, int, interface{}, error) {
	if s.killRing == nil {
		return text, pos, rune(esc), nil
	}

	lineStart := text[:pos]
	lineEnd := text[pos:]
	var line []rune

	for {
		value := s.killRing.Value.([]rune)
		line = make([]rune, 0)
		line = append(line, lineStart...)
		line = append(line, value...)
		line = append(line, lineEnd...)

		pos = len(lineStart) + len(value)
		s.refresh(p, line, pos)

		next, err := s.readNext()
		if err != nil {
			return line, pos, next, err
		}

		switch v := next.(type) {
		case rune:
			return line, pos, next, nil
		case action:
			switch v {
			case altY:
				s.killRing = s.killRing.Prev()
			default:
				return line, pos, next, nil
			}
		}
	}

	return line, pos, esc, nil
}

// Prompt displays p, and then waits for user input. Prompt allows line editing
// if the terminal supports it.
func (s *State) Prompt(prompt string) (string, error) {
	err := ErrPromptAborted
	line := ""

	for err == ErrPromptAborted {
		line, err = s.AbortablePrompt(prompt, "")
	}

	return line, err
}

// AbortablePrompt displays p, and then waits for user input. If the user
// presses Ctrl-C AbortablePrompt returns ErrPromptAborted. AbortablePrompt
// allows line editing if the terminal supports it.
func (s *State) AbortablePrompt(prompt, aborted string) (string, error) {
	if !s.terminalOutput {
		return "", errNotTerminalOutput
	}
	if !s.terminalSupported {
		return s.promptUnsupported(prompt)
	}

	s.historyMutex.RLock()
	defer s.historyMutex.RUnlock()

	s.startPrompt()
	defer s.stopPrompt()
	s.getColumns()

	fmt.Print(prompt)
	p := []rune(prompt)
	var line []rune
	pos := 0
	var historyEnd string
	prefixHistory := s.getHistoryByPrefix(string(line))
	historyPos := len(prefixHistory)
	var historyAction bool // used to mark history related actions
	var killAction int = 0 // used to mark kill related actions
	var status error = nil
mainLoop:
	for {
		next, err := s.readNext()
	haveNext:
		if err != nil {
			return "", err
		}

		historyAction = false
		switch v := next.(type) {
		case rune:
			switch v {
			case ctrlC: // reset
				line = line[:0]
				status = ErrPromptAborted
				fmt.Print(aborted)
				fallthrough
			case cr, lf:
				fmt.Println()
				break mainLoop
			case ctrlA: // Start of line
				pos = 0
				s.refresh(p, line, pos)
			case ctrlE: // End of line
				pos = len(line)
				s.refresh(p, line, pos)
			case ctrlB: // left
				if pos > 0 {
					pos -= len(getSuffixGlyphs(line[:pos], 1))
					s.refresh(p, line, pos)
				} else {
					fmt.Print(beep)
				}
			case ctrlF: // right
				if pos < len(line) {
					pos += len(getPrefixGlyphs(line[pos:], 1))
					s.refresh(p, line, pos)
				} else {
					fmt.Print(beep)
				}
			case ctrlD: // del
				if pos == 0 && len(line) == 0 {
					// exit
					return "", io.EOF
				}

				// ctrlD is a potential EOF, so the rune reader shuts down.
				// Therefore, if it isn't actually an EOF, we must re-startPrompt.
				s.restartPrompt()

				if pos >= len(line) {
					fmt.Print(beep)
				} else {
					n := len(getPrefixGlyphs(line[pos:], 1))
					line = append(line[:pos], line[pos+n:]...)
					s.refresh(p, line, pos)
				}
			case ctrlK: // delete remainder of line
				if pos >= len(line) {
					fmt.Print(beep)
				} else {
					if killAction > 0 {
						s.addToKillRing(line[pos:], 1) // Add in apend mode
					} else {
						s.addToKillRing(line[pos:], 0) // Add in normal mode
					}

					killAction = 2 // Mark that there was a kill action
					line = line[:pos]
					s.refresh(p, line, pos)
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
					s.refresh(p, line, pos)
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
					s.refresh(p, line, pos)
				} else {
					fmt.Print(beep)
				}
			case ctrlT: // transpose prev glyph with glyph under cursor
				if len(line) < 2 || pos < 1 {
					fmt.Print(beep)
				} else {
					if pos == len(line) {
						pos -= len(getSuffixGlyphs(line, 1))
					}
					prev := getSuffixGlyphs(line[:pos], 1)
					next := getPrefixGlyphs(line[pos:], 1)
					scratch := make([]rune, len(prev))
					copy(scratch, prev)
					copy(line[pos-len(prev):], next)
					copy(line[pos-len(prev)+len(next):], scratch)
					pos += len(next)
					s.refresh(p, line, pos)
				}
			case ctrlL: // clear screen
				s.eraseScreen()
				s.refresh(p, line, pos)
			case ctrlH, bs: // Backspace
				if pos <= 0 {
					fmt.Print(beep)
				} else {
					n := len(getSuffixGlyphs(line[:pos], 1))
					line = append(line[:pos-n], line[pos:]...)
					pos -= n
					s.refresh(p, line, pos)
				}
			case ctrlU: // Erase line before cursor
				if killAction > 0 {
					s.addToKillRing(line[:pos], 2) // Add in prepend mode
				} else {
					s.addToKillRing(line[:pos], 0) // Add in normal mode
				}

				killAction = 2 // Mark that there was some killing
				line = line[pos:]
				pos = 0
				s.refresh(p, line, pos)
			case ctrlW: // Erase word
				if pos == 0 {
					fmt.Print(beep)
					break
				}
				// Remove whitespace to the left
				buf := make([]rune, 0) // Store the deleted chars in a buffer
				for {
					if pos == 0 || !unicode.IsSpace(line[pos-1]) {
						break
					}
					buf = append(buf, line[pos-1])
					line = append(line[:pos-1], line[pos:]...)
					pos--
				}
				// Remove non-whitespace to the left
				for {
					if pos == 0 || unicode.IsSpace(line[pos-1]) {
						break
					}
					buf = append(buf, line[pos-1])
					line = append(line[:pos-1], line[pos:]...)
					pos--
				}
				// Invert the buffer and save the result on the killRing
				newBuf := make([]rune, 0)
				for i := len(buf) - 1; i >= 0; i-- {
					newBuf = append(newBuf, buf[i])
				}
				if killAction > 0 {
					s.addToKillRing(newBuf, 2) // Add in prepend mode
				} else {
					s.addToKillRing(newBuf, 0) // Add in normal mode
				}
				killAction = 2 // Mark that there was some killing

				s.refresh(p, line, pos)
			case ctrlY: // Paste from Yank buffer
				line, pos, next, err = s.yank(p, line, pos)
				goto haveNext
			case ctrlR: // Reverse Search
				line, pos, next, err = s.reverseISearch(line, pos)
				s.refresh(p, line, pos)
				goto haveNext
			case tab: // Tab completion
				line, pos, next, err = s.tabComplete(p, line, pos)
				goto haveNext
			// Catch keys that do nothing, but you don't want them to beep
			case esc:
				// DO NOTHING
			// Unused keys
			case ctrlG, ctrlO, ctrlQ, ctrlS, ctrlV, ctrlX, ctrlZ:
				fallthrough
			// Catch unhandled control codes (anything <= 31)
			case 0, 28, 29, 30, 31:
				fmt.Print(beep)
			default:
				if pos == len(line) && len(p)+len(line) < s.columns-1 {
					line = append(line, v)
					fmt.Printf("%c", v)
					pos++
				} else {
					line = append(line[:pos], append([]rune{v}, line[pos:]...)...)
					pos++
					s.refresh(p, line, pos)
				}
			}
		case action:
			switch v {
			case del:
				if pos >= len(line) {
					fmt.Print(beep)
				} else {
					n := len(getPrefixGlyphs(line[pos:], 1))
					line = append(line[:pos], line[pos+n:]...)
				}
			case left:
				if pos > 0 {
					pos -= len(getSuffixGlyphs(line[:pos], 1))
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
					pos += len(getPrefixGlyphs(line[pos:], 1))
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
			s.refresh(p, line, pos)
		}
		if !historyAction {
			prefixHistory = s.getHistoryByPrefix(string(line))
			historyPos = len(prefixHistory)
		}
		if killAction > 0 {
			killAction--
		}
	}
	return string(line), status
}

// PasswordPrompt displays p, and then waits for user input. The input typed by
// the user is not displayed in the terminal.
func (s *State) PasswordPrompt(prompt string) (string, error) {
	if !s.terminalOutput {
		return "", errNotTerminalOutput
	}
	if !s.terminalSupported {
		return "", errors.New("liner: function not supported in this terminal")
	}

	s.startPrompt()
	defer s.stopPrompt()
	s.getColumns()

	fmt.Print(prompt)
	p := []rune(prompt)
	var line []rune
	pos := 0

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
				fmt.Println()
				break mainLoop
			case ctrlD: // del
				if pos == 0 && len(line) == 0 {
					// exit
					return "", io.EOF
				}

				// ctrlD is a potential EOF, so the rune reader shuts down.
				// Therefore, if it isn't actually an EOF, we must re-startPrompt.
				s.restartPrompt()
			case ctrlL: // clear screen
				s.eraseScreen()
				s.refresh(p, []rune{}, 0)
			case ctrlH, bs: // Backspace
				if pos <= 0 {
					fmt.Print(beep)
				} else {
					n := len(getSuffixGlyphs(line[:pos], 1))
					line = append(line[:pos-n], line[pos:]...)
					pos -= n
				}
			case ctrlC:
				fmt.Println()
				line = line[:0]
				pos = 0
				fmt.Print(prompt)
			// Unused keys
			case esc, tab, ctrlA, ctrlB, ctrlE, ctrlF, ctrlG, ctrlK, ctrlN, ctrlO, ctrlP, ctrlQ, ctrlR, ctrlS,
				ctrlT, ctrlU, ctrlV, ctrlW, ctrlX, ctrlY, ctrlZ:
				fallthrough
			// Catch unhandled control codes (anything <= 31)
			case 0, 28, 29, 30, 31:
				fmt.Print(beep)
			default:
				line = append(line[:pos], append([]rune{v}, line[pos:]...)...)
				pos++
			}
		}
	}
	return string(line), nil
}

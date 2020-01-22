package liner

import (
	"unsafe"
)

type coord struct {
	x, y int16
}
type smallRect struct {
	left, top, right, bottom int16
}

type consoleScreenBufferInfo struct {
	dwSize              coord
	dwCursorPosition    coord
	wAttributes         int16
	srWindow            smallRect
	dwMaximumWindowSize coord
}

func (s *State) cursorPos(x int) {
	var sbi consoleScreenBufferInfo
	procGetConsoleScreenBufferInfo.Call(s.outfd, uintptr(unsafe.Pointer(&sbi)))
	procSetConsoleCursorPosition.Call(s.outfd,
		uintptr(int(x)&0xFFFF|int(sbi.dwCursorPosition.y)<<16))
}

func (s *State) eraseLine() {
	var sbi consoleScreenBufferInfo
	procGetConsoleScreenBufferInfo.Call(s.outfd, uintptr(unsafe.Pointer(&sbi)))
	var numWritten uint32
	procFillConsoleOutputCharacter.Call(s.outfd, uintptr(' '),
		uintptr(sbi.dwSize.x-sbi.dwCursorPosition.x),
		uintptr(int(sbi.dwCursorPosition.x)&0xFFFF|int(sbi.dwCursorPosition.y)<<16),
		uintptr(unsafe.Pointer(&numWritten)))
}

func (s *State) eraseScreen() {
	var sbi consoleScreenBufferInfo
	procGetConsoleScreenBufferInfo.Call(s.outfd, uintptr(unsafe.Pointer(&sbi)))
	var numWritten uint32
	procFillConsoleOutputCharacter.Call(s.outfd, uintptr(' '),
		uintptr(sbi.dwSize.x)*uintptr(sbi.dwSize.y),
		0,
		uintptr(unsafe.Pointer(&numWritten)))
	procSetConsoleCursorPosition.Call(s.outfd, 0)
}

func (s *State) moveUp(lines int) {
	var sbi consoleScreenBufferInfo
	procGetConsoleScreenBufferInfo.Call(s.outfd, uintptr(unsafe.Pointer(&sbi)))
	procSetConsoleCursorPosition.Call(s.outfd,
		uintptr(int(sbi.dwCursorPosition.x)&0xFFFF|(int(sbi.dwCursorPosition.y)-lines)<<16))
}

func (s *State) moveDown(lines int) {
	var sbi consoleScreenBufferInfo
	procGetConsoleScreenBufferInfo.Call(s.outfd, uintptr(unsafe.Pointer(&sbi)))
	procSetConsoleCursorPosition.Call(s.outfd,
		uintptr(int(sbi.dwCursorPosition.x)&0xFFFF|(int(sbi.dwCursorPosition.y)+lines)<<16))
}

func (s *State) emitNewLine() {
	// windows doesn't need to omit a new line
}

func (s *State) getColumns() {
	var sbi consoleScreenBufferInfo
	procGetConsoleScreenBufferInfo.Call(s.outfd, uintptr(unsafe.Pointer(&sbi)))
	s.columns = int(sbi.dwSize.x)
}

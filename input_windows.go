package liner

import (
	"errors"
	"syscall"
	"unsafe"
)

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procGetStdHandle               = kernel32.NewProc("GetStdHandle")
	procReadConsoleInput           = kernel32.NewProc("ReadConsoleInputW")
	procGetConsoleMode             = kernel32.NewProc("GetConsoleMode")
	procSetConsoleMode             = kernel32.NewProc("SetConsoleMode")
	procSetConsoleCursorPosition   = kernel32.NewProc("SetConsoleCursorPosition")
	procGetConsoleScreenBufferInfo = kernel32.NewProc("GetConsoleScreenBufferInfo")
	procFillConsoleOutputCharacter = kernel32.NewProc("FillConsoleOutputCharacterW")
)

const (
	std_input_handle  = uint32(-10 & 0xFFFFFFFF)
	std_output_handle = uint32(-11 & 0xFFFFFFFF)
	std_error_handle  = uint32(-12 & 0xFFFFFFFF)
)

type State struct {
	commonState
	handle   syscall.Handle
	hOut     syscall.Handle
	origMode uint32
	key      interface{}
	repeat   uint16
}

const (
	enableEchoInput      = 0x4
	enableInsertMode     = 0x20
	enableLineInput      = 0x2
	enableMouseInput     = 0x10
	enableProcessedInput = 0x1
	enableQuickEditMode  = 0x40
	enableWindowInput    = 0x8
)

func NewLiner() *State {
	var s State
	hIn, _, _ := procGetStdHandle.Call(uintptr(std_input_handle))
	s.handle = syscall.Handle(hIn)
	hOut, _, _ := procGetStdHandle.Call(uintptr(std_output_handle))
	s.hOut = syscall.Handle(hOut)

	ok, _, _ := procGetConsoleMode.Call(hIn, uintptr(unsafe.Pointer(&s.origMode)))
	if ok != 0 {
		mode := s.origMode
		mode &^= enableEchoInput
		mode &^= enableInsertMode
		mode &^= enableLineInput
		mode &^= enableMouseInput
		mode |= enableWindowInput
		procSetConsoleMode.Call(hIn, uintptr(mode))
	}

	s.supported = true
	return &s
}

const (
	focus_event              = 0x0010
	key_event                = 0x0001
	menu_event               = 0x0008
	mouse_event              = 0x0002
	window_buffer_size_event = 0x0004
)

type input_record struct {
	eventType uint16
	pad       uint16
	blob      [16]byte
}

type key_event_record struct {
	KeyDown         int32
	RepeatCount     uint16
	VirtualKeyCode  uint16
	VirtualScanCode uint16
	Char            int16
	ControlKeyState uint32
}

const (
	vk_tab    = 0x09
	vk_prior  = 0x21
	vk_next   = 0x22
	vk_end    = 0x23
	vk_home   = 0x24
	vk_left   = 0x25
	vk_up     = 0x26
	vk_right  = 0x27
	vk_down   = 0x28
	vk_insert = 0x2d
	vk_delete = 0x2e
	vk_f1     = 0x70
	vk_f2     = 0x71
	vk_f3     = 0x72
	vk_f4     = 0x73
	vk_f5     = 0x74
	vk_f6     = 0x75
	vk_f7     = 0x76
	vk_f8     = 0x77
	vk_f9     = 0x78
	vk_f10    = 0x79
	vk_f11    = 0x7a
	vk_f12    = 0x7b
)

const (
	shiftPressed     = 0x0010
	leftAltPressed   = 0x0002
	leftCtrlPressed  = 0x0008
	rightAltPressed  = 0x0001
	rightCtrlPressed = 0x0004

	modKeys = shiftPressed | leftAltPressed | rightAltPressed | leftCtrlPressed | rightCtrlPressed
)

func (s *State) readNext() (interface{}, error) {
	if s.repeat > 0 {
		s.repeat--
		return s.key, nil
	}

	var input input_record
	pbuf := uintptr(unsafe.Pointer(&input))
	var rv uint32
	prv := uintptr(unsafe.Pointer(&rv))

	for {
		ok, _, err := procReadConsoleInput.Call(uintptr(s.handle), pbuf, 1, prv)

		if ok == 0 {
			return nil, err
		}

		if input.eventType == window_buffer_size_event {
			xy := (*coord)(unsafe.Pointer(&input.blob[0]))
			s.columns = int(xy.x)
			return winch, nil
		}
		if input.eventType != key_event {
			continue
		}
		ke := (*key_event_record)(unsafe.Pointer(&input.blob[0]))
		if ke.KeyDown == 0 {
			continue
		}

		if ke.VirtualKeyCode == vk_tab && ke.ControlKeyState&modKeys == shiftPressed {
			s.key = shiftTab
		} else if ke.Char > 0 {
			s.key = rune(ke.Char)
		} else {
			switch ke.VirtualKeyCode {
			case vk_prior:
				s.key = pageUp
			case vk_next:
				s.key = pageDown
			case vk_end:
				s.key = end
			case vk_home:
				s.key = home
			case vk_left:
				s.key = left
			case vk_right:
				s.key = right
			case vk_up:
				s.key = up
			case vk_down:
				s.key = down
			case vk_insert:
				s.key = insert
			case vk_delete:
				s.key = del
			case vk_f1:
				s.key = f1
			case vk_f2:
				s.key = f2
			case vk_f3:
				s.key = f3
			case vk_f4:
				s.key = f4
			case vk_f5:
				s.key = f5
			case vk_f6:
				s.key = f6
			case vk_f7:
				s.key = f7
			case vk_f8:
				s.key = f8
			case vk_f9:
				s.key = f9
			case vk_f10:
				s.key = f10
			case vk_f11:
				s.key = f11
			case vk_f12:
				s.key = f12
			default:
				// Eat modifier keys
				// TODO: return Action(Unknown) if the key isn't a
				// modifier.
				continue
			}
		}

		if ke.RepeatCount > 1 {
			s.repeat = ke.RepeatCount - 1
		}
		return s.key, nil
	}
	return unknown, nil
}

func (s *State) promptUnsupported(p string) (string, error) {
	return "", errors.New("Internal Error: Always supported on Windows")
}

func (s *State) Close() error {
	procSetConsoleMode.Call(uintptr(s.handle), uintptr(s.origMode))
	return nil
}

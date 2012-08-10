package line

import (
	"syscall"
	"unsafe"
)

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procGetStdHandle     = kernel32.NewProc("GetStdHandle")
	procReadConsoleInput = kernel32.NewProc("ReadConsoleInputW")
)

const (
	std_input_handle  = uint32(-10 & 0xFFFFFFFF)
	std_output_handle = uint32(-11 & 0xFFFFFFFF)
	std_error_handle  = uint32(-12 & 0xFFFFFFFF)
)

type State struct {
	handle syscall.Handle
	key    interface{}
	repeat uint16
}

func NewSimpleLine() *State {
	var s State
	h, _, _ := procGetStdHandle.Call(uintptr(std_input_handle))
	s.handle = syscall.Handle(h)
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

		if input.eventType != key_event {
			continue
		}
		ke := (*key_event_record)(unsafe.Pointer(&input))
		if ke.KeyDown == 0 {
			continue
		}

		if ke.Char > 0 {
			s.key = rune(ke.Char)
		} else {
			switch ke.VirtualKeyCode {
			case vk_prior:
				s.key = PageUp
			case vk_next:
				s.key = PageDown
			case vk_end:
				s.key = End
			case vk_home:
				s.key = Home
			case vk_left:
				s.key = Left
			case vk_right:
				s.key = Right
			case vk_up:
				s.key = Up
			case vk_down:
				s.key = Down
			case vk_insert:
				s.key = Insert
			case vk_delete:
				s.key = Delete
			case vk_f1:
				s.key = F1
			case vk_f2:
				s.key = F2
			case vk_f3:
				s.key = F3
			case vk_f4:
				s.key = F4
			case vk_f5:
				s.key = F5
			case vk_f6:
				s.key = F6
			case vk_f7:
				s.key = F7
			case vk_f8:
				s.key = F8
			case vk_f9:
				s.key = F9
			case vk_f10:
				s.key = F10
			case vk_f11:
				s.key = F11
			case vk_f12:
				s.key = F12
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
	return Unknown, nil
}

//go:build windows
// +build windows

package liner

import (
	"testing"
	"unsafe"
)

func makeKeyEvent(keyDown int32, vk uint16, char uint16, controlState uint32) input_record {
	var rec input_record
	rec.eventType = key_event
	ke := (*key_event_record)(unsafe.Pointer(&rec.blob[0]))
	ke.KeyDown = keyDown
	ke.VirtualKeyCode = vk
	ke.Char = char
	ke.ControlKeyState = controlState
	return rec
}

func TestIsActionable(t *testing.T) {
	// Normal characters (KeyDown)
	charEvent := makeKeyEvent(1, 'A', 'A', 0)
	if !isActionable(&charEvent) {
		t.Errorf("expected KeyDown 'A' to be actionable")
	}

	// Normal characters with Shift (KeyDown)
	shiftBEvent := makeKeyEvent(1, 'B', 'B', shiftPressed)
	if !isActionable(&shiftBEvent) {
		t.Errorf("expected KeyDown 'B' with Shift to be actionable")
	}

	// KeyUp events for normal characters and modifiers should NOT be actionable
	charUpEvent := makeKeyEvent(0, 'B', 'B', 0)
	if isActionable(&charUpEvent) {
		t.Errorf("expected KeyUp 'B' to NOT be actionable")
	}

	shiftUpEvent := makeKeyEvent(0, 0x10, 0, 0)
	if isActionable(&shiftUpEvent) {
		t.Errorf("expected KeyUp Shift to NOT be actionable")
	}

	ctrlUpEvent := makeKeyEvent(0, 0x11, 0, 0)
	if isActionable(&ctrlUpEvent) {
		t.Errorf("expected KeyUp Ctrl to NOT be actionable")
	}

	altUpEvent := makeKeyEvent(0, 0x12, 0, 0)
	if isActionable(&altUpEvent) {
		t.Errorf("expected KeyUp Alt to NOT be actionable")
	}

	// Modifier key down alone (Shift, Ctrl, Alt) should NOT be actionable
	shiftDownEvent := makeKeyEvent(1, 0x10, 0, shiftPressed)
	if isActionable(&shiftDownEvent) {
		t.Errorf("expected KeyDown Shift to NOT be actionable")
	}

	ctrlDownEvent := makeKeyEvent(1, 0x11, 0, leftCtrlPressed)
	if isActionable(&ctrlDownEvent) {
		t.Errorf("expected KeyDown Ctrl to NOT be actionable")
	}

	// Special navigation keys (KeyDown) should be actionable
	specialKeys := []uint16{
		vk_left, vk_right, vk_up, vk_down,
		vk_home, vk_end, vk_insert, vk_delete,
		vk_prior, vk_next,
	}
	for _, vk := range specialKeys {
		specialEvent := makeKeyEvent(1, vk, 0, 0)
		if !isActionable(&specialEvent) {
			t.Errorf("expected KeyDown VK 0x%x to be actionable", vk)
		}
	}

	// Function keys (KeyDown) should be actionable
	functionKeys := []uint16{
		vk_f1, vk_f2, vk_f3, vk_f4, vk_f5, vk_f6,
		vk_f7, vk_f8, vk_f9, vk_f10, vk_f11, vk_f12,
	}
	for _, vk := range functionKeys {
		fEvent := makeKeyEvent(1, vk, 0, 0)
		if !isActionable(&fEvent) {
			t.Errorf("expected KeyDown VK 0x%x to be actionable", vk)
		}
	}

	// Alt combinations
	altBEvent := makeKeyEvent(1, bKey, 0, leftAltPressed)
	if !isActionable(&altBEvent) {
		t.Errorf("expected KeyDown Alt+B to be actionable")
	}

	// Alt-numpad unicode on KeyUp
	altNumpadEvent := makeKeyEvent(0, vk_menu, 'X', 0)
	if !isActionable(&altNumpadEvent) {
		t.Errorf("expected KeyUp Alt-numpad to be actionable")
	}

	// Window buffer size event
	var winEvent input_record
	winEvent.eventType = window_buffer_size_event
	if !isActionable(&winEvent) {
		t.Errorf("expected window_buffer_size_event to be actionable")
	}

	// Non-key events like mouse or focus
	var mouseEvent input_record
	mouseEvent.eventType = mouse_event
	if isActionable(&mouseEvent) {
		t.Errorf("expected mouse_event to NOT be actionable")
	}

	var focusEvent input_record
	focusEvent.eventType = focus_event
	if isActionable(&focusEvent) {
		t.Errorf("expected focus_event to NOT be actionable")
	}
}

func TestHasActionable(t *testing.T) {
	// Scenario from issue #163:
	// After typing 'B' (Shift+B), readNext() has consumed KeyDown Shift and KeyDown 'B'.
	// In the console input buffer, 2 events remain: KeyUp 'B' and KeyUp Shift.
	// Previously, inputWaiting returned true because num > 1 (2 > 1), preventing refresh.
	shiftBKeyUpQueue := []input_record{
		makeKeyEvent(0, 'B', 'B', shiftPressed),
		makeKeyEvent(0, 0x10, 0, 0),
	}
	if hasActionable(shiftBKeyUpQueue) {
		t.Errorf("expected KeyUp events in buffer to not be considered actionable input")
	}

	// Single keyup in buffer
	singleKeyUpQueue := []input_record{
		makeKeyEvent(0, 'a', 'a', 0),
	}
	if hasActionable(singleKeyUpQueue) {
		t.Errorf("expected single KeyUp event in buffer to not be considered actionable input")
	}

	// Paste scenario: multiple keydown and keyup events in buffer
	pasteQueue := []input_record{
		makeKeyEvent(0, 'H', 'H', 0),
		makeKeyEvent(1, 'e', 'e', 0),
		makeKeyEvent(0, 'e', 'e', 0),
	}
	if !hasActionable(pasteQueue) {
		t.Errorf("expected paste queue containing KeyDown to be considered actionable input")
	}
}

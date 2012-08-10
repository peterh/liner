// +build !windows

package line

import (
	"bufio"
	"bytes"
	"testing"
)

func (s *State) expectRune(t *testing.T, r rune) {
	item, err := s.readNext()
	if err != nil {
		t.Fatal("Expected rune '%c', got error %s\n", r, err)
	}
	if v, ok := item.(rune); !ok {
		t.Fatal("Expected rune '%c', got non-rune %v\n", r, v)
	} else {
		if v != r {
			t.Fatal("Expected rune '%c', got rune '%c'\n", r, v)
		}
	}
}

func (s *State) expectAction(t *testing.T, a Action) {
	item, err := s.readNext()
	if err != nil {
		t.Fatal("Expected Action %d, got error %s\n", a, err)
	}
	if v, ok := item.(Action); !ok {
		t.Fatal("Expected Action %d, got non-Action %v\n", a, v)
	} else {
		if v != a {
			t.Fatal("Expected Action %d, got Action %d\n", a, v)
		}
	}
}

func TestTypes(t *testing.T) {
	input := []byte{'A', 27, 'B', 27, 91, 68}
	var s State
	s.r = bufio.NewReader(bytes.NewBuffer(input))

	s.expectRune(t, 'A')
	s.expectRune(t, 27)
	s.expectRune(t, 'B')
	s.expectAction(t, Left)
}

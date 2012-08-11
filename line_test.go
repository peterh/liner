// +build !windows

package line

import (
	"bytes"
	"strings"
	"testing"
)

func TestHistory(t *testing.T) {
	input := `foo
bar
baz
quux
dingle`

	var s State
	num, err := s.ReadHistory(strings.NewReader(input))
	if err != nil {
		t.Fatal("Unexpected error reading history", err)
	}
	if num != 5 {
		t.Fatal("Wrong number of history entries read")
	}

	var out bytes.Buffer
	num, err = s.WriteHistory(&out)
	if err != nil {
		t.Fatal("Unexpected error writing history", err)
	}
	if num != 5 {
		t.Fatal("Wrong number of history entries written")
	}
	if strings.TrimSpace(out.String()) != input {
		t.Fatal("Round-trip failure")
	}

	// Test reading with a trailing newline present
	var s2 State
	num, err = s2.ReadHistory(&out)
	if err != nil {
		t.Fatal("Unexpected error reading history the 2nd time", err)
	}
	if num != 5 {
		t.Fatal("Wrong number of history entries read the 2nd time")
	}
}

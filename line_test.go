package liner

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func TestAppend(t *testing.T) {
	var s State
	s.AppendHistory("foo")
	s.AppendHistory("bar")

	var out bytes.Buffer
	num, err := s.WriteHistory(&out)
	if err != nil {
		t.Fatal("Unexpected error writing history", err)
	}
	if num != 2 {
		t.Fatalf("Expected 2 history entries, got %d", num)
	}

	s.AppendHistory("baz")
	num, err = s.WriteHistory(&out)
	if err != nil {
		t.Fatal("Unexpected error writing history", err)
	}
	if num != 3 {
		t.Fatalf("Expected 3 history entries, got %d", num)
	}

	s.AppendHistory("baz")
	num, err = s.WriteHistory(&out)
	if err != nil {
		t.Fatal("Unexpected error writing history", err)
	}
	if num != 3 {
		t.Fatalf("Expected 3 history entries after duplicate append, got %d", num)
	}

	s.AppendHistory("baz")

}

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

	num, err = s.ReadHistory(strings.NewReader(input + "\n\xff"))
	if err == nil {
		t.Fatal("Unexpected success reading corrupted history", err)
	}
	if num != 5 {
		t.Fatal("Wrong number of history entries read the 3rd time")
	}
}

func TestPrompt(t *testing.T) {
	var s State
	s.terminalSupported = true

	input := []byte{'M','e','e','h', '\n'}
	s.r = bufio.NewReader(bytes.NewBuffer(input))
	s.next = make(chan nexter)

	got, err := s.Prompt("> ")
	if err != nil {
		t.Fatal("Unexpected error on prompt", err)
	}
	if got != strings.TrimSpace(string(input)) {
		t.Fatal(fmt.Sprintf("Prompt returned unexpected data %q != %q", got, string(input)))
	}

	
}

func TestPasswordPrompt(t *testing.T) {
	var s State
	s.terminalSupported = true

	input := []byte{'s','h','e','h', '\n'}
	s.r = bufio.NewReader(bytes.NewBuffer(input))
	s.next = make(chan nexter)

	realStdout := os.Stdout
	defer func() {os.Stdout = realStdout}()
	tmpf, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal("error creating tmpfile", err)
	}
	os.Stdout = tmpf
	
	
	got, err := s.PasswordPrompt("> ")
	if err != nil {
		t.Fatal("Unexpected error on prompt", err)
	}
	if got != strings.TrimSpace(string(input)) {
		t.Fatal(fmt.Sprintf("Prompt returned unexpected data %q != %q", got, string(input)))
	}

	tmpf.Seek(0, 0)
	output, err := ioutil.ReadAll(tmpf)
	if err != nil {
		t.Fatal("Unexpected error on read", err)
	}
	if strings.Contains(string(output), string(input)) {
		t.Fatal("password in output")
	}
}

package liner

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
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

	// clear the history and re-write
	s.ClearHistory()
	num, err = s.WriteHistory(&out)
	if err != nil {
		t.Fatal("Unexpected error writing history", err)
	}
	if num != 0 {
		t.Fatal("Wrong number of history entries written, expected none")
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

func TestColumns(t *testing.T) {
	list := []string{"foo", "food", "This entry is quite a bit longer than the typical entry"}

	output := []struct {
		width, columns, rows, maxWidth int
	}{
		{80, 1, 3, len(list[2]) + 1},
		{120, 2, 2, len(list[2]) + 1},
		{800, 14, 1, 0},
		{8, 1, 3, 7},
	}

	for i, o := range output {
		col, row, max := calculateColumns(o.width, list)
		if col != o.columns {
			t.Fatalf("Wrong number of columns, %d != %d, in TestColumns %d\n", col, o.columns, i)
		}
		if row != o.rows {
			t.Fatalf("Wrong number of rows, %d != %d, in TestColumns %d\n", row, o.rows, i)
		}
		if max != o.maxWidth {
			t.Fatalf("Wrong column width, %d != %d, in TestColumns %d\n", max, o.maxWidth, i)
		}
	}
}

// This example demonstrates a way to retrieve the current
// history buffer without using a file.
func ExampleState_WriteHistory() {
	var s State
	s.AppendHistory("foo")
	s.AppendHistory("bar")

	buf := new(bytes.Buffer)
	_, err := s.WriteHistory(buf)
	if err == nil {
		history := strings.Split(strings.TrimSpace(buf.String()), "\n")
		for i, line := range history {
			fmt.Println("History entry", i, ":", line)
		}
	}
	// Output:
	// History entry 0 : foo
	// History entry 1 : bar
}

func TestPromptTooNarrowFallback(t *testing.T) {
	origStdout := os.Stdout
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = pw
	defer func() {
		os.Stdout = origStdout
		pw.Close()
		pr.Close()
	}()

	var s State
	s.terminalSupported = true
	s.columns = 20
	// Prompt glyph count 11 + minWorkingSpace (10) = 21 > columns (20),
	// which forces Liner into tooNarrow fallback mode (reading input directly).
	prompt := "12345678901"
	s.r = bufio.NewReader(strings.NewReader("fallback test input\n"))

	line, err := s.Prompt(prompt)
	if err != nil {
		t.Fatalf("Unexpected error from Prompt in tooNarrow fallback: %v", err)
	}
	if line != "fallback test input" {
		t.Fatalf("Expected 'fallback test input', got %q", line)
	}

	pw.Close()
	var printed bytes.Buffer
	printed.ReadFrom(pr)
	if printed.String() != prompt {
		t.Fatalf("Expected prompt %q to be printed, got %q", prompt, printed.String())
	}
}

func TestAppendHistoryMultiline(t *testing.T) {
	var s State
	s.AppendHistory("normal line 1")
	s.AppendHistory("multi\nline")
	s.AppendHistory("multi\r\nline")
	s.AppendHistory("multi\rline")
	s.AppendHistory("\n")
	s.AppendHistory("\r\n")
	s.AppendHistory("normal line 2")

	var out bytes.Buffer
	num, err := s.WriteHistory(&out)
	if err != nil {
		t.Fatal("Unexpected error writing history", err)
	}
	if num != 2 {
		t.Fatalf("Expected 2 history entries, got %d", num)
	}
	expected := "normal line 1\nnormal line 2\n"
	if out.String() != expected {
		t.Fatalf("Expected history %q, got %q", expected, out.String())
	}
}

func runTestPrompt(t *testing.T, input string) string {
	origStdout := os.Stdout
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = pw
	defer func() {
		os.Stdout = origStdout
		pw.Close()
		pr.Close()
	}()

	var s State
	s.terminalSupported = true
	s.columns = 80
	s.r = bufio.NewReader(strings.NewReader(input))

	done := make(chan struct{})
	go func() {
		io.Copy(io.Discard, pr)
		close(done)
	}()

	line, err := s.Prompt("")
	if err != nil {
		t.Fatalf("Unexpected error from Prompt for input %q: %v", input, err)
	}
	pw.Close()
	<-done
	return line
}

func TestPromptMultilinePaste(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single line with lf",
			input:    "single line\n",
			expected: "single line",
		},
		{
			name:     "single line with crlf",
			input:    "single line\r\n",
			expected: "single line",
		},
		{
			name:     "single line with cr",
			input:    "single line\r",
			expected: "single line",
		},
		{
			name:     "two lines with lf",
			input:    "first line\nsecond line\n",
			expected: "first line\nsecond line",
		},
		{
			name:     "two lines with crlf",
			input:    "first line\r\nsecond line\r\n",
			expected: "first line\nsecond line",
		},
		{
			name:     "three lines with lf",
			input:    "line1\nline2\nline3\n",
			expected: "line1\nline2\nline3",
		},
		{
			name:     "three lines with crlf",
			input:    "line1\r\nline2\r\nline3\r\n",
			expected: "line1\nline2\nline3",
		},
		{
			name:     "empty line between lines",
			input:    "first\n\nsecond\n",
			expected: "first\n\nsecond",
		},
		{
			name:     "empty line between lines crlf",
			input:    "first\r\n\r\nsecond\r\n",
			expected: "first\n\nsecond",
		},
		{
			name:     "multiline without trailing newline followed by enter",
			input:    "foo\nbar\r",
			expected: "foo\nbar",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := runTestPrompt(t, tc.input)
			if got != tc.expected {
				t.Fatalf("Expected %q, got %q", tc.expected, got)
			}
		})
	}
}


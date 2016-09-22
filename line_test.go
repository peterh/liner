package liner

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
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

func BenchmarkInput1(b *testing.B)    { benchWithInput(b, 1) }
func BenchmarkInput10(b *testing.B)   { benchWithInput(b, 10) }
func BenchmarkInput100(b *testing.B)  { benchWithInput(b, 100) }
func BenchmarkInput1000(b *testing.B) { benchWithInput(b, 1000) }

func benchWithInput(b *testing.B, lineLength int) {
	inpR, inpW := io.Pipe()
	outR, outW := io.Pipe()

	s := newLiner(inpR, outW, -1, -1)
	s.inputRedirected = false
	s.outputRedirected = false
	s.terminalSupported = true

	go func() {
		// discard any output from the prompt reader
		io.Copy(ioutil.Discard, outR)
	}()

	buf := bytes.Buffer{}
	for i := 0; i < lineLength; i++ {
		buf.WriteByte('A')
	}
	buf.WriteByte('\n')
	go func() {
		for i := 0; i < b.N; i++ {
			inpW.Write(buf.Bytes())
		}
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		line, err := s.Prompt("test")
		if err != nil {
			b.Errorf("expected no errrors, got %s", err)
		}
		if len(line) != lineLength {
			b.Errorf("expected to read %d bytes, got %d", b.N, len(line))
		}
	}
	b.StopTimer()
	inpW.Close()
	outW.Close()
}

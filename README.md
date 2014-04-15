Liner
=====

Liner is a command line editor with history. It was inspired by linenoise;
everything Unix-like is a VT100 (or is trying very hard to be). If your
terminal is not pretending to be a VT100, change it. Liner also support
Windows.

Liner is released under the X11 license (which is similar to the new BSD
license).

Getting started
-----------------

```go
package main

import (
	"log"
	"os"
	"strings"

	"github.com/peterh/liner"
)

var (
	history_fn = "/tmp/.liner_history"
	names      = []string{"john", "james", "mary", "nancy"}
)

func main() {
	line := liner.NewLiner()
	defer line.Close()

	line.SetCompleter(func(line string) (c []string) {
		for _, n := range names {
			if strings.HasPrefix(n, strings.ToLower(line)) {
				c = append(c, n)
			}
		}
		return
	})

	if f, err := os.Open(history_fn); err == nil {
		line.ReadHistory(f)
		f.Close()
	}

	if name, err := line.Prompt("What is your name? "); err != nil {
		log.Print("Error reading line: ", err)
	} else {
		log.Print("Got: ", name)
		line.AppendHistory(name)
	}

	if f, err := os.Create(history_fn); err != nil {
		log.Print("Error writing history file: ", err)
	} else {
		line.WriteHistory(f)
		f.Close()
	}
}
```

For documentation, see http://godoc.org/github.com/peterh/liner

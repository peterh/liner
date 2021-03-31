// +build linux darwin openbsd freebsd netbsd

package liner

import (
	"os"
	"syscall"
)

func handleCtrZ() {
	p, err := os.FindProcess(os.Getpid())
	if err == nil {
		p.Signal(syscall.SIGTSTP)
	}
}

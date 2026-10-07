//go:build linux

// Source: src/util/debug.cpp:88-107.
package util

import (
	"os"
	"os/exec"
	"syscall"
)

const debugHostLinux = true

func (a *DebugActionContext) hostRunCommand(command string) (int, error) {
	out, e := a.streams.Stdout()
	if e != nil {
		return 0, e
	}
	errout, e := a.streams.Stderr()
	if e != nil {
		return 0, e
	}
	if e = a.input.open(); e != nil {
		return 0, e
	}
	if e = out.lock(); e != nil {
		return 0, e
	}
	out.mu.Unlock()
	if e = errout.lock(); e != nil {
		return 0, e
	}
	errout.mu.Unlock()
	// /bin/sh is the concrete system() adapter. The command is literal source
	// text; status reflects the actual child, with no injected result provider.
	cmd := exec.Command("/bin/sh", "-c", command)
	cmd.Stdin = a.input.file
	cmd.Stdout = out.File
	cmd.Stderr = errout.File
	if e = cmd.Run(); e != nil {
		return 1, nil
	}
	return 0, nil
}
func (a *DebugActionContext) hostStop() error {
	if e := syscall.Kill(os.Getpid(), syscall.SIGSEGV); e != nil {
		return e
	}
	select {} // the actual Go signal handler terminates this process.
}

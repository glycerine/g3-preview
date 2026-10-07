//go:build !linux

// Source: src/util/debug.cpp:72-118; src/util/debug.h:42-56.
package util

const debugHostLinux = false

func (a *DebugActionContext) hostRunCommand(string) (int, error) { return 0, ErrDebugHostUnsupported }
func (a *DebugActionContext) hostStop() error                    { return ErrDebugHostUnsupported }

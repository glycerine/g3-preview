//go:build !linux

// Source: src/util/debug.cpp:72-118; src/util/debug.h:42-56.
// Source author: Leonardo de Moura (2006-09-11).
// Licensed under the MIT license; see LICENSE.txt.
package g3

const debugHostLinux = false

func (a *DebugActionContext) hostRunCommand(string) (int, error) { return 0, ErrDebugHostUnsupported }
func (a *DebugActionContext) hostStop() error                    { return ErrDebugHostUnsupported }

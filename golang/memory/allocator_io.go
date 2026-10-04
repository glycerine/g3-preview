// Licensed under the MIT license.
// One borrowed cold dispatcher. Byte formatting remains in the caller package.
package memory

import "errors"

type AllocatorIOEvent uint8

const (
	AllocatorIOLeak AllocatorIOEvent = iota + 1
	AllocatorIOVerboseStart
	AllocatorIOVerboseEnd
)

type AllocatorIO interface {
	Emit(*SmallAllocator, AllocatorIOEvent) error
}

var ErrAllocatorIO = errors.New("memory: missing allocator diagnostic transport")

func (a *SmallAllocator) emit(event AllocatorIOEvent) error {
	if a.config.IO == nil {
		return ErrAllocatorIO
	}
	return a.config.IO.Emit(a, event)
}
func (a *SmallAllocator) diagnostic() error {
	if !a.config.Profile.Debug || a.allocSize == 0 {
		return nil
	}
	return a.emit(AllocatorIOLeak)
}
func (a *SmallAllocator) verbose(end bool) error {
	if a.config.Verbosity < 20 {
		return nil
	}
	if end {
		return a.emit(AllocatorIOVerboseEnd)
	}
	return a.emit(AllocatorIOVerboseStart)
}

// Source: src/util/memory_manager.cpp:33-72; src/util/memory_manager.h:49-71.
// Source: src/util/error_codes.h; src/util/z3_exception.h.
// Licensed under the MIT license. See LICENSE.txt.
// Source authors: Nikolaj Bjorner (nbjorner), 2007; Leonardo de Moura (leonardo), 2007/2011.
// Main compatibility/error bridge to the single concrete lower runtime.
package util

type utilMemoryRuntime = Runtime
type utilMemoryRuntimeConfig = RuntimeConfig
type utilMemoryBlock = Block
type memoryRuntimeState = RuntimeState
type memoryRuntimeStats = RuntimeStats

var ErrMemoryRuntimeOwner = ErrRuntimeOwner
var ErrMemoryRuntimeLive = ErrRuntimeLive
var ErrMemoryRuntimeUninitialized = ErrRuntimeUninitialized

func memoryErrorTransport(f Failure) error {
	switch f.Kind {
	case FailureMemout:
		return Z3Error{ErrorMemout}
	case FailureAllocExceeded:
		return Z3Error{ErrorAllocExceeded}
	case FailureVectorOverflow:
		return NewDefaultException(f.Message())
	}
	return ErrFailureTransport
}
func newUtilMemoryRuntime(c utilMemoryRuntimeConfig) (*utilMemoryRuntime, error) {
	if c.Transport == nil && c.FatalPolicy == nil {
		c.Transport = memoryErrorTransport
	}
	return NewRuntime(c)
}

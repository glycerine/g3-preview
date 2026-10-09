// Source: src/util/memory_manager.cpp:33-72; src/util/memory_manager.h:49-71.
// Source: src/util/error_codes.h; src/util/z3_exception.h.
// Main compatibility/error bridge to the single concrete lower runtime.
package util

import "github.com/glycerine/g3/pkg/util/z3exception"

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
		return z3exception.MustZ3Error(z3exception.ErrorMemout)
	case FailureAllocExceeded:
		return z3exception.MustZ3Error(z3exception.ErrorAllocExceeded)
	case FailureVectorOverflow:
		return z3exception.NewDefaultException(f.Message())
	}
	return ErrFailureTransport
}
func newUtilMemoryRuntime(c utilMemoryRuntimeConfig) (*utilMemoryRuntime, error) {
	if c.Transport == nil && c.FatalPolicy == nil {
		c.Transport = memoryErrorTransport
	}
	return NewRuntime(c)
}

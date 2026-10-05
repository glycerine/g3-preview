// Source: src/util/memory_manager.cpp:33-72; src/util/memory_manager.h:49-71.
// Source: src/util/error_codes.h; src/util/z3_exception.h.
// Licensed under the MIT license. See LICENSE.txt.
// Source authors: Nikolaj Bjorner (nbjorner), 2007; Leonardo de Moura (leonardo), 2007/2011.
// Main compatibility/error bridge to the single concrete lower runtime.
package g3

import "github.com/glycerine/g3/golang/memory"

type utilMemoryRuntime = memory.Runtime
type utilMemoryRuntimeConfig = memory.RuntimeConfig
type utilMemoryBlock = memory.Block
type memoryRuntimeState = memory.RuntimeState
type memoryRuntimeStats = memory.RuntimeStats

var ErrMemoryRuntimeOwner = memory.ErrRuntimeOwner
var ErrMemoryRuntimeLive = memory.ErrRuntimeLive
var ErrMemoryRuntimeUninitialized = memory.ErrRuntimeUninitialized

func memoryErrorTransport(f memory.Failure) error {
	switch f.Kind {
	case memory.FailureMemout:
		return Z3Error{ErrorMemout}
	case memory.FailureAllocExceeded:
		return Z3Error{ErrorAllocExceeded}
	case memory.FailureVectorOverflow:
		return NewDefaultException(f.Message())
	}
	return memory.ErrFailureTransport
}
func newUtilMemoryRuntime(c utilMemoryRuntimeConfig) (*utilMemoryRuntime, error) {
	if c.Transport == nil && c.FatalPolicy == nil {
		c.Transport = memoryErrorTransport
	}
	return memory.NewRuntime(c)
}

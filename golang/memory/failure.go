// Source: src/util/memory_manager.cpp:33-72; src/util/memory_manager.h:49-71.
// Source: src/util/error_codes.h.
// Licensed under the MIT license. See LICENSE.txt.
// Source authors: Nikolaj Bjorner, 2007; Leonardo de Moura, 2007.
package memory

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
)

type FailureKind uint8

const (
	FailureMemout FailureKind = iota + 1
	FailureAllocExceeded
	FailureVectorOverflow
)

type Failure struct{ Kind FailureKind }

func (f Failure) Message() string {
	switch f.Kind {
	case FailureMemout:
		return "out of memory"
	case FailureAllocExceeded:
		return "number of configured allocations exceeded"
	case FailureVectorOverflow:
		return "Overflow encountered when expanding vector"
	}
	return "invalid memory failure"
}
func (f Failure) Error() string { return f.Message() }
func (f Failure) ErrorCode() uint32 {
	switch f.Kind {
	case FailureMemout:
		return 101
	case FailureAllocExceeded:
		return 113
	}
	return 0
}
func (f Failure) HasErrorCode() bool { return f.ErrorCode() != 0 }

// Transport is a single borrowed named/static function. Returned exceptions
// are cold caller-owned values; no captured environment is allocated here.
type ErrorTransport func(Failure) error

// Cold borrowed source policy; vector overflow remains a distinct transport.
type FatalPolicy interface {
	MemoryBudget() *arena.Budget
	HandleMemoryFailure(*Runtime, Failure) error
}

var ErrFailureTransport = errors.New("memory: invalid failure transport")

func (m *Runtime) emitFailure(kind FailureKind) error {
	f := Failure{Kind: kind}
	if m.config.FatalPolicy != nil && kind != FailureVectorOverflow {
		e := m.config.FatalPolicy.HandleMemoryFailure(m, f)
		if e == nil {
			return ErrFailureTransport
		}
		return e
	}
	if m.config.Transport != nil {
		e := m.config.Transport(f)
		if e == nil {
			return ErrFailureTransport
		}
		return e
	}
	return f
}

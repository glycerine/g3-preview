// Copyright (c) 2006, 2011 Microsoft Corporation. Licensed under the MIT license.
package memory

import "errors"

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

var ErrFailureTransport = errors.New("memory: invalid failure transport")

func (m *Runtime) emitFailure(kind FailureKind) error {
	f := Failure{Kind: kind}
	if m.config.Transport != nil {
		e := m.config.Transport(f)
		if e == nil {
			return ErrFailureTransport
		}
		return e
	}
	return f
}

// Concrete error classification and nonformatted util/z3_exception values.
package util

import (
	"errors"
)

type ErrorCode uint32

const (
	ErrorOK                ErrorCode = 0
	ErrorMemout            ErrorCode = 101
	ErrorTimeout           ErrorCode = 102
	ErrorParser            ErrorCode = 103
	ErrorUnsoundness       ErrorCode = 104
	ErrorIncompleteness    ErrorCode = 105
	ErrorINIFile           ErrorCode = 106
	ErrorNotImplementedYet ErrorCode = 107
	ErrorOpenFile          ErrorCode = 108
	ErrorCmdLine           ErrorCode = 109
	ErrorInternalFatal     ErrorCode = 110
	ErrorTypeCheck         ErrorCode = 111
	ErrorUnknownResult     ErrorCode = 112
	ErrorAllocExceeded     ErrorCode = 113
)

var ErrExceptionPrecondition = errors.New("g3: exception source precondition")

// Z3Exception is a cold error surface. Exceptions and immutable message storage
// are caller-owned return values, never boxed hot records or owner-budget data.
type Z3Exception interface {
	error
	Message() string
	ErrorCode() ErrorCode
	HasErrorCode() bool
}
type Z3Error struct{ code ErrorCode }

func NewZ3Error(code ErrorCode) (Z3Error, error) {
	if code == ErrorOK {
		return Z3Error{}, ErrExceptionPrecondition
	}
	return Z3Error{code}, nil
}
func (e Z3Error) Message() string {
	switch e.code {
	case ErrorMemout:
		return "out of memory"
	case ErrorTimeout:
		return "timeout"
	case ErrorParser:
		return "parser error"
	case ErrorUnsoundness:
		return "unsoundess"
	case ErrorIncompleteness:
		return "incompleteness"
	case ErrorINIFile:
		return "invalid INI file"
	case ErrorNotImplementedYet:
		return "not implemented yet"
	case ErrorOpenFile:
		return "open file"
	case ErrorCmdLine:
		return "invalid command line"
	case ErrorInternalFatal:
		return "internal error"
	case ErrorTypeCheck:
		return "type error"
	case ErrorAllocExceeded:
		return "number of configured allocations exceeded"
	default:
		return "unknown error"
	}
}
func (e Z3Error) Error() string        { return e.Message() }
func (e Z3Error) ErrorCode() ErrorCode { return e.code }
func (e Z3Error) HasErrorCode() bool   { return e.ErrorCode() != ErrorOK }

// DefaultException retains std::string bytes including embedded NUL; Message
// observes the source msg() C-string prefix. Immutable Go strings keep their
// backing live automatically; no borrowed owner is retained by an exception.
// The fmt/va_list constructor is pending warning.cpp's printf formatter port.
type DefaultException struct{ MessageField string }

func NewDefaultException(message string) DefaultException { return DefaultException{message} }
func (e DefaultException) Message() string                { return e.MessageField[:cSymbolLength(e.MessageField)] }
func (e DefaultException) Error() string                  { return e.Message() }
func (e DefaultException) ErrorCode() ErrorCode           { return ErrorOK }
func (e DefaultException) HasErrorCode() bool             { return e.ErrorCode() != ErrorOK }

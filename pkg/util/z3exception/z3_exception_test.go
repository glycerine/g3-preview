package z3exception

import (
	"errors"
	"strings"
	"testing"
)

func TestZ3ExceptionSourceCodesAndLabels(t *testing.T) {
	codes := []ErrorCode{ErrorOK, ErrorMemout, ErrorTimeout, ErrorParser, ErrorUnsoundness, ErrorIncompleteness, ErrorINIFile, ErrorNotImplementedYet, ErrorOpenFile, ErrorCmdLine, ErrorInternalFatal, ErrorTypeCheck, ErrorUnknownResult, ErrorAllocExceeded}
	labels := []string{"out of memory", "timeout", "parser error", "unsoundess", "incompleteness", "invalid INI file", "not implemented yet", "open file", "invalid command line", "internal error", "type error", "unknown error", "number of configured allocations exceeded"}
	if codes[0] != 0 {
		t.Fatal(codes)
	}
	if _, e := NewZ3Error(ErrorOK); e != ErrExceptionPrecondition {
		t.Fatal(e)
	}
	for i, c := range codes[1:] {
		if c != ErrorCode(101+i) {
			t.Fatal(c)
		}
		z, e := NewZ3Error(c)
		if e != nil || z.Message() != labels[i] || z.Error() != labels[i] || z.ErrorCode() != c || !z.HasErrorCode() {
			t.Fatal(c, z, e)
		}
		var base Z3Exception = z
		if !base.HasErrorCode() {
			t.Fatal(base)
		}
	}
	for _, c := range []ErrorCode{1, 100, 114, ^ErrorCode(0)} {
		z, e := NewZ3Error(c)
		if e != nil || z.Message() != "unknown error" || z.ErrorCode() != c {
			t.Fatal(c, z, e)
		}
	}
}
func TestDefaultExceptionSourceStringAndEmbeddedNUL(t *testing.T) {
	for _, message := range []string{"", "plain", "prefix\x00suffix", "\x00hidden", "\xff\xfe", "line\nnext"} {
		x := NewDefaultException(message)
		want := strings.SplitN(message, "\x00", 2)[0]
		if x.Message() != want || x.Error() != want || x.ErrorCode() != ErrorOK || x.HasErrorCode() {
			t.Fatal(message, x)
		}
		if x.MessageField != message {
			t.Fatal("std::string storage must retain bytes")
		}
		var err error = x
		var target DefaultException
		if !errors.As(err, &target) || target.Message() != want {
			t.Fatal(err)
		}
	}
}

package cmdexception

import (
	"errors"
	"github.com/glycerine/g3/pkg/util/z3exception"
	"testing"
)

func TestSharedCmdExceptionBaseClassificationAndLocation(t *testing.T) {
	value := NewCmdExceptionAt("invalid polynomial", 23, 29)
	var err error = value
	var command CmdException
	var base z3exception.DefaultException
	var source z3exception.Z3Exception
	if !errors.As(err, &command) || !errors.As(err, &base) || !errors.As(err, &source) {
		t.Fatal("source exception bases missing", err)
	}
	line, le := command.Line()
	pos, pe := command.Pos()
	if le != nil || pe != nil || line != 23 || pos != 29 || base.Error() != value.Error() || source.HasErrorCode() {
		t.Fatal("source exception payload changed", line, pos, le, pe)
	}
	var stop StopParserException
	if errors.As(err, &stop) {
		t.Fatal("command exception became stop-parser signal")
	}
	err = StopParserException{}
	if !errors.As(err, &stop) || errors.As(err, &command) || errors.As(err, &base) {
		t.Fatal("stop parser lost distinct source type")
	}
}

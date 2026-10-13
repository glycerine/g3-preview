package shell

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestShellInputFileSourceBranches(t *testing.T) {
	var output bytes.Buffer
	warnings := diagnostic.NewWarningService(&output)
	option := NewInputFileOption(warnings)
	if _, ok := option.Selected(); ok {
		t.Fatal("unselected input should be null")
	}
	if err := option.AcceptPositional("first.smt2\x00ignored"); err != nil {
		t.Fatal(err)
	}
	if err := option.AcceptAfterDoubleDash([]string{"-second.smt2", "tail\x00ignored"}); err != nil {
		t.Fatal(err)
	}
	if err := option.AcceptPositional("third.smt2"); err != nil {
		t.Fatal(err)
	}
	if name, ok := option.Selected(); !ok || name != "first.smt2" {
		t.Fatal("duplicate changed first filename", name, ok)
	}
	if want := "WARNING: input file was already specified.\nWARNING: input file was already specified.\n"; output.String() != want {
		t.Fatalf("warning order: got %q, want %q", output.String(), want)
	}

	option = NewInputFileOption(warnings)
	if err := option.AcceptAfterDoubleDash([]string{"-a.smt2", "b c.smt2", "d\x00ignored"}); err != nil {
		t.Fatal(err)
	}
	if name, ok := option.Selected(); !ok || name != "-a.smt2 b c.smt2 d" {
		t.Fatal("terminal filename join", name, ok)
	}
	option = NewInputFileOption(warnings)
	if err := option.AcceptAfterDoubleDash(nil); err != nil {
		t.Fatal(err)
	}
	if name, ok := option.Selected(); !ok || name != "" {
		t.Fatal("empty terminal filename must still be selected", name, ok)
	}
}

type refusedWarningWriter struct{}

func (refusedWarningWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestShellInputFileWarningRefusalPreservesSelection(t *testing.T) {
	option := NewInputFileOption(diagnostic.NewWarningService(refusedWarningWriter{}))
	if err := option.AcceptPositional("first"); err != nil {
		t.Fatal(err)
	}
	if err := option.AcceptAfterDoubleDash([]string{"second"}); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatal("warning refusal", err)
	}
	if name, ok := option.Selected(); !ok || name != "first" {
		t.Fatal("source first selection lost", name, ok)
	}
	var absent *InputFileOption
	if err := absent.AcceptPositional("x"); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatal("nil receiver", err)
	}
}

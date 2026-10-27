package smt2

import (
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/cmd_context"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

type commandGlobalsReader struct {
	input *strings.Reader
	check func()
}

func (r *commandGlobalsReader) Read(p []byte) (int, error) {
	if r.check != nil {
		check := r.check
		r.check = nil
		check()
	}
	return r.input.Read(p)
}

func TestSMT2CommandSessionGlobalBindingAndRestoration(t *testing.T) {
	outer, err := cmd_context.NewStandaloneContext()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := outer.Close(); err != nil {
			t.Error(err)
		}
	}()
	inner, err := cmd_context.NewStandaloneContext()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := inner.Close(); err != nil {
			t.Error(err)
		}
	}()
	outerGlobals, err := outer.GlobalParameters()
	if err != nil {
		t.Fatal(err)
	}
	innerGlobals, err := inner.GlobalParameters()
	if err != nil {
		t.Fatal(err)
	}
	previous, err := uparams.BindProcessGlobalParamsService(innerGlobals)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := uparams.BindProcessGlobalParamsService(previous); err != nil {
			t.Error(err)
		}
	}()
	for _, script := range []string{"(exit)", "(assert )"} {
		reader := &commandGlobalsReader{input: strings.NewReader(script)}
		reader.check = func() {
			if uparams.BorrowProcessGlobalParamsService() != outerGlobals {
				t.Fatal("outer reader reached before binding")
			}
			if err := ParseCommands(inner, strings.NewReader("(exit)"), io.Discard); err != nil {
				t.Fatal(err)
			}
			if uparams.BorrowProcessGlobalParamsService() != outerGlobals {
				t.Fatal("nested parse did not restore enclosing session")
			}
		}
		err := ParseCommands(outer, reader, io.Discard)
		if (err != nil) != (script == "(assert )") {
			t.Fatal("parse result", script, err)
		}
		if uparams.BorrowProcessGlobalParamsService() != innerGlobals {
			t.Fatal("session failed to restore prior binding", script)
		}
	}
}

package smt2

// Source differential: src/cmd_context/basic_cmds.cpp get-assignment.

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/cmd_context"
)

func TestSMT2GetAssignmentBooleanMacrosInDeclarationOrder(t *testing.T) {
	context, err := cmd_context.NewStandaloneContext()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := context.Close(); err != nil {
			t.Error(err)
		}
	})
	script := `(set-option :produce-models true)(set-logic QF_UF)
(declare-const a Bool)
(declare-sort U 0)(declare-const u U)
(define-const |p q| Bool (and a true))
(define-fun q () Bool (not a))
(define-const ignored U u)
(define-fun parameterized ((x Bool)) Bool x)
(assert a)(check-sat)(get-assignment)
(push)(define-const r Bool false)(check-sat)(get-assignment)
(pop)(check-sat)(get-assignment)`
	var output bytes.Buffer
	if err = ParseCommands(context, strings.NewReader(script), &output); err != nil {
		t.Fatal(err, output.String())
	}
	want := "sat\n((|p q| true) (q false))\nsat\n((|p q| true) (q false) (r false))\nsat\n((|p q| true) (q false))\n"
	if output.String() != want {
		t.Fatal("get-assignment output", output.String(), "want", want)
	}
}

func TestSMT2GetAssignmentRequiresAvailableModel(t *testing.T) {
	context, err := cmd_context.NewStandaloneContext()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := context.Close(); err != nil {
			t.Error(err)
		}
	})
	var output bytes.Buffer
	err = ParseCommands(context, strings.NewReader(`(get-assignment)`), &output)
	if err == nil || !strings.Contains(err.Error(), "model is not available") || output.Len() != 0 {
		t.Fatal("get-assignment model refusal", err, output.String())
	}
}

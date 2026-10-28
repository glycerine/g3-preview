package smt2

// Source differential: src/cmd_context/simplify_cmd.cpp.

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/cmd_context"
)

func TestSMT2SimplifyCommandEndToEnd(t *testing.T) {
	context, err := cmd_context.NewStandaloneContext()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := context.Close(); err != nil {
			t.Error(err)
		}
	})
	script := `(set-logic QF_LIA)
(declare-const x Int)(declare-const y Int)
(simplify (+ 0 (* 1 x) 2 3))
(simplify (* (+ x y) (+ x y)) :som true)
(simplify (and true (= (+ 1 2) 3)))
(simplify (+ x 0) :print false)`
	var output bytes.Buffer
	if err = ParseCommands(context, strings.NewReader(script), &output); err != nil {
		t.Fatal(err, output.String())
	}
	want := "(+ 5 x)\n(+ (* x x) (* 2 x y) (* y y))\ntrue\n"
	if output.String() != want {
		t.Fatal("simplify output", output.String(), "want", want)
	}
}

func TestSMT2SimplifyCommandRejectsUnknownParameterBeforeOutput(t *testing.T) {
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
	err = ParseCommands(context, strings.NewReader(`(simplify (+ 1 2) :not-a-parameter true)`), &output)
	if err == nil || !strings.Contains(err.Error(), "unknown parameter") || output.Len() != 0 {
		t.Fatal("simplify parameter refusal", err, output.String())
	}
}

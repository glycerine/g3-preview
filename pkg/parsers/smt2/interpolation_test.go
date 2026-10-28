package smt2

// Source differential: src/cmd_context/interpolant_cmds.cpp.

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/cmd_context"
)

func TestSMT2ComputeAndGetInterpolantUseNativeBackend(t *testing.T) {
	context, err := cmd_context.NewStandaloneContext()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := context.Close(); err != nil {
			t.Error(err)
		}
	})
	script := `(declare-const q Bool)(compute-interpolant q (not q))(reset)
(set-option :produce-interpolants true)(declare-const p Bool)
(compute-interpolant p)
(compute-interpolant p (not p))
(assert p)(assert (not p))(check-sat)(get-interpolant p (not p))
(reset)(declare-const x Int)(compute-interpolant (< x 0) (> x 2))`
	var output bytes.Buffer
	if err = ParseCommands(context, strings.NewReader(script), &output); err != nil {
		t.Fatal(err, output.String())
	}
	want := "unsat\n(interpolants\n q)\nsat\nunsat\n(interpolants\n p)\nunsat\n(interpolants\n p)\nunsat\n(interpolants\n (not (>= x 0)))\n"
	if output.String() != want {
		t.Fatal("interpolation output", output.String(), "want", want)
	}
}

func TestSMT2InterpolantCommandGuards(t *testing.T) {
	for name, script := range map[string]string{
		"empty-compute": `(compute-interpolant)`,
		"disabled-get":  `(get-interpolant true)`,
		"no-proof":      `(set-option :produce-interpolants true)(get-interpolant true)`,
	} {
		t.Run(name, func(t *testing.T) {
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
			if err = ParseCommands(context, strings.NewReader(script), &output); err == nil {
				t.Fatal("interpolation guard accepted", script, output.String())
			}
		})
	}
}

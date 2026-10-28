package smt2

// Source differential: src/cmd_context/basic_cmds.cpp get-unsat-assumptions.

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/cmd_context"
)

func TestSMT2GetUnsatAssumptionsUsesDirectAssumptionCore(t *testing.T) {
	context, err := cmd_context.NewStandaloneContext()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := context.Close(); err != nil {
			t.Error(err)
		}
	})
	script := `(set-option :produce-unsat-assumptions true)(set-logic QF_UF)
(declare-const a Bool)(declare-const b Bool)(assert (not a))
(check-sat-assuming (a b))(get-unsat-assumptions)
(push)(assert (not b))(check-sat-assuming (b))(get-unsat-assumptions)(pop)
(push)(assert b)(check-sat-assuming ((not b)))(get-unsat-assumptions)(pop)
(reset)(set-logic QF_UF)(declare-const a Bool)(assert (not a))
(check-sat-assuming (a))(get-unsat-assumptions)`
	var output bytes.Buffer
	if err = ParseCommands(context, strings.NewReader(script), &output); err != nil {
		t.Fatal(err, output.String())
	}
	want := "unsat\n(a)\nunsat\n(b)\nunsat\n((not b))\nunsat\n(a)\n"
	if output.String() != want {
		t.Fatal("get-unsat-assumptions output", output.String(), "want", want)
	}
}

func TestSMT2GetUnsatAssumptionsGuards(t *testing.T) {
	for name, row := range map[string]struct {
		script  string
		message string
	}{
		"disabled":    {`(get-unsat-assumptions)`, "unsat assumptions construction is not enabled"},
		"unavailable": {`(set-option :produce-unsat-assumptions true)(get-unsat-assumptions)`, "unsat assumptions is not available"},
		"initialized": {`(declare-const a Bool)(set-option :produce-unsat-assumptions true)`, "option value cannot be modified after initialization"},
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
			err = ParseCommands(context, strings.NewReader(row.script), &output)
			if err == nil || !strings.Contains(err.Error(), row.message) || output.Len() != 0 {
				t.Fatal("get-unsat-assumptions guard", err, output.String())
			}
		})
	}
}

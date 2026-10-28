package smt2

// Source differential: src/cmd_context/basic_cmds.cpp labels_cmd.

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/cmd_context"
)

func TestSMT2LabelsCommandAndAvailability(t *testing.T) {
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
	if err = ParseCommands(context, strings.NewReader(`(set-logic QF_UF)(check-sat)(labels)
(reset)(set-logic QF_FD)(declare-const x Int)(assert (> x 0))(check-sat)(labels)`), &output); err != nil {
		t.Fatal(err)
	}
	if output.String() != "sat\n(labels)\nunknown\n(labels)\n" {
		t.Fatal("labels output", output.String())
	}
}

func TestSMT2LabelsCommandGuards(t *testing.T) {
	for name, script := range map[string]string{
		"before-check": `(labels)`,
		"after-unsat":  `(assert false)(check-sat)(labels)`,
		"extra":        `(check-sat)(labels extra)`,
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
			err = ParseCommands(context, strings.NewReader(script), &output)
			if err == nil {
				t.Fatal("labels guard accepted", script, output.String())
			}
			if name != "extra" && !strings.Contains(err.Error(), "labels are not available") {
				t.Fatal("labels availability error", err)
			}
		})
	}
}

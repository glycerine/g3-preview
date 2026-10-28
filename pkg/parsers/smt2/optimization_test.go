package smt2

// Source: src/opt/opt_cmds.cpp maximize/minimize/get-objectives and
// src/opt/opt_context.cpp scoped arithmetic objectives.

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/cmd_context"
	_ "github.com/glycerine/g3/pkg/engine"
)

func TestSMT2BoundedIntegerOptimizationEndToEnd(t *testing.T) {
	for _, tc := range []struct {
		command, value string
	}{
		{"maximize", "3"},
		{"minimize", "0"},
	} {
		t.Run(tc.command, func(t *testing.T) {
			context, err := cmd_context.NewStandaloneContext()
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := context.Close(); err != nil {
					t.Error(err)
				}
			})
			script := `(set-option :produce-models true)
(set-logic QF_LIA)
(declare-const x Int)
(assert (and (>= x 0) (<= x 3)))
(` + tc.command + ` x)
(check-sat)
(get-objectives)
(get-value (x))`
			var output bytes.Buffer
			if err = ParseCommands(context, strings.NewReader(script), &output); err != nil {
				t.Fatal(err, output.String())
			}
			want := "sat (objectives (x " + tc.value + ") ) ((x " + tc.value + "))"
			if got := strings.Join(strings.Fields(output.String()), " "); got != want {
				t.Fatal("optimization output", got, "want", want)
			}
		})
	}
}

func TestSMT2OptimizationObjectivesFollowScopes(t *testing.T) {
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
	script := `(set-logic QF_LIA)(declare-const x Int)(assert (= x 1))
(push)(maximize x)(pop)(check-sat)(get-objectives)`
	if err = ParseCommands(context, strings.NewReader(script), &output); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(strings.Fields(output.String()), " "); got != "sat (objectives )" {
		t.Fatal("popped objective survived", got)
	}
}

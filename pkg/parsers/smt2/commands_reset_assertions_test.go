package smt2

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/cmd_context"
)

func TestSMT2ResetAssertionsSyntaxAndParserLifetimes(t *testing.T) {
	for _, global := range []string{"true", "false"} {
		t.Run(global, func(t *testing.T) {
			c, err := cmd_context.NewStandaloneContext()
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := c.Close(); err != nil {
					t.Error(err)
				}
			})
			var out bytes.Buffer
			setup := "(set-option :global-declarations " + global + ")(set-logic AUFLIA)(define-fun id ((x Int)) Int (let ((y x)) y))(declare-const p Int)(assert (= p 7))(push 1)(check-sat-using smt)"
			if err = ParseCommands(c, strings.NewReader(setup), &out); err != nil {
				t.Fatal(err)
			}
			if err = ParseCommands(c, strings.NewReader("(reset-assertions extra)"), &out); err == nil || !strings.Contains(err.Error(), "too many arguments") {
				t.Fatal("reset-assertions argument diagnostic", err)
			}
			if err = ParseCommands(c, strings.NewReader("(get-value (p))"), &out); err != nil {
				t.Fatal("malformed command invalidated result", err)
			}
			if !strings.HasSuffix(out.String(), "((p 7))\n") {
				t.Fatal(out.String())
			}
			if global == "true" {
				if err = ParseCommands(c, strings.NewReader("(reset-assertions)(pop 1)(assert (= (id p) 4))(check-sat-using smt)(get-value (p))"), &out); err != nil {
					t.Fatal(err, out.String())
				}
				if !strings.HasSuffix(out.String(), "sat\n((p 4))\n") {
					t.Fatal(out.String())
				}
			} else {
				if err = ParseCommands(c, strings.NewReader("(reset-assertions)(set-logic QF_UF)(define-fun id ((x Bool)) Bool (let ((y x)) y))(declare-const p Bool)(assert (id p))(check-sat-using smt)(get-value (p))"), &out); err != nil {
					t.Fatal(err, out.String())
				}
				if !strings.HasSuffix(out.String(), "sat\n((p true))\n") {
					t.Fatal(out.String())
				}
			}
			m, err := c.Manager()
			if err != nil {
				t.Fatal(err)
			}
			budget := m.Budget()
			if err = c.Close(); err != nil {
				t.Fatal(err)
			}
			if budget.Used() != 0 {
				t.Fatal("reset-assertions retained parser/session bytes", budget.Used())
			}
		})
	}
}

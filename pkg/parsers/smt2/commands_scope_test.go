package smt2

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/cmd_context"
)

func TestSMT2OptionalScopeCountsAndNestedRollback(t *testing.T) {
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
	script := `(set-logic QF_UF)(declare-const p Bool)
(push)(assert p)(push 0)(pop 0)
(push)(assert (not p))(check-sat)
(pop)(check-sat)(pop)(assert (not p))(check-sat)`
	if err = ParseCommands(c, strings.NewReader(script), &out); err != nil {
		t.Fatal(err, out.String())
	}
	if out.String() != "unsat\nsat\nsat\n" {
		t.Fatal("optional scope-count semantics", out.String())
	}
}

func TestSMT2ScopeCommandsPreserveSourceMutationBeforeTrailingError(t *testing.T) {
	for _, operation := range []string{"push", "pop"} {
		t.Run(operation, func(t *testing.T) {
			c, err := cmd_context.NewStandaloneContext()
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := c.Close(); err != nil {
					t.Error(err)
				}
			})
			if operation == "pop" {
				if err = c.Push(1); err != nil {
					t.Fatal(err)
				}
			}
			var out bytes.Buffer
			if err = ParseCommands(c, strings.NewReader("(set-option :print-success true)"), &out); err != nil {
				t.Fatal(err)
			}
			out.Reset()
			if err = ParseCommands(c, strings.NewReader("("+operation+" 1 trailing)"), &out); err == nil {
				t.Fatal("malformed scope command was accepted")
			}
			popErr := c.Pop(1)
			if (operation == "push" && popErr != nil) || (operation == "pop" && popErr == nil) {
				t.Fatal("scope mutation did not precede source check_rparen", operation, popErr)
			}
			if out.Len() != 0 {
				t.Fatal("failed scope command printed success", out.String())
			}
		})
	}
}

func TestSMT2InvalidScopeCountsRefuseBeforeMutation(t *testing.T) {
	for _, count := range []string{"-1", "4294967296", "1.5", "oops"} {
		t.Run(count, func(t *testing.T) {
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
			if err = ParseCommands(c, strings.NewReader("(push "+count+")"), &out); err == nil {
				t.Fatal("invalid scope count was accepted")
			}
			if err = c.Pop(1); err == nil {
				t.Fatal("invalid count changed context scope depth")
			}
		})
	}
}

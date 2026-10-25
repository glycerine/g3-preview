package smt2

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/cmd_context"
)

func TestSMT2RecursiveDefinitionsKeepDeclarationsAfterBodyErrors(t *testing.T) {
	for _, input := range []string{
		"(define-fun-rec bad ((x Bool)) Bool 1)",
		"(define-fun-rec bad ((x Bool)) Bool (forall ((y Bool)) y))",
		"(define-funs-rec ((bad ((x Bool)) Bool) (other ((x Bool)) Bool)) (true))",
	} {
		t.Run(input, func(t *testing.T) {
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
			if err := ParseCommands(c, strings.NewReader(input), &out); err == nil {
				t.Fatal("malformed recursive body accepted")
			}
			// Source publishes declarations before bodies and unwinds the local
			// binder environment on a failed body without deleting declarations.
			if err := ParseCommands(c, strings.NewReader("(assert (bad true))"), &out); err != nil {
				t.Fatal("body failure lost its declaration or leaked locals", err)
			}
		})
	}
}

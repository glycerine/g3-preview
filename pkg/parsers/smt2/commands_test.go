package smt2

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/cmd_context"
	_ "github.com/glycerine/g3/pkg/engine"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSMT2CommandsRejectMalformedInputBeforeEffects(t *testing.T) {
	for _, command := range []string{"(assert false extra)", "(declare-fun p () Bool extra)", "(push 1 extra)", "(assert (forall ((x Bool))"} {
		t.Run(command, func(t *testing.T) {
			context, err := cmd_context.NewStandaloneContext()
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := context.Close(); err != nil {
					t.Fatal(err)
				}
			}()
			var out bytes.Buffer
			if err := ParseCommands(context, strings.NewReader(command), &out); err == nil {
				t.Fatal("malformed command was accepted")
			}
			if strings.HasPrefix(command, "(assert") {
				status, err := context.CheckSat(nil)
				if err != nil || status != lbool.True {
					t.Fatal("malformed assertion changed solver state", status, err)
				}
			}
			if strings.HasPrefix(command, "(declare") {
				if _, err := context.MakeApp("p", nil, nil, ast.ASTHandle{}); err == nil {
					t.Fatal("malformed declaration was installed")
				}
			}
			if strings.HasPrefix(command, "(push") && context.Pop(1) == nil {
				t.Fatal("malformed push opened a scope")
			}
		})
	}
}

func TestSMT2ExitStopsBeforeTrailingInputAndPrintsSuccess(t *testing.T) {
	context, err := cmd_context.NewStandaloneContext()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := context.Close(); err != nil {
			t.Fatal(err)
		}
	}()
	var out bytes.Buffer
	if err := ParseCommands(context, strings.NewReader("(set-option :print-success true)(exit)["), &out); err != nil {
		t.Fatal(err)
	}
	if out.String() != "success\nsuccess\n" {
		t.Fatal(out.String())
	}
}

func TestSMT2GetValuePreservesParsedTermTextAcrossQueries(t *testing.T) {
	context, err := cmd_context.NewStandaloneContext()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := context.Close(); err != nil {
			t.Fatal(err)
		}
	})
	input := `(set-logic QF_LIA)
(set-option :produce-models true)
(define-fun |a b| () Int 3)
(define-fun id ((n Int)) Int n)
(check-sat)
(get-value (|a b| (id   (+ 1 2)) (let ((q |a b|)) (+ q 0))))
(get-value ((+ 4 5)))`
	var out bytes.Buffer
	if err := ParseCommands(context, strings.NewReader(input), &out); err != nil {
		t.Fatal(err, out.String())
	}
	want := "sat\n((|a b| 3)\n ((id   (+ 1 2)) 3)\n ((let ((q |a b|)) (+ q 0)) 3))\n(((+ 4 5) 9))\n"
	if out.String() != want {
		t.Fatal("get-value must echo the original terms", out.String(), want)
	}
}

func TestSMT2GetValueRejectsEmptyAndQuantifiedTermsBeforeOutput(t *testing.T) {
	for _, command := range []string{"(get-value ())", "(get-value ((forall ((x Bool)) x)))", "(get-value ((and true (exists ((x Bool)) x))))"} {
		t.Run(command, func(t *testing.T) {
			context, err := cmd_context.NewStandaloneContext()
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := context.Close(); err != nil {
					t.Fatal(err)
				}
			})
			var out bytes.Buffer
			err = ParseCommands(context, strings.NewReader("(set-option :produce-models true)(check-sat)"+command), &out)
			if err == nil || !strings.Contains(err.Error(), "invalid get-value") || out.String() != "sat\n" {
				t.Fatal("invalid query was accepted or partially printed", err, out.String())
			}
		})
	}
}

func TestSMT2FunctionDefinitionFailuresAndRecovery(t *testing.T) {
	for _, script := range []string{
		"(define-fun f ((x Int)) Real x)",
		"(define-const f Real 1)",
		"(define-fun f ((x Bool)) Bool x extra)",
		"(define-fun f ((x Bool)) Bool x)(define-fun f ((y Bool)) Bool (not y))",
		"(declare-fun f (Bool) Bool)(define-fun f ((x Bool)) Bool x)",
		"(define-fun f ((x Bool)) Bool x)(declare-fun f (Bool) Bool)",
		"(define-fun and ((x Bool)) Bool x)",
	} {
		t.Run(script, func(t *testing.T) {
			c, err := cmd_context.NewStandaloneContext()
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := c.Close(); err != nil {
					t.Fatal(err)
				}
			}()
			var out bytes.Buffer
			if err := ParseCommands(c, strings.NewReader(script), &out); err == nil {
				t.Fatal("invalid definition accepted")
			}
			// A failed definition must leave the command context usable, including
			// releasing parser locals and restoring bound-variable scope.
			if err := ParseCommands(c, strings.NewReader("(define-fun g ((x Bool)) Bool x)(assert (g true))(check-sat)"), &out); err != nil {
				t.Fatal(err)
			}
			if out.String() != "sat\n" {
				t.Fatal(out.String())
			}
		})
	}
}

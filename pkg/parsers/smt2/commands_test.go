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
	for _, command := range []string{"(assert false extra)", "(declare-fun p () Bool extra)", "(declare-const p (Array))", "(declare-const p (Array Bool))", "(push 1 extra)", "(assert (forall ((x Bool))"} {
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

func TestSMT2UnsignedOptionsAndLiveRelevancyUpdates(t *testing.T) {
	context, err := cmd_context.NewStandaloneContext()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := context.Close(); err != nil {
			t.Fatal(err)
		}
	})
	input := `(set-option :smt.relevancy 0)
(set-option :rlimit 4294967295)
(set-option :produce-models true)
(declare-const p Bool)
(assert p)
(check-sat)
(get-value (p))
(set-option :smt.relevancy 2)
(check-sat)
(get-value (p))`
	var out bytes.Buffer
	if err := ParseCommands(context, strings.NewReader(input), &out); err != nil {
		t.Fatal(err, out.String())
	}
	if want := "sat\n((p true))\nsat\n((p true))\n"; out.String() != want {
		t.Fatal("unsigned options or relevancy update", out.String(), want)
	}
}

func TestSMT2UnsignedOptionsRejectRangeAndTrailingArguments(t *testing.T) {
	for _, value := range []string{"4294967296", "0 extra", "0.5", "-1"} {
		t.Run(value, func(t *testing.T) {
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
			if err := ParseCommands(context, strings.NewReader("(set-option :smt.relevancy "+value+")"), &out); err == nil || out.Len() != 0 {
				t.Fatal("invalid unsigned option accepted or printed output", err, out.String())
			}
		})
	}
}

func TestSMT2RelevancyUpdatesRestoreCachedBooleanAndTermITEWatches(t *testing.T) {
	for _, formula := range []string{"(or a b)", "(not (and a b))", "(ite c a b)", "(= (ite c 1 2) x)"} {
		t.Run(formula, func(t *testing.T) {
			context, err := cmd_context.NewStandaloneContext()
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := context.Close(); err != nil {
					t.Fatal(err)
				}
			})
			query := "(check-sat)(get-value (" + formula + "))"
			input := "(set-option :smt.relevancy 0)(set-option :produce-models true)" +
				"(declare-const a Bool)(declare-const b Bool)(declare-const c Bool)(declare-const x Int)" +
				"(assert " + formula + ")" + query +
				"(push 1)(set-option :smt.relevancy 2)" + query +
				"(push 1)(assert (not c))" + query +
				"(pop 1)" + query + "(pop 1)" + query
			var out bytes.Buffer
			if err := ParseCommands(context, strings.NewReader(input), &out); err != nil {
				t.Fatal(err, out.String())
			}
			want := strings.Repeat("sat\n(("+formula+" true))\n", 5)
			if out.String() != want {
				t.Fatal("cached relevance watches lost an asserted gate or term ITE", out.String(), want)
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

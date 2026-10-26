package smt2

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/cmd_context"
)

func TestSMT2ResetInvalidSyntaxPreservesSession(t *testing.T) {
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
	if err = ParseCommands(c, strings.NewReader("(set-logic QF_UF)(declare-const p Bool)(assert p)(push 1)"), &out); err != nil {
		t.Fatal(err)
	}
	if err = ParseCommands(c, strings.NewReader("(reset extra)"), &out); err == nil {
		t.Fatal("reset accepted an argument")
	}
	if err = ParseCommands(c, strings.NewReader("(pop 1)(assert (not p))(check-sat)"), &out); err != nil {
		t.Fatal(err)
	}
	if out.String() != "unsat\n" {
		t.Fatal("invalid reset changed declarations/assertions/scopes", out.String())
	}
}

func TestSMT2ResetDropsManagerBoundParserOwners(t *testing.T) {
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
	script := `(set-logic AUFLIA)
 (define-fun id ((x Int)) Int (let ((y x)) y))
 (assert (forall ((x Int)) (= (id x) x)))
 (check-sat-using smt)
 (reset)
 (set-logic ALL)
 (define-fun id ((x Bool)) Bool (let ((y x)) y))
 (assert (forall ((x Bool)) (= (id x) x)))
 (check-sat-using smt)
 (reset)
 (declare-const id Int)
 (assert (= id 7))
 (check-sat-using smt)
 (get-value (id))`
	if err = ParseCommands(c, strings.NewReader(script), &out); err != nil {
		t.Fatal(err, out.String())
	}
	if out.String() != "sat\nsat\nsat\n((id 7))\n" {
		t.Fatal("parser retained old manager/definitions", out.String())
	}
	manager, err := c.Manager()
	if err != nil {
		t.Fatal(err)
	}
	budget := manager.Budget()
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if used := budget.Used(); used != 0 {
		t.Fatal("parser reset retained bytes", used)
	}
}

func TestSMT2ResetPrintsSuccessBeforeNextScannerRefusal(t *testing.T) {
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
	if err = ParseCommands(c, strings.NewReader("(set-option :print-success true)(declare-const p Bool)(reset) \"unfinished"), &out); err == nil {
		t.Fatal("unterminated next string accepted")
	}
	if out.String() != "success\nsuccess\nsuccess\n" {
		t.Fatal("reset success was not printed before next-token failure", out.String())
	}
	if err = ParseCommands(c, strings.NewReader("(set-logic QF_LIA)(declare-const p Int)(assert (= p 1))(check-sat-using smt)"), &out); err != nil {
		t.Fatal("completed reset did not survive parser failure", err)
	}
	if !strings.HasSuffix(out.String(), "sat\n") {
		t.Fatal(out.String())
	}
}

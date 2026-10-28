package smt2

// Source: src/parsers/smt2/smt2parser.cpp:568-760,2098-2192.

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/cmd_context"
)

func TestSMT2ParametricSortDeclarationsAndDefinitions(t *testing.T) {
	context, err := cmd_context.NewStandaloneContext()
	if err != nil {
		t.Fatal(err)
	}
	defer context.Close()
	var output bytes.Buffer
	script := "(declare-sort Pair 2)" +
		"(define-sort BoolPair (T) (Pair Bool T))" +
		"(define-sort Set (T) (Array T Bool))" +
		"(define-sort Truth () Bool)" +
		"(declare-const p (BoolPair Int))" +
		"(declare-const s (Set Int))" +
		"(declare-const b Truth)" +
		"(assert (= p p))(assert (= (select (store s 3 true) 3) b))" +
		"(check-sat)"
	if err = ParseCommands(context, strings.NewReader(script), &output); err != nil {
		t.Fatal(err, output.String())
	}
	if output.String() != "sat\n" {
		t.Fatal(output.String())
	}
}

func TestSMT2ParametricSortArityAndScopeRefusals(t *testing.T) {
	for _, script := range []string{
		"(declare-sort F 1)(declare-const x F)",
		"(declare-sort F 1)(declare-const x (F Bool Bool))",
		"(define-sort F (T) T)(declare-const x (F))",
		"(define-sort F (T T) T)",
		"(push 1)(declare-sort F 1)(pop 1)(declare-const x (F Bool))",
	} {
		context, err := cmd_context.NewStandaloneContext()
		if err != nil {
			t.Fatal(err)
		}
		var output bytes.Buffer
		err = ParseCommands(context, strings.NewReader(script), &output)
		_ = context.Close()
		if err == nil {
			t.Fatal("accepted invalid parametric sort script", script)
		}
	}
}

package smt2

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/cmd_context"
)

func TestSMT2ConcreteDatatypeFormsAndDeclarationScopes(t *testing.T) {
	for _, declaration := range []string{
		"(declare-datatype Nat ((zero) (succ (pred Nat))))",
		"(declare-datatypes () ((Nat zero (succ (pred Nat)))))",
		"(declare-datatypes ((Nat 0)) ((zero (succ (pred Nat)))))",
	} {
		t.Run(declaration, func(t *testing.T) {
			context, err := cmd_context.NewStandaloneContext()
			if err != nil {
				t.Fatal(err)
			}
			defer context.Close()
			var output bytes.Buffer
			if err = ParseCommands(context, strings.NewReader("(push 1)"+declaration+"(declare-const n Nat)(assert (= (pred (succ n)) n))(check-sat-using (then simplify smt))"), &output); err != nil {
				t.Fatal(err, output.String())
			}
			if output.String() != "sat\n" {
				t.Fatal(output.String())
			}
			if err = ParseCommands(context, strings.NewReader("(pop 1)"), &output); err != nil {
				t.Fatal(err)
			}
			if _, err = context.LookupSort("Nat"); err == nil {
				t.Fatal("popped datatype sort remained visible")
			}
			if _, err = context.MakeApp("zero", nil, nil, ast.ASTHandle{}); err == nil {
				t.Fatal("popped constructor remained visible")
			}
		})
	}
}

func TestSMT2DatatypeRefusalsBeforeDeclarationPublication(t *testing.T) {
	for _, declaration := range []string{
		"(declare-datatype D ())",
		"(declare-datatype D ((make (field Missing))))",
		"(declare-datatype D ((a (field Bool)) (b (field Bool))))",
		"(declare-datatype D ((make)) extra)",
		"(declare-datatypes ((D 0) (E 0)) (((make))))",
		"(declare-datatypes ((D 0)) (((make)) ((other))))",
		"(declare-datatypes ((D 1)) ((par (T) ((make (field T))))))",
	} {
		t.Run(declaration, func(t *testing.T) {
			context, err := cmd_context.NewStandaloneContext()
			if err != nil {
				t.Fatal(err)
			}
			defer context.Close()
			var output bytes.Buffer
			if err = ParseCommands(context, strings.NewReader(declaration), &output); err == nil {
				t.Fatal("invalid/unported declaration accepted")
			}
			if _, err = context.LookupSort("D"); err == nil {
				t.Fatal("rejected declaration published its sort")
			}
			if _, err = context.MakeApp("make", nil, nil, ast.ASTHandle{}); err == nil {
				t.Fatal("rejected declaration published its constructor")
			}
		})
	}
}

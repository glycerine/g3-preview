package smt2

// Source differential: src/cmd_context/basic_cmds.cpp declare_map_cmd.

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/cmd_context"
)

func TestSMT2DeclareMapInstallsArrayAxiomAndScopes(t *testing.T) {
	context, err := cmd_context.NewStandaloneContext()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := context.Close(); err != nil {
			t.Error(err)
		}
	})
	script := `(set-logic ALL)(declare-map map-or (Int) (or (Bool Bool) Bool))
(declare-const a (Array Int Bool))(declare-const b (Array Int Bool))
(assert (not (= (select (map-or a b) 0) (or (select a 0) (select b 0)))))(check-sat)
(push)(declare-map map-not (Int) (not (Bool) Bool))(pop)`
	var output bytes.Buffer
	if err = ParseCommands(context, strings.NewReader(script), &output); err != nil {
		t.Fatal(err, output.String())
	}
	if output.String() != "unsat\n" {
		t.Fatal("declare-map result", output.String())
	}
	if _, err = context.FindFunctionDeclaration("map-or", nil, nil, ast.ASTHandle{}); err != nil {
		t.Fatal("persistent map declaration", err)
	}
	if _, err = context.FindFunctionDeclaration("map-not", nil, nil, ast.ASTHandle{}); err == nil {
		t.Fatal("scoped map declaration survived pop")
	}
}

func TestSMT2DeclareMapRejectsEmptyIndicesAndNullaryFunctions(t *testing.T) {
	for name, script := range map[string]string{
		"empty-indices": `(declare-map bad () (or (Bool Bool) Bool))`,
		"nullary":       `(declare-const k Bool)(declare-map bad (Int) k)`,
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
			if err == nil || !strings.Contains(err.Error(), "invalid map declaration") {
				t.Fatal("declare-map guard", err, output.String())
			}
		})
	}
}

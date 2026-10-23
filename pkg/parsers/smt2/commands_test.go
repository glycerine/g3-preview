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

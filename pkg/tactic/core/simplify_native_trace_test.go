package core

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func simplifyBoolFormula(t *testing.T, manager *ast.ASTManager) ast.ASTHandle {
	t.Helper()
	boolean, err := manager.MkBoolSort()
	simplifyMust(t, err)
	var atoms []ast.ASTHandle
	for _, name := range []string{"p", "q", "r"} {
		symbol, err := manager.Symbols.Intern(name)
		simplifyMust(t, err)
		atom, err := manager.MkConst(symbol, boolean)
		simplifyMust(t, err)
		simplifyPin(t, manager, atom)
		atoms = append(atoms, atom)
	}
	and, err := manager.MkAnd(atoms[0], atoms[1])
	simplifyMust(t, err)
	formula, err := manager.MkOr(and, atoms[2])
	simplifyMust(t, err)
	simplifyPin(t, manager, formula)
	return formula
}

func TestSimplifyTacticCPPApplyReference(t *testing.T) {
	reference := os.Getenv("G3_CPP_REFERENCE")
	if reference == "" {
		t.Skip("set G3_CPP_REFERENCE to compare native apply")
	}
	for _, eliminateAnd := range []bool{false, true} {
		t.Run(fmt.Sprint(eliminateAnd), func(t *testing.T) {
			manager, context, globals := simplifyFixture(t)
			var runner tactic.Tactic
			var err error
			name := "simplify"
			if eliminateAnd {
				runner, err = MakeEliminateAndTactic(manager, context, globals, nil)
				name = "elim-and"
			} else {
				runner, err = MakeSimplifyTactic(manager, context, globals, nil)
			}
			simplifyMust(t, err)
			simplifyMust(t, runner.IncRef())
			t.Cleanup(func() { simplifyMust(t, runner.DecRef()) })
			goal, err := tactic.NewGoal(manager, false, true, false)
			simplifyMust(t, err)
			t.Cleanup(func() { simplifyMust(t, goal.Close()) })
			simplifyMust(t, goal.Assert(simplifyBoolFormula(t, manager)))
			var result tactic.TacticResult
			t.Cleanup(func() { simplifyMust(t, result.Close()) })
			simplifyMust(t, runner.Apply(goal, &result))
			printer, err := ast.NewLegacySMTPrinter(manager)
			simplifyMust(t, err)
			var output bytes.Buffer
			output.WriteString("(goals (goal ")
			size, err := goal.Size()
			simplifyMust(t, err)
			for i := uint32(0); i < size; i++ {
				formula, err := goal.Form(i)
				simplifyMust(t, err)
				simplifyMust(t, printer.DisplayExpr(&output, formula, nil))
				output.WriteByte(' ')
			}
			depth, err := goal.Depth()
			simplifyMust(t, err)
			fmt.Fprintf(&output, ":precision precise :depth %d))", depth)
			command := exec.Command(reference, "-in")
			command.Stdin = strings.NewReader("(set-logic ALL)(declare-const p Bool)(declare-const q Bool)(declare-const r Bool)(assert (or (and p q) r))(apply " + name + ")")
			native, err := command.CombinedOutput()
			if err != nil {
				t.Fatal(err, string(native))
			}
			// These cases contain only unquoted Boolean symbols. Compare
			// S-expression tokens, independently of printer whitespace.
			tokens := func(s string) string {
				return strings.Join(strings.Fields(strings.NewReplacer("(", " ( ", ")", " ) ").Replace(s)), " ")
			}
			if tokens(output.String()) != tokens(string(native)) {
				t.Fatalf("simplifier apply differs from C++\nGo: %s\nC++: %s", output.String(), native)
			}
		})
	}
}

type simplifyRefusedWriter struct{}

func (simplifyRefusedWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestSimplifyTacticTraceStagesAndStreamRefusal(t *testing.T) {
	manager, context, globals := simplifyFixture(t)
	runner, err := NewSimplifyTactic(manager, context, globals, nil)
	simplifyMust(t, err)
	t.Cleanup(func() { simplifyMust(t, runner.Close()) })
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	for _, tag := range []string{"before_simplifier", "after_simplifier_bug", "after_simplifier", "after_simplifier_detail"} {
		tags.Enable(tag)
	}
	simplifyMust(t, runner.BindTrace(tags))
	goal, err := tactic.NewGoal(manager, false, true, false)
	simplifyMust(t, err)
	t.Cleanup(func() { simplifyMust(t, goal.Close()) })
	simplifyMust(t, goal.Assert(simplifyFalseEquality(t, manager)))
	var before bytes.Buffer
	simplifyMust(t, goal.Display(&before))
	var result tactic.TacticResult
	t.Cleanup(func() { simplifyMust(t, result.Close()) })
	simplifyMust(t, runner.Apply(goal, &result))
	if !tags.ShouldTrace("before_simplifier") {
		if output.Len() != 0 {
			t.Fatal("erased simplifier traces wrote output")
		}
	} else {
		var after bytes.Buffer
		simplifyMust(t, goal.Display(&after))
		record := func(tag string, line int, body string) string {
			return fmt.Sprintf("-------- [%s] operator() src/tactic/core/simplify_tactic.cpp:%d ---------\n%s------------------------------------------------\n", tag, line, body)
		}
		// imp::operator() traces dependencies before the public tactic
		// increments the goal depth.
		wantDetails := "(goal\n  |-\n  false\n  :precision precise :depth 0)\n"
		want := record("before_simplifier", 48, before.String()) + record("after_simplifier_bug", 67, after.String()) + record("after_simplifier", 69, after.String()) + record("after_simplifier_detail", 70, wantDetails)
		if output.String() != want {
			t.Fatalf("source simplifier trace mismatch\nGo: %q\nwant: %q", output.String(), want)
		}
	}
	refused := diagnostic.NewTraceTagsWithWriter(simplifyRefusedWriter{})
	refused.EnableAll(true)
	simplifyMust(t, runner.BindTrace(refused))
	simplifyMust(t, runner.Apply(goal, &result))
	if !goal.IsDecidedUnsat() {
		t.Fatal("trace stream refusal changed the goal")
	}
	copy := *runner
	if err := copy.Close(); err == nil {
		t.Fatal("copied tactic accepted ownership of the original rewriter")
	}
}

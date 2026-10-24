package aig

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/tactic"
	uparams "github.com/glycerine/g3/pkg/util/params"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func aigTacticFixture(t *testing.T) (*ast.ASTManager, *Tactic) {
	t.Helper()
	m, _ := aigFixture(t)
	close := aigTestBindSMT2Params(t, m.Symbols, m.NumeralService(), m.Budget())
	t.Cleanup(close)
	c := uparams.BorrowProcessGlobalParamsService().Context
	r, err := NewTactic(c, nil, nil)
	aigMust(t, err)
	t.Cleanup(func() { aigMust(t, r.Close()) })
	return m, r
}
func aigAdapterGoal(t *testing.T, m *ast.ASTManager, form string, cores bool) *tactic.Goal {
	t.Helper()
	sort, err := m.MkBoolSort()
	aigMust(t, err)
	var values []ast.ASTHandle
	for _, name := range []string{"p", "q", "r"} {
		sym, err := m.Symbols.Intern(name)
		aigMust(t, err)
		h, err := m.MkConst(sym, sort)
		aigMust(t, err)
		aigPinExpr(t, m, h)
		values = append(values, h)
	}
	p, q, r := values[0], values[1], values[2]
	var h ast.ASTHandle
	switch form {
	case "xor":
		h, err = m.MkXor(p, q)
	case "ite":
		h, err = m.MkIte(p, q, r)
	case "iff":
		h, err = m.MkIff(p, q)
	case "sharing":
		a, e := m.MkAnd(p, q)
		aigMust(t, e)
		aigPinExpr(t, m, a)
		b, e := m.MkAnd(p, r)
		aigMust(t, e)
		aigPinExpr(t, m, b)
		h, err = m.MkOr(a, b)
	case "demorgan":
		a, e := m.MkOr(p, q)
		aigMust(t, e)
		aigPinExpr(t, m, a)
		h, err = m.MkNot(a)
	}
	aigMust(t, err)
	goal, err := tactic.NewGoal(m, false, true, cores)
	aigMust(t, err)
	t.Cleanup(func() { aigMust(t, goal.Close()) })
	aigMust(t, goal.Assert(h))
	return goal
}
func aigAdapterWrite(t *testing.T, m *ast.ASTManager, w *bytes.Buffer, h ast.ASTHandle) {
	t.Helper()
	g := uparams.BorrowProcessGlobalParamsService()
	f, err := m.SMT2PPExprTree(h, nil)
	aigMust(t, err)
	out, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: m.Budget()})
	aigMust(t, err)
	defer func() { aigMust(t, out.Close()) }()
	p := g.Context.EmptyValue()
	aigMust(t, m.PrintFormat(out, f, g.Context, &p, g))
	text, err := out.String()
	aigMust(t, err)
	w.WriteString(text)
}

func TestAIGTacticCPPGoalShapes(t *testing.T) {
	reference := os.Getenv("G3_CPP_REFERENCE")
	if reference == "" {
		t.Skip("set G3_CPP_REFERENCE to prebuilt z3")
	}
	for _, per := range []bool{false, true} {
		for _, form := range []string{"xor", "ite", "iff", "sharing", "demorgan"} {
			t.Run(fmt.Sprint(per, "/", form), func(t *testing.T) {
				m, r := aigTacticFixture(t)
				p, err := r.context.NewValue()
				aigMust(t, err)
				defer func() { aigMust(t, r.context.Del(&p)) }()
				key := "aig_per_assertion"
				aigMust(t, r.context.SetBoolName(&p, &key, per))
				aigMust(t, r.UpdateParams(&p))
				goal := aigAdapterGoal(t, m, form, false)
				var source bytes.Buffer
				source.WriteString("(set-logic QF_UF)(declare-const p Bool)(declare-const q Bool)(declare-const r Bool)")
				inputSize, err := goal.Size()
				aigMust(t, err)
				for i := uint32(0); i < inputSize; i++ {
					h, err := goal.Form(i)
					aigMust(t, err)
					source.WriteString("(assert ")
					aigAdapterWrite(t, m, &source, h)
					source.WriteString(")")
				}
				fmt.Fprintf(&source, "(apply (using-params aig :aig_per_assertion %v))", per)
				var result tactic.TacticResult
				defer func() { aigMust(t, result.Close()) }()
				aigMust(t, r.Apply(goal, &result))
				var actual bytes.Buffer
				actual.WriteString("(goals (goal ")
				n, err := goal.Size()
				aigMust(t, err)
				for i := uint32(0); i < n; i++ {
					h, err := goal.Form(i)
					aigMust(t, err)
					aigAdapterWrite(t, m, &actual, h)
					actual.WriteByte(' ')
				}
				depth, err := goal.Depth()
				aigMust(t, err)
				fmt.Fprintf(&actual, ":precision precise :depth %d))", depth)
				cmd := exec.Command(reference, "-in")
				cmd.Stdin = bytes.NewReader(source.Bytes())
				native, err := cmd.CombinedOutput()
				if err != nil {
					t.Fatal(err, string(native))
				}
				canonical := func(s string) string {
					return strings.Join(strings.Fields(strings.NewReplacer("(", " ( ", ")", " ) ").Replace(s)), " ")
				}
				if canonical(actual.String()) != canonical(string(native)) {
					t.Fatalf("Go: %s\nC++: %s", actual.String(), native)
				}
			})
		}
	}
}

func TestAIGTacticBorrowedGoalsPolicyAndTranslation(t *testing.T) {
	m, r := aigTacticFixture(t)
	goal := aigAdapterGoal(t, m, "sharing", true)
	form, err := goal.Form(0)
	aigMust(t, err)
	dep, err := m.DependencyLeaf(form)
	aigMust(t, err)
	aigMust(t, goal.Update(0, form, ast.ASTHandle{}, dep))
	refs := goal.RefCount
	var result tactic.TacticResult
	defer func() { aigMust(t, result.Close()) }()
	aigMust(t, r.Apply(goal, &result))
	if goal.RefCount != refs || len(result.Goals) != 1 || result.Goals[0] != goal {
		t.Fatal("tactic must borrow the caller's goal", goal.RefCount, refs)
	}
	gotDep, err := goal.Dependency(0)
	aigMust(t, err)
	if gotDep != dep {
		t.Fatal("dependency lost")
	}
	aigMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
	proofGoal, err := tactic.NewGoal(m, true, false, false)
	aigMust(t, err)
	t.Cleanup(func() { aigMust(t, proofGoal.Close()) })
	if err := r.Apply(proofGoal, &result); err == nil || err.Error() != "aig does not support proof production" {
		t.Fatal("source proof refusal", err)
	}
	if len(result.Goals) != 1 || result.Goals[0] != goal {
		t.Fatal("proof refusal reset the previous result")
	}
	aigMust(t, result.Reset())
	aigMust(t, goal.CheckOwner())
	p, err := r.context.NewValue()
	aigMust(t, err)
	defer func() { aigMust(t, r.context.Del(&p)) }()
	key := "aig_per_assertion"
	aigMust(t, r.context.SetBoolName(&p, &key, false))
	key = "aig_default_gate_encoding"
	aigMust(t, r.context.SetBoolName(&p, &key, false))
	aigMust(t, r.UpdateParams(&p))
	key = "aig_per_assertion"
	aigMust(t, r.context.SetBoolName(&p, &key, true)) // scalar policy must already be copied
	if r.pass.PerAssertion || r.pass.DefaultGateEncoding {
		t.Fatal("policy was not read immediately")
	}
	size, err := goal.Size()
	aigMust(t, err)
	if err := r.Apply(goal, &result); err == nil || err.Error() != "aig does not support unsat core production" {
		t.Fatal("whole-goal core refusal", err)
	}
	after, err := goal.Size()
	aigMust(t, err)
	if size != after || len(result.Goals) != 0 {
		t.Fatal("whole-goal refusal changed input or retained result", size, after)
	}
	destination, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: m.Symbols, Numerals: m.NumeralService(), Budget: m.Budget()})
	aigMust(t, err)
	t.Cleanup(func() { aigMust(t, destination.Close()) })
	other, err := r.Translate(destination)
	aigMust(t, err)
	translated := other.(*Tactic)
	t.Cleanup(func() { aigMust(t, translated.Close()) })
	aigMust(t, r.Close())
	if translated.pass.PerAssertion || translated.pass.DefaultGateEncoding {
		t.Fatal("translation lost scalar policy")
	}
	targetGoal := aigAdapterGoal(t, destination, "ite", false)
	var targetResult tactic.TacticResult
	defer func() { aigMust(t, targetResult.Close()) }()
	aigMust(t, translated.Apply(targetGoal, &targetResult))
	aigMust(t, translated.Cleanup())
	if translated.pass.manager != nil || len(translated.pass.pendingRefs) != 0 {
		t.Fatal("successful cleanup retained invocation")
	}
	aigMust(t, translated.IncRef())
	aigMust(t, translated.DecRef())
	if err := translated.Cleanup(); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("closed owner", err)
	}
}

func TestAIGTacticAllocationFrontiers(t *testing.T) {
	for _, per := range []bool{false, true} {
		success := false
		for stop := int64(0); stop < 1024; stop++ {
			t.Run(fmt.Sprint(per, "/", stop), func(t *testing.T) {
				m, r := aigTacticFixture(t)
				r.pass.PerAssertion = per
				goal := aigAdapterGoal(t, m, "ite", per)
				var result tactic.TacticResult
				defer func() { aigMust(t, result.Close()) }()
				m.Budget().FailAfter(stop)
				err := r.Apply(goal, &result)
				m.Budget().FailAfter(-1)
				if err == nil {
					success = true
				} else if !errors.Is(err, arena.ErrInjectedFailure) {
					t.Fatal(err)
				}
				aigMust(t, r.Cleanup())
			})
			if success {
				break
			}
			if t.Failed() {
				t.Fatal("failed frontier", per, stop)
			}
		}
		if !success {
			t.Fatal("no successful frontier", per)
		}
	}
	for stop := int64(0); stop < 64; stop++ {
		m, r := aigTacticFixture(t)
		m.Budget().FailAfter(stop)
		fresh, err := NewTactic(r.context, nil, nil)
		m.Budget().FailAfter(-1)
		if fresh != nil {
			aigMust(t, fresh.Close())
		}
		if err == nil {
			return
		}
		if !errors.Is(err, arena.ErrInjectedFailure) {
			t.Fatal(err)
		}
	}
	t.Fatal("no successful constructor frontier")
}

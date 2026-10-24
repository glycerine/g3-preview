package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/ast/macros"
	"github.com/glycerine/g3/pkg/util/lbool"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestSMTAssertedDriverCancellationAndResume(t *testing.T) {
	m, s := smtAssertedNNFFixture(t, false)
	b, err := m.MkBoolSort()
	astTestMust(t, err)
	p, err := m.MkConst(astTestSymbol(t, m, "driver-cancel-p"), b)
	astTestMust(t, err)
	astTestMust(t, s.Assert(p))
	k := s.kernel
	phase := k.assertedElimAnd
	m.Limit.Cancel()
	astTestMust(t, k.internalizePending())
	if k.assertedHead != 0 || len(k.assertions) != 1 || k.assertedElimAnd != phase {
		t.Fatal("canceled preprocessing committed or changed its pending assertions")
	}
	m.Limit.ResetCancel()
	status, err := s.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.True || k.assertedHead != len(k.assertions) {
		t.Fatal("pending assertion was not resumed", status, k.assertedHead)
	}
}

func TestSMTAssertedDriverQuantifierFlagSurvivesRewriteAndPop(t *testing.T) {
	m, s := smtAssertedNNFFixture(t, false)
	k := s.kernel
	astTestMust(t, s.Push())
	b, err := m.MkBoolSort()
	astTestMust(t, err)
	truth, err := m.MkTrue()
	astTestMust(t, err)
	q, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{b}, Names: []symbol.Symbol{symbol.Numerical(0)}, Body: truth})
	astTestMust(t, err)
	astTestMust(t, s.Assert(q))
	if !k.assertedHasQuantifiers || len(k.assertions) != 0 {
		t.Fatal("original quantified input must set the flag even when rewritten to true")
	}
	astTestMust(t, s.Pop(1))
	if !k.assertedHasQuantifiers {
		t.Fatal("scope pop cleared the source's sticky quantifier flag")
	}
	k.preprocessState.MacroFinder = true
	if !k.shouldApplyAssertedPass(assertedMacros) {
		t.Fatal("pass guard ignored a previously seen quantifier")
	}
}

func TestSMTAssertedDriverInconsistencyStopsAssertionAndRestoresScope(t *testing.T) {
	for _, proofs := range []bool{false, true} {
		m, s := smtAssertedNNFFixture(t, proofs)
		k := s.kernel
		astTestMust(t, s.Push())
		falsity, err := m.MkFalse()
		astTestMust(t, err)
		astTestMust(t, s.Assert(falsity))
		b, err := m.MkBoolSort()
		astTestMust(t, err)
		q, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{b}, Names: []symbol.Symbol{symbol.Numerical(0)}, Body: falsity})
		astTestMust(t, err)
		astTestMust(t, s.Assert(q))
		if len(k.assertions) != 1 || k.assertedHasQuantifiers {
			t.Fatal("assertion after inconsistency changed the formula manager")
		}
		before := m.Limit.Count()
		ready, err := k.reduceAssertedPending()
		astTestMust(t, err)
		if !ready || m.Limit.Count() != before {
			t.Fatal("inconsistent driver should stop before cancellation or passes")
		}
		status, err := s.CheckSat(nil)
		astTestMust(t, err)
		if status != lbool.False {
			t.Fatal("false assertion was not internalized", status)
		}
		astTestMust(t, s.Pop(1))
		if k.assertedFormulasInconsistent() {
			t.Fatal("popping false assertion retained inconsistency")
		}
		status, err = s.CheckSat(nil)
		astTestMust(t, err)
		if status != lbool.True {
			t.Fatal("solver did not recover after popping inconsistency", status)
		}
	}
}

func TestSMTAssertedDriverSkippedPassDoesNotPollCancellation(t *testing.T) {
	m, s := smtAssertedNNFFixture(t, false)
	k := s.kernel
	k.preprocessState.MacroFinder = true
	m.Limit.Cancel()
	defer m.Limit.ResetCancel()
	before := m.Limit.Count()
	proceed, canceled, err := k.invokeAssertedPass(assertedMacros)
	astTestMust(t, err)
	if !proceed || canceled || m.Limit.Count() != before {
		t.Fatal("skipped pass polled cancellation")
	}
	proceed, canceled, err = k.invokeAssertedPass(assertedReduce)
	astTestMust(t, err)
	if proceed || !canceled || m.Limit.Count() != before+1 {
		t.Fatal("active pass omitted its final cancellation check")
	}
}

func TestSMTAssertedDriverStopsBeforeNNFOnPropagationConflict(t *testing.T) {
	m, s := smtAssertedNNFFixture(t, false)
	b, err := m.MkBoolSort()
	astTestMust(t, err)
	p, err := m.MkConst(astTestSymbol(t, m, "driver-stop-p"), b)
	astTestMust(t, err)
	astTestMust(t, s.Assert(p))
	np, err := m.MkNot(p)
	astTestMust(t, err)
	astTestMust(t, s.Assert(np))
	k := s.kernel
	k.preprocessState.PropagateValues = true
	ready, err := k.reduceAssertedPending()
	astTestMust(t, err)
	if !ready || !k.assertedFormulasInconsistent() || k.assertedElimAnd {
		t.Fatal("propagation conflict did not stop before the post-NNF phase")
	}
}

func TestSMTAssertedMacroPostOperationDoesNotPropagateValues(t *testing.T) {
	m, s := smtAssertedNNFFixture(t, false)
	b, err := m.MkBoolSort()
	astTestMust(t, err)
	p, err := m.MkConst(astTestSymbol(t, m, "driver-post-p"), b)
	astTestMust(t, err)
	np, err := m.MkNot(p)
	astTestMust(t, err)
	batch, err := macros.NewFormulaBatch(m)
	astTestMust(t, err)
	defer func() { astTestMust(t, batch.Close()) }()
	for _, h := range []ast.ASTHandle{p, np} {
		astTestMust(t, batch.Insert(macros.Formula{Expression: h}))
	}
	astTestMust(t, s.kernel.replacePendingMacroBatch(batch))
	if s.kernel.assertedFormulasInconsistent() || len(s.kernel.assertions) != 2 {
		t.Fatal("macro post-operation ran an extra propagation pass")
	}
	status, err := s.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.False {
		t.Fatal("normal preprocessing missed the contradiction", status)
	}
}

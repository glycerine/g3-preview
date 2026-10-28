package smtcore

// Source: src/smt/theory_seq.h stats and
// src/smt/theory_seq.cpp collect_statistics.

import (
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
	"github.com/glycerine/g3/pkg/util/statistics"
)

func smtSequenceStatisticsFixture(t *testing.T) (*SMTSequenceTheory, *SMTInternalizerState) {
	t.Helper()
	m := astTestManager(t)
	family, err := m.SequenceFamilyID()
	astTestMust(t, err)
	theory, err := NewSMTSequenceTheory(family)
	astTestMust(t, err)
	_, _, state := smtInternalizerFixture(t, m, nil)
	astTestMust(t, state.RegisterTheory(theory))
	return theory, state
}

func TestSMTSequenceStatisticsSourceKeysOrderAndGuards(t *testing.T) {
	theory, _ := smtSequenceStatisticsFixture(t)
	theory.statistics = smtSequenceStatistics{
		numSplits:            1,
		numReductions:        2,
		propagateAutomata:    3,
		checkLengthCoherence: 4,
		branchVariable:       5,
		solveDisequalities:   6,
		solveEqualities:      7,
		addAxiom:             8,
		extensionality:       9,
		fixedLength:          10,
		propagateContains:    99,
		integerString:        11,
	}
	out := statistics.New()
	astTestMust(t, theory.CollectStatistics(out))
	want := []string{
		"seq num splits",
		"seq num reductions",
		"seq unfold def",
		"seq length coherence",
		"seq branch",
		"seq solve !=",
		"seq solve =",
		"seq add axiom",
		"seq extensionality",
		"seq fixed length",
		"seq int.to.str",
	}
	if out.Size() != len(want) {
		t.Fatal("source sequence statistic count", out.Size())
	}
	for i, key := range want {
		if string(out.Key(i)) != key || !out.IsUint(i) || out.UintValue(i) != uint32(i+1) {
			t.Fatal("source sequence statistic key/order/value", i, out.Key(i), out.UintValue(i))
		}
	}
	if err := theory.CollectStatistics(nil); err != ast.ErrASTHandle {
		t.Fatal("nil statistics target", err)
	}
	theory.ResetStatistics()
	out.Reset()
	astTestMust(t, theory.CollectStatistics(out))
	if out.Size() != 0 {
		t.Fatal("reset sequence statistics", out.Size())
	}
	uninitialized, err := NewSMTSequenceTheory(theory.FamilyID())
	astTestMust(t, err)
	if err = uninitialized.CollectStatistics(statistics.New()); err != ast.ErrASTHandle {
		t.Fatal("uninitialized sequence statistics owner", err)
	}
}

func TestSMTSequenceStatisticsCountUniqueAxiomsAndReductions(t *testing.T) {
	theory, state := smtSequenceStatisticsFixture(t)
	boolean, err := state.manager.MkBoolSort()
	astTestMust(t, err)
	formula, err := state.manager.MkConst(astTestSymbol(t, state.manager, "sequence-statistic-axiom"), boolean)
	astTestMust(t, err)
	theory.statistics.addAxiom = math.MaxUint32
	astTestMust(t, theory.axioms.Push(formula))
	astTestMust(t, theory.axioms.Push(formula))
	if theory.statistics.addAxiom != 0 {
		t.Fatal("deduplicated sequence axiom count", theory.statistics.addAxiom)
	}
	word, err := state.manager.MkString("a")
	astTestMust(t, err)
	astTestMust(t, theory.equations.Push(1, []ast.ASTHandle{word}, []ast.ASTHandle{word}, nil))
	theory.statistics.numReductions = math.MaxUint32
	astTestMust(t, theory.removeEquation(0))
	if theory.statistics.numReductions != 0 {
		t.Fatal("sequence equation reduction count", theory.statistics.numReductions)
	}
}

func TestSMTSequenceStatisticsFlowThroughSolverCollection(t *testing.T) {
	m, config := smtModelGeneratorFixture(t)
	sample, err := m.MkString("")
	astTestMust(t, err)
	sort, err := m.GetSort(sample)
	astTestMust(t, err)
	x, err := m.MkConst(astTestSymbol(t, m, "sequence-statistic-solver-x"), sort)
	astTestMust(t, err)
	length, err := m.MkSequenceLength(x)
	astTestMust(t, err)
	zero, err := m.ArithmeticIntegerNumeralText("0")
	astTestMust(t, err)
	nonnegative, err := m.MkArithmeticGE(length, zero)
	astTestMust(t, err)
	solver, err := NewSMTSolver(m, nil, "ALL", SMTSolverConfig{Params: config.Context, ModelEvaluator: config})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, solver.Close()) })
	astTestMust(t, solver.Assert(nonnegative))
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.True {
		t.Fatal("sequence statistic solver fixture", status)
	}
	out := statistics.New()
	astTestMust(t, solver.CollectStatistics(out))
	found := false
	for i := 0; i < out.Size(); i++ {
		if out.Key(i) == "seq add axiom" {
			found = out.IsUint(i) && out.UintValue(i) != 0
		}
	}
	if !found {
		t.Fatal("solver omitted sequence theory statistics")
	}
}

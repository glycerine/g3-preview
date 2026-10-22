package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSMTEmptySequenceTheoryScopesAndModelFactory(t *testing.T) {
	m := astTestManager(t)
	_, err := m.MkString("")
	astTestMust(t, err)
	family, err := m.SequenceFamilyID()
	astTestMust(t, err)
	theory, err := NewSMTEmptySequenceTheory(family)
	astTestMust(t, err)
	_, _, state := smtInternalizerFixture(t, m, nil)
	astTestMust(t, state.RegisterTheory(theory))
	if theory.Name() != "seq-empty" || theory.UseDisequalities() {
		t.Fatal("empty sequence theory metadata", theory.Name(), theory.UseDisequalities())
	}
	status, err := theory.FinalCheck()
	astTestMust(t, err)
	if status != SMTFinalCheckDone {
		t.Fatal("unused sequence theory", status)
	}
	astTestMust(t, state.PushScope())
	if internalized, e := theory.InternalizeTerm(ast.ASTHandle{}); e != nil || internalized {
		t.Fatal("empty sequence term", internalized, e)
	}
	status, err = theory.FinalCheck()
	astTestMust(t, err)
	if status != SMTFinalCheckGiveUp {
		t.Fatal("used sequence theory", status)
	}
	astTestMust(t, state.PopScopes(1))
	status, err = theory.FinalCheck()
	astTestMust(t, err)
	if status != SMTFinalCheckDone {
		t.Fatal("sequence usage did not roll back", status)
	}
	fresh, err := theory.Fresh()
	astTestMust(t, err)
	if fresh == theory || fresh.FamilyID() != family {
		t.Fatal("fresh sequence theory", fresh)
	}
}

func TestSMTSequenceLogicsInstallSafeCheckpointTheory(t *testing.T) {
	for _, logic := range []string{"QF_BVRE", "QF_S", "ALL"} {
		t.Run(logic, func(t *testing.T) {
			m, modelConfig := smtModelGeneratorFixture(t)
			value, err := m.MkString("a")
			astTestMust(t, err)
			sort, err := m.GetSort(value)
			astTestMust(t, err)
			x, err := m.MkConst(astTestSymbol(t, m, "sequence-checkpoint-"+logic), sort)
			astTestMust(t, err)
			equality, err := m.MkEq(x, value)
			astTestMust(t, err)
			solver, err := NewSMTSolver(m, nil, logic, SMTSolverConfig{Params: modelConfig.Context, ModelEvaluator: modelConfig})
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, solver.Close()) })
			family, err := m.SequenceFamilyID()
			astTestMust(t, err)
			if _, ok := solver.kernel.internalizer.theories[family].(*SMTSequenceTheory); !ok {
				t.Fatal("sequence theory was not installed")
			}
			astTestMust(t, solver.Assert(equality))
			status, err := solver.CheckSat(nil)
			astTestMust(t, err)
			if status != lbool.True {
				t.Fatal("sequence equality fragment did not return sat", status)
			}
		})
	}
}

func TestSMTSequenceTheoryLiteralConflictAndUnsupportedBoundary(t *testing.T) {
	for _, logic := range []string{"QF_S", "ALL"} {
		t.Run(logic, func(t *testing.T) {
			m, modelConfig := smtModelGeneratorFixture(t)
			a, err := m.MkString("a")
			astTestMust(t, err)
			b, err := m.MkString("b")
			astTestMust(t, err)
			equality, err := m.MkEq(a, b)
			astTestMust(t, err)
			solver, err := NewSMTSolver(m, nil, logic, SMTSolverConfig{Params: modelConfig.Context, ModelEvaluator: modelConfig})
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, solver.Close()) })
			astTestMust(t, solver.Assert(equality))
			status, err := solver.CheckSat(nil)
			astTestMust(t, err)
			if status != lbool.False {
				t.Fatal("distinct sequence literals", status)
			}
		})
	}

	m, modelConfig := smtModelGeneratorFixture(t)
	a, err := m.MkString("a")
	astTestMust(t, err)
	sort, err := m.GetSort(a)
	astTestMust(t, err)
	y, err := m.MkConst(astTestSymbol(t, m, "sequence-symbolic-part"), sort)
	astTestMust(t, err)
	concat, err := m.MkSequenceConcat(y, a)
	astTestMust(t, err)
	x, err := m.MkConst(astTestSymbol(t, m, "sequence-unsupported"), sort)
	astTestMust(t, err)
	equality, err := m.MkEq(x, concat)
	astTestMust(t, err)
	solver, err := NewSMTSolver(m, nil, "QF_S", SMTSolverConfig{Params: modelConfig.Context, ModelEvaluator: modelConfig})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, solver.Close()) })
	astTestMust(t, solver.Assert(equality))
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.Undef {
		t.Fatal("unported concatenation returned a definitive result", status)
	}
}

func TestSMTSequenceTheoryConcreteWordEquationConflict(t *testing.T) {
	m, modelConfig := smtModelGeneratorFixture(t)
	a, err := m.MkString("a")
	astTestMust(t, err)
	ba, err := m.MkString("ba")
	astTestMust(t, err)
	c, err := m.MkString("c")
	astTestMust(t, err)
	sort, err := m.GetSort(a)
	astTestMust(t, err)
	x, err := m.MkConst(astTestSymbol(t, m, "sequence-word-x"), sort)
	astTestMust(t, err)
	word, err := m.MkSequenceConcat(x, a)
	astTestMust(t, err)
	wordEquality, err := m.MkEq(word, ba)
	astTestMust(t, err)
	valueEquality, err := m.MkEq(x, c)
	astTestMust(t, err)
	solver, err := NewSMTSolver(m, nil, "QF_S", SMTSolverConfig{Params: modelConfig.Context, ModelEvaluator: modelConfig})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, solver.Close()) })
	astTestMust(t, solver.Assert(wordEquality))
	astTestMust(t, solver.Assert(valueEquality))
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.False {
		t.Fatal("concrete sequence word conflict", status)
	}
}

func TestSMTSequenceTheoryConcreteWordDisequationConflict(t *testing.T) {
	m, modelConfig := smtModelGeneratorFixture(t)
	a, err := m.MkString("a")
	astTestMust(t, err)
	ba, err := m.MkString("ba")
	astTestMust(t, err)
	b, err := m.MkString("b")
	astTestMust(t, err)
	sort, err := m.GetSort(a)
	astTestMust(t, err)
	x, err := m.MkConst(astTestSymbol(t, m, "sequence-diseq-x"), sort)
	astTestMust(t, err)
	word, err := m.MkSequenceConcat(x, a)
	astTestMust(t, err)
	wordEquality, err := m.MkEq(word, ba)
	astTestMust(t, err)
	wordDisequality, err := m.MkNot(wordEquality)
	astTestMust(t, err)
	valueEquality, err := m.MkEq(x, b)
	astTestMust(t, err)
	solver, err := NewSMTSolver(m, nil, "QF_S", SMTSolverConfig{Params: modelConfig.Context, ModelEvaluator: modelConfig})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, solver.Close()) })
	astTestMust(t, solver.Assert(wordDisequality))
	astTestMust(t, solver.Assert(valueEquality))
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.False {
		t.Fatal("concrete sequence word disequality conflict", status)
	}
}

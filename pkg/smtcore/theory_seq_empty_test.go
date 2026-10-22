package smtcore

import (
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSMTSequenceStructuredModelValues(t *testing.T) {
	m, config := smtModelGeneratorFixture(t)
	a, err := m.MkString("a")
	astTestMust(t, err)
	sort, err := m.GetSort(a)
	astTestMust(t, err)
	x, err := m.MkConst(astTestSymbol(t, m, "sequence-model-x"), sort)
	astTestMust(t, err)
	concat, err := m.MkSequenceConcat(x, a)
	astTestMust(t, err)
	family, err := m.SequenceFamilyID()
	astTestMust(t, err)
	theory, err := NewSMTSequenceTheory(family)
	astTestMust(t, err)
	context, _, state := smtInternalizerFixture(t, m, nil)
	astTestMust(t, state.RegisterTheory(theory))
	astTestMust(t, state.Internalize(concat, false))
	astTestMust(t, context.SetRelevant(x, true))
	astTestMust(t, context.SetRelevant(concat, true))
	generator, err := NewSMTModelGenerator(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, generator.Close()) })
	astTestMust(t, generator.SetContext(context))
	astTestMust(t, generator.InitializeModel(config))
	astTestMust(t, generator.BuildStateValues(state))
	xValue, err := generator.Value(context.enodes[x])
	astTestMust(t, err)
	concatValue, err := generator.Value(context.enodes[concat])
	astTestMust(t, err)
	xText, xString, err := m.SequenceStringValue(xValue)
	astTestMust(t, err)
	concatText, concatString, err := m.SequenceStringValue(concatValue)
	astTestMust(t, err)
	if !xString || !concatString || concatText != xText+"a" {
		t.Fatal("structured concat model value", xText, concatText)
	}

	byteValue, err := m.BitVectorNumeralUnsigned('A', 8)
	astTestMust(t, err)
	integerValue, err := m.ArithmeticNumeralRat(big.NewRat(12, 1), true)
	astTestMust(t, err)
	procedure := &smtSequenceValueProcedure{
		sort:         sort,
		sources:      []smtSequenceModelSource{{kind: smtSequenceModelUnit}, {kind: smtSequenceModelInteger}},
		dependencies: make([]SMTModelValueDependency, 2),
	}
	value, err := procedure.MakeValue(generator, []ast.ASTHandle{byteValue, integerValue})
	astTestMust(t, err)
	if text, matched, textErr := m.SequenceStringValue(value); textErr != nil || !matched || text != "A12" {
		t.Fatal("unit and integer model sources", text, matched, textErr)
	}
}

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

func TestSMTSequenceTheoryConcreteNonContainsConflict(t *testing.T) {
	m, modelConfig := smtModelGeneratorFixture(t)
	abc, err := m.MkString("abc")
	astTestMust(t, err)
	b, err := m.MkString("b")
	astTestMust(t, err)
	sort, err := m.GetSort(abc)
	astTestMust(t, err)
	x, err := m.MkConst(astTestSymbol(t, m, "sequence-contains-x"), sort)
	astTestMust(t, err)
	contains, err := m.MkSequenceContains(x, b)
	astTestMust(t, err)
	notContains, err := m.MkNot(contains)
	astTestMust(t, err)
	valueEquality, err := m.MkEq(x, abc)
	astTestMust(t, err)
	solver, err := NewSMTSolver(m, nil, "QF_S", SMTSolverConfig{Params: modelConfig.Context, ModelEvaluator: modelConfig})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, solver.Close()) })
	astTestMust(t, solver.Assert(notContains))
	astTestMust(t, solver.Assert(valueEquality))
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.False {
		t.Fatal("concrete non-contains conflict", status)
	}
}

func TestSMTSequenceTheoryCancelsWordBoundaryAndPropagatesSolution(t *testing.T) {
	m, modelConfig := smtModelGeneratorFixture(t)
	a, err := m.MkString("a")
	astTestMust(t, err)
	ab, err := m.MkString("ab")
	astTestMust(t, err)
	b, err := m.MkString("b")
	astTestMust(t, err)
	sort, err := m.GetSort(a)
	astTestMust(t, err)
	x, err := m.MkConst(astTestSymbol(t, m, "sequence-cancel-x"), sort)
	astTestMust(t, err)
	word, err := m.MkSequenceConcat(a, x)
	astTestMust(t, err)
	wordEquality, err := m.MkEq(word, ab)
	astTestMust(t, err)
	xEqualsB, err := m.MkEq(x, b)
	astTestMust(t, err)
	xNotB, err := m.MkNot(xEqualsB)
	astTestMust(t, err)
	solver, err := NewSMTSolver(m, nil, "QF_S", SMTSolverConfig{Params: modelConfig.Context, ModelEvaluator: modelConfig})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, solver.Close()) })
	astTestMust(t, solver.Assert(wordEquality))
	astTestMust(t, solver.Assert(xNotB))
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.False {
		t.Fatal("cancelled word solution did not conflict", status)
	}
}

func TestSMTSequenceTheoryCancelsEquivalentEnodeBoundaries(t *testing.T) {
	m, modelConfig := smtModelGeneratorFixture(t)
	a, err := m.MkString("a")
	astTestMust(t, err)
	b, err := m.MkString("b")
	astTestMust(t, err)
	sort, err := m.GetSort(a)
	astTestMust(t, err)
	x, err := m.MkConst(astTestSymbol(t, m, "sequence-enode-cancel-x"), sort)
	astTestMust(t, err)
	y, err := m.MkConst(astTestSymbol(t, m, "sequence-enode-cancel-y"), sort)
	astTestMust(t, err)
	left, err := m.MkSequenceConcat(x, a)
	astTestMust(t, err)
	right, err := m.MkSequenceConcat(y, b)
	astTestMust(t, err)
	variablesEqual, err := m.MkEq(x, y)
	astTestMust(t, err)
	wordsEqual, err := m.MkEq(left, right)
	astTestMust(t, err)
	solver, err := NewSMTSolver(m, nil, "QF_S", SMTSolverConfig{Params: modelConfig.Context, ModelEvaluator: modelConfig})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, solver.Close()) })
	astTestMust(t, solver.Assert(variablesEqual))
	astTestMust(t, solver.Assert(wordsEqual))
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.False {
		t.Fatal("equivalent e-node boundary cancellation", status)
	}
}

func TestSMTSequenceTheoryReducesEqualLengthBoundaries(t *testing.T) {
	m, modelConfig := smtModelGeneratorFixture(t)
	a, err := m.MkString("a")
	astTestMust(t, err)
	b, err := m.MkString("b")
	astTestMust(t, err)
	sort, err := m.GetSort(a)
	astTestMust(t, err)
	x, err := m.MkConst(astTestSymbol(t, m, "sequence-length-cancel-x"), sort)
	astTestMust(t, err)
	y, err := m.MkConst(astTestSymbol(t, m, "sequence-length-cancel-y"), sort)
	astTestMust(t, err)
	xLength, err := m.MkSequenceLength(x)
	astTestMust(t, err)
	yLength, err := m.MkSequenceLength(y)
	astTestMust(t, err)
	lengthsEqual, err := m.MkEq(xLength, yLength)
	astTestMust(t, err)
	left, err := m.MkSequenceConcat(x, a)
	astTestMust(t, err)
	right, err := m.MkSequenceConcat(y, b)
	astTestMust(t, err)
	wordsEqual, err := m.MkEq(left, right)
	astTestMust(t, err)
	solver, err := NewSMTSolver(m, nil, "QF_S", SMTSolverConfig{Params: modelConfig.Context, ModelEvaluator: modelConfig})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, solver.Close()) })
	astTestMust(t, solver.Assert(lengthsEqual))
	astTestMust(t, solver.Assert(wordsEqual))
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.False {
		t.Fatal("equal-length boundary reduction", status)
	}
}

func TestSMTSequenceTheoryBranchesBinaryUnitEquation(t *testing.T) {
	m, modelConfig := smtModelGeneratorFixture(t)
	a, err := m.MkString("a")
	astTestMust(t, err)
	b, err := m.MkString("b")
	astTestMust(t, err)
	sort, err := m.GetSort(a)
	astTestMust(t, err)
	x, err := m.MkConst(astTestSymbol(t, m, "sequence-binary-x"), sort)
	astTestMust(t, err)
	y, err := m.MkConst(astTestSymbol(t, m, "sequence-binary-y"), sort)
	astTestMust(t, err)
	left, err := m.MkSequenceConcat(x, a)
	astTestMust(t, err)
	right, err := m.MkSequenceConcat(b, y)
	astTestMust(t, err)
	wordEquality, err := m.MkEq(left, right)
	astTestMust(t, err)
	tailName, err := m.Symbols.Intern("seq.binary.tail")
	astTestMust(t, err)
	tail, err := m.MkSequenceSkolem(tailName, []ast.ASTHandle{x, y, a, b}, sort)
	astTestMust(t, err)
	xValue, err := m.MkSequenceConcat(b, tail)
	astTestMust(t, err)
	xEquality, err := m.MkEq(x, xValue)
	astTestMust(t, err)
	notXEquality, err := m.MkNot(xEquality)
	astTestMust(t, err)
	solver, err := NewSMTSolver(m, nil, "QF_S", SMTSolverConfig{Params: modelConfig.Context, ModelEvaluator: modelConfig})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, solver.Close()) })
	astTestMust(t, solver.Assert(wordEquality))
	astTestMust(t, solver.Assert(notXEquality))
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.False {
		t.Fatal("binary unit equation branch", status)
	}
}

func TestSMTSequenceTheoryBranchesFixedStringOverlaps(t *testing.T) {
	m, modelConfig := smtModelGeneratorFixture(t)
	a, err := m.MkString("abc")
	astTestMust(t, err)
	b, err := m.MkString("ab")
	astTestMust(t, err)
	empty, err := m.MkString("")
	astTestMust(t, err)
	c, err := m.MkString("c")
	astTestMust(t, err)
	sort, err := m.GetSort(a)
	astTestMust(t, err)
	x, err := m.MkConst(astTestSymbol(t, m, "sequence-overlap-x"), sort)
	astTestMust(t, err)
	y, err := m.MkConst(astTestSymbol(t, m, "sequence-overlap-y"), sort)
	astTestMust(t, err)
	left, err := m.MkSequenceConcat(x, a)
	astTestMust(t, err)
	right, err := m.MkSequenceConcat(b, y)
	astTestMust(t, err)
	wordEquality, err := m.MkEq(left, right)
	astTestMust(t, err)
	shortLeft, err := m.MkEq(x, empty)
	astTestMust(t, err)
	shortRight, err := m.MkEq(y, c)
	astTestMust(t, err)
	shortAlternative, err := m.MkAnd(shortLeft, shortRight)
	astTestMust(t, err)
	notShort, err := m.MkNot(shortAlternative)
	astTestMust(t, err)
	tailName, err := m.Symbols.Intern("seq.binary.tail")
	astTestMust(t, err)
	tail, err := m.MkSequenceSkolem(tailName, []ast.ASTHandle{x, y, a, b}, sort)
	astTestMust(t, err)
	longX, err := m.MkSequenceConcat(b, tail)
	astTestMust(t, err)
	longY, err := m.MkSequenceConcat(tail, a)
	astTestMust(t, err)
	longLeft, err := m.MkEq(x, longX)
	astTestMust(t, err)
	longRight, err := m.MkEq(y, longY)
	astTestMust(t, err)
	longAlternative, err := m.MkAnd(longLeft, longRight)
	astTestMust(t, err)
	notLong, err := m.MkNot(longAlternative)
	astTestMust(t, err)
	solver, err := NewSMTSolver(m, nil, "QF_S", SMTSolverConfig{Params: modelConfig.Context, ModelEvaluator: modelConfig})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, solver.Close()) })
	astTestMust(t, solver.Assert(wordEquality))
	astTestMust(t, solver.Assert(notShort))
	astTestMust(t, solver.Assert(notLong))
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.False {
		t.Fatal("fixed string overlap alternatives", status)
	}
}

func TestSMTSequenceTheoryConcreteRelationConflicts(t *testing.T) {
	for _, test := range []struct {
		name  string
		op    int32
		left  string
		right string
		value bool
	}{
		{name: "prefix-positive", op: 3, left: "bc", right: "abc", value: true},
		{name: "suffix-negative", op: 4, left: "bc", right: "abc", value: false},
		{name: "contains-positive", op: 5, left: "abc", right: "z", value: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			m, modelConfig := smtModelGeneratorFixture(t)
			left, err := m.MkString(test.left)
			astTestMust(t, err)
			right, err := m.MkString(test.right)
			astTestMust(t, err)
			sort, err := m.GetSort(left)
			astTestMust(t, err)
			x, err := m.MkConst(astTestSymbol(t, m, "sequence-relation-"+test.name), sort)
			astTestMust(t, err)
			var relation, value ast.ASTHandle
			switch test.op {
			case 3:
				relation, err = m.MkSequencePrefix(left, x)
				value = right
			case 4:
				relation, err = m.MkSequenceSuffix(left, x)
				value = right
			case 5:
				relation, err = m.MkSequenceContains(x, right)
				value = left
			}
			astTestMust(t, err)
			binding, err := m.MkEq(x, value)
			astTestMust(t, err)
			assertion := relation
			if !test.value {
				assertion, err = m.MkNot(relation)
				astTestMust(t, err)
			}
			solver, err := NewSMTSolver(m, nil, "QF_S", SMTSolverConfig{Params: modelConfig.Context, ModelEvaluator: modelConfig})
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, solver.Close()) })
			astTestMust(t, solver.Assert(assertion))
			astTestMust(t, solver.Assert(binding))
			status, err := solver.CheckSat(nil)
			astTestMust(t, err)
			if status != lbool.False {
				t.Fatal("concrete relation contradiction", status)
			}
		})
	}
}

func TestSMTSequenceTheoryRelationWitnessAndNonemptyAxioms(t *testing.T) {
	for _, test := range []struct {
		name      string
		operation int32
		skolem    string
	}{
		{name: "prefix", operation: 3, skolem: "seq.p.suffix"},
		{name: "suffix", operation: 4, skolem: "seq.s.prefix"},
		{name: "contains", operation: 5, skolem: "seq.indexof_left"},
	} {
		t.Run(test.name, func(t *testing.T) {
			m, modelConfig := smtModelGeneratorFixture(t)
			empty, err := m.MkString("")
			astTestMust(t, err)
			sort, err := m.GetSort(empty)
			astTestMust(t, err)
			left, err := m.MkConst(astTestSymbol(t, m, "sequence-relation-left-"+test.name), sort)
			astTestMust(t, err)
			right, err := m.MkConst(astTestSymbol(t, m, "sequence-relation-right-"+test.name), sort)
			astTestMust(t, err)
			var relation ast.ASTHandle
			switch test.operation {
			case 3:
				relation, err = m.MkSequencePrefix(left, right)
			case 4:
				relation, err = m.MkSequenceSuffix(left, right)
			case 5:
				relation, err = m.MkSequenceContains(left, right)
			}
			astTestMust(t, err)
			name, err := m.Symbols.Intern(test.skolem)
			astTestMust(t, err)
			first, err := m.MkSequenceSkolem(name, []ast.ASTHandle{left, right}, sort)
			astTestMust(t, err)
			var word ast.ASTHandle
			switch test.operation {
			case 3:
				word, err = m.MkSequenceConcat(left, first)
			case 4:
				word, err = m.MkSequenceConcat(first, left)
			case 5:
				rightName, symbolErr := m.Symbols.Intern("seq.indexof_right")
				astTestMust(t, symbolErr)
				last, makeErr := m.MkSequenceSkolem(rightName, []ast.ASTHandle{left, right}, sort)
				astTestMust(t, makeErr)
				tail, makeErr := m.MkSequenceConcat(right, last)
				astTestMust(t, makeErr)
				word, err = m.MkSequenceConcat(first, tail)
			}
			astTestMust(t, err)
			target := right
			if test.operation == 5 {
				target = left
			}
			definition, err := m.MkEq(word, target)
			astTestMust(t, err)
			notDefinition, err := m.MkNot(definition)
			astTestMust(t, err)
			solver, err := NewSMTSolver(m, nil, "QF_S", SMTSolverConfig{Params: modelConfig.Context, ModelEvaluator: modelConfig})
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, solver.Close()) })
			astTestMust(t, solver.Assert(relation))
			astTestMust(t, solver.Assert(notDefinition))
			status, err := solver.CheckSat(nil)
			astTestMust(t, err)
			if status != lbool.False {
				t.Fatal("relation witness contradiction", status)
			}
		})
	}

	m, modelConfig := smtModelGeneratorFixture(t)
	empty, err := m.MkString("")
	astTestMust(t, err)
	sort, err := m.GetSort(empty)
	astTestMust(t, err)
	y, err := m.MkConst(astTestSymbol(t, m, "sequence-empty-prefix-right"), sort)
	astTestMust(t, err)
	prefix, err := m.MkSequencePrefix(empty, y)
	astTestMust(t, err)
	notPrefix, err := m.MkNot(prefix)
	astTestMust(t, err)
	solver, err := NewSMTSolver(m, nil, "QF_S", SMTSolverConfig{Params: modelConfig.Context, ModelEvaluator: modelConfig})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, solver.Close()) })
	astTestMust(t, solver.Assert(notPrefix))
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.False {
		t.Fatal("empty sequence is always a prefix", status)
	}
}

func TestSMTSequenceTheoryNonContainsExcludesBoundaryMatches(t *testing.T) {
	for _, boundary := range []string{"prefix", "suffix"} {
		t.Run(boundary, func(t *testing.T) {
			m, modelConfig := smtModelGeneratorFixture(t)
			empty, err := m.MkString("")
			astTestMust(t, err)
			sort, err := m.GetSort(empty)
			astTestMust(t, err)
			haystack, err := m.MkConst(astTestSymbol(t, m, "sequence-nc-haystack-"+boundary), sort)
			astTestMust(t, err)
			needle, err := m.MkConst(astTestSymbol(t, m, "sequence-nc-needle-"+boundary), sort)
			astTestMust(t, err)
			contains, err := m.MkSequenceContains(haystack, needle)
			astTestMust(t, err)
			notContains, err := m.MkNot(contains)
			astTestMust(t, err)
			var relation ast.ASTHandle
			if boundary == "prefix" {
				relation, err = m.MkSequencePrefix(needle, haystack)
			} else {
				relation, err = m.MkSequenceSuffix(needle, haystack)
			}
			astTestMust(t, err)
			solver, err := NewSMTSolver(m, nil, "QF_S", SMTSolverConfig{Params: modelConfig.Context, ModelEvaluator: modelConfig})
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, solver.Close()) })
			astTestMust(t, solver.Assert(notContains))
			astTestMust(t, solver.Assert(relation))
			status, err := solver.CheckSat(nil)
			astTestMust(t, err)
			if status != lbool.False {
				t.Fatal("non-containment boundary match", status)
			}
		})
	}
}

func TestSMTSequenceTheoryNegativeBoundaryWitnesses(t *testing.T) {
	for _, suffix := range []bool{false, true} {
		name := "prefix"
		if suffix {
			name = "suffix"
		}
		t.Run(name, func(t *testing.T) {
			m, modelConfig := smtModelGeneratorFixture(t)
			empty, err := m.MkString("")
			astTestMust(t, err)
			sort, err := m.GetSort(empty)
			astTestMust(t, err)
			elementSort, sequence, err := m.SequenceSortElement(sort)
			astTestMust(t, err)
			if !sequence {
				t.Fatal("string sequence sort")
			}
			boundary, err := m.MkConst(astTestSymbol(t, m, "sequence-negative-boundary-"+name), sort)
			astTestMust(t, err)
			container, err := m.MkConst(astTestSymbol(t, m, "sequence-negative-container-"+name), sort)
			astTestMust(t, err)
			var relation ast.ASTHandle
			if suffix {
				relation, err = m.MkSequenceSuffix(boundary, container)
			} else {
				relation, err = m.MkSequencePrefix(boundary, container)
			}
			astTestMust(t, err)
			notRelation, err := m.MkNot(relation)
			astTestMust(t, err)
			containerEmpty, err := m.MkEq(container, empty)
			astTestMust(t, err)
			containerNonempty, err := m.MkNot(containerEmpty)
			astTestMust(t, err)
			prefix := "seq.prefix."
			if suffix {
				prefix = "seq.suffix."
			}
			arguments := []ast.ASTHandle{boundary, container}
			xName, err := m.Symbols.Intern(prefix + "x")
			astTestMust(t, err)
			x, err := m.MkSequenceSkolem(xName, arguments, sort)
			astTestMust(t, err)
			yName, err := m.Symbols.Intern(prefix + "y")
			astTestMust(t, err)
			y, err := m.MkSequenceSkolem(yName, arguments, sort)
			astTestMust(t, err)
			cName, err := m.Symbols.Intern(prefix + "c")
			astTestMust(t, err)
			c, err := m.MkSequenceSkolem(cName, arguments, elementSort)
			astTestMust(t, err)
			unit, err := m.MkSequenceUnit(c)
			astTestMust(t, err)
			var word ast.ASTHandle
			if suffix {
				tail, makeErr := m.MkSequenceConcat(unit, x)
				astTestMust(t, makeErr)
				word, err = m.MkSequenceConcat(y, tail)
			} else {
				tail, makeErr := m.MkSequenceConcat(unit, y)
				astTestMust(t, makeErr)
				word, err = m.MkSequenceConcat(x, tail)
			}
			astTestMust(t, err)
			definition, err := m.MkEq(boundary, word)
			astTestMust(t, err)
			notDefinition, err := m.MkNot(definition)
			astTestMust(t, err)
			solver, err := NewSMTSolver(m, nil, "QF_S", SMTSolverConfig{Params: modelConfig.Context, ModelEvaluator: modelConfig})
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, solver.Close()) })
			astTestMust(t, solver.Assert(notRelation))
			astTestMust(t, solver.Assert(containerNonempty))
			astTestMust(t, solver.Assert(notDefinition))
			status, err := solver.CheckSat(nil)
			astTestMust(t, err)
			if status != lbool.False {
				t.Fatal("negative boundary witness", status)
			}
		})
	}
}

func TestSMTSequenceTheoryAtBoundaryAxioms(t *testing.T) {
	for _, test := range []struct {
		name  string
		index string
		word  string
	}{
		{name: "negative-index", index: "-1", word: "symbolic"},
		{name: "past-end", index: "1", word: ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			m, modelConfig := smtModelGeneratorFixture(t)
			seed, err := m.MkString(test.word)
			astTestMust(t, err)
			sort, err := m.GetSort(seed)
			astTestMust(t, err)
			sequence := seed
			if test.word == "symbolic" {
				sequence, err = m.MkConst(astTestSymbol(t, m, "sequence-at-negative"), sort)
				astTestMust(t, err)
			}
			index, err := m.ArithmeticIntegerNumeralText(test.index)
			astTestMust(t, err)
			at, err := m.MkSequenceAt(sequence, index)
			astTestMust(t, err)
			empty, err := m.MkSequenceEmpty(sort)
			astTestMust(t, err)
			equality, err := m.MkEq(at, empty)
			astTestMust(t, err)
			notEmpty, err := m.MkNot(equality)
			astTestMust(t, err)
			solver, err := NewSMTSolver(m, nil, "QF_S", SMTSolverConfig{Params: modelConfig.Context, ModelEvaluator: modelConfig})
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, solver.Close()) })
			astTestMust(t, solver.Assert(notEmpty))
			status, err := solver.CheckSat(nil)
			astTestMust(t, err)
			if status != lbool.False {
				t.Fatal("at boundary value", status)
			}
		})
	}

	m, modelConfig := smtModelGeneratorFixture(t)
	empty, err := m.MkString("")
	astTestMust(t, err)
	sort, err := m.GetSort(empty)
	astTestMust(t, err)
	sequence, err := m.MkConst(astTestSymbol(t, m, "sequence-at-valid"), sort)
	astTestMust(t, err)
	zero, err := m.ArithmeticIntegerNumeralText("0")
	astTestMust(t, err)
	one, err := m.ArithmeticIntegerNumeralText("1")
	astTestMust(t, err)
	at, err := m.MkSequenceAt(sequence, zero)
	astTestMust(t, err)
	prefixName, err := m.Symbols.Intern("seq.pre")
	astTestMust(t, err)
	prefix, err := m.MkSequenceSkolem(prefixName, []ast.ASTHandle{sequence, zero}, sort)
	astTestMust(t, err)
	suffixName, err := m.Symbols.Intern("seq.post")
	astTestMust(t, err)
	suffix, err := m.MkSequenceSkolem(suffixName, []ast.ASTHandle{sequence, zero}, sort)
	astTestMust(t, err)
	tail, err := m.MkSequenceConcat(at, suffix)
	astTestMust(t, err)
	word, err := m.MkSequenceConcat(prefix, tail)
	astTestMust(t, err)
	definition, err := m.MkEq(sequence, word)
	astTestMust(t, err)
	notDefinition, err := m.MkNot(definition)
	astTestMust(t, err)
	length, err := m.MkSequenceLength(sequence)
	astTestMust(t, err)
	lengthOne, err := m.MkEq(length, one)
	astTestMust(t, err)
	solver, err := NewSMTSolver(m, nil, "QF_S", SMTSolverConfig{Params: modelConfig.Context, ModelEvaluator: modelConfig})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, solver.Close()) })
	astTestMust(t, solver.Assert(lengthOne))
	astTestMust(t, solver.Assert(notDefinition))
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.False {
		t.Fatal("at in-bounds decomposition", status)
	}
}

func TestSMTSequenceTheoryExtractBoundaryAndDecompositionAxioms(t *testing.T) {
	for _, test := range []struct{ name, offset, length string }{
		{name: "negative-offset", offset: "-1", length: "1"},
		{name: "zero-length", offset: "0", length: "0"},
		{name: "negative-length", offset: "0", length: "-1"},
	} {
		t.Run(test.name, func(t *testing.T) {
			m, modelConfig := smtModelGeneratorFixture(t)
			empty, err := m.MkString("")
			astTestMust(t, err)
			sort, err := m.GetSort(empty)
			astTestMust(t, err)
			sequence, err := m.MkConst(astTestSymbol(t, m, "sequence-extract-"+test.name), sort)
			astTestMust(t, err)
			offset, err := m.ArithmeticIntegerNumeralText(test.offset)
			astTestMust(t, err)
			length, err := m.ArithmeticIntegerNumeralText(test.length)
			astTestMust(t, err)
			extract, err := m.MkSequenceExtract(sequence, offset, length)
			astTestMust(t, err)
			equality, err := m.MkEq(extract, empty)
			astTestMust(t, err)
			notEmpty, err := m.MkNot(equality)
			astTestMust(t, err)
			solver, err := NewSMTSolver(m, nil, "QF_S", SMTSolverConfig{Params: modelConfig.Context, ModelEvaluator: modelConfig})
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, solver.Close()) })
			astTestMust(t, solver.Assert(notEmpty))
			status, err := solver.CheckSat(nil)
			astTestMust(t, err)
			if status != lbool.False {
				t.Fatal("extract boundary value", status)
			}
		})
	}

	m, modelConfig := smtModelGeneratorFixture(t)
	empty, err := m.MkString("")
	astTestMust(t, err)
	sort, err := m.GetSort(empty)
	astTestMust(t, err)
	sequence, err := m.MkConst(astTestSymbol(t, m, "sequence-extract-valid"), sort)
	astTestMust(t, err)
	zero, err := m.ArithmeticIntegerNumeralText("0")
	astTestMust(t, err)
	one, err := m.ArithmeticIntegerNumeralText("1")
	astTestMust(t, err)
	extract, err := m.MkSequenceExtract(sequence, zero, one)
	astTestMust(t, err)
	sequenceLength, err := m.MkSequenceLength(sequence)
	astTestMust(t, err)
	remaining, err := m.MkArithmeticSub(sequenceLength, zero)
	astTestMust(t, err)
	postLength, err := m.MkArithmeticSub(remaining, one)
	astTestMust(t, err)
	prefixName, err := m.Symbols.Intern("seq.pre")
	astTestMust(t, err)
	prefix, err := m.MkSequenceSkolem(prefixName, []ast.ASTHandle{sequence, zero}, sort)
	astTestMust(t, err)
	suffixName, err := m.Symbols.Intern("seq.post")
	astTestMust(t, err)
	suffix, err := m.MkSequenceSkolem(suffixName, []ast.ASTHandle{sequence, postLength}, sort)
	astTestMust(t, err)
	tail, err := m.MkSequenceConcat(extract, suffix)
	astTestMust(t, err)
	word, err := m.MkSequenceConcat(prefix, tail)
	astTestMust(t, err)
	definition, err := m.MkEq(sequence, word)
	astTestMust(t, err)
	notDefinition, err := m.MkNot(definition)
	astTestMust(t, err)
	nonnegative, err := m.MkArithmeticGE(zero, zero)
	astTestMust(t, err)
	pastEnd, err := m.MkArithmeticGE(zero, sequenceLength)
	astTestMust(t, err)
	inside, err := m.MkNot(pastEnd)
	astTestMust(t, err)
	solver, err := NewSMTSolver(m, nil, "QF_S", SMTSolverConfig{Params: modelConfig.Context, ModelEvaluator: modelConfig})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, solver.Close()) })
	astTestMust(t, solver.Assert(nonnegative))
	astTestMust(t, solver.Assert(inside))
	astTestMust(t, solver.Assert(notDefinition))
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.False {
		t.Fatal("extract in-bounds decomposition", status)
	}
}

func TestSMTSequenceTheoryIndexAxioms(t *testing.T) {
	for _, test := range []struct {
		name   string
		offset string
		empty  bool
		absent bool
		want   string
	}{
		{name: "negative-offset", offset: "-1", want: "-1"},
		{name: "empty-needle", offset: "0", empty: true, want: "0"},
		{name: "absent", offset: "0", absent: true, want: "-1"},
	} {
		t.Run(test.name, func(t *testing.T) {
			m, modelConfig := smtModelGeneratorFixture(t)
			empty, err := m.MkString("")
			astTestMust(t, err)
			sort, err := m.GetSort(empty)
			astTestMust(t, err)
			source, err := m.MkConst(astTestSymbol(t, m, "sequence-index-source-"+test.name), sort)
			astTestMust(t, err)
			needle := empty
			if !test.empty {
				needle, err = m.MkConst(astTestSymbol(t, m, "sequence-index-needle-"+test.name), sort)
				astTestMust(t, err)
			}
			offset, err := m.ArithmeticIntegerNumeralText(test.offset)
			astTestMust(t, err)
			index, err := m.MkSequenceIndex(source, needle, offset)
			astTestMust(t, err)
			want, err := m.ArithmeticIntegerNumeralText(test.want)
			astTestMust(t, err)
			equality, err := m.MkEq(index, want)
			astTestMust(t, err)
			notEquality, err := m.MkNot(equality)
			astTestMust(t, err)
			solver, err := NewSMTSolver(m, nil, "QF_S", SMTSolverConfig{Params: modelConfig.Context, ModelEvaluator: modelConfig})
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, solver.Close()) })
			if test.absent {
				contains, makeErr := m.MkSequenceContains(source, needle)
				astTestMust(t, makeErr)
				notContains, makeErr := m.MkNot(contains)
				astTestMust(t, makeErr)
				astTestMust(t, solver.Assert(notContains))
			}
			astTestMust(t, solver.Assert(notEquality))
			status, err := solver.CheckSat(nil)
			astTestMust(t, err)
			if status != lbool.False {
				t.Fatal("index boundary result", status)
			}
		})
	}

	m, modelConfig := smtModelGeneratorFixture(t)
	empty, err := m.MkString("")
	astTestMust(t, err)
	sort, err := m.GetSort(empty)
	astTestMust(t, err)
	source, err := m.MkConst(astTestSymbol(t, m, "sequence-index-witness-source"), sort)
	astTestMust(t, err)
	needle, err := m.MkConst(astTestSymbol(t, m, "sequence-index-witness-needle"), sort)
	astTestMust(t, err)
	zero, err := m.ArithmeticIntegerNumeralText("0")
	astTestMust(t, err)
	index, err := m.MkSequenceIndex(source, needle, zero)
	astTestMust(t, err)
	contains, err := m.MkSequenceContains(source, needle)
	astTestMust(t, err)
	prefixName, err := m.Symbols.Intern("seq.indexof_left")
	astTestMust(t, err)
	prefix, err := m.MkSequenceSkolem(prefixName, []ast.ASTHandle{source, needle}, sort)
	astTestMust(t, err)
	prefixLength, err := m.MkSequenceLength(prefix)
	astTestMust(t, err)
	indexLength, err := m.MkEq(index, prefixLength)
	astTestMust(t, err)
	notIndexLength, err := m.MkNot(indexLength)
	astTestMust(t, err)
	emptyNeedle, err := m.MkEq(needle, empty)
	astTestMust(t, err)
	notEmptyNeedle, err := m.MkNot(emptyNeedle)
	astTestMust(t, err)
	solver, err := NewSMTSolver(m, nil, "QF_S", SMTSolverConfig{Params: modelConfig.Context, ModelEvaluator: modelConfig})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, solver.Close()) })
	astTestMust(t, solver.Assert(contains))
	astTestMust(t, solver.Assert(notEmptyNeedle))
	astTestMust(t, solver.Assert(notIndexLength))
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.False {
		t.Fatal("zero-offset index witness", status)
	}
}

func TestSMTSequenceTheoryReplaceAxioms(t *testing.T) {
	t.Run("empty-needle", func(t *testing.T) {
		m, modelConfig := smtModelGeneratorFixture(t)
		empty, err := m.MkString("")
		astTestMust(t, err)
		sort, err := m.GetSort(empty)
		astTestMust(t, err)
		source, err := m.MkConst(astTestSymbol(t, m, "sequence-replace-empty-source"), sort)
		astTestMust(t, err)
		replacement, err := m.MkConst(astTestSymbol(t, m, "sequence-replace-empty-value"), sort)
		astTestMust(t, err)
		result, err := m.MkSequenceReplace(source, empty, replacement)
		astTestMust(t, err)
		word, err := m.MkSequenceConcat(replacement, source)
		astTestMust(t, err)
		equality, err := m.MkEq(result, word)
		astTestMust(t, err)
		contradiction, err := m.MkNot(equality)
		astTestMust(t, err)
		solver, err := NewSMTSolver(m, nil, "QF_S", SMTSolverConfig{Params: modelConfig.Context, ModelEvaluator: modelConfig})
		astTestMust(t, err)
		t.Cleanup(func() { astTestMust(t, solver.Close()) })
		astTestMust(t, solver.Assert(contradiction))
		status, err := solver.CheckSat(nil)
		astTestMust(t, err)
		if status != lbool.False {
			t.Fatal("empty needle replacement", status)
		}
	})

	for _, found := range []bool{false, true} {
		name := "absent"
		if found {
			name = "found"
		}
		t.Run(name, func(t *testing.T) {
			m, modelConfig := smtModelGeneratorFixture(t)
			empty, err := m.MkString("")
			astTestMust(t, err)
			sort, err := m.GetSort(empty)
			astTestMust(t, err)
			source, err := m.MkConst(astTestSymbol(t, m, "sequence-replace-source-"+name), sort)
			astTestMust(t, err)
			needle, err := m.MkConst(astTestSymbol(t, m, "sequence-replace-needle-"+name), sort)
			astTestMust(t, err)
			replacement, err := m.MkConst(astTestSymbol(t, m, "sequence-replace-value-"+name), sort)
			astTestMust(t, err)
			result, err := m.MkSequenceReplace(source, needle, replacement)
			astTestMust(t, err)
			contains, err := m.MkSequenceContains(source, needle)
			astTestMust(t, err)
			assertion := contains
			want := source
			if found {
				prefixName, makeErr := m.Symbols.Intern("seq.indexof_left")
				astTestMust(t, makeErr)
				prefix, makeErr := m.MkSequenceSkolem(prefixName, []ast.ASTHandle{source, needle}, sort)
				astTestMust(t, makeErr)
				suffixName, makeErr := m.Symbols.Intern("seq.indexof_right")
				astTestMust(t, makeErr)
				suffix, makeErr := m.MkSequenceSkolem(suffixName, []ast.ASTHandle{source, needle}, sort)
				astTestMust(t, makeErr)
				tail, makeErr := m.MkSequenceConcat(replacement, suffix)
				astTestMust(t, makeErr)
				want, makeErr = m.MkSequenceConcat(prefix, tail)
				astTestMust(t, makeErr)
			} else {
				assertion, err = m.MkNot(contains)
				astTestMust(t, err)
			}
			equality, err := m.MkEq(result, want)
			astTestMust(t, err)
			notEquality, err := m.MkNot(equality)
			astTestMust(t, err)
			solver, err := NewSMTSolver(m, nil, "QF_S", SMTSolverConfig{Params: modelConfig.Context, ModelEvaluator: modelConfig})
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, solver.Close()) })
			astTestMust(t, solver.Assert(assertion))
			if found {
				sourceEmpty, makeErr := m.MkEq(source, empty)
				astTestMust(t, makeErr)
				needleEmpty, makeErr := m.MkEq(needle, empty)
				astTestMust(t, makeErr)
				notSourceEmpty, makeErr := m.MkNot(sourceEmpty)
				astTestMust(t, makeErr)
				notNeedleEmpty, makeErr := m.MkNot(needleEmpty)
				astTestMust(t, makeErr)
				astTestMust(t, solver.Assert(notSourceEmpty))
				astTestMust(t, solver.Assert(notNeedleEmpty))
			}
			astTestMust(t, solver.Assert(notEquality))
			status, err := solver.CheckSat(nil)
			astTestMust(t, err)
			if status != lbool.False {
				t.Fatal("replace definition", status)
			}
		})
	}
}

func TestSMTSequenceTheoryRegexMembershipReduction(t *testing.T) {
	t.Run("singleton-symbolic", func(t *testing.T) {
		m, modelConfig := smtModelGeneratorFixture(t)
		literal, err := m.MkString("ab")
		astTestMust(t, err)
		sort, err := m.GetSort(literal)
		astTestMust(t, err)
		x, err := m.MkConst(astTestSymbol(t, m, "sequence-regex-singleton"), sort)
		astTestMust(t, err)
		regex, err := m.MkSequenceToRegex(literal)
		astTestMust(t, err)
		membership, err := m.MkSequenceInRegex(x, regex)
		astTestMust(t, err)
		equality, err := m.MkEq(x, literal)
		astTestMust(t, err)
		disequality, err := m.MkNot(equality)
		astTestMust(t, err)
		solver, err := NewSMTSolver(m, nil, "QF_S", SMTSolverConfig{Params: modelConfig.Context, ModelEvaluator: modelConfig})
		astTestMust(t, err)
		t.Cleanup(func() { astTestMust(t, solver.Close()) })
		astTestMust(t, solver.Assert(membership))
		astTestMust(t, solver.Assert(disequality))
		status, err := solver.CheckSat(nil)
		astTestMust(t, err)
		if status != lbool.False {
			t.Fatal("singleton membership reduction", status)
		}
	})

	t.Run("concrete-union-rejection", func(t *testing.T) {
		m, modelConfig := smtModelGeneratorFixture(t)
		input, err := m.MkString("c")
		astTestMust(t, err)
		a, err := m.MkString("a")
		astTestMust(t, err)
		b, err := m.MkString("b")
		astTestMust(t, err)
		ra, err := m.MkSequenceToRegex(a)
		astTestMust(t, err)
		rb, err := m.MkSequenceToRegex(b)
		astTestMust(t, err)
		union, err := m.MkRegexUnion(ra, rb)
		astTestMust(t, err)
		membership, err := m.MkSequenceInRegex(input, union)
		astTestMust(t, err)
		solver, err := NewSMTSolver(m, nil, "QF_S", SMTSolverConfig{Params: modelConfig.Context, ModelEvaluator: modelConfig})
		astTestMust(t, err)
		t.Cleanup(func() { astTestMust(t, solver.Close()) })
		astTestMust(t, solver.Assert(membership))
		status, err := solver.CheckSat(nil)
		astTestMust(t, err)
		if status != lbool.False {
			t.Fatal("concrete regex rejection", status)
		}
	})
}

func TestSMTSequenceTheoryLengthAxioms(t *testing.T) {
	for _, test := range []struct {
		name  string
		build func(*testing.T, *ast.ASTManager) ast.ASTHandle
	}{
		{name: "string", build: func(t *testing.T, m *ast.ASTManager) ast.ASTHandle {
			value, err := m.MkString("abc")
			astTestMust(t, err)
			length, err := m.MkSequenceLength(value)
			astTestMust(t, err)
			three, err := m.ArithmeticIntegerNumeralText("3")
			astTestMust(t, err)
			equality, err := m.MkEq(length, three)
			astTestMust(t, err)
			result, err := m.MkNot(equality)
			astTestMust(t, err)
			return result
		}},
		{name: "concat", build: func(t *testing.T, m *ast.ASTManager) ast.ASTHandle {
			literal, err := m.MkString("ab")
			astTestMust(t, err)
			sort, err := m.GetSort(literal)
			astTestMust(t, err)
			x, err := m.MkConst(astTestSymbol(t, m, "sequence-length-x"), sort)
			astTestMust(t, err)
			word, err := m.MkSequenceConcat(x, literal)
			astTestMust(t, err)
			wordLength, err := m.MkSequenceLength(word)
			astTestMust(t, err)
			xLength, err := m.MkSequenceLength(x)
			astTestMust(t, err)
			two, err := m.ArithmeticIntegerNumeralText("2")
			astTestMust(t, err)
			sum, err := m.MkArithmeticAdd(xLength, two)
			astTestMust(t, err)
			equality, err := m.MkEq(wordLength, sum)
			astTestMust(t, err)
			result, err := m.MkNot(equality)
			astTestMust(t, err)
			return result
		}},
		{name: "nonnegative", build: func(t *testing.T, m *ast.ASTManager) ast.ASTHandle {
			empty, err := m.MkString("")
			astTestMust(t, err)
			sort, err := m.GetSort(empty)
			astTestMust(t, err)
			x, err := m.MkConst(astTestSymbol(t, m, "sequence-nonnegative-x"), sort)
			astTestMust(t, err)
			length, err := m.MkSequenceLength(x)
			astTestMust(t, err)
			zero, err := m.ArithmeticIntegerNumeralText("0")
			astTestMust(t, err)
			nonnegative, err := m.MkArithmeticGE(length, zero)
			astTestMust(t, err)
			result, err := m.MkNot(nonnegative)
			astTestMust(t, err)
			return result
		}},
		{name: "zero-iff-empty", build: func(t *testing.T, m *ast.ASTManager) ast.ASTHandle {
			empty, err := m.MkString("")
			astTestMust(t, err)
			sort, err := m.GetSort(empty)
			astTestMust(t, err)
			x, err := m.MkConst(astTestSymbol(t, m, "sequence-zero-length-x"), sort)
			astTestMust(t, err)
			length, err := m.MkSequenceLength(x)
			astTestMust(t, err)
			zero, err := m.ArithmeticIntegerNumeralText("0")
			astTestMust(t, err)
			zeroLength, err := m.MkEq(length, zero)
			astTestMust(t, err)
			emptyValue, err := m.MkEq(x, empty)
			astTestMust(t, err)
			notEmpty, err := m.MkNot(emptyValue)
			astTestMust(t, err)
			result, err := m.MkAnd(zeroLength, notEmpty)
			astTestMust(t, err)
			return result
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			m, modelConfig := smtModelGeneratorFixture(t)
			assertion := test.build(t, m)
			solver, err := NewSMTSolver(m, nil, "QF_S", SMTSolverConfig{Params: modelConfig.Context, ModelEvaluator: modelConfig})
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, solver.Close()) })
			astTestMust(t, solver.Assert(assertion))
			status, err := solver.CheckSat(nil)
			astTestMust(t, err)
			if status != lbool.False {
				t.Fatal("sequence length axiom contradiction", status)
			}
		})
	}
}

func TestSMTSequenceTheoryIntegerStringAxioms(t *testing.T) {
	for _, test := range []struct {
		name  string
		build func(*testing.T, *ast.ASTManager) []ast.ASTHandle
	}{
		{name: "negative-is-empty", build: func(t *testing.T, m *ast.ASTManager) []ast.ASTHandle {
			minusOne, err := m.ArithmeticIntegerNumeralText("-1")
			astTestMust(t, err)
			sort, err := m.GetSort(minusOne)
			astTestMust(t, err)
			n, err := m.MkConst(astTestSymbol(t, m, "sequence-itos-negative"), sort)
			astTestMust(t, err)
			binding, err := m.MkEq(n, minusOne)
			astTestMust(t, err)
			converted, err := m.MkIntToString(n)
			astTestMust(t, err)
			empty, err := m.MkString("")
			astTestMust(t, err)
			equality, err := m.MkEq(converted, empty)
			astTestMust(t, err)
			result, err := m.MkNot(equality)
			astTestMust(t, err)
			return []ast.ASTHandle{binding, result}
		}},
		{name: "nonnegative-roundtrip", build: func(t *testing.T, m *ast.ASTManager) []ast.ASTHandle {
			twelve, err := m.ArithmeticIntegerNumeralText("12")
			astTestMust(t, err)
			sort, err := m.GetSort(twelve)
			astTestMust(t, err)
			n, err := m.MkConst(astTestSymbol(t, m, "sequence-itos-positive"), sort)
			astTestMust(t, err)
			binding, err := m.MkEq(n, twelve)
			astTestMust(t, err)
			converted, err := m.MkIntToString(n)
			astTestMust(t, err)
			back, err := m.MkStringToInt(converted)
			astTestMust(t, err)
			equality, err := m.MkEq(back, twelve)
			astTestMust(t, err)
			result, err := m.MkNot(equality)
			astTestMust(t, err)
			return []ast.ASTHandle{binding, result}
		}},
		{name: "stoi-lower-bound", build: func(t *testing.T, m *ast.ASTManager) []ast.ASTHandle {
			value, err := m.MkString("not-a-number")
			astTestMust(t, err)
			sort, err := m.GetSort(value)
			astTestMust(t, err)
			x, err := m.MkConst(astTestSymbol(t, m, "sequence-stoi-string"), sort)
			astTestMust(t, err)
			binding, err := m.MkEq(x, value)
			astTestMust(t, err)
			converted, err := m.MkStringToInt(x)
			astTestMust(t, err)
			minusOne, err := m.ArithmeticIntegerNumeralText("-1")
			astTestMust(t, err)
			lowerBound, err := m.MkArithmeticGE(converted, minusOne)
			astTestMust(t, err)
			result, err := m.MkNot(lowerBound)
			astTestMust(t, err)
			return []ast.ASTHandle{binding, result}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			m, modelConfig := smtModelGeneratorFixture(t)
			assertions := test.build(t, m)
			solver, err := NewSMTSolver(m, nil, "QF_S", SMTSolverConfig{Params: modelConfig.Context, ModelEvaluator: modelConfig})
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, solver.Close()) })
			for _, assertion := range assertions {
				astTestMust(t, solver.Assert(assertion))
			}
			status, err := solver.CheckSat(nil)
			astTestMust(t, err)
			if status != lbool.False {
				t.Fatal("integer/string axiom contradiction", status)
			}
		})
	}
}

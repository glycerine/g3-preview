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

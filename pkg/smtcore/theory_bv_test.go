package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
	uparams "github.com/glycerine/g3/pkg/util/params"
	"github.com/glycerine/g3/pkg/util/statistics"
)

func smtBVSolver(t *testing.T) (*ast.ASTManager, ast.FamilyID, *SMTSolver) {
	t.Helper()
	m := astTestManager(t)
	name, err := m.Symbols.Intern("bv")
	if err != nil {
		t.Fatal("intern BV family:", err)
	}
	family, err := m.FamilyIDForSymbol(name)
	if err != nil {
		t.Fatal("resolve BV family:", err)
	}
	params, err := uparams.NewParamsContext(m.Symbols, m.NumeralService(), uparams.ParamsConfig{Budget: m.Budget()})
	if err != nil {
		t.Fatal("create params:", err)
	}
	solver, err := NewSMTSolver(m, nil, "QF_BV", SMTSolverConfig{Params: params})
	if err != nil {
		t.Fatal("create BV solver:", err)
	}
	t.Cleanup(func() {
		astTestMust(t, solver.Close())
		astTestMust(t, params.Close())
	})
	return m, family, solver
}

func TestSMTBitVectorNumeralEquality(t *testing.T) {
	for _, test := range []struct {
		name        string
		left, right uint64
		want        lbool.LBool
	}{
		{name: "same", left: 5, right: 5, want: lbool.True},
		{name: "different", left: 3, right: 5, want: lbool.False},
	} {
		t.Run(test.name, func(t *testing.T) {
			m, _, solver := smtBVSolver(t)
			left, err := m.BitVectorNumeralUint64(test.left, 4)
			astTestMust(t, err)
			right, err := m.BitVectorNumeralUint64(test.right, 4)
			astTestMust(t, err)
			equality, err := m.MkEq(left, right)
			astTestMust(t, err)
			astTestMust(t, solver.Assert(equality))
			status, err := solver.CheckSat(nil)
			astTestMust(t, err)
			if status != test.want {
				reason, _ := solver.ReasonUnknown()
				t.Fatal("numeral equality", status, "want", test.want, reason)
			}
		})
	}
}

func TestSMTBitVectorNumeralBit2Bool(t *testing.T) {
	m, _, solver := smtBVSolver(t)
	value, err := m.BitVectorNumeralUint64(2, 3)
	astTestMust(t, err)
	zeroBit, err := m.MkBitVectorBit2Bool(0, value)
	astTestMust(t, err)
	astTestMust(t, solver.Assert(zeroBit))
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.False {
		t.Fatal("false numeral bit", status)
	}
}

func TestSMTBitVectorAdditionCircuit(t *testing.T) {
	m, _, solver := smtBVSolver(t)
	one, err := m.BitVectorNumeralUint64(1, 4)
	astTestMust(t, err)
	sort, err := m.GetSort(one)
	astTestMust(t, err)
	left, err := m.MkConst(astTestSymbol(t, m, "unsupported-bv-left"), sort)
	astTestMust(t, err)
	sum, err := m.MkBitVectorPolynomialAdd(sort, ast.SliceArguments([]ast.ASTHandle{left, one}))
	astTestMust(t, err)
	right, err := m.MkConst(astTestSymbol(t, m, "unsupported-bv-right"), sort)
	astTestMust(t, err)
	equality, err := m.MkEq(sum, right)
	astTestMust(t, err)
	leftValue, err := m.MkEq(left, one)
	astTestMust(t, err)
	two, err := m.BitVectorNumeralUint64(2, 4)
	astTestMust(t, err)
	rightValue, err := m.MkEq(right, two)
	astTestMust(t, err)
	wrongRightValue, err := m.MkNot(rightValue)
	astTestMust(t, err)
	astTestMust(t, solver.Assert(equality))
	astTestMust(t, solver.Assert(leftValue))
	astTestMust(t, solver.Assert(wrongRightValue))
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.False {
		t.Fatal("BV addition circuit", status)
	}
	stats := statistics.New()
	astTestMust(t, solver.CollectStatistics(stats))
	found := false
	for i := 0; i < stats.Size(); i++ {
		if stats.Key(i) == "bv dynamic eqs" {
			found = stats.IsUint(i) && stats.UintValue(i) != 0
		}
	}
	if !found {
		t.Fatal("missing BV equality statistics")
	}
	astTestMust(t, solver.kernel.ResetStatistics())
	stats.Reset()
	astTestMust(t, solver.CollectStatistics(stats))
	for i := 0; i < stats.Size(); i++ {
		if stats.Key(i) == "bv dynamic eqs" {
			t.Fatal("BV statistics survived reset")
		}
	}
}

func TestSMTBitVectorExpandsDynamicDisequality(t *testing.T) {
	m, _, solver := smtBVSolver(t)
	zero, err := m.BitVectorNumeralUint64(0, 4)
	astTestMust(t, err)
	sort, err := m.GetSort(zero)
	astTestMust(t, err)
	left, err := m.MkConst(astTestSymbol(t, m, "bv-diseq-left"), sort)
	astTestMust(t, err)
	right, err := m.MkConst(astTestSymbol(t, m, "bv-diseq-right"), sort)
	astTestMust(t, err)
	equality, err := m.MkEq(left, right)
	astTestMust(t, err)
	disequality, err := m.MkNot(equality)
	astTestMust(t, err)
	astTestMust(t, solver.Assert(disequality))
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.True {
		t.Fatal("bit-vector disequality", status)
	}
	stats := statistics.New()
	astTestMust(t, solver.CollectStatistics(stats))
	found := false
	for i := 0; i < stats.Size(); i++ {
		if stats.Key(i) == "bv dynamic diseqs" {
			found = stats.IsUint(i) && stats.UintValue(i) != 0
		}
	}
	if !found {
		t.Fatal("missing dynamic bit-vector disequality statistic")
	}
}

func TestSMTBitVectorExpandsNewEquality(t *testing.T) {
	m, family, solver := smtBVSolver(t)
	family, err := m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	zero, err := m.BitVectorNumeralUint64(0, 4)
	astTestMust(t, err)
	sort, err := m.GetSort(zero)
	astTestMust(t, err)
	left, err := m.MkConst(astTestSymbol(t, m, "bv-new-eq-left"), sort)
	astTestMust(t, err)
	right, err := m.MkConst(astTestSymbol(t, m, "bv-new-eq-right"), sort)
	astTestMust(t, err)
	state := solver.kernel.internalizer
	astTestMust(t, state.Internalize(left, false))
	astTestMust(t, state.Internalize(right, false))
	leftNode, rightNode := state.context.enodes[left], state.context.enodes[right]
	theory, ok := state.theories[family].(*SMTBitVectorTheory)
	if !ok || theory == nil {
		t.Fatal("missing bit-vector theory")
	}
	before := len(state.Clauses())
	astTestMust(t, theory.NewEquality(leftNode.TheoryVar(family), rightNode.TheoryVar(family)))
	if len(state.Clauses()) <= before || theory.statistics.eqDynamic == 0 {
		t.Fatal("new equality did not publish bit equivalence", before, len(state.Clauses()), theory.statistics.eqDynamic)
	}
}

func TestSMTBitVectorDivisionByZeroValueIsUninterpreted(t *testing.T) {
	m, _, solver := smtBVSolver(t)
	one, err := m.BitVectorNumeralUint64(1, 4)
	astTestMust(t, err)
	sort, err := m.GetSort(one)
	astTestMust(t, err)
	left, err := m.MkConst(astTestSymbol(t, m, "unsupported-div-left"), sort)
	astTestMust(t, err)
	division, err := m.MkBitVectorSRemZero(left)
	astTestMust(t, err)
	astTestMust(t, m.IncRef(division))
	equality, err := m.MkEq(division, left)
	astTestMust(t, err)
	astTestMust(t, m.DecRef(division))
	astTestMust(t, solver.Assert(equality))
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.True {
		t.Fatal("uninterpreted BV division-by-zero value", status)
	}
	family, err := m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	theory, ok := solver.kernel.internalizer.theories[family].(*SMTBitVectorTheory)
	if !ok || theory == nil {
		t.Fatal("missing bit-vector theory")
	}
	declaration, err := m.GetDecl(division)
	astTestMust(t, err)
	leftDeclaration, err := m.GetDecl(left)
	astTestMust(t, err)
	if !theory.IncludeFunctionInterpretation(declaration) || theory.IncludeFunctionInterpretation(leftDeclaration) {
		t.Fatal("division-by-zero model interpretation selection")
	}
}

func TestSMTBitVectorCloseReleasesBitOwners(t *testing.T) {
	m, _, solver := smtBVSolver(t)
	family, err := m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	one, err := m.BitVectorNumeralUint64(1, 4)
	astTestMust(t, err)
	sort, err := m.GetSort(one)
	astTestMust(t, err)
	value, err := m.MkConst(astTestSymbol(t, m, "bv-close-value"), sort)
	astTestMust(t, err)
	equality, err := m.MkEq(value, one)
	astTestMust(t, err)
	astTestMust(t, solver.Assert(equality))
	_, err = solver.CheckSat(nil)
	astTestMust(t, err)
	theory, ok := solver.kernel.internalizer.theories[family].(*SMTBitVectorTheory)
	if !ok || len(theory.bitOwners) == 0 {
		t.Fatal("missing retained bit owners")
	}
	astTestMust(t, solver.Close())
	if len(theory.bitOwners) != 0 || len(theory.bits) != 0 || len(theory.bitTrail) != 0 {
		t.Fatal("bit owners survived theory close")
	}
}

func TestSMTBitVectorRelevantTermMarksBits(t *testing.T) {
	m, _, solver := smtBVSolver(t)
	family, err := m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	one, err := m.BitVectorNumeralUint64(1, 4)
	astTestMust(t, err)
	sort, err := m.GetSort(one)
	astTestMust(t, err)
	value, err := m.MkConst(astTestSymbol(t, m, "bv-relevant-value"), sort)
	astTestMust(t, err)
	state := solver.kernel.internalizer
	astTestMust(t, state.Internalize(value, false))
	theory, ok := state.theories[family].(*SMTBitVectorTheory)
	if !ok || theory == nil || state.relevancy == nil {
		t.Fatal("missing BV relevancy fixture")
	}
	bits, err := theory.bitsFor(value)
	astTestMust(t, err)
	astTestMust(t, theory.Relevant(value))
	for i, bit := range bits {
		if !state.relevancy.IsRelevant(bit) {
			t.Fatal("bit was not marked relevant", i)
		}
	}
}

func TestSMTBitVectorBlastLimitGivesUnknown(t *testing.T) {
	m, _, solver := smtBVSolver(t)
	family, err := m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	theory, ok := solver.kernel.internalizer.theories[family].(*SMTBitVectorTheory)
	if !ok || theory == nil {
		t.Fatal("missing bit-vector theory")
	}
	astTestMust(t, theory.ConfigureBitVector(SMTBVState{BlastMaxSize: 2, EnableInt2BV: true}))
	one, err := m.BitVectorNumeralUint64(1, 4)
	astTestMust(t, err)
	sort, err := m.GetSort(one)
	astTestMust(t, err)
	value, err := m.MkConst(astTestSymbol(t, m, "bv-blast-limit-value"), sort)
	astTestMust(t, err)
	equality, err := m.MkEq(value, one)
	astTestMust(t, err)
	astTestMust(t, solver.Assert(equality))
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.Undef {
		t.Fatal("large bit-vector approximation", status)
	}
}

func TestSMTBitVectorToIntAxiom(t *testing.T) {
	m := astTestManager(t)
	params, err := uparams.NewParamsContext(m.Symbols, m.NumeralService(), uparams.ParamsConfig{Budget: m.Budget()})
	astTestMust(t, err)
	solver, err := NewSMTSolver(m, nil, "QF_BVRE", SMTSolverConfig{Params: params})
	astTestMust(t, err)
	t.Cleanup(func() {
		astTestMust(t, solver.Close())
		astTestMust(t, params.Close())
	})
	fiveBV, err := m.BitVectorNumeralUint64(5, 4)
	astTestMust(t, err)
	sort, err := m.GetSort(fiveBV)
	astTestMust(t, err)
	value, err := m.MkConst(astTestSymbol(t, m, "bv2int-value"), sort)
	astTestMust(t, err)
	converted, err := m.MkBitVectorToInt(value)
	astTestMust(t, err)
	fiveInt, err := m.ArithmeticIntegerNumeralText("5")
	astTestMust(t, err)
	valueConstraint, err := m.MkEq(value, fiveBV)
	astTestMust(t, err)
	convertedConstraint, err := m.MkEq(converted, fiveInt)
	astTestMust(t, err)
	wrongConverted, err := m.MkNot(convertedConstraint)
	astTestMust(t, err)
	astTestMust(t, solver.Assert(valueConstraint))
	astTestMust(t, solver.Assert(wrongConverted))
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.False {
		t.Fatal("bv2int axiom", status)
	}
}

func TestSMTIntToBitVectorAxiom(t *testing.T) {
	m := astTestManager(t)
	params, err := uparams.NewParamsContext(m.Symbols, m.NumeralService(), uparams.ParamsConfig{Budget: m.Budget()})
	astTestMust(t, err)
	solver, err := NewSMTSolver(m, nil, "QF_BVRE", SMTSolverConfig{Params: params})
	astTestMust(t, err)
	t.Cleanup(func() {
		astTestMust(t, solver.Close())
		astTestMust(t, params.Close())
	})
	nineteen, err := m.ArithmeticIntegerNumeralText("19")
	astTestMust(t, err)
	integerSort, err := m.GetSort(nineteen)
	astTestMust(t, err)
	value, err := m.MkConst(astTestSymbol(t, m, "int2bv-value"), integerSort)
	astTestMust(t, err)
	converted, err := m.MkIntToBitVector(4, value)
	astTestMust(t, err)
	three, err := m.BitVectorNumeralUint64(3, 4)
	astTestMust(t, err)
	valueConstraint, err := m.MkEq(value, nineteen)
	astTestMust(t, err)
	convertedConstraint, err := m.MkEq(converted, three)
	astTestMust(t, err)
	wrongConverted, err := m.MkNot(convertedConstraint)
	astTestMust(t, err)
	astTestMust(t, solver.Assert(valueConstraint))
	astTestMust(t, solver.Assert(wrongConverted))
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.False {
		t.Fatal("int2bv axiom", status)
	}
}

func TestSMTBitVectorUnsignedDivisionCircuit(t *testing.T) {
	m, _, solver := smtBVSolver(t)
	two, err := m.BitVectorNumeralUint64(2, 4)
	astTestMust(t, err)
	sort, err := m.GetSort(two)
	astTestMust(t, err)
	left, err := m.MkConst(astTestSymbol(t, m, "bv-div-left"), sort)
	astTestMust(t, err)
	right, err := m.MkConst(astTestSymbol(t, m, "bv-div-right"), sort)
	astTestMust(t, err)
	quotient, err := m.MkBitVectorUDivInternal(left, right)
	astTestMust(t, err)
	six, err := m.BitVectorNumeralUint64(6, 4)
	astTestMust(t, err)
	three, err := m.BitVectorNumeralUint64(3, 4)
	astTestMust(t, err)
	leftValue, err := m.MkEq(left, six)
	astTestMust(t, err)
	rightValue, err := m.MkEq(right, two)
	astTestMust(t, err)
	quotientValue, err := m.MkEq(quotient, three)
	astTestMust(t, err)
	wrongQuotient, err := m.MkNot(quotientValue)
	astTestMust(t, err)
	for _, assertion := range []ast.ASTHandle{leftValue, rightValue, wrongQuotient} {
		astTestMust(t, solver.Assert(assertion))
	}
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.False {
		t.Fatal("unsigned division circuit", status)
	}
}

func TestSMTBitVectorSignedDivisionCircuits(t *testing.T) {
	for _, test := range []struct {
		name        string
		left, right uint64
		want        uint64
		build       func(*ast.ASTManager, ast.ASTHandle, ast.ASTHandle) (ast.ASTHandle, error)
	}{
		{"division", 9, 3, 14, func(m *ast.ASTManager, a, b ast.ASTHandle) (ast.ASTHandle, error) {
			return m.MkBitVectorSDivInternal(a, b)
		}},
		{"remainder", 9, 3, 15, func(m *ast.ASTManager, a, b ast.ASTHandle) (ast.ASTHandle, error) {
			return m.MkBitVectorSRemInternal(a, b)
		}},
		{"modulus", 9, 3, 2, func(m *ast.ASTManager, a, b ast.ASTHandle) (ast.ASTHandle, error) {
			return m.MkBitVectorSModInternal(a, b)
		}},
		{"negative-divisor-modulus", 7, 13, 14, func(m *ast.ASTManager, a, b ast.ASTHandle) (ast.ASTHandle, error) {
			return m.MkBitVectorSModInternal(a, b)
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			m, _, solver := smtBVSolver(t)
			leftValue, err := m.BitVectorNumeralUint64(test.left, 4)
			astTestMust(t, err)
			sort, err := m.GetSort(leftValue)
			astTestMust(t, err)
			left, err := m.MkConst(astTestSymbol(t, m, "bv-signed-div-left"), sort)
			astTestMust(t, err)
			right, err := m.MkConst(astTestSymbol(t, m, "bv-signed-div-right"), sort)
			astTestMust(t, err)
			operation, err := test.build(m, left, right)
			astTestMust(t, err)
			rightValue, err := m.BitVectorNumeralUint64(test.right, 4)
			astTestMust(t, err)
			want, err := m.BitVectorNumeralUint64(test.want, 4)
			astTestMust(t, err)
			leftEquality, err := m.MkEq(left, leftValue)
			astTestMust(t, err)
			rightEquality, err := m.MkEq(right, rightValue)
			astTestMust(t, err)
			resultEquality, err := m.MkEq(operation, want)
			astTestMust(t, err)
			wrongResult, err := m.MkNot(resultEquality)
			astTestMust(t, err)
			for _, assertion := range []ast.ASTHandle{leftEquality, rightEquality, wrongResult} {
				astTestMust(t, solver.Assert(assertion))
			}
			status, err := solver.CheckSat(nil)
			astTestMust(t, err)
			if status != lbool.False {
				t.Fatal("signed division-family circuit", status)
			}
		})
	}
}

func TestSMTBitVectorOrdinaryDivisionFamily(t *testing.T) {
	for _, test := range []struct {
		name              string
		left, right, want uint64
		build             func(*ast.ASTManager, ast.ASTHandle, ast.ASTHandle) (ast.ASTHandle, error)
	}{
		{"signed-division", 9, 3, 14, func(m *ast.ASTManager, a, b ast.ASTHandle) (ast.ASTHandle, error) { return m.MkBitVectorSDiv(a, b) }},
		{"signed-remainder", 9, 3, 15, func(m *ast.ASTManager, a, b ast.ASTHandle) (ast.ASTHandle, error) { return m.MkBitVectorSRem(a, b) }},
		{"signed-modulus", 9, 3, 2, func(m *ast.ASTManager, a, b ast.ASTHandle) (ast.ASTHandle, error) { return m.MkBitVectorSMod(a, b) }},
		{"unsigned-division", 14, 3, 4, func(m *ast.ASTManager, a, b ast.ASTHandle) (ast.ASTHandle, error) { return m.MkBitVectorUDiv(a, b) }},
		{"unsigned-remainder", 14, 3, 2, func(m *ast.ASTManager, a, b ast.ASTHandle) (ast.ASTHandle, error) { return m.MkBitVectorURem(a, b) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			m, _, solver := smtBVSolver(t)
			leftNumeral, err := m.BitVectorNumeralUint64(test.left, 4)
			astTestMust(t, err)
			sort, err := m.GetSort(leftNumeral)
			astTestMust(t, err)
			left, err := m.MkConst(astTestSymbol(t, m, "bv-ordinary-div-left"), sort)
			astTestMust(t, err)
			right, err := m.MkConst(astTestSymbol(t, m, "bv-ordinary-div-right"), sort)
			astTestMust(t, err)
			result, err := test.build(m, left, right)
			astTestMust(t, err)
			rightNumeral, err := m.BitVectorNumeralUint64(test.right, 4)
			astTestMust(t, err)
			want, err := m.BitVectorNumeralUint64(test.want, 4)
			astTestMust(t, err)
			leftValue, err := m.MkEq(left, leftNumeral)
			astTestMust(t, err)
			rightValue, err := m.MkEq(right, rightNumeral)
			astTestMust(t, err)
			resultValue, err := m.MkEq(result, want)
			astTestMust(t, err)
			wrongResult, err := m.MkNot(resultValue)
			astTestMust(t, err)
			for _, assertion := range []ast.ASTHandle{leftValue, rightValue, wrongResult} {
				astTestMust(t, solver.Assert(assertion))
			}
			status, err := solver.CheckSat(nil)
			astTestMust(t, err)
			if status != lbool.False {
				t.Fatal("ordinary division-family operation", status)
			}
		})
	}
}

func TestSMTBitVectorMultiplicationCircuit(t *testing.T) {
	m, _, solver := smtBVSolver(t)
	two, err := m.BitVectorNumeralUint64(2, 4)
	astTestMust(t, err)
	sort, err := m.GetSort(two)
	astTestMust(t, err)
	left, err := m.MkConst(astTestSymbol(t, m, "bv-mul-left"), sort)
	astTestMust(t, err)
	right, err := m.MkConst(astTestSymbol(t, m, "bv-mul-right"), sort)
	astTestMust(t, err)
	product, err := m.MkBitVectorMul(left, right)
	astTestMust(t, err)
	three, err := m.BitVectorNumeralUint64(3, 4)
	astTestMust(t, err)
	six, err := m.BitVectorNumeralUint64(6, 4)
	astTestMust(t, err)
	leftValue, err := m.MkEq(left, two)
	astTestMust(t, err)
	rightValue, err := m.MkEq(right, three)
	astTestMust(t, err)
	productValue, err := m.MkEq(product, six)
	astTestMust(t, err)
	wrongProduct, err := m.MkNot(productValue)
	astTestMust(t, err)
	for _, assertion := range []ast.ASTHandle{leftValue, rightValue, wrongProduct} {
		astTestMust(t, solver.Assert(assertion))
	}
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.False {
		t.Fatal("multiplication circuit", status)
	}
}

func TestSMTBitVectorUnsignedComparisonCircuit(t *testing.T) {
	m, _, solver := smtBVSolver(t)
	two, err := m.BitVectorNumeralUint64(2, 4)
	astTestMust(t, err)
	sort, err := m.GetSort(two)
	astTestMust(t, err)
	left, err := m.MkConst(astTestSymbol(t, m, "bv-ule-left"), sort)
	astTestMust(t, err)
	right, err := m.MkConst(astTestSymbol(t, m, "bv-ule-right"), sort)
	astTestMust(t, err)
	comparison, err := m.MkBitVectorULE(left, right)
	astTestMust(t, err)
	astTestMust(t, solver.Assert(comparison))
	one, err := m.BitVectorNumeralUint64(1, 4)
	astTestMust(t, err)
	leftValue, err := m.MkEq(left, two)
	astTestMust(t, err)
	rightValue, err := m.MkEq(right, one)
	astTestMust(t, err)
	astTestMust(t, solver.Assert(leftValue))
	astTestMust(t, solver.Assert(rightValue))
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.False {
		t.Fatal("unsigned comparison circuit", status)
	}
}

func TestSMTBitVectorShiftCircuit(t *testing.T) {
	m, _, solver := smtBVSolver(t)
	one, err := m.BitVectorNumeralUint64(1, 4)
	astTestMust(t, err)
	sort, err := m.GetSort(one)
	astTestMust(t, err)
	value, err := m.MkConst(astTestSymbol(t, m, "bv-shift-value"), sort)
	astTestMust(t, err)
	shifted, err := m.MkBitVectorSHL(value, one)
	astTestMust(t, err)
	three, err := m.BitVectorNumeralUint64(3, 4)
	astTestMust(t, err)
	six, err := m.BitVectorNumeralUint64(6, 4)
	astTestMust(t, err)
	valueConstraint, err := m.MkEq(value, three)
	astTestMust(t, err)
	shiftConstraint, err := m.MkEq(shifted, six)
	astTestMust(t, err)
	wrongShift, err := m.MkNot(shiftConstraint)
	astTestMust(t, err)
	astTestMust(t, solver.Assert(valueConstraint))
	astTestMust(t, solver.Assert(wrongShift))
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.False {
		t.Fatal("shift circuit", status)
	}
}

func TestSMTBitVectorPublishesNumeralModelValue(t *testing.T) {
	m, modelConfig := smtModelGeneratorFixture(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	family, err := m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	five, err := m.BitVectorNumeralUint64(5, 4)
	astTestMust(t, err)
	sort, err := m.GetSort(five)
	astTestMust(t, err)
	value, err := m.MkConst(astTestSymbol(t, m, "bv-model-value"), sort)
	astTestMust(t, err)
	context, err := NewSMTCheckerContext(m)
	astTestMust(t, err)
	state, err := NewSMTInternalizerState(m, context, nil)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, state.Close()) })
	theory, err := NewSMTBitVectorTheory(family)
	astTestMust(t, err)
	astTestMust(t, state.RegisterTheory(theory))
	astTestMust(t, state.Internalize(value, false))
	for index := uint32(0); index < 4; index++ {
		bit, bitErr := m.MkBitVectorBit2Bool(index, value)
		astTestMust(t, bitErr)
		astTestMust(t, state.Internalize(bit, true))
		variable, found := state.BoolVar(bit)
		if !found {
			t.Fatal("missing BV model bit", index)
		}
		literal := NewSMTLiteral(variable, false)
		if index == 1 || index == 3 {
			literal = literal.Negated()
		}
		astTestMust(t, state.AssignLiteral(literal, NewSMTBooleanAxiom()))
	}
	astTestMust(t, context.SetRelevant(value, true))
	generator, err := NewSMTModelGenerator(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, generator.Close()) })
	astTestMust(t, generator.SetContext(context))
	astTestMust(t, generator.InitializeModel(modelConfig))
	astTestMust(t, generator.BuildStateValues(state))
	node := context.enodes[value]
	interpretation, err := generator.Value(node)
	astTestMust(t, err)
	text, matched, err := m.BitVectorNumeralText(interpretation)
	astTestMust(t, err)
	if !matched || text != "5" {
		t.Fatal("BV model value", text, matched)
	}
}

func TestSMTBitVectorExtendedRotateCircuit(t *testing.T) {
	m, _, solver := smtBVSolver(t)
	one, err := m.BitVectorNumeralUint64(1, 4)
	astTestMust(t, err)
	sort, err := m.GetSort(one)
	astTestMust(t, err)
	value, err := m.MkConst(astTestSymbol(t, m, "bv-rotate-value"), sort)
	astTestMust(t, err)
	rotated, err := m.MkBitVectorExtendedRotateLeft(value, one)
	astTestMust(t, err)
	nine, err := m.BitVectorNumeralUint64(9, 4)
	astTestMust(t, err)
	three, err := m.BitVectorNumeralUint64(3, 4)
	astTestMust(t, err)
	valueConstraint, err := m.MkEq(value, nine)
	astTestMust(t, err)
	rotateConstraint, err := m.MkEq(rotated, three)
	astTestMust(t, err)
	wrongRotation, err := m.MkNot(rotateConstraint)
	astTestMust(t, err)
	astTestMust(t, solver.Assert(valueConstraint))
	astTestMust(t, solver.Assert(wrongRotation))
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.False {
		t.Fatal("extended rotate circuit", status)
	}
}

func TestSMTBitVectorUnsignedMulOverflowCircuit(t *testing.T) {
	m, _, solver := smtBVSolver(t)
	two, err := m.BitVectorNumeralUint64(2, 4)
	astTestMust(t, err)
	sort, err := m.GetSort(two)
	astTestMust(t, err)
	left, err := m.MkConst(astTestSymbol(t, m, "bv-overflow-left"), sort)
	astTestMust(t, err)
	right, err := m.MkConst(astTestSymbol(t, m, "bv-overflow-right"), sort)
	astTestMust(t, err)
	noOverflow, err := m.MkBitVectorUnsignedMulNoOverflow(left, right)
	astTestMust(t, err)
	eight, err := m.BitVectorNumeralUint64(8, 4)
	astTestMust(t, err)
	leftValue, err := m.MkEq(left, eight)
	astTestMust(t, err)
	rightValue, err := m.MkEq(right, two)
	astTestMust(t, err)
	astTestMust(t, solver.Assert(leftValue))
	astTestMust(t, solver.Assert(rightValue))
	astTestMust(t, solver.Assert(noOverflow))
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.False {
		t.Fatal("unsigned multiplication overflow", status)
	}
}

func TestSMTBitVectorSignedMulBounds(t *testing.T) {
	for _, test := range []struct {
		name        string
		left, right uint64
		build       func(*ast.ASTManager, ast.ASTHandle, ast.ASTHandle) (ast.ASTHandle, error)
	}{
		{name: "positive-overflow", left: 7, right: 2, build: (*ast.ASTManager).MkBitVectorSignedMulNoOverflow},
		{name: "negative-underflow", left: 8, right: 2, build: (*ast.ASTManager).MkBitVectorSignedMulNoUnderflow},
	} {
		t.Run(test.name, func(t *testing.T) {
			m, _, solver := smtBVSolver(t)
			leftValue, err := m.BitVectorNumeralUint64(test.left, 4)
			astTestMust(t, err)
			sort, err := m.GetSort(leftValue)
			astTestMust(t, err)
			left, err := m.MkConst(astTestSymbol(t, m, "bv-signed-bound-left"), sort)
			astTestMust(t, err)
			right, err := m.MkConst(astTestSymbol(t, m, "bv-signed-bound-right"), sort)
			astTestMust(t, err)
			bound, err := test.build(m, left, right)
			astTestMust(t, err)
			rightValue, err := m.BitVectorNumeralUint64(test.right, 4)
			astTestMust(t, err)
			leftConstraint, err := m.MkEq(left, leftValue)
			astTestMust(t, err)
			rightConstraint, err := m.MkEq(right, rightValue)
			astTestMust(t, err)
			astTestMust(t, solver.Assert(leftConstraint))
			astTestMust(t, solver.Assert(rightConstraint))
			astTestMust(t, solver.Assert(bound))
			status, err := solver.CheckSat(nil)
			astTestMust(t, err)
			if status != lbool.False {
				t.Fatal("signed multiplication bound", status)
			}
		})
	}
}

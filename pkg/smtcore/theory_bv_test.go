package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
	uparams "github.com/glycerine/g3/pkg/util/params"
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
}

func TestSMTBitVectorUnsupportedOperationGivesUnknown(t *testing.T) {
	m, _, solver := smtBVSolver(t)
	one, err := m.BitVectorNumeralUint64(1, 4)
	astTestMust(t, err)
	sort, err := m.GetSort(one)
	astTestMust(t, err)
	left, err := m.MkConst(astTestSymbol(t, m, "unsupported-div-left"), sort)
	astTestMust(t, err)
	right, err := m.MkConst(astTestSymbol(t, m, "unsupported-div-right"), sort)
	astTestMust(t, err)
	division, err := m.MkBitVectorUDivInternal(left, right)
	astTestMust(t, err)
	astTestMust(t, m.IncRef(division))
	equality, err := m.MkEq(division, left)
	astTestMust(t, err)
	astTestMust(t, m.DecRef(division))
	astTestMust(t, solver.Assert(equality))
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.Undef {
		t.Fatal("unsupported BV operation", status)
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

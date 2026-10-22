package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

func TestSMTPseudoBooleanQFFDExactCardinality(t *testing.T) {
	for _, atLeast := range []bool{false, true} {
		t.Run(map[bool]string{false: "at-most", true: "at-least"}[atLeast], func(t *testing.T) {
			m := astTestManager(t)
			family, err := m.RegisterPseudoBooleanPlugin()
			astTestMust(t, err)
			params, err := uparams.NewParamsContext(m.Symbols, m.NumeralService(), uparams.ParamsConfig{Budget: m.Budget()})
			astTestMust(t, err)
			solver, err := NewSMTSolver(m, nil, "QF_FD", SMTSolverConfig{Params: params})
			astTestMust(t, err)
			t.Cleanup(func() {
				astTestMust(t, solver.Close())
				astTestMust(t, params.Close())
			})
			if _, ok := solver.kernel.internalizer.theories[family].(*SMTPseudoBooleanTheory); !ok {
				t.Fatal("QF_FD did not install the pseudo-Boolean theory")
			}
			boolean, err := m.MkBoolSort()
			astTestMust(t, err)
			a, err := m.MkConst(astTestSymbol(t, m, "smt-pb-a"), boolean)
			astTestMust(t, err)
			b, err := m.MkConst(astTestSymbol(t, m, "smt-pb-b"), boolean)
			astTestMust(t, err)
			var constraint ast.ASTHandle
			if atLeast {
				constraint, err = m.MkPseudoBooleanAtLeast([]ast.ASTHandle{a, b}, 2)
			} else {
				constraint, err = m.MkPseudoBooleanAtMost([]ast.ASTHandle{a, b}, 1)
			}
			astTestMust(t, err)
			astTestMust(t, solver.Assert(constraint))
			astTestMust(t, solver.Assert(a))
			if !atLeast {
				astTestMust(t, solver.Assert(b))
			} else {
				notB, assertionErr := m.MkNot(b)
				astTestMust(t, assertionErr)
				astTestMust(t, solver.Assert(notB))
			}
			status, err := solver.CheckSat(nil)
			astTestMust(t, err)
			if status != lbool.False {
				t.Fatal("pseudo-Boolean cardinality violation accepted", status)
			}
		})
	}
}

func TestSMTPseudoBooleanWeightedEqualityAndPolarity(t *testing.T) {
	for _, negate := range []bool{false, true} {
		t.Run(map[bool]string{false: "positive", true: "negative"}[negate], func(t *testing.T) {
			m := astTestManager(t)
			_, err := m.RegisterPseudoBooleanPlugin()
			astTestMust(t, err)
			params, err := uparams.NewParamsContext(m.Symbols, m.NumeralService(), uparams.ParamsConfig{Budget: m.Budget()})
			astTestMust(t, err)
			solver, err := NewSMTSolver(m, nil, "QF_FD", SMTSolverConfig{Params: params})
			astTestMust(t, err)
			t.Cleanup(func() {
				astTestMust(t, solver.Close())
				astTestMust(t, params.Close())
			})
			boolean, err := m.MkBoolSort()
			astTestMust(t, err)
			a, err := m.MkConst(astTestSymbol(t, m, "smt-pb-weight-a"), boolean)
			astTestMust(t, err)
			b, err := m.MkConst(astTestSymbol(t, m, "smt-pb-weight-b"), boolean)
			astTestMust(t, err)
			service := m.NumeralService()
			first, err := service.NewValue()
			astTestMust(t, err)
			second, err := service.NewValue()
			astTestMust(t, err)
			bound, err := service.NewValue()
			astTestMust(t, err)
			t.Cleanup(func() {
				astTestMust(t, service.Del(&bound))
				astTestMust(t, service.Del(&second))
				astTestMust(t, service.Del(&first))
			})
			astTestMust(t, service.SetInt64(&first, 2))
			astTestMust(t, service.SetInt64(&second, 3))
			astTestMust(t, service.SetInt64(&bound, 5))
			constraint, err := m.MkPseudoBooleanEQ([]unumeral.OwnedRational{first, second}, []ast.ASTHandle{a, b}, &bound)
			astTestMust(t, err)
			assertion := constraint
			if negate {
				assertion, err = m.MkNot(constraint)
				astTestMust(t, err)
			}
			astTestMust(t, solver.Assert(assertion))
			astTestMust(t, solver.Assert(a))
			if negate {
				astTestMust(t, solver.Assert(b))
			} else {
				notB, assertionErr := m.MkNot(b)
				astTestMust(t, assertionErr)
				astTestMust(t, solver.Assert(notB))
			}
			status, err := solver.CheckSat(nil)
			astTestMust(t, err)
			if status != lbool.False {
				t.Fatal("weighted pseudo-Boolean polarity violation accepted", status)
			}
		})
	}
}

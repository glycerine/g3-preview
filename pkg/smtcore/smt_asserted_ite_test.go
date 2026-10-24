package smtcore

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
	"github.com/glycerine/g3/pkg/util/symbol"
)

type smtAssertedITEFixture struct {
	m                            *ast.ASTManager
	solver                       *SMTSolver
	formula, condition, selected ast.ASTHandle
}

func smtNewAssertedITEFixture(t *testing.T, proofs, double, nonground bool) smtAssertedITEFixture {
	t.Helper()
	m, solver := smtAssertedNNFFixture(t, proofs)
	smtDisableAutoConfig(t, solver)
	u, err := m.MkUninterpretedSort(astTestSymbol(t, m, "asserted-ite-U"), nil)
	astTestMust(t, err)
	b, err := m.MkBoolSort()
	astTestMust(t, err)
	constant := func(name string, sort ast.ASTHandle) ast.ASTHandle {
		value, err := m.MkConst(astTestSymbol(t, m, name), sort)
		astTestMust(t, err)
		return value
	}
	c, c2 := constant("asserted-ite-c", b), constant("asserted-ite-c2", b)
	if nonground {
		c, err = m.MkVar(0, b)
		astTestMust(t, err)
	}
	a, z := constant("asserted-ite-a", u), constant("asserted-ite-z", u)
	d, e := constant("asserted-ite-d", u), constant("asserted-ite-e", u)
	term, err := m.MkIte(c, a, z)
	astTestMust(t, err)
	second := d
	if double {
		second, err = m.MkIte(c2, d, e)
		astTestMust(t, err)
	}
	f, err := m.MkFuncDecl(astTestSymbol(t, m, "asserted-ite-f"), []ast.ASTHandle{u, u}, b, ast.ASTFuncDeclInfo{})
	astTestMust(t, err)
	formula, err := m.MkApp(f, []ast.ASTHandle{term, second})
	astTestMust(t, err)
	if nonground {
		formula, err = m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{b}, Names: []symbol.Symbol{symbol.Numerical(1)}, Body: formula})
		astTestMust(t, err)
	}
	selected, err := m.MkApp(f, []ast.ASTHandle{a, d})
	astTestMust(t, err)
	for _, value := range []ast.ASTHandle{formula, selected} {
		ref, err := m.NewRef(value)
		astTestMust(t, err)
		t.Cleanup(func() { astTestMust(t, m.DelRef(&ref)) })
	}
	return smtAssertedITEFixture{m, solver, formula, c, selected}
}

func TestSMTAssertedITELiftingModesAndProofs(t *testing.T) {
	cases := []struct {
		name                                         string
		lift, ng                                     uint32
		double, nonground, ngPass, disabled, changed bool
	}{
		{name: "none"},
		{name: "conservative-single", lift: smtLiftITEConservative, changed: true},
		{name: "conservative-double", lift: smtLiftITEConservative, double: true},
		{name: "full-double", lift: smtLiftITEFull, double: true, changed: true},
		{name: "nonground", ng: smtLiftITEConservative, nonground: true, ngPass: true, changed: true},
		{name: "nonground-refuses-ground", ng: smtLiftITEFull, ngPass: true},
		{name: "full-suppresses-nonground", lift: smtLiftITEFull, ng: smtLiftITEFull, ngPass: true, nonground: true},
		{name: "conservative-suppresses-nonground", lift: smtLiftITEConservative, ng: smtLiftITEConservative, ngPass: true, nonground: true},
		{name: "preprocessing-disabled", lift: smtLiftITEFull, disabled: true},
	}
	for _, proofs := range []bool{false, true} {
		for _, tc := range cases {
			t.Run(fmt.Sprint(proofs, "/", tc.name), func(t *testing.T) {
				x := smtNewAssertedITEFixture(t, proofs, tc.double, tc.nonground)
				k := x.solver.kernel
				astTestMust(t, x.solver.Assert(x.formula))
				// These source fields have no retained public parameter accessor.
				k.preprocessState.LiftITE, k.preprocessState.NonGroundLiftITE = tc.lift, tc.ng
				k.preprocess = !tc.disabled
				original := k.assertions[0].Formula
				astTestMust(t, k.liftPendingITEs(tc.ngPass))
				changed := len(k.assertions) != 1 || k.assertions[0].Formula != original
				if changed != tc.changed {
					t.Fatal("ITE lifting did not follow the source mode guard", changed)
				}
				if proofs {
					for _, row := range k.assertions {
						fact, err := x.m.GetProofFact(row.Proof)
						astTestMust(t, err)
						if fact != row.Formula {
							t.Fatal("ITE lifting lost the assertion proof")
						}
					}
				}
			})
		}
	}
}

func TestSMTAssertedTermITEEliminationDefinitionsAndGuards(t *testing.T) {
	for _, proofs := range []bool{false, true} {
		for _, mode := range []string{"enabled", "disabled", "full-lift", "no-preprocessing"} {
			t.Run(fmt.Sprint(proofs, "/", mode), func(t *testing.T) {
				x := smtNewAssertedITEFixture(t, proofs, false, false)
				k := x.solver.kernel
				astTestMust(t, x.solver.Assert(x.formula))
				k.preprocessState.EliminateTermITE = mode != "disabled"
				if mode == "full-lift" {
					k.preprocessState.LiftITE = smtLiftITEFull
				}
				if mode == "no-preprocessing" {
					k.preprocess = false
				}
				// The retained driver invokes term elimination after the NNF
				// phase has enabled conjunction elimination.
				astTestMust(t, k.setAssertedEliminateAnd(true))
				astTestMust(t, k.eliminatePendingTermITEs())
				n, err := k.definedNames.NumNames()
				astTestMust(t, err)
				if mode != "enabled" {
					if n != 0 || len(k.assertions) != 1 || k.assertions[0].Formula != x.formula {
						t.Fatal("disabled elimination changed assertions or defined names")
					}
					return
				}
				if n != 1 || len(k.assertions) != 3 {
					t.Fatal("missing defined term or its two branch constraints", n, len(k.assertions))
				}
				decl, err := x.m.GetDecl(k.assertions[0].Formula)
				astTestMust(t, err)
				originalDecl, err := x.m.GetDecl(x.formula)
				astTestMust(t, err)
				if decl != originalDecl {
					t.Fatal("definitions preceded the transformed original assertion")
				}
				if proofs {
					for _, row := range k.assertions {
						fact, err := x.m.GetProofFact(row.Proof)
						astTestMust(t, err)
						if fact != row.Formula {
							t.Fatal("term ITE elimination lost a formula or definition proof")
						}
					}
				}
			})
		}
	}
}

func TestSMTAssertedITEPassesScopedConflict(t *testing.T) {
	for _, proofs := range []bool{false, true} {
		for _, eliminate := range []bool{false, true} {
			t.Run(fmt.Sprint(proofs, "/", eliminate), func(t *testing.T) {
				x := smtNewAssertedITEFixture(t, proofs, false, false)
				k := x.solver.kernel
				k.preprocessState.PropagateValues = false
				k.preprocessState.EliminateTermITE = eliminate
				if !eliminate {
					k.preprocessState.LiftITE = smtLiftITEConservative
				}
				for cycle := 0; cycle < 2; cycle++ {
					astTestMust(t, x.solver.Push())
					astTestMust(t, x.solver.Assert(x.formula))
					negative, err := x.m.MkNot(x.selected)
					astTestMust(t, err)
					astTestMust(t, x.solver.Assert(negative))
					astTestMust(t, x.solver.Assert(x.condition))
					status, err := x.solver.CheckSat(nil)
					astTestMust(t, err)
					if status != lbool.False {
						t.Fatal("ITE branch contradiction was not refuted", status)
					}
					n, err := k.definedNames.NumNames()
					astTestMust(t, err)
					if eliminate && n == 0 {
						t.Fatal("solver did not run term ITE elimination")
					}
					if proofs {
						proof, err := x.solver.Proof()
						astTestMust(t, err)
						fact, err := x.m.GetProofFact(proof)
						astTestMust(t, err)
						if !x.m.IsFalse(fact) {
							t.Fatal("ITE contradiction lost its proof")
						}
					}
					astTestMust(t, x.solver.Pop(1))
					n, err = k.definedNames.NumNames()
					astTestMust(t, err)
					if n != 0 || len(k.assertions) != 0 {
						t.Fatal("ITE definitions survived scope rollback", n)
					}
				}
			})
		}
	}
}

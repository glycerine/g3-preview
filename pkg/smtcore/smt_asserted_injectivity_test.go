package smtcore

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func smtAssertedInjectivityFixture(t *testing.T, proofs bool) (*ast.ASTManager, *SMTSolver, ast.ASTHandle, ast.ASTHandle, ast.ASTHandle) {
	t.Helper()
	m, solver := smtAssertedNNFFixture(t, proofs)
	u, err := m.MkUninterpretedSort(astTestSymbol(t, m, "asserted-inj-U"), nil)
	astTestMust(t, err)
	x, err := m.MkVar(0, u)
	astTestMust(t, err)
	y, err := m.MkVar(1, u)
	astTestMust(t, err)
	f, err := m.MkFuncDecl(astTestSymbol(t, m, "asserted-inj-f"), []ast.ASTHandle{u}, u, ast.ASTFuncDeclInfo{})
	astTestMust(t, err)
	fx, err := m.MkApp(f, []ast.ASTHandle{x})
	astTestMust(t, err)
	fy, err := m.MkApp(f, []ast.ASTHandle{y})
	astTestMust(t, err)
	equal, err := m.MkEq(fx, fy)
	astTestMust(t, err)
	different, err := m.MkNot(equal)
	astTestMust(t, err)
	equal, err = m.MkEq(x, y)
	astTestMust(t, err)
	body, err := m.MkOr(different, equal)
	astTestMust(t, err)
	q, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{u, u}, Names: []symbol.Symbol{symbol.Numerical(0), symbol.Numerical(1)}, Body: body})
	astTestMust(t, err)
	astTestMust(t, solver.Assert(q))
	return m, solver, q, f, u
}

func TestSMTAssertedInjectivityRefinementAndGuards(t *testing.T) {
	for _, proofs := range []bool{false, true} {
		for _, mode := range []string{"enabled", "disabled", "no-preprocessing"} {
			t.Run(fmt.Sprint(proofs, "/", mode), func(t *testing.T) {
				m, solver, original, _, _ := smtAssertedInjectivityFixture(t, proofs)
				if mode == "disabled" {
					local, err := solver.kernel.paramContext.NewValue()
					astTestMust(t, err)
					defer func() { astTestMust(t, solver.kernel.paramContext.Del(&local)) }()
					name := "refine_inj_axioms"
					astTestMust(t, solver.kernel.paramContext.SetBoolName(&local, &name, false))
					astTestMust(t, solver.UpdateParams(&local))
				}
				if mode == "no-preprocessing" {
					solver.kernel.preprocess = false
				}
				astTestMust(t, solver.kernel.refinePendingInjectivityAxioms())
				if len(solver.kernel.assertions) != 1 {
					t.Fatal("unexpected assertion count", len(solver.kernel.assertions))
				}
				row := solver.kernel.assertions[0]
				if mode != "enabled" {
					if row.Formula != original {
						t.Fatal("disabled refinement changed the axiom")
					}
					return
				}
				n, err := m.GetNumDeclarations(row.Formula)
				astTestMust(t, err)
				patterns, err := m.GetNumPatterns(row.Formula)
				astTestMust(t, err)
				if n != 1 || patterns != 1 {
					t.Fatal("inverse axiom lost its reduced binder or trigger", n, patterns)
				}
				if proofs {
					fact, err := m.GetProofFact(row.Proof)
					astTestMust(t, err)
					if fact != row.Formula {
						t.Fatal("rewrite proof does not justify the replacement")
					}
				}
				// A second pass must preserve an already refined axiom.
				astTestMust(t, solver.kernel.refinePendingInjectivityAxioms())
				if solver.kernel.assertions[0].Formula != row.Formula {
					t.Fatal("inverse axiom was rewritten again")
				}
			})
		}
	}
}

func TestSMTAssertedInjectivityConflict(t *testing.T) {
	for _, proofs := range []bool{false, true} {
		t.Run(fmt.Sprint(proofs), func(t *testing.T) {
			m, solver, _, f, u := smtAssertedInjectivityFixture(t, proofs)
			a, err := m.MkConst(astTestSymbol(t, m, "asserted-inj-a"), u)
			astTestMust(t, err)
			b, err := m.MkConst(astTestSymbol(t, m, "asserted-inj-b"), u)
			astTestMust(t, err)
			fa, err := m.MkApp(f, []ast.ASTHandle{a})
			astTestMust(t, err)
			fb, err := m.MkApp(f, []ast.ASTHandle{b})
			astTestMust(t, err)
			equal, err := m.MkEq(fa, fb)
			astTestMust(t, err)
			astTestMust(t, solver.Assert(equal))
			equal, err = m.MkEq(a, b)
			astTestMust(t, err)
			different, err := m.MkNot(equal)
			astTestMust(t, err)
			astTestMust(t, solver.Assert(different))
			status, err := solver.CheckSat(nil)
			astTestMust(t, err)
			if status != lbool.False {
				t.Fatal("injectivity contradiction was not refuted", status)
			}
			if proofs {
				proof, err := solver.Proof()
				astTestMust(t, err)
				fact, err := m.GetProofFact(proof)
				astTestMust(t, err)
				if !m.IsFalse(fact) {
					t.Fatal("injectivity contradiction lost its proof")
				}
			}
		})
	}
}

func TestSMTDisequalityConflictProofAcrossEqualityBranches(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		t.Run(fmt.Sprint(reverse), func(t *testing.T) {
			m, solver := smtAssertedNNFFixture(t, true)
			solver.kernel.preprocess = false
			u, err := m.MkUninterpretedSort(astTestSymbol(t, m, "diseq-proof-U"), nil)
			astTestMust(t, err)
			constants := make([]ast.ASTHandle, 4)
			for i := range constants {
				constants[i], err = m.MkConst(astTestSymbol(t, m, fmt.Sprint("diseq-proof-", i)), u)
				astTestMust(t, err)
			}
			a, b, c, d := constants[0], constants[1], constants[2], constants[3]
			left, right := a, b
			if reverse {
				left, right = b, a
			}
			equal, err := m.MkEq(left, right)
			astTestMust(t, err)
			different, err := m.MkNot(equal)
			astTestMust(t, err)
			astTestMust(t, solver.Assert(different))
			for _, pair := range [][2]ast.ASTHandle{{c, a}, {d, b}, {c, d}} {
				equal, err := m.MkEq(pair[0], pair[1])
				astTestMust(t, err)
				astTestMust(t, solver.Assert(equal))
			}
			status, err := solver.CheckSat(nil)
			astTestMust(t, err)
			if status != lbool.False {
				t.Fatal("equality chain did not contradict the disequality", status)
			}
			proof, err := solver.Proof()
			astTestMust(t, err)
			fact, err := m.GetProofFact(proof)
			astTestMust(t, err)
			if !m.IsFalse(fact) {
				t.Fatal("equality chain lost its conflict proof")
			}
		})
	}
}

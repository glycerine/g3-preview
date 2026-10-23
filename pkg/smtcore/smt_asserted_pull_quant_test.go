package smtcore

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func smtAssertedPullFixture(t *testing.T, proofs, enabled bool) (*ast.ASTManager, *SMTSolver, ast.ASTHandle, ast.ASTHandle, ast.ASTHandle) {
	t.Helper()
	m, solver := smtAssertedNNFFixture(t, proofs)
	local, err := solver.kernel.paramContext.NewValue()
	astTestMust(t, err)
	name := "pull_nested_quantifiers"
	astTestMust(t, solver.kernel.paramContext.SetBoolName(&local, &name, enabled))
	astTestMust(t, solver.UpdateParams(&local))
	astTestMust(t, solver.kernel.paramContext.Del(&local))
	b, err := m.MkBoolSort()
	astTestMust(t, err)
	x, err := m.MkVar(0, b)
	astTestMust(t, err)
	y, err := m.MkVar(1, b)
	astTestMust(t, err)
	p := smtQuickPredicate(t, m, "asserted-pull-p", b, b)
	r, err := m.MkFuncDecl(astTestSymbol(t, m, "asserted-pull-r"), []ast.ASTHandle{b, b}, b, ast.ASTFuncDeclInfo{})
	astTestMust(t, err)
	px, err := m.MkApp(p, []ast.ASTHandle{x})
	astTestMust(t, err)
	ryx, err := m.MkApp(r, []ast.ASTHandle{y, x})
	astTestMust(t, err)
	inner, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{b}, Names: []symbol.Symbol{symbol.Numerical(2)}, Body: ryx})
	astTestMust(t, err)
	body, err := m.MkOr(px, inner)
	astTestMust(t, err)
	outer, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{b}, Names: []symbol.Symbol{symbol.Numerical(1)}, Body: body})
	astTestMust(t, err)
	ref, err := m.NewRef(outer)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.DelRef(&ref)) })
	return m, solver, outer, p, r
}

func TestSMTAssertedPullQuantifierGuardsAndProofs(t *testing.T) {
	for _, proofs := range []bool{false, true} {
		for _, mode := range []string{"enabled", "disabled", "no-preprocessing"} {
			t.Run(fmt.Sprint(proofs, "/", mode), func(t *testing.T) {
				m, solver, original, _, _ := smtAssertedPullFixture(t, proofs, mode != "disabled")
				astTestMust(t, solver.Assert(original))
				if mode == "no-preprocessing" {
					solver.kernel.preprocess = false
				}
				astTestMust(t, solver.kernel.pullPendingNestedQuantifiers())
				row := solver.kernel.assertions[0]
				if mode != "enabled" {
					if row.Formula != original {
						t.Fatal("disabled pulling changed the assertion")
					}
					return
				}
				n, err := m.GetNumDeclarations(row.Formula)
				astTestMust(t, err)
				body, err := m.GetQuantifierBody(row.Formula)
				astTestMust(t, err)
				nested, err := m.HasQuantifiers(body)
				astTestMust(t, err)
				if n != 2 || nested {
					t.Fatal("nested universal was not pulled into the outer binder", n, nested)
				}
				if proofs {
					fact, err := m.GetProofFact(row.Proof)
					astTestMust(t, err)
					if fact != row.Formula {
						t.Fatal("asserted pull lost its composed proof")
					}
				}
			})
		}
	}
}

func TestSMTAssertedPullQuantifierScopedConflict(t *testing.T) {
	for _, proofs := range []bool{false, true} {
		t.Run(fmt.Sprint(proofs), func(t *testing.T) {
			m, solver, original, p, r := smtAssertedPullFixture(t, proofs, true)
			for cycle := 0; cycle < 2; cycle++ {
				astTestMust(t, solver.Push())
				astTestMust(t, solver.Assert(original))
				falsity, err := m.MkFalse()
				astTestMust(t, err)
				for _, app := range []struct {
					decl ast.ASTHandle
					args []ast.ASTHandle
				}{{p, []ast.ASTHandle{falsity}}, {r, []ast.ASTHandle{falsity, falsity}}} {
					value, err := m.MkApp(app.decl, app.args)
					astTestMust(t, err)
					negative, err := m.MkNot(value)
					astTestMust(t, err)
					astTestMust(t, solver.Assert(negative))
				}
				status, err := solver.CheckSat(nil)
				astTestMust(t, err)
				if status != lbool.False {
					t.Fatal("pulled universal did not refute its counterexample", status)
				}
				if proofs {
					proof, err := solver.Proof()
					astTestMust(t, err)
					fact, err := m.GetProofFact(proof)
					astTestMust(t, err)
					if !m.IsFalse(fact) {
						t.Fatal("scoped pull contradiction lost its proof")
					}
				}
				astTestMust(t, solver.Pop(1))
				if len(solver.kernel.assertions) != 0 || !solver.kernel.internalizer.quantifiers.Empty() {
					t.Fatal("pulled assertion survived scope rollback")
				}
			}
		})
	}
}

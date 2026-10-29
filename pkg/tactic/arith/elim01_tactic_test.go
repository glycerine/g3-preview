package arith

// Source: src/tactic/arith/elim01_tactic.cpp finite-range conversion and
// bool2int model reconstruction.

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/tactic"
)

func elim01Fixture(t *testing.T, proofs bool) *Elim01Tactic {
	t.Helper()
	base, m := fixDLFixture(t, proofs)
	r, err := NewElim01Tactic(m, base.context, base.globals, nil, base.services)
	equationMust(t, err)
	t.Cleanup(func() { equationMust(t, r.Close()) })
	return r
}

func elim01BooleanConstants(t *testing.T, m *ast.ASTManager, goal *tactic.Goal) []ast.ASTHandle {
	t.Helper()
	seen := make(map[ast.ASTHandle]bool)
	var result []ast.ASTHandle
	size, err := goal.Size()
	equationMust(t, err)
	for i := uint32(0); i < size; i++ {
		root, e := goal.Form(i)
		equationMust(t, e)
		stack := []ast.ASTHandle{root}
		for len(stack) != 0 {
			n := len(stack) - 1
			h := stack[n]
			stack = stack[:n]
			if seen[h] {
				continue
			}
			seen[h] = true
			kind, e := m.GetKind(h)
			equationMust(t, e)
			if kind != ast.ASTApp {
				continue
			}
			count, e := m.GetNumArgs(h)
			equationMust(t, e)
			if count == 0 {
				decl, e := m.GetDecl(h)
				equationMust(t, e)
				family, e := m.GetFamilyID(decl)
				equationMust(t, e)
				sort, e := m.GetSort(h)
				equationMust(t, e)
				boolSort, e := m.MkBoolSort()
				equationMust(t, e)
				if family == ast.NullFamilyID && sort == boolSort {
					result = append(result, decl)
				}
			}
			for j := uint32(0); j < count; j++ {
				arg, e := m.GetArg(h, j)
				equationMust(t, e)
				stack = append(stack, arg)
			}
		}
	}
	return result
}

func TestElim01FiniteRangeRewriteAndModel(t *testing.T) {
	r := elim01Fixture(t, false)
	m := r.manager
	x := fixDLConst(t, m, true, "x")
	zero, three := fixDLNumber(t, m, true, "0"), fixDLNumber(t, m, true, "3")
	goal := fixDLGoal(t, m, false, true, false,
		fixDLApp(t, m, 3, x, zero),
		fixDLApp(t, m, 2, x, three),
		func() ast.ASTHandle {
			eq, err := m.MkEq(x, three)
			equationMust(t, err)
			fixDLPin(t, m, eq)
			return eq
		}(),
	)
	var result tactic.TacticResult
	t.Cleanup(func() { equationMust(t, result.Close()) })
	equationMust(t, r.Apply(goal, &result))
	depth, err := goal.Depth()
	equationMust(t, err)
	if depth != 1 || result.ModelConverter == nil {
		t.Fatal("finite-range conversion missing", depth)
	}
	size, err := goal.Size()
	equationMust(t, err)
	for i := uint32(0); i < size; i++ {
		form, e := goal.Form(i)
		equationMust(t, e)
		occurs, e := m.OccursExpr(x, form)
		equationMust(t, e)
		if occurs {
			t.Fatal("original integer remained in converted goal", i)
		}
	}
	bits := elim01BooleanConstants(t, m, goal)
	if len(bits) != 2 {
		t.Fatal("expected two Boolean digits", bits)
	}
	md, err := model.NewModel(m)
	equationMust(t, err)
	truth, err := m.MkTrue()
	equationMust(t, err)
	for _, bit := range bits {
		equationMust(t, md.CoreField.RegisterConstant(bit, truth))
	}
	var converted tactic.ModelRef
	equationMust(t, converted.Set(&md.CoreField))
	t.Cleanup(func() { equationMust(t, converted.Close()) })
	equationMust(t, result.ModelConverter.ApplyModel(&converted, 0))
	gotModel, err := converted.Get()
	equationMust(t, err)
	xDecl, err := m.GetDecl(x)
	equationMust(t, err)
	value, err := gotModel.ConstantInterpretation(xDecl)
	equationMust(t, err)
	text, integer, matched, err := m.ArithmeticNumeralText(value)
	equationMust(t, err)
	if !matched || !integer || text != "3" {
		t.Fatal("integer reconstruction", text, integer, matched)
	}
	for _, bit := range bits {
		present, e := gotModel.HasInterpretation(bit)
		equationMust(t, e)
		if present {
			t.Fatal("Boolean helper escaped model filter", bit)
		}
	}
}

func TestElim01ProofsAndNonTargetNoOp(t *testing.T) {
	for _, target := range []bool{false, true} {
		t.Run(map[bool]string{false: "wide", true: "target"}[target], func(t *testing.T) {
			r := elim01Fixture(t, true)
			m := r.manager
			x := fixDLConst(t, m, true, "x")
			upper := "9"
			if target {
				upper = "3"
			}
			goal := fixDLGoal(t, m, true, false, false,
				fixDLApp(t, m, 3, x, fixDLNumber(t, m, true, "0")),
				fixDLApp(t, m, 2, x, fixDLNumber(t, m, true, upper)),
			)
			var result tactic.TacticResult
			t.Cleanup(func() { equationMust(t, result.Close()) })
			equationMust(t, r.Apply(goal, &result))
			depth, err := goal.Depth()
			equationMust(t, err)
			wantDepth := uint32(0)
			if target {
				wantDepth = 1
			}
			if depth != wantDepth {
				t.Fatal("conversion depth", depth, wantDepth)
			}
			if target {
				size, e := goal.Size()
				equationMust(t, e)
				for i := uint32(0); i < size; i++ {
					proof, e := goal.Proof(i)
					equationMust(t, e)
					fact, e := m.GetProofFact(proof)
					equationMust(t, e)
					form, e := goal.Form(i)
					equationMust(t, e)
					if fact != form {
						t.Fatal("proof fact mismatch", i)
					}
				}
			}
		})
	}
}

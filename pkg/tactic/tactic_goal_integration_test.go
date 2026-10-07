package

// Source: src/tactic/goal.cpp:26-30,111-190,240-329,498-631,656-707;
// src/tactic/goal.h:39-143.
// Licensed under the MIT license. Source author: Leonardo de Moura (2011).
tactic_test

import (
	"github.com/glycerine/g3/pkg/sat"
	sattactic "github.com/glycerine/g3/pkg/sat/tactic"
	"github.com/glycerine/g3/pkg/util"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestTacticGoalAssertionsToSATAndValidatedModel(t *testing.T) {
	for bits := 0; bits < 8; bits++ {
		t.Run(string(rune('0'+bits)), func(t *testing.T) {
			m := astTestManager(t)
			p := goalTestAtom(t, m, "p")
			q := goalTestAtom(t, m, "q")
			r := goalTestAtom(t, m, "r")
			formula := goalTestApp(t, m, ast.ASTOpAnd, goalTestApp(t, m, ast.ASTOpOr, p, q), goalTestApp(t, m, ast.ASTOpNot, goalTestApp(t, m, ast.ASTOpAnd, p, q)), goalTestApp(t, m, ast.ASTOpIte, r, p, q))
			g := goalTestNew(t, m, false, false)
			astTestMust(t, g.Assert(formula))
			for i, h := range []ast.ASTHandle{p, q, r} {
				if bits&(1<<i) == 0 {
					h = goalTestApp(t, m, ast.ASTOpNot, h)
				}
				astTestMust(t, g.Assert(h))
			}
			astTestMust(t, g.ElimRedundancies())
			formulas, e := g.AppendFormulas(nil)
			astTestMust(t, e)
			s, _, _ := satSolverCheckFixture(t)
			tr, e := sattactic.NewGoalTranslator(m, s)
			satOK(t, e)
			t.Cleanup(func() { satOK(t, tr.Close()) })
			satOK(t, tr.Compile(formulas))
			got, e := s.Check(nil)
			satOK(t, e)
			pv, qv, rv := bits&1 != 0, bits&2 != 0, bits&4 != 0
			selected := qv
			if rv {
				selected = pv
			}
			want := util.LFalse
			if (pv || qv) && !(pv && qv) && selected {
				want = util.LTrue
			}
			if got != want {
				t.Fatal("original goal assertions", got, want)
			}
			if got == util.LTrue {
				model, e := s.AppendModel(nil)
				satOK(t, e)
				for i, h := range []ast.ASTHandle{p, q, r} {
					v, e := tr.Variable(h)
					satOK(t, e)
					if v == sat.NullBoolVar || int(v) >= len(model) || model[v] == util.LUndef || (model[v] == util.LTrue) != (bits&(1<<i) != 0) {
						t.Fatal("model violates goal assertion", i, v)
					}
				}
			}
		})
	}
}

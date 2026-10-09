// Source: src/sat/tactic/goal2sat.cpp:425-485;
// src/sat/tactic/atom2bool_var.cpp:88-111.
package tactic

import (
	"fmt"
	"github.com/glycerine/g3/pkg/util/lbool"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/sat"
)

func TestSATGoalDependenciesSignedAssumptionsAndCore(t *testing.T) {
	for _, negative := range []bool{false, true} {
		t.Run(fmt.Sprint(negative), func(t *testing.T) {
			m := astTestManager(t)
			s, _, _ := satSolverCheckFixture(t)
			tr, e := NewGoalTranslator(m, s)
			satOK(t, e)
			t.Cleanup(func() { satOK(t, tr.Close()) })
			p := goalTestAtom(t, m, "label")
			d := p
			if negative {
				d = goalTestApp(t, m, ast.ASTOpNot, p)
			}
			leaf, e := m.DependencyLeaf(d)
			satOK(t, e)
			g := goalTestNew(t, m, false, true)
			f, e := m.MkFalse()
			satOK(t, e)
			satOK(t, g.AssertWithProof(f, ast.ASTHandle{}, leaf))
			satOK(t, tr.CompileGoal(g))
			assumptions, e := tr.AppendDependencyAssumptions(nil)
			satOK(t, e)
			if len(assumptions) != 1 || assumptions[0].Sign() != negative {
				t.Fatal("dependency sign", assumptions)
			}
			r, e := s.Check(assumptions)
			satOK(t, e)
			if r != lbool.False {
				t.Fatal("guarded false must be unsat under label", r)
			}
			core, e := s.BorrowCore()
			satOK(t, e)
			n, e := core.Size()
			satOK(t, e)
			if n != 1 {
				t.Fatal("source assumption core", n)
			}
			l, e := core.Read(0)
			satOK(t, e)
			if l != assumptions[0] {
				t.Fatal("core did not preserve signed label", l)
			}
			r, e = s.Check(nil)
			satOK(t, e)
			if r != lbool.True {
				t.Fatal("guarded false should permit disabled label", r)
			}
		})
	}
}
func TestSATGoalDependenciesFormulaLabelShortcutAndInterface(t *testing.T) {
	m := astTestManager(t)
	s, _, _ := satSolverCheckFixture(t)
	tr, e := NewGoalTranslator(m, s)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, tr.Close()) })
	p := goalTestAtom(t, m, "self-label")
	leaf, e := m.DependencyLeaf(p)
	satOK(t, e)
	g := goalTestNew(t, m, false, true)
	satOK(t, g.AssertWithProof(p, ast.ASTHandle{}, leaf))
	satOK(t, tr.CompileGoal(g))
	v, e := tr.Variable(p)
	satOK(t, e)
	value, e := s.SourceValueVar(v)
	satOK(t, e)
	if value != lbool.Undef {
		t.Fatal("source skip_dep emitted a root unit", value)
	}
	external, e := s.SourceIsExternal(v)
	satOK(t, e)
	if !external {
		t.Fatal("dependency interface constant was not external")
	}
	a, e := tr.AppendDependencyAssumptions(nil)
	satOK(t, e)
	r, e := s.Check(a)
	satOK(t, e)
	if r != lbool.True {
		t.Fatal(r)
	}
	model, e := s.AppendModel(nil)
	satOK(t, e)
	if model[v] != lbool.True {
		t.Fatal("assumption not enforced")
	}
}
func TestSATGoalDependenciesCompoundGuardOriginalFreshnessAndTruthTable(t *testing.T) {
	for bits := 0; bits < 4; bits++ {
		t.Run(fmt.Sprint(bits), func(t *testing.T) {
			m := astTestManager(t)
			s, _, _ := satSolverCheckFixture(t)
			tr, e := NewGoalTranslator(m, s)
			satOK(t, e)
			t.Cleanup(func() { satOK(t, tr.Close()) })
			p := goalTestAtom(t, m, "p")
			q := goalTestAtom(t, m, "q")
			label := goalTestApp(t, m, ast.ASTOpOr, p, q)
			leaf, e := m.DependencyLeaf(label)
			satOK(t, e)
			g := goalTestNew(t, m, false, true)
			satOK(t, g.AssertWithProof(p, ast.ASTHandle{}, leaf))
			satOK(t, g.AssertWithProof(q, ast.ASTHandle{}, leaf))
			satOK(t, tr.CompileGoal(g))
			if len(tr.dependencyTrail) != 2 || tr.dependencyTrail[0] == tr.dependencyTrail[1] {
				t.Fatal("source repeated compound labels must create fresh guards")
			}
			a, e := tr.AppendDependencyAssumptions(nil)
			satOK(t, e)
			if len(tr.dependencyTrail) != 2 {
				t.Fatal("query retired the source dependency trail")
			}
			if len(a) != 1 {
				t.Fatal("source dep map overwrites same label", a)
			}
			for i, h := range []ast.ASTHandle{p, q} {
				v, e := tr.Variable(h)
				satOK(t, e)
				a = append(a, sat.MakeLiteral(v, bits&(1<<i) == 0))
			}
			r, e := s.Check(a)
			satOK(t, e)
			want := lbool.False
			if bits == 3 {
				want = lbool.True
			}
			if r != want {
				t.Fatal("independent label/guard truth table", bits, r, want)
			}
		})
	}
}

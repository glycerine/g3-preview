package portfolio_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/solver"
	"github.com/glycerine/g3/pkg/tactic/portfolio"
	"github.com/glycerine/g3/pkg/util/lbool"
	"github.com/glycerine/g3/pkg/util/statistics"
)

func pbSolverFixture(t *testing.T) (*enumFixture, *portfolio.PB2BVSolver) {
	t.Helper()
	f := enumSolverFixture(t)
	s, e := portfolio.NewPB2BVSolver(f.manager, f.services.Context, nil, f.s)
	enumMust(t, e)
	t.Cleanup(func() { enumMust(t, s.Close()) })
	return f, s
}
func pbAtoms(t *testing.T, m *ast.ASTManager, n int) []ast.ASTHandle {
	t.Helper()
	sort, e := m.MkBoolSort()
	enumMust(t, e)
	var out []ast.ASTHandle
	for i := 0; i < n; i++ {
		name, e := m.Symbols.Intern(fmt.Sprint("pb-b", i))
		enumMust(t, e)
		h, e := m.MkConst(name, sort)
		enumMust(t, e)
		enumPin(t, m, h)
		out = append(out, h)
	}
	return out
}
func pbStat(s *statistics.Statistics, key statistics.Key) uint32 {
	var out uint32
	for i := 0; i < s.Size(); i++ {
		if s.IsUint(i) && s.Key(i) == key {
			out += s.UintValue(i)
		}
	}
	return out
}
func TestPB2BVSolverBuffersAndFlushesAtSourceBoundaries(t *testing.T) {
	for _, boundary := range []string{"check", "push", "count", "assertion", "consequences"} {
		t.Run(boundary, func(t *testing.T) {
			f, s := pbSolverFixture(t)
			m := f.manager
			vars := pbAtoms(t, m, 3)
			h, e := m.MkPseudoBooleanAtLeast(vars, 1)
			enumMust(t, e)
			enumPin(t, m, h)
			enumMust(t, s.Assert(h))
			n, e := f.backend.NumAssertions()
			enumMust(t, e)
			if n != 0 {
				t.Fatal("assertion forwarded eagerly")
			}
			var stats statistics.Statistics
			enumMust(t, s.CollectStatistics(&stats))
			n, e = f.backend.NumAssertions()
			enumMust(t, e)
			if n != 0 {
				t.Fatal("statistics flushed assertion buffer")
			}
			switch boundary {
			case "check":
				status, e := s.CheckSat(nil)
				enumMust(t, e)
				if status != lbool.True {
					t.Fatal(status)
				}
			case "push":
				enumMust(t, s.Push())
				enumMust(t, s.Pop(1))
			case "count":
				_, e = s.NumAssertions()
				enumMust(t, e)
			case "assertion":
				_, e = s.Assertion(0)
				enumMust(t, e)
			case "consequences":
				_, _, e = s.GetConsequences(nil, vars[:1])
				enumMust(t, e)
			}
			n, e = f.backend.NumAssertions()
			enumMust(t, e)
			if n != 1 {
				t.Fatal("flush did not assert exactly one rewritten OR", boundary, n)
			}
			got, e := f.backend.Assertion(0)
			enumMust(t, e)
			kind, e := m.GetDeclKind(got)
			enumMust(t, e)
			if kind != ast.ASTOpOr {
				t.Fatal("PB assertion was not compiled before forwarding", kind)
			}
			_, e = s.NumAssertions()
			enumMust(t, e)
			n, e = f.backend.NumAssertions()
			enumMust(t, e)
			if n != 1 {
				t.Fatal("successful flush kept pending assertions", n)
			}
		})
	}
}
func TestPB2BVSolverPopDiscardsPendingAndPreservesZeroPop(t *testing.T) {
	f, s := pbSolverFixture(t)
	m := f.manager
	p := pbAtoms(t, m, 1)[0]
	enumMust(t, s.Push())
	falsity, e := m.MkFalse()
	enumMust(t, e)
	enumMust(t, s.Assert(falsity))
	enumMust(t, s.Pop(1))
	status, e := s.CheckSat(nil)
	enumMust(t, e)
	if status != lbool.True {
		t.Fatal("popped pending assertion reached backend", status)
	}
	enumMust(t, s.Assert(p))
	enumMust(t, s.Pop(0))
	n, e := s.NumAssertions()
	enumMust(t, e)
	if n != 1 {
		t.Fatal("zero pop discarded buffer", n)
	}
}
func TestPB2BVSolverFreshModelFilteringAfterDestruction(t *testing.T) {
	f, s := pbSolverFixture(t)
	m := f.manager
	vars := pbAtoms(t, m, 7)
	h, e := m.MkPseudoBooleanAtMost(vars, 2)
	enumMust(t, e)
	enumPin(t, m, h)
	enumMust(t, s.Assert(h))
	enumMust(t, s.Assert(vars[0]))
	enumMust(t, s.Assert(vars[1]))
	status, e := s.CheckSat(nil)
	enumMust(t, e)
	if status != lbool.True {
		t.Fatal(status)
	}
	var stats statistics.Statistics
	enumMust(t, s.CollectStatistics(&stats))
	if pbStat(&stats, "pb-aux-variables") == 0 || pbStat(&stats, "pb-aux-clauses") == 0 {
		t.Fatal("fixture did not compile fresh variables and side clauses")
	}
	md, e := s.AcquireModel()
	enumMust(t, e)
	if md == nil {
		t.Fatal("missing filtered model")
	}
	t.Cleanup(func() { enumMust(t, md.DecRef()) })
	raw, e := f.backend.AcquireModel()
	enumMust(t, e)
	if raw == nil {
		t.Fatal("missing backend model")
	}
	rawN, e := raw.NumConstants()
	enumMust(t, e)
	n, e := md.NumConstants()
	enumMust(t, e)
	if rawN <= n {
		t.Fatal("fixture did not expose model filtering", rawN, n)
	}
	enumMust(t, raw.DecRef())
	enumMust(t, s.Close())
	for i := uint32(0); i < n; i++ {
		d, e := md.Constant(i)
		enumMust(t, e)
		name, e := m.GetASTName(d)
		enumMust(t, e)
		text, e := m.Symbols.String(name)
		enumMust(t, e)
		if strings.HasPrefix(text, "sn") {
			t.Fatal("private network declaration escaped filtering", text)
		}
	}
	evaluator, e := model.NewModelEvaluator(md, f.services)
	enumMust(t, e)
	t.Cleanup(func() { enumMust(t, evaluator.Close()) })
	out, e := m.NewRef(ast.ASTHandle{})
	enumMust(t, e)
	t.Cleanup(func() { enumMust(t, m.DelRef(&out)) })
	enumMust(t, evaluator.Evaluate(h, out))
	value, e := m.RefGet(out)
	enumMust(t, e)
	yes := m.IsTrue(value)
	if !yes {
		t.Fatal("filtered model violates original PB constraint")
	}
}
func TestPB2BVSolverNamedCoreAndNoFreshModelIdentity(t *testing.T) {
	f, s := pbSolverFixture(t)
	m := f.manager
	p := pbAtoms(t, m, 1)[0]
	first := enumTag(t, m, "pb-first")
	second := enumTag(t, m, "pb-second")
	enumMust(t, s.AssertTracked(p, first))
	enumMust(t, s.Push())
	not, e := m.MkNot(p)
	enumMust(t, e)
	enumMust(t, s.AssertTracked(not, second))
	status, e := s.CheckSat(nil)
	enumMust(t, e)
	if status != lbool.False {
		t.Fatal(status)
	}
	core, e := s.UnsatCore()
	enumMust(t, e)
	seen := map[ast.ASTHandle]bool{}
	for _, h := range core {
		seen[h] = true
	}
	if !seen[first] || !seen[second] {
		t.Fatal("named core lost a buffered assertion", core)
	}
	enumMust(t, s.Pop(1))
	status, e = s.CheckSat(nil)
	enumMust(t, e)
	if status != lbool.True {
		t.Fatal(status)
	}
	underlying, e := f.s.AcquireModel()
	enumMust(t, e)
	// Enum2bv itself returns a filtered model on every call, so compare the
	// PB wrapper's identity behavior with an independent plain SMT backend.
	enumMust(t, underlying.DecRef())
	plain, e := portfolio.NewPB2BVSolver(m, f.services.Context, nil, f.backend)
	enumMust(t, e)
	t.Cleanup(func() { enumMust(t, plain.Close()) })
	md, e := f.backend.AcquireModel()
	enumMust(t, e)
	got, e := plain.AcquireModel()
	enumMust(t, e)
	if got != md {
		t.Fatal("empty fresh vector unnecessarily replaced model")
	}
	enumMust(t, got.DecRef())
	enumMust(t, md.DecRef())
}

var _ solver.Solver = (*portfolio.PB2BVSolver)(nil)

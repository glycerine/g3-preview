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

func boundedFixture(t *testing.T) (*enumFixture, *portfolio.BoundedInt2BVSolver) {
	t.Helper()
	f := enumSolverFixture(t)
	s, err := portfolio.NewBoundedInt2BVSolver(f.manager, f.services.Context, nil, f.s, f.services)
	enumMust(t, err)
	t.Cleanup(func() { enumMust(t, s.Close()) })
	return f, s
}
func boundedConst(t *testing.T, m *ast.ASTManager, name string) ast.ASTHandle {
	t.Helper()
	family, err := m.RegisterArithmeticPlugin()
	enumMust(t, err)
	sort, err := m.MkFamilySort(family, 1, nil)
	enumMust(t, err)
	symbol, err := m.Symbols.Intern(name)
	enumMust(t, err)
	h, err := m.MkConst(symbol, sort)
	enumMust(t, err)
	enumPin(t, m, h)
	return h
}
func boundedNumber(t *testing.T, m *ast.ASTManager, n int64) ast.ASTHandle {
	t.Helper()
	h, err := m.ArithmeticIntegerNumeralText(fmt.Sprint(n))
	enumMust(t, err)
	enumPin(t, m, h)
	return h
}
func boundedOp(t *testing.T, m *ast.ASTManager, op int32, args ...ast.ASTHandle) ast.ASTHandle {
	t.Helper()
	family, err := m.RegisterArithmeticPlugin()
	enumMust(t, err)
	h, err := m.MkFamilyApp(family, op, nil, args)
	enumMust(t, err)
	enumPin(t, m, h)
	return h
}
func boundedBounds(t *testing.T, m *ast.ASTManager, s solver.Solver, x ast.ASTHandle, lo, hi int64) {
	t.Helper()
	enumMust(t, s.Assert(boundedOp(t, m, 3, x, boundedNumber(t, m, lo))))
	enumMust(t, s.Assert(boundedOp(t, m, 2, x, boundedNumber(t, m, hi))))
}
func boundedValue(t *testing.T, m *ast.ASTManager, md *model.ModelCore, services model.ModelEvaluatorConfig, x ast.ASTHandle) string {
	t.Helper()
	ev, err := model.NewModelEvaluator(md, services)
	enumMust(t, err)
	defer func() { enumMust(t, ev.Close()) }()
	ref, err := ev.EvaluateValue(x)
	enumMust(t, err)
	defer func() { enumMust(t, m.DelRef(&ref)) }()
	h, err := m.RefGet(ref)
	enumMust(t, err)
	text, integer, ok, err := m.ArithmeticNumeralText(h)
	enumMust(t, err)
	if !integer || !ok {
		t.Fatal("model did not reconstruct integer", h)
	}
	return text
}
func TestBoundedInt2BVSolverFlushBoundariesAndModels(t *testing.T) {
	for _, boundary := range []string{"count", "assertion", "push", "check", "consequences"} {
		t.Run(boundary, func(t *testing.T) {
			f := enumSolverFixture(t)
			recorder := &pbRecordingBackend{Solver: f.s}
			s, err := portfolio.NewBoundedInt2BVSolver(f.manager, f.services.Context, nil, recorder, f.services)
			enumMust(t, err)
			t.Cleanup(func() { enumMust(t, s.Close()) })
			m := f.manager
			x := boundedConst(t, m, "bounded-x")
			boundedBounds(t, m, s, x, -3, 5)
			n, err := f.backend.NumAssertions()
			enumMust(t, err)
			if n != 0 {
				t.Fatal("eager assertion forwarding")
			}
			var stats statistics.Statistics
			enumMust(t, s.CollectStatistics(&stats))
			n, err = f.backend.NumAssertions()
			enumMust(t, err)
			if n != 0 {
				t.Fatal("statistics flushed buffer")
			}
			switch boundary {
			case "count":
				_, err = s.NumAssertions()
			case "assertion":
				_, err = s.Assertion(0)
			case "push":
				err = s.Push()
			case "check":
				var status lbool.LBool
				status, err = s.CheckSat(nil)
				if status != lbool.True {
					t.Fatal("bounded check", status, err)
				}
			case "consequences":
				_, _, err = s.GetConsequences(nil, []ast.ASTHandle{x})
			}
			enumMust(t, err)
			n, err = f.backend.NumAssertions()
			enumMust(t, err)
			if len(recorder.asserted) != 3 {
				t.Fatal("expected two converted bounds and source range guard", len(recorder.asserted), "internal backend count", n)
			}
			eq, err := m.MkEq(x, boundedNumber(t, m, 2))
			enumMust(t, err)
			enumPin(t, m, eq)
			enumMust(t, s.Assert(eq))
			status, err := s.CheckSat(nil)
			enumMust(t, err)
			if status != lbool.True {
				t.Fatal("constrained model", status)
			}
			md, err := s.AcquireModel()
			enumMust(t, err)
			if md == nil {
				t.Fatal("missing model")
			}
			defer func() { enumMust(t, md.DecRef()) }()
			if got := boundedValue(t, m, md, f.services, x); got != "2" {
				t.Fatal("offset model reconstruction", got)
			}
			count, err := md.NumConstants()
			enumMust(t, err)
			for i := uint32(0); i < count; i++ {
				d, err := md.Constant(i)
				enumMust(t, err)
				name, err := m.GetASTName(d)
				enumMust(t, err)
				text, err := m.Symbols.String(name)
				enumMust(t, err)
				if strings.HasPrefix(text, "b!") {
					t.Fatal("helper escaped filter", text)
				}
			}
			if boundary == "check" {
				enumMust(t, s.Close())
				if got := boundedValue(t, m, md, f.services, x); got != "2" {
					t.Fatal("model depended on closed wrapper", got)
				}
			}
		})
	}
}
func TestBoundedInt2BVSolverConjunctionOrderAndFailurePrefix(t *testing.T) {
	f := enumSolverFixture(t)
	m := f.manager
	b := &pbRecordingBackend{Solver: f.s}
	s, err := portfolio.NewBoundedInt2BVSolver(m, f.services.Context, nil, b, f.services)
	enumMust(t, err)
	t.Cleanup(func() { enumMust(t, s.Close()) })
	p := pbAtoms(t, m, 4)
	inner, err := m.MkAndArgs([]ast.ASTHandle{p[1], p[2]})
	enumMust(t, err)
	enumPin(t, m, inner)
	root, err := m.MkAndArgs([]ast.ASTHandle{p[0], inner, p[3]})
	enumMust(t, err)
	enumPin(t, m, root)
	enumMust(t, s.Assert(root))
	_, err = s.NumAssertions()
	enumMust(t, err)
	want := []ast.ASTHandle{p[3], p[0], p[2], p[1]}
	if len(b.asserted) != len(want) {
		t.Fatal("flattened count", len(b.asserted))
	}
	for i, h := range want {
		if b.asserted[i] != h {
			t.Fatal("source AND tail replacement order", i)
		}
	}
}
func TestBoundedInt2BVSolverScopedBoundsAndConsequences(t *testing.T) {
	f, s := boundedFixture(t)
	m := f.manager
	x := boundedConst(t, m, "scope-x")
	boundedBounds(t, m, s, x, 0, 10)
	enumMust(t, s.Push())
	enumMust(t, s.Assert(boundedOp(t, m, 2, x, boundedNumber(t, m, 3))))
	enumMust(t, s.Assert(boundedOp(t, m, 3, x, boundedNumber(t, m, 3))))
	status, cons, err := s.GetConsequences(nil, []ast.ASTHandle{x})
	enumMust(t, err)
	if status != lbool.True || len(cons) != 1 {
		t.Fatal("integer consequence", status, len(cons))
	}
	head, err := m.GetArg(cons[0], 1)
	enumMust(t, err)
	lhs, err := m.GetArg(head, 0)
	enumMust(t, err)
	if lhs != x {
		t.Fatal("consequence not projected to original integer")
	}
	rhs, err := m.GetArg(head, 1)
	enumMust(t, err)
	text, _, ok, err := m.ArithmeticNumeralText(rhs)
	enumMust(t, err)
	if !ok || text != "3" {
		t.Fatal("projected consequence value", text)
	}
	// This local variable's helper must disappear when its declaring scope pops.
	y := boundedConst(t, m, "local-y")
	boundedBounds(t, m, s, y, -2, 2)
	status, err = s.CheckSat(nil)
	enumMust(t, err)
	if status != lbool.True {
		t.Fatal("inner scope", status)
	}
	enumMust(t, s.Pop(1))
	enumMust(t, s.Assert(boundedOp(t, m, 3, x, boundedNumber(t, m, 8))))
	status, err = s.CheckSat(nil)
	enumMust(t, err)
	if status != lbool.True {
		t.Fatal("popped local bounds survived", status)
	}
	md, err := s.AcquireModel()
	enumMust(t, err)
	defer func() { enumMust(t, md.DecRef()) }()
	decl, err := m.GetDecl(y)
	enumMust(t, err)
	found, err := md.HasInterpretation(decl)
	enumMust(t, err)
	if found {
		t.Fatal("popped integer map survived")
	}
}

func TestBoundedInt2BVSolverNamedCoreAndUnconvertedBounds(t *testing.T) {
	t.Run("named-core", func(t *testing.T) {
		f, s := boundedFixture(t)
		m := f.manager
		x := boundedConst(t, m, "core-x")
		boundedBounds(t, m, s, x, 0, 2)
		tag := enumTag(t, m, "bounded-core")
		enumMust(t, s.Push())
		enumMust(t, s.AssertTracked(boundedOp(t, m, 3, x, boundedNumber(t, m, 3)), tag))
		status, err := s.CheckSat(nil)
		enumMust(t, err)
		if status != lbool.False {
			t.Fatal("expected conflict", status)
		}
		core, err := s.UnsatCore()
		enumMust(t, err)
		if len(core) != 1 || core[0] != tag {
			t.Fatal("named core not retained", core)
		}
		enumMust(t, s.Pop(1))
		status, err = s.CheckSat(nil)
		enumMust(t, err)
		if status != lbool.True {
			t.Fatal("popped named conflict survived", status)
		}
	})
	for _, contradictory := range []bool{false, true} {
		t.Run(fmt.Sprint("unconverted-", contradictory), func(t *testing.T) {
			f := enumSolverFixture(t)
			m := f.manager
			b := &pbRecordingBackend{Solver: f.s}
			s, err := portfolio.NewBoundedInt2BVSolver(m, f.services.Context, nil, b, f.services)
			enumMust(t, err)
			t.Cleanup(func() { enumMust(t, s.Close()) })
			x := boundedConst(t, m, "unconverted-x")
			lower := boundedOp(t, m, 3, x, boundedNumber(t, m, 5))
			enumMust(t, s.Assert(lower))
			var upper ast.ASTHandle
			if contradictory {
				upper = boundedOp(t, m, 2, x, boundedNumber(t, m, 3))
				enumMust(t, s.Assert(upper))
			}
			_, err = s.NumAssertions()
			enumMust(t, err)
			if b.asserted[0] != lower || contradictory && b.asserted[1] != upper {
				t.Fatal("missing/inconsistent bounds were converted")
			}
		})
	}
}

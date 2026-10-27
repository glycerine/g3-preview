package portfolio_test

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/solver"
	"github.com/glycerine/g3/pkg/tactic/portfolio"
	"github.com/glycerine/g3/pkg/util/lbool"
	"github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/statistics"
)

func fdFixture(t *testing.T) (*enumFixture, solver.Solver) {
	t.Helper()
	f := enumSolverFixture(t)
	s, err := portfolio.NewFDSolver(f.manager, nil, f.services)
	enumMust(t, err)
	t.Cleanup(func() { enumMust(t, s.Close()) })
	return f, s
}
func TestFDSolverRetainedMixedStackAndScopes(t *testing.T) {
	f, s := fdFixture(t)
	m := f.manager
	if _, ok := s.(*portfolio.BoundedInt2BVSolver); !ok {
		t.Fatal("wrong outer source wrapper")
	}
	_, enumFormula, _ := enumGoal(t, f)
	integer := boundedConst(t, m, "fd-integer")
	boundedBounds(t, m, s, integer, -3, 3)
	square := boundedOp(t, m, 9, integer, integer)
	squareEq, err := m.MkEq(square, boundedNumber(t, m, 4))
	enumMust(t, err)
	enumPin(t, m, squareEq)
	bools := pbAtoms(t, m, 3)
	pb, err := m.MkPseudoBooleanAtLeast(bools, 2)
	enumMust(t, err)
	enumPin(t, m, pb)
	neg, err := m.MkNot(bools[2])
	enumMust(t, err)
	enumPin(t, m, neg)
	forms := []ast.ASTHandle{enumFormula, squareEq, pb, neg}
	for _, h := range forms {
		enumMust(t, s.Assert(h))
	}
	status, err := s.CheckSat(nil)
	enumMust(t, err)
	if status != lbool.True {
		reason, _ := s.ReasonUnknown()
		t.Fatal("mixed retained stack", status, reason)
	}
	md, err := s.AcquireModel()
	enumMust(t, err)
	if md == nil {
		t.Fatal("missing FD model")
	}
	defer func() { enumMust(t, md.DecRef()) }()
	ev, err := model.NewModelEvaluator(md, f.services)
	enumMust(t, err)
	defer func() { enumMust(t, ev.Close()) }()
	for _, h := range forms {
		ref, err := ev.EvaluateValue(h)
		enumMust(t, err)
		v, err := m.RefGet(ref)
		enumMust(t, err)
		enumMust(t, m.DelRef(&ref))
		if !m.IsTrue(v) {
			t.Fatal("reconstructed mixed model violates original assertion")
		}
	}
	enumMust(t, s.Push())
	enumMust(t, s.Assert(boundedOp(t, m, 3, integer, boundedNumber(t, m, 3))))
	status, err = s.CheckSat(nil)
	enumMust(t, err)
	if status != lbool.False {
		t.Fatal("scoped nonlinear conflict", status)
	}
	enumMust(t, s.Pop(1))
	status, err = s.CheckSat(nil)
	enumMust(t, err)
	if status != lbool.True {
		t.Fatal("SAT user scope did not pop", status)
	}
}
func TestFDSolverConstructorRefusals(t *testing.T) {
	f := enumSolverFixture(t)
	m := f.manager
	warm, err := portfolio.NewFDSolver(m, nil, f.services)
	enumMust(t, err)
	enumMust(t, warm.Close())
	baseline := m.Budget().Used()
	references := f.services.Context.Stats().References
	completed := false
	for ordinal := int64(0); ordinal < 1024; ordinal++ {
		m.Budget().FailAfter(ordinal)
		s, err := portfolio.NewFDSolver(m, nil, f.services)
		m.Budget().FailAfter(-1)
		if err != nil && !errors.Is(err, arena.ErrInjectedFailure) {
			t.Fatal("unexpected factory error", ordinal, err)
		}
		if s != nil {
			enumMust(t, s.Close())
		}
		var failure memory.ConstructionFailure
		if errors.As(err, &failure) {
			if cleanup := failure.Cleanup(); cleanup != nil {
				t.Fatalf("factory cleanup at refusal %d: %T: %v", ordinal, failure, cleanup)
			}
		}
		if used := m.Budget().Used(); used != baseline {
			t.Fatal("factory leaked constructor prefix", ordinal, used, baseline)
		}
		if f.services.Context.Stats().References != references {
			t.Fatal("factory retained parameters", ordinal)
		}
		if err == nil {
			completed = true
			break
		}
	}
	if !completed {
		t.Fatal("factory census did not reach success")
	}
}

func TestFDSolverMixedConsequencesAndModelAfterClose(t *testing.T) {
	f, s := fdFixture(t)
	m := f.manager
	_, enumFormula, colour := enumGoal(t, f)
	enumMust(t, s.Assert(enumFormula))
	x := boundedConst(t, m, "fd-consequence-x")
	boundedBounds(t, m, s, x, -5, 5)
	eq, err := m.MkEq(x, boundedNumber(t, m, -2))
	enumMust(t, err)
	enumPin(t, m, eq)
	enumMust(t, s.Assert(eq))
	p := pbAtoms(t, m, 1)[0]
	enumMust(t, s.Assert(p))
	status, out, err := s.GetConsequences(nil, []ast.ASTHandle{colour, x, p})
	enumMust(t, err)
	if status != lbool.True || len(out) != 3 {
		t.Fatal("mixed consequences", status, len(out))
	}
	seen := map[ast.ASTHandle]bool{}
	for _, h := range out {
		head, err := m.GetArg(h, 1)
		enumMust(t, err)
		if head == p {
			seen[p] = true
			continue
		}
		left, err := m.GetArg(head, 0)
		enumMust(t, err)
		seen[left] = true
		if left == x {
			right, err := m.GetArg(head, 1)
			enumMust(t, err)
			text, _, ok, err := m.ArithmeticNumeralText(right)
			enumMust(t, err)
			if !ok || text != "-2" {
				t.Fatal("integer consequence offset", text)
			}
		}
	}
	if !seen[colour] || !seen[x] || !seen[p] {
		t.Fatal("helper variable escaped consequence projection")
	}
	status, err = s.CheckSat(nil)
	enumMust(t, err)
	if status != lbool.True {
		t.Fatal(status)
	}
	md, err := s.AcquireModel()
	enumMust(t, err)
	defer func() { enumMust(t, md.DecRef()) }()
	enumMust(t, s.Close())
	if value := boundedValue(t, m, md, f.services, x); value != "-2" {
		t.Fatal("FD model depended on dead solver", value)
	}
}

func TestFDSolverWideWeightedPseudoBooleanTruthTable(t *testing.T) {
	f, s := fdFixture(t)
	m := f.manager
	vars := pbAtoms(t, m, 3)
	numbers := make([]unumeral.OwnedRational, 4)
	for i, value := range []int64{4294967296, 4294967296, 1, 4294967296} {
		q, err := m.NumeralService().NewValue()
		enumMust(t, err)
		numbers[i] = q
		enumMust(t, m.NumeralService().SetInt64(&numbers[i], value))
		t.Cleanup(func() { enumMust(t, m.NumeralService().Del(&numbers[i])) })
	}
	h, err := m.MkPseudoBooleanLE(numbers[:3], vars, &numbers[3])
	enumMust(t, err)
	enumPin(t, m, h)
	enumMust(t, s.Assert(h))
	for mask := 0; mask < 8; mask++ {
		var assumptions []ast.ASTHandle
		sum := int64(0)
		for i, p := range vars {
			if mask&(1<<i) != 0 {
				assumptions = append(assumptions, p)
				if i < 2 {
					sum += 4294967296
				} else {
					sum++
				}
			} else {
				neg, err := m.MkNot(p)
				enumMust(t, err)
				enumPin(t, m, neg)
				assumptions = append(assumptions, neg)
			}
		}
		status, err := s.CheckSat(assumptions)
		enumMust(t, err)
		want := lbool.False
		if sum <= 4294967296 {
			want = lbool.True
		}
		if status != want {
			reason, _ := s.ReasonUnknown()
			t.Fatal("weighted full-stack truth table", mask, status, want, reason)
		}
	}
}

func TestFDSolverSortingNetworkStatisticsAndFiltering(t *testing.T) {
	f, s := fdFixture(t)
	m := f.manager
	vars := pbAtoms(t, m, 7)
	h, err := m.MkPseudoBooleanAtMost(vars, 2)
	enumMust(t, err)
	enumPin(t, m, h)
	enumMust(t, s.Assert(h))
	enumMust(t, s.Assert(vars[0]))
	enumMust(t, s.Assert(vars[1]))
	status, err := s.CheckSat(nil)
	enumMust(t, err)
	if status != lbool.True {
		t.Fatal(status)
	}
	var stats statistics.Statistics
	enumMust(t, s.CollectStatistics(&stats))
	// The prebuilt C++ FD stack reports these two component counters for
	// exactly this seven-input/two-limit sorting network and fixed inputs.
	if pbStat(&stats, "pb-aux-variables") != 9 || pbStat(&stats, "pb-aux-clauses") != 54 {
		t.Fatal("missing or different source PB compilation statistics", pbStat(&stats, "pb-aux-variables"), pbStat(&stats, "pb-aux-clauses"))
	}
	md, err := s.AcquireModel()
	enumMust(t, err)
	defer func() { enumMust(t, md.DecRef()) }()
	n, err := md.NumConstants()
	enumMust(t, err)
	if n != 7 {
		t.Fatal("network helpers escaped full-stack model", n)
	}
}

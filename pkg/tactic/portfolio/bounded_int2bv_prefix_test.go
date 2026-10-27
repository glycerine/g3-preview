package portfolio_test

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/tactic/portfolio"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestBoundedInt2BVSolverFailurePrefixAndPopDiscard(t *testing.T) {
	f := enumSolverFixture(t)
	m := f.manager
	b := &pbRecordingBackend{Solver: f.s, rejectAt: 2, stop: errors.New("bounded flush stopped")}
	s, err := portfolio.NewBoundedInt2BVSolver(m, f.services.Context, nil, b, f.services)
	enumMust(t, err)
	t.Cleanup(func() { enumMust(t, s.Close()) })
	vars := pbAtoms(t, m, 2)
	for _, h := range vars {
		enumMust(t, s.Assert(h))
	}
	if _, err = s.NumAssertions(); !errors.Is(err, b.stop) {
		t.Fatal("missing flush error", err)
	}
	b.rejectAt = 0
	_, err = s.NumAssertions()
	enumMust(t, err)
	if len(b.asserted) != 4 || b.asserted[2] != vars[0] || b.asserted[3] != vars[1] {
		t.Fatal("failed flush lost buffer or forwarded prefix")
	}
	enumMust(t, s.Push())
	enumMust(t, s.Assert(vars[0]))
	enumMust(t, s.Pop(0))
	before := len(b.asserted)
	_, err = s.NumAssertions()
	enumMust(t, err)
	if len(b.asserted) != before+1 {
		t.Fatal("pop0 discarded source buffer")
	}
	enumMust(t, s.Assert(vars[1]))
	b.popError = errors.New("bounded pop stopped")
	if err = s.Pop(1); !errors.Is(err, b.popError) {
		t.Fatal("missing pop error", err)
	}
	before = len(b.asserted)
	_, err = s.NumAssertions()
	enumMust(t, err)
	if len(b.asserted) != before {
		t.Fatal("stopped backend pop preserved pending inputs")
	}
	b.popError = nil
	enumMust(t, s.Pop(1))
}
func TestBoundedInt2BVSolverConstructorRefusalsParametersAndTranslate(t *testing.T) {
	f := enumSolverFixture(t)
	m, c := f.manager, f.services.Context
	enumMust(t, f.s.IncRef())
	t.Cleanup(func() { enumMust(t, f.s.DecRef()) })
	p, err := c.NewValue()
	enumMust(t, err)
	t.Cleanup(func() { enumMust(t, c.Del(&p)) })
	key := "bounded-constructor-param"
	enumMust(t, c.SetUIntName(&p, &key, 17))
	b := &pbRecordingBackend{Solver: f.s}
	warm, err := portfolio.NewBoundedInt2BVSolver(m, c, &p, b, f.services)
	enumMust(t, err)
	enumMust(t, warm.Close())
	baseline := m.Budget().Used()
	refs := c.Stats().References
	complete := false
	for ordinal := int64(0); ordinal < 128; ordinal++ {
		m.Budget().FailAfter(ordinal)
		s, err := portfolio.NewBoundedInt2BVSolver(m, c, &p, b, f.services)
		m.Budget().FailAfter(-1)
		if err != nil && !errors.Is(err, arena.ErrInjectedFailure) {
			t.Fatal("unexpected constructor refusal", ordinal, err)
		}
		if s != nil {
			enumMust(t, s.Close())
		}
		if m.Budget().Used() != baseline || c.Stats().References != refs {
			t.Fatal("constructor owner retained", ordinal, m.Budget().Used(), baseline)
		}
		if err == nil {
			complete = true
			break
		}
	}
	if !complete {
		t.Fatal("constructor refusal census did not complete")
	}
	if b.updates != 0 {
		t.Fatal("constructor called virtual backend update")
	}
	s, err := portfolio.NewBoundedInt2BVSolver(m, c, &p, b, f.services)
	enumMust(t, err)
	t.Cleanup(func() { enumMust(t, s.Close()) })
	enumMust(t, c.SetUIntName(&p, &key, 29))
	params, err := s.Params()
	enumMust(t, err)
	value, err := c.GetUIntName(params, &key, 0)
	enumMust(t, err)
	if value != 17 {
		t.Fatal("base params were borrowed")
	}
	enumMust(t, s.UpdateParams(&p))
	if b.updates != 1 {
		t.Fatal("update not forwarded")
	}
	atom := pbAtoms(t, m, 1)[0]
	enumMust(t, s.Assert(atom))
	target, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: m.Symbols, Numerals: m.NumeralService(), Budget: m.Budget()})
	enumMust(t, err)
	t.Cleanup(func() { enumMust(t, target.Close()) })
	translated, err := s.Translate(target, nil)
	enumMust(t, err)
	t.Cleanup(func() { enumMust(t, translated.Close()) })
	n, err := translated.NumAssertions()
	enumMust(t, err)
	if n != 0 {
		t.Fatal("translate copied pending input", n)
	}
	if len(b.asserted) != 0 {
		t.Fatal("translate flushed source")
	}
	n, err = s.NumAssertions()
	enumMust(t, err)
	if n != 1 {
		t.Fatal("source lost pending input", n)
	}
}
func TestBoundedInt2BVSolverNonzeroOffsetConsequence(t *testing.T) {
	f, s := boundedFixture(t)
	m := f.manager
	x := boundedConst(t, m, "offset-x")
	boundedBounds(t, m, s, x, -9, -2)
	eq, err := m.MkEq(x, boundedNumber(t, m, -4))
	enumMust(t, err)
	enumPin(t, m, eq)
	enumMust(t, s.Assert(eq))
	_, out, err := s.GetConsequences(nil, []ast.ASTHandle{x})
	enumMust(t, err)
	if len(out) != 1 {
		t.Fatal("missing consequence", len(out))
	}
	head, err := m.GetArg(out[0], 1)
	enumMust(t, err)
	lhs, err := m.GetArg(head, 0)
	enumMust(t, err)
	rhs, err := m.GetArg(head, 1)
	enumMust(t, err)
	text, _, ok, err := m.ArithmeticNumeralText(rhs)
	enumMust(t, err)
	if lhs != x || !ok || text != "-4" {
		t.Fatal("source num+offset projection", text)
	}
}

func TestBoundedInt2BVSolverEmptyInspectionAndIndependentStatus(t *testing.T) {
	f, s := boundedFixture(t)
	allocations := testing.AllocsPerRun(10, func() { _, e := s.NumAssertions(); enumMust(t, e) })
	if allocations != 0 {
		t.Fatal("empty source inspection allocated", allocations)
	}
	f.manager.Budget().FailAfter(0)
	n, err := s.NumAssertions()
	f.manager.Budget().FailAfter(-1)
	enumMust(t, err)
	if n != 0 {
		t.Fatal("empty wrapper has assertions", n)
	}
	own, err := s.Status()
	enumMust(t, err)
	if own != lbool.Undef {
		t.Fatal("initial base status", own)
	}
	result, err := s.CheckSat(nil)
	enumMust(t, err)
	if result != lbool.True {
		t.Fatal("empty solver", result)
	}
	own, err = s.Status()
	enumMust(t, err)
	if own != lbool.Undef {
		t.Fatal("CheckSat updated source base status", own)
	}
	enumMust(t, s.SetStatus(lbool.False))
	backend, err := f.backend.Status()
	enumMust(t, err)
	if backend != lbool.True {
		t.Fatal("SetStatus forwarded to backend", backend)
	}
}

func TestBoundedInt2BVSolverWideOffsetModelsAndConsequences(t *testing.T) {
	f, s := boundedFixture(t)
	m := f.manager
	x := boundedConst(t, m, "wide-offset-x")
	number := func(text string) ast.ASTHandle {
		h, err := m.ArithmeticIntegerNumeralText(text)
		enumMust(t, err)
		enumPin(t, m, h)
		return h
	}
	lo, hi, value := number("36893488147419103232"), number("36893488147419103235"), number("36893488147419103233")
	enumMust(t, s.Assert(boundedOp(t, m, 3, x, lo)))
	enumMust(t, s.Assert(boundedOp(t, m, 2, x, hi)))
	eq, err := m.MkEq(x, value)
	enumMust(t, err)
	enumPin(t, m, eq)
	enumMust(t, s.Assert(eq))
	status, err := s.CheckSat(nil)
	enumMust(t, err)
	if status != lbool.True {
		t.Fatal("wide offset status", status)
	}
	md, err := s.AcquireModel()
	enumMust(t, err)
	defer func() { enumMust(t, md.DecRef()) }()
	if got := boundedValue(t, m, md, f.services, x); got != "36893488147419103233" {
		t.Fatal("wide model offset truncated", got)
	}
	_, out, err := s.GetConsequences(nil, []ast.ASTHandle{x})
	enumMust(t, err)
	if len(out) != 1 {
		t.Fatal("wide consequence count", len(out))
	}
	head, err := m.GetArg(out[0], 1)
	enumMust(t, err)
	rhs, err := m.GetArg(head, 1)
	enumMust(t, err)
	text, _, ok, err := m.ArithmeticNumeralText(rhs)
	enumMust(t, err)
	if !ok || text != "36893488147419103233" {
		t.Fatal("wide projected offset truncated", text)
	}
}

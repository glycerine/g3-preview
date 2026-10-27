package portfolio_test

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/solver"
	"github.com/glycerine/g3/pkg/tactic/portfolio"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

type pbRecordingBackend struct {
	solver.Solver
	asserted []ast.ASTHandle
	rejectAt int
	stop     error
	popError error
	updates  uint32
}

func (b *pbRecordingBackend) Assert(h ast.ASTHandle) error {
	b.asserted = append(b.asserted, h)
	if b.rejectAt != 0 && len(b.asserted) == b.rejectAt {
		return b.stop
	}
	return b.Solver.Assert(h)
}
func (b *pbRecordingBackend) Pop(n uint32) error {
	if b.popError != nil {
		return b.popError
	}
	return b.Solver.Pop(n)
}
func (b *pbRecordingBackend) UpdateParams(*uparams.ParamsRef) error { b.updates++; return nil }
func TestPB2BVSolverFlushFailureKeepsPublishedPrefixAndBuffer(t *testing.T) {
	f := enumSolverFixture(t)
	m := f.manager
	b := &pbRecordingBackend{Solver: f.s, rejectAt: 2, stop: errors.New("backend assertion stopped")}
	s, e := portfolio.NewPB2BVSolver(m, f.services.Context, nil, b)
	enumMust(t, e)
	t.Cleanup(func() { enumMust(t, s.Close()) })
	vars := pbAtoms(t, m, 2)
	for _, h := range vars {
		enumMust(t, s.Assert(h))
	}
	if _, e = s.NumAssertions(); !errors.Is(e, b.stop) {
		t.Fatal("flush error", e)
	}
	if len(b.asserted) != 2 || b.asserted[0] != vars[0] || b.asserted[1] != vars[1] {
		t.Fatal("source forwarding prefix", b.asserted)
	}
	b.rejectAt = 0
	_, e = s.NumAssertions()
	enumMust(t, e)
	if len(b.asserted) != 4 || b.asserted[2] != vars[0] || b.asserted[3] != vars[1] {
		t.Fatal("failed flush reset or rolled back buffer", b.asserted)
	}
	_, e = s.NumAssertions()
	enumMust(t, e)
	if len(b.asserted) != 4 {
		t.Fatal("successful flush did not reset buffer")
	}
}
func TestPB2BVSolverPopFailureStillDiscardsPending(t *testing.T) {
	f := enumSolverFixture(t)
	m := f.manager
	b := &pbRecordingBackend{Solver: f.s}
	s, e := portfolio.NewPB2BVSolver(m, f.services.Context, nil, b)
	enumMust(t, e)
	t.Cleanup(func() { enumMust(t, s.Close()) })
	enumMust(t, s.Push())
	p := pbAtoms(t, m, 1)[0]
	enumMust(t, s.Assert(p))
	b.popError = errors.New("backend pop stopped")
	if e = s.Pop(1); !errors.Is(e, b.popError) {
		t.Fatal("pop error", e)
	}
	// Source resets the pending vector before attempting the backend pop.
	_, e = s.NumAssertions()
	enumMust(t, e)
	if len(b.asserted) != 0 {
		t.Fatal("stopped pop kept pending assertions")
	}
	b.popError = nil
	enumMust(t, s.Pop(1))
}
func TestPB2BVSolverConstructorParametersAndRefusals(t *testing.T) {
	f := enumSolverFixture(t)
	m, c := f.manager, f.services.Context
	enumMust(t, f.s.IncRef())
	t.Cleanup(func() { enumMust(t, f.s.DecRef()) })
	p, e := c.NewValue()
	enumMust(t, e)
	t.Cleanup(func() { enumMust(t, c.Del(&p)) })
	key := "pb-wrapper-parameter"
	enumMust(t, c.SetUIntName(&p, &key, 17))
	b := &pbRecordingBackend{Solver: f.s}
	warm, e := portfolio.NewPB2BVSolver(m, c, &p, b)
	enumMust(t, e)
	enumMust(t, warm.Close())
	baseline := m.Budget().Used()
	refs := c.Stats().References
	complete := false
	for ordinal := int64(0); ordinal < 128; ordinal++ {
		m.Budget().FailAfter(ordinal)
		s, err := portfolio.NewPB2BVSolver(m, c, &p, b)
		m.Budget().FailAfter(-1)
		if err != nil && !errors.Is(err, arena.ErrInjectedFailure) {
			t.Fatal("constructor prefix", ordinal, err)
		}
		if s != nil {
			enumMust(t, s.Close())
		}
		if m.Budget().Used() != baseline || c.Stats().References != refs {
			t.Fatal("constructor retained owner prefix", ordinal, m.Budget().Used(), baseline, c.Stats())
		}
		if err == nil {
			complete = true
			break
		}
	}
	if !complete {
		t.Fatal("constructor census did not finish")
	}
	s, e := portfolio.NewPB2BVSolver(m, c, &p, b)
	enumMust(t, e)
	t.Cleanup(func() { enumMust(t, s.Close()) })
	if b.updates != 0 {
		t.Fatal("constructor invoked virtual backend parameter update")
	}
	enumMust(t, c.SetUIntName(&p, &key, 29))
	enumMust(t, s.UpdateParams(&p))
	local, e := s.Params()
	enumMust(t, e)
	value, e := c.GetUIntName(local, &key, 0)
	enumMust(t, e)
	if value != 29 || b.updates != 1 {
		t.Fatal("parameter forwarding", value, b.updates)
	}
}
func TestPB2BVSolverTranslateDoesNotFlushOrCopyPending(t *testing.T) {
	f, s := pbSolverFixture(t)
	m := f.manager
	p := pbAtoms(t, m, 1)[0]
	enumMust(t, s.Assert(p))
	target, e := ast.NewASTManager(ast.ASTManagerConfig{Symbols: m.Symbols, Numerals: m.NumeralService(), Budget: m.Budget()})
	enumMust(t, e)
	t.Cleanup(func() { enumMust(t, target.Close()) })
	translated, e := s.Translate(target, nil)
	enumMust(t, e)
	t.Cleanup(func() { enumMust(t, translated.Close()) })
	n, e := translated.NumAssertions()
	enumMust(t, e)
	if n != 0 {
		t.Fatal("translate flushed/copied pending source assertions", n)
	}
	n, e = f.backend.NumAssertions()
	enumMust(t, e)
	if n != 0 {
		t.Fatal("translation mutated original backend", n)
	}
	n, e = s.NumAssertions()
	enumMust(t, e)
	if n != 1 {
		t.Fatal("translation lost original pending buffer", n)
	}
}

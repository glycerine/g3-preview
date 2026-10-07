// Source: src/sat/sat_solver.h:286,445; src/sat/sat_solver.cpp:2818-2838.
package sat

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestSATSolverUserBinaryPairPreservesFullLiteralWords(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	satOK(t, s.userBinClauses.Push(newSATSolverBinPair(Literal(0x80000001), Literal(0xfffffffe))))
	p, e := s.userBinClauses.Get(0)
	satOK(t, e)
	if p.Literal1() != Literal(0x80000001) || p.Literal2() != Literal(0xfffffffe) {
		t.Fatal("source pair has two complete uint32 words", p.Literal1(), p.Literal2())
	}
}

// Exact public raw TLS signed-overflow boundary, not ordinary allocation OOM.
func TestSATSolverBinaryPairPostphysicalSourceStopRetainsOwningCursor(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := util.NewRuntime(util.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	satOK(t, e)
	t.Cleanup(func() { satOK(t, r.Close()) })
	v, e := NewBinClauseBuffer(r, b)
	satOK(t, e)
	for i := uint32(0); i < 20000; i++ {
		satOK(t, v.Push(newSATSolverBinPair(Literal(i), Literal(i+1))))
	}
	first, e := r.Allocate(8)
	satOK(t, e)
	second, e := r.Allocate(8)
	satOK(t, e)
	delta := int64(-1000000)
	request := uint64(16) + uint64(delta) - 8
	if e = r.Reallocate(&first, request); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	satOK(t, r.Deallocate(&first))
	state := r.State()
	target := int64(math.MinInt64) + 1000 - state.GlobalSize + 16
	request = uint64(target) + 16 - 8
	if e = r.Reallocate(&second, request); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	satOK(t, r.Deallocate(&second))
	before := b.Used()
	e = v.Close()
	if !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	if v.raw.Block != (util.Block{}) {
		t.Fatal("actual physical free did not advance")
	}
	if !v.closing || v.closed || b.Used() == before {
		t.Fatal("stopped owner/cursor", v.closing, v.closed, b.Used(), before)
	}
	if er := v.Close(); !errors.Is(er, e) {
		t.Error("source cause lost on ordinary retry", er, e)
	}
	satOK(t, v.CleanupStopped())
	if !v.closed {
		t.Fatal("owning abort did not finish")
	}
}

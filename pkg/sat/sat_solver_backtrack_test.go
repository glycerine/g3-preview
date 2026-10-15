package sat

import (
	"errors"
	"github.com/glycerine/g3/pkg/util/lbool"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestSATSolverBacktrackRealTrailQueueAndReinit(t *testing.T) {
	s, p, v, _ := satLifecycleFixture(t, 3)
	satOK(t, v.queue.DelVar(0))
	satOK(t, v.queue.DelVar(1))
	satOK(t, v.queue.DelVar(2))
	satOK(t, s.push())
	satOK(t, p.assign(1, satJustification{}))
	satOK(t, p.assign(3, satJustification{}))
	ok, e := p.propagateCore(false)
	satOK(t, e)
	if !ok {
		t.Fatal("scope prefix")
	}
	satOK(t, s.mkBinClause(0, 4, false))
	if len(s.reinit) != 1 {
		t.Fatal("actual binary reinit")
	}
	ok, e = p.propagateCore(false)
	satOK(t, e)
	if !ok {
		t.Fatal("propagated")
	}
	satOK(t, s.pop(1))
	n, e := v.trail.Size(v.trailRef)
	satOK(t, e)
	if n != 0 || v.qhead != 0 || v.scopeLevel != 0 || len(s.reinit) != 0 {
		t.Fatal(n, v.qhead, v.scopeLevel)
	}
	for i := uint32(0); i < 3; i++ {
		got, e := v.valueVar(BoolVar(i))
		satOK(t, e)
		if got != lbool.Undef || !v.queue.Contains(BoolVar(i)) {
			t.Fatal("actual unassign callback", i, got)
		}
	}
}
func TestSATSolverBacktrackExtensionAndOwnerPrefixes(t *testing.T) {
	s, p, v, _ := satLifecycleFixture(t, 1)
	p.extensionPresent = true
	if e := s.push(); !errors.Is(e, ErrSATMissingSourceExtBinding) || v.scopeLevel != 1 || len(s.scopes) != 1 {
		t.Fatal(e, v.scopeLevel)
	}
	if e := s.pop(1); !errors.Is(e, ErrSATMissingSourceExtBinding) || v.scopeLevel != 1 {
		t.Fatal(e)
	}
	p.extensionPresent = false
	satOK(t, s.pop(1))
	copy := *s
	if e := copy.push(); e == nil {
		t.Fatal("copied owner")
	}
	satOK(t, p.Close())
	if e := s.push(); e == nil {
		t.Fatal("borrowed propagation owner")
	}
}

func TestSATSolverUnassignInvalidOldActualReadPrefix(t *testing.T) {
	s, p, v, _ := satLifecycleFixture(t, 2)
	satOK(t, p.assign(0, satJustification{}))
	satOK(t, p.assign(2, satJustification{}))
	ok, e := p.propagateCore(false)
	satOK(t, e)
	if !ok {
		t.Fatal("fixture")
	}
	e = s.unassignVars(3)
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) || v.assignment[0] != lbool.True || v.assignment[2] != lbool.True {
			t.Fatal("actual source SASSERT before any mutation", e)
		}
	} else {
		if e == nil || v.assignment[0] != lbool.Undef || v.assignment[2] != lbool.Undef || v.qhead != 2 {
			t.Fatal("reverse prefix before unsigned invalid read", e, uint32(len(v.assignment)))
		}
	}
}

func TestSATSolverReinitNativeGrowthAndLogicalBounds(t *testing.T) {
	s, _, _, b := satLifecycleFixture(t, 2)
	before := b.Used()
	b.FailAfter(0)
	for i := 0; i < 129; i++ {
		satOK(t, s.pushWrapper(newSATBinaryWrapper(Literal(i*2), Literal(i*2+1))))
	}
	if len(s.reinit) != 129 || b.Used() != before {
		t.Fatal("native wrapper growth", len(s.reinit), b.Used(), before)
	}
	b.FailAfter(-1)
	s.reinit = s.reinit[:0]
	satOK(t, s.pushWrapper(newSATBinaryWrapper(0, 2)))
	satOK(t, s.reinitClauses(0))
	if len(s.reinit) != 0 {
		t.Fatal("source compact")
	}
	e := s.reinitClauses(1)
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) || len(s.reinit) != 0 {
			t.Fatal("source logical assertion", e)
		}
	} else {
		if !errors.Is(e, arena.ErrBounds) || len(s.reinit) != 0 {
			t.Fatal("native logical bounds", e)
		}
	}
}

func TestSATSolverPushActualStatementReadTiming(t *testing.T) {
	s, p, v, _ := satLifecycleFixture(t, 1)
	satOK(t, v.trail.Free(v.trailRef))
	p.inconsistent = true
	e := s.push()
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) || len(s.scopes) != 0 || v.scopeLevel != 0 {
			t.Fatal("first actual assertion", e)
		}
	} else {
		if !errors.Is(e, arena.ErrInvalidSlot) || len(s.scopes) != 1 || v.scopeLevel != 1 {
			t.Fatal("source publish before trail-limit read", e, len(s.scopes), v.scopeLevel)
		}
	}
}

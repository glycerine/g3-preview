// Source: src/sat/sat_mus.cpp:149–174; independent checked lifetime fixtures.
// Licensed under the MIT license. Source author: Nikolaj Bjorner, 2014.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestSATMUSEmptyRecursionGuardPreservesFirstSplitPublication(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	var a, b satMUSLiteralSet
	a.bind(s.runtime.memory)
	b.bind(s.runtime.memory)
	t.Cleanup(func() { satOK(t, b.close()); satOK(t, a.close()) })
	satOK(t, a.insert(2))
	satOK(t, a.reset())
	oldMembership, oldValues := a.membership.block, a.values.raw.block
	_, e := s.mus.qx(&a, &b, false)
	if !errors.Is(e, ErrSATMUSEmptyRecursion) {
		t.Fatal(e)
	}
	// Empty qx does NOT succeed. Its first split assigns an empty temporary to
	// assignment before recursive repetition: old buffers are already destroyed.
	if a.membership.block != (memory.Block{}) || a.values.raw.block != (memory.Block{}) {
		t.Fatal("source split publication skipped")
	}
	if _, e = s.runtime.memory.Read(oldMembership, 0); e == nil {
		t.Fatal("old membership survives source assignment")
	}
	if _, e = s.runtime.memory.Read(oldValues, 0); e == nil {
		t.Fatal("old values survive source assignment")
	}
	if s.mus.frames != nil {
		t.Fatal("guard loses lexical cleanup")
	}
	satFullSolverRuntimeAudit(t, s)
}

// Checked-domain protocol fixture: caller actually frees the reached inner
// membership block. It is no ordinary source solver state or malloc failure.
func TestSATMUSInnerCoreReleaseGuardRetainsOlderScopedAppend(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	f, e := s.mus.frame(1, false)
	satOK(t, e)
	satOK(t, f.sets[0].insert(2))
	satOK(t, s.mus.mus.push(0))
	satOK(t, s.mus.mus.push(2))
	f.appendActive = true
	f.appendOld = 0
	f.innerCoreCompleted = true
	oldValues := f.sets[0].values.raw.block
	actual := f.sets[0].membership.block
	satOK(t, s.runtime.memory.Deallocate(&actual))
	before := s.budget.Used()
	e = s.mus.drop(f, nil)
	if e == nil {
		t.Fatal("actual stale inner release accepted")
	}
	n, e := s.mus.mus.size()
	satOK(t, e)
	if n != 2 || !f.appendActive || s.mus.frames != f {
		t.Fatal("older scoped_append released before stopped inner core", n)
	}
	if _, e = s.runtime.memory.Read(oldValues, 0); e == nil {
		t.Fatal("completed inner values release was replayed or omitted")
	}
	if s.budget.Used() >= before {
		t.Fatal("source completed inner free prefix absent")
	}
	// Explicit caller acknowledgment of its already completed physical free.
	// Owning continuation consumes remaining fields, not reconstructed bytes.
	f.sets[0].membership.block = memory.Block{}
	satOK(t, s.mus.cleanup())
	n, e = s.mus.mus.size()
	satOK(t, e)
	if n != 0 || s.mus.frames != nil {
		t.Fatal("scoped continuation", n)
	}
	satFullSolverRuntimeAudit(t, s)
}

func TestSATMUSCopiedOwnerRejectedBeforeSourceFieldPublication(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	copied := s.mus
	copied.maxNumRestarts = 17
	before := s.budget.Used()
	_, e := copied.run()
	if !errors.Is(e, ErrSATOwner) || copied.maxNumRestarts != 17 || s.budget.Used() != before {
		t.Fatal("copied owner published source fields", e, copied.maxNumRestarts)
	}
	satOK(t, s.Close())
	if _, e = s.mus.run(); e == nil {
		t.Fatal("closed MUS owner accepted")
	}
}

// Actual source scoped_append destructor semantics when a caller mutates the
// referenced vector during its lifetime. The source class uses resize, so a
// grown logical suffix is value-initialized, not restored from retained bytes.
func TestSATMUSScopedAppendDestructorValueInitializesGrownSuffix(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	for _, v := range []uint32{11, 13, 15} {
		satOK(t, s.mus.mus.push(v))
	}
	f, e := s.mus.frame(0, false)
	satOK(t, e)
	f.appendActive = true
	f.appendOld = 3
	satOK(t, s.mus.mus.shrink(0))
	satOK(t, s.mus.drop(f, nil))
	for i := uint32(0); i < 3; i++ {
		v, e := s.mus.mus.read(i)
		satOK(t, e)
		if v != 0 {
			t.Fatal("source resize default placement omitted", i, v)
		}
	}
	satFullSolverRuntimeAudit(t, s)
}

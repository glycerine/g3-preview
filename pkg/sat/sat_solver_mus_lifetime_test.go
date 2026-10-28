package sat

// Source: src/sat/sat_mus.cpp:149–174; independent checked lifetime fixtures.

import (
	"errors"
	"testing"
)

func TestSATMUSEmptyRecursionGuardPreservesFirstSplitPublication(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	var a, b satMUSLiteralSet
	t.Cleanup(func() { b.close(); a.close() })
	satOK(t, a.insert(2))
	a.reset()
	_, e := s.mus.qx(&a, &b, false)
	if !errors.Is(e, ErrSATMUSEmptyRecursion) {
		t.Fatal(e)
	}
	// Empty qx does NOT succeed. Its first split assigns an empty temporary to
	// assignment before recursive repetition: old buffers are already destroyed.
	if len(a.membership) != 0 || len(a.values) != 0 {
		t.Fatal("source split publication skipped")
	}
	if s.mus.frames != nil {
		t.Fatal("guard loses lexical cleanup")
	}
	satFullSolverRuntimeAudit(t, s)
}

func TestSATMUSInnerCoreNativeCleanupRestoresScopedAppend(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	f, e := s.mus.frame(1, false)
	satOK(t, e)
	satOK(t, f.sets[0].insert(2))
	satOK(t, s.mus.mus.Push(0))
	satOK(t, s.mus.mus.Push(2))
	f.appendActive = true
	f.appendOld = 0
	f.innerCoreCompleted = true
	satOK(t, s.mus.drop(f, nil))
	n, e := s.mus.mus.Size()
	satOK(t, e)
	if n != 0 || s.mus.frames != nil || f.sets[0].values != nil || f.sets[0].membership != nil {
		t.Fatal("native inner cleanup/scoped append", n)
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
		satOK(t, s.mus.mus.Push(v))
	}
	f, e := s.mus.frame(0, false)
	satOK(t, e)
	f.appendActive = true
	f.appendOld = 3
	satOK(t, s.mus.mus.Shrink(0))
	satOK(t, s.mus.drop(f, nil))
	for i := uint32(0); i < 3; i++ {
		v, e := s.mus.mus.Read(i)
		satOK(t, e)
		if v != 0 {
			t.Fatal("source resize default placement omitted", i, v)
		}
	}
	satFullSolverRuntimeAudit(t, s)
}

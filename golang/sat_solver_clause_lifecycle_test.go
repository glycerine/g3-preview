// Copyright (c) 2011 Microsoft Corporation. Licensed under the MIT license.
package g3

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/golang/arena"
	"reflect"
	"testing"
)

func satLifecycleFixture(t *testing.T, n uint32) (*satSolverClauseLifecycle, *satSolverPropagationState, *satSolverVariableState, *arena.Budget) {
	t.Helper()
	p, v, b := satPropagationFixture(t, n)
	s, e := newSATSolverClauseLifecycle(p, b)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, s.Close()) })
	return s, p, v, b
}
func TestSATSolverClauseLifecycleActualArityAndOwnership(t *testing.T) {
	s, p, v, _ := satLifecycleFixture(t, 8)
	for n := uint32(0); n <= 5; n++ {
		lits := []satLiteral{2, 4, 6, 8, 10}
		if n == 1 {
			lits[0] = 0
		}
		if n == 0 {
			continue
		}
		c, e := s.mkClauseCore(n, lits, false)
		satOK(t, e)
		if n < 3 && c != (satStoredClauseRef{}) {
			t.Fatal(n, c)
		}
		if n >= 3 && !p.clauses.IsLive(c) {
			t.Fatal(n, c)
		}
	}
	if s.mkBinary != 1 || s.mkTernary != 1 || s.mkNary != 2 || s.nonLearnedGeneration != 5 {
		t.Fatal("source stats", s.mkBinary, s.mkTernary, s.mkNary, s.nonLearnedGeneration)
	}
	if len(s.original) != 3 || len(s.learned) != 0 || s.pendingCandidate != (satStoredClauseRef{}) {
		t.Fatal("actual ownership")
	}
	satOK(t, v.checkPhysicalOwners())
	satOK(t, s.Close())
	if s.nonLearnedGeneration != 7 {
		t.Fatal("both del_clauses empty/nonempty calls")
	}
	if _, e := s.mkClauseCore(0, nil, false); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}
func TestSATSolverClauseLifecyclePendingCandidateRefusal(t *testing.T) {
	s, p, _, b := satLifecycleFixture(t, 3)
	// Allocator succeeds, first watch growth/candidate publication may refuse.

	// Exact targeted refusal at original-vector growth after real attachment.
	c, e := p.clauses.MkClause([]satLiteral{0, 2, 4}, false)
	satOK(t, e)
	s.pendingCandidate = c
	satOK(t, s.attachClause(c))
	b.FailAfter(0)
	e = s.publishCandidate(false)
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	if s.pendingCandidate != c || !p.clauses.IsLive(c) || len(s.original) != 0 {
		t.Fatal("retained allocated clause")
	}
	b.FailAfter(-1)
	satOK(t, s.publishCandidate(false))
	if s.pendingCandidate != (satStoredClauseRef{}) || len(s.original) != 1 {
		t.Fatal("publication")
	}
}

// Independent named ownership walk: liveness in the allocator registry alone
// never makes an anonymous raw candidate part of the solver ownership graph.
func satLifecycleAssertOwned(t *testing.T, s *satSolverClauseLifecycle) {
	t.Helper()
	a := s.propagation.clauses
	refs := append([]satStoredClauseRef(nil), s.original...)
	refs = append(refs, s.learned...)
	if s.pendingCandidate != (satStoredClauseRef{}) {
		refs = append(refs, s.pendingCandidate)
	}
	for i, r := range refs {
		if !a.IsLive(r) {
			t.Fatal("named stale owned clause", r)
		}
		for j := 0; j < i; j++ {
			if refs[j] == r {
				t.Fatal("duplicate owning clause", r)
			}
		}
	}
	// These bounded fixtures use small_allocator clauses only, including direct
	// debug storage. Every actual live clause descriptor must have a named root.
	if a.storage.Stats().Live != uint32(len(refs)) {
		t.Fatal("anonymous actual clause descriptor", a.storage.Stats().Live, len(refs))
	}
	want := s.ownerBytes + uint64(cap(s.original)+cap(s.learned))*uint64(reflect.TypeOf(satStoredClauseRef{}).Size()) + uint64(cap(s.reinit))*uint64(reflect.TypeOf(satClauseWrapper{}).Size()) + uint64(cap(s.scopes))*uint64(reflect.TypeOf(satSolverScope{}).Size())
	p := s.propagation
	v := p.variables
	want += p.ownerBytes + uint64(cap(p.diffLevels.data)) + v.Stats().TotalBytes
	todo := v.elimTodo
	want += todo.ownerBytes + uint64(cap(todo.inSet)) + 4*uint64(cap(todo.values))
	small := a.storage.Stats()
	want += a.ownerBytes + 4*uint64(cap(a.ids.free)) + a.largeBytes + a.largeDirectory + small.OwnerBytes + small.PayloadBytes + small.MetadataBytes + small.DirectoryBytes
	mc := p.converter
	want += mc.ownerBytes + mc.entryBytes + mc.pendingBytes + mc.store.ownerBytes + mc.store.directoryBytes + satWatchGraphBytes(mc.store.words.Stats())
	r := p.limit
	want += uint64(reflect.TypeOf(utilReslimit{}).Size()) + 8*uint64(cap(r.limits)) + uint64(reflect.TypeOf((*utilReslimit)(nil)).Size())*uint64(cap(r.children)) + p.memory.Stats().TotalBytes
	if s.budget.Used() != want {
		t.Fatal("actual once-only entire bound owner graph", s.budget.Used(), want)
	}
	satOK(t, v.checkPhysicalOwners())
}
func TestSATSolverClauseActualAllocationPublicationFaultPrefixes(t *testing.T) {
	failures, success := 0, 0
	for fault := int64(0); fault < 96; fault++ {
		done := false
		t.Run(fmt.Sprint(fault), func(t *testing.T) {
			s, p, _, b := satLifecycleFixture(t, 4)
			b.FailAfter(fault)
			c, e := s.mkClauseCore(4, []satLiteral{0, 2, 4, 6}, false)
			satLifecycleAssertOwned(t, s)
			if e != nil {
				if !errors.Is(e, arena.ErrInjectedFailure) {
					t.Fatal(e)
				}
				failures++
				if c != (satStoredClauseRef{}) && (s.pendingCandidate != c || !p.clauses.IsLive(c)) {
					t.Fatal("allocated source prefix lost", c)
				}
			} else {
				success++
				done = true
				if len(s.original) != 1 || s.original[0] != c || s.pendingCandidate != (satStoredClauseRef{}) {
					t.Fatal("final source publication")
				}
			}
			b.FailAfter(-1)
		})
		if done {
			break
		}
	}
	if failures == 0 || success != 1 {
		t.Fatal(failures, success)
	}
	t.Log("actual refusal prefixes", failures)
}
func TestSATSolverClauseCandidateCleanupPreservesPublishedBorrowers(t *testing.T) {
	s, p, v, _ := satLifecycleFixture(t, 4)
	c, e := p.clauses.MkClause([]satLiteral{0, 2, 4, 6}, false)
	satOK(t, e)
	s.pendingCandidate = c
	satOK(t, s.attachClause(c))
	satOK(t, s.pushReinitStack(c))
	satLifecycleAssertOwned(t, s)
	satOK(t, s.cleanupCandidate())
	if len(s.reinit) != 1 {
		t.Fatal("must preserve source reinit prefix")
	}
	w, e := v.watches.Get(1)
	satOK(t, e)
	n, e := w.Size()
	satOK(t, e)
	if n != 1 {
		t.Fatal("must preserve source watch prefix")
	}
	satOK(t, p.assign(1, satJustification{}))
	_, e = p.propagateCore(false)
	if !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("actual stale clause guard", e)
	}
	if e = s.reinitClauses(0); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("actual stale wrapper guard", e)
	}
	satLifecycleAssertOwned(t, s)
}
func TestSATSolverClauseEmptyTautologyAndDeleteCounters(t *testing.T) {
	s, p, _, _ := satLifecycleFixture(t, 2)
	c, e := s.mkClauseCore(2, []satLiteral{0, 1}, false)
	satOK(t, e)
	if c != (satStoredClauseRef{}) || s.nonLearnedGeneration != 0 || p.inconsistent {
		t.Fatal("tautology")
	}
	_, e = s.mkClauseCore(0, nil, false)
	satOK(t, e)
	if !p.inconsistent || !p.conflict.None() || p.notLiteral != satDefaultLiteral() || s.nonLearnedGeneration != 1 {
		t.Fatal("empty source conflict")
	}
	c, e = p.clauses.MkClause([]satLiteral{0, 2, 0}, false)
	satOK(t, e)
	satOK(t, s.delClause(c))
	if s.nonLearnedGeneration != 2 || s.deleted != 1 {
		t.Fatal("actual delete source counters")
	}
	satLifecycleAssertOwned(t, s)
}

// Source: src/util/hashtable.h:139–620; src/util/map.h:24–163; src/sat/sat_solver.h:481–483.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package g3

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/golang/memory"
	"reflect"
	"testing"
)

func TestSATSolverConsequenceIndexSetActualPhysicalIterationAndDeepMapCopy(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	var set satSolverIndexSet
	satOK(t, set.init(s.runtime.memory))
	t.Cleanup(func() { satOK(t, set.close()) })
	// u_hash is identity. All three keys initially collide in bucket1 of the
	// actual eight-entry table, establishing source physical probe order.
	for _, key := range []uint32{17, 1, 9} {
		satOK(t, set.insert(key))
	}
	it, e := set.capture()
	satOK(t, e)
	for i, want := range []uint32{17, 1, 9} {
		got, e := it.next()
		satOK(t, e)
		if !got.present || got.value != want {
			t.Fatal("actual USED traversal", i, got, want)
		}
	}
	satOK(t, s.antecedents.insert(7, &set))
	// The source map owns an actual nested COPY, not the caller's inner table.
	old := set.raw
	satOK(t, set.reset())
	stored, e := s.antecedents.find(7)
	satOK(t, e)
	if stored.raw == old {
		t.Fatal("map value stole/borrowed the caller's table")
	}
	for _, key := range []uint32{17, 1, 9} {
		yes, e := stored.contains(key)
		satOK(t, e)
		if !yes {
			t.Fatal("nested copy lost value", key)
		}
	}
	// Every FREE map entry still has a genuinely constructed owning index_set;
	// reset marks the map FREE but does not automatically destroy its values.
	owned, e := s.antecedents.innerOwnerCount()
	satOK(t, e)
	if owned != 8 {
		t.Fatal("default-constructed inactive source values", owned)
	}
	satOK(t, s.antecedents.reset())
	owned, e = s.antecedents.innerOwnerCount()
	satOK(t, e)
	if owned != 8 {
		t.Fatal("reset discarded inactive owning values", owned)
	}
	yes, e := stored.contains(17)
	satOK(t, e)
	if !yes {
		t.Fatal("nonshrinking reset unexpectedly destroyed mapped field")
	}
}

// Explicit malformed source-capacity domain: the physical nested core header,
// not a cached cold count, controls the reached source pointer arithmetic.
func TestSATSolverConsequenceNestedIndexSetReadsActualHeaderAtReachedSite(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	var set satSolverIndexSet
	satOK(t, set.init(s.runtime.memory))
	t.Cleanup(func() { satOK(t, set.close()) })
	satOK(t, set.insert(17))
	satOK(t, s.antecedents.insert(7, &set))
	stored, e := s.antecedents.find(7)
	satOK(t, e)
	satOK(t, s.runtime.memory.WriteUint32(stored.parent, stored.base+8, 0))
	_, e = stored.contains(17)
	if e == nil {
		t.Fatal("source capacity0 must reach endangered17th physical bucket, not cached8")
	}
	// Restore the explicit fixture corruption before ordinary source teardown.
	satOK(t, s.runtime.memory.WriteUint32(stored.parent, stored.base+8, 8))
}

func TestSATSolverConsequenceMapFailedSecondTemporaryUnwindsCompletedFirst(t *testing.T) {
	reached := false
	for fault := int64(0); fault < 12 && !reached; fault++ {
		t.Run(fmt.Sprint(fault), func(t *testing.T) {
			s, _, _ := satSolverCheckFixture(t)
			f, e := s.consequenceFrame()
			satOK(t, e)
			satOK(t, f.indexSet(s))
			set := &f.index
			t.Cleanup(func() { satOK(t, s.consequenceDrop(f, nil)) })
			satOK(t, set.insert(17))
			s.budget.FailAfter(fault)
			e = s.antecedents.insert(7, set)
			m := s.antecedents
			if e != nil && m.tempUsed == 2 && m.temps[1].raw == (memory.Block{}) {
				reached = true
				if m.temps[0].raw != (memory.Block{}) {
					t.Error("completed key_data temporary must destruct after failed second constructor")
				}
			}
			satFullSolverRuntimeAudit(t, s)
			s.budget.FailAfter(-1)
		})
	}
	if !reached {
		t.Fatal("did not reach actual second-copy refusal")
	}
}

func TestSATSolverConsequenceIndexSetAndMapGrowBeforeDuplicate(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	f, e := s.consequenceFrame()
	satOK(t, e)
	satOK(t, f.indexSet(s))
	t.Cleanup(func() { satOK(t, s.consequenceDrop(f, nil)) })
	set := &f.index
	for k := uint32(0); k < 7; k++ {
		satOK(t, set.insert(k))
		satOK(t, s.antecedents.insert(k, set))
	}
	oldSet, oldMap := set.raw, s.antecedents.raw
	it, e := set.capture()
	satOK(t, e)
	satOK(t, set.insert(0))
	satOK(t, s.antecedents.insert(0, set))
	if set.capacity != 16 || s.antecedents.capacity != 16 {
		t.Fatal("strict source75% duplicate growth", set.capacity, s.antecedents.capacity)
	}
	if set.size != 7 || s.antecedents.size != 7 {
		t.Fatal("duplicate changed size")
	}
	for _, b := range []memory.Block{oldSet, oldMap} {
		if _, e = s.runtime.memory.Read(b, 0); e == nil {
			t.Fatal("old actual table survived growth")
		}
	}
	if _, e = it.next(); e == nil {
		t.Fatal("captured old iterator survived relocation")
	}
	n, e := s.antecedents.innerOwnerCount()
	satOK(t, e)
	if n != 16 {
		t.Fatal("all inactive new entries constructed", n)
	}
	for k := uint32(0); k < 7; k++ {
		v, e := s.antecedents.find(k)
		satOK(t, e)
		for j := uint32(0); j <= k; j++ {
			yes, e := v.contains(j)
			satOK(t, e)
			if !yes {
				t.Fatal(k, j)
			}
		}
	}
	satFullSolverRuntimeAudit(t, s)
}

func TestSATSolverConsequenceMapGrowthEveryActualRefusalRetainsNestedOwners(t *testing.T) {
	fixture := func(t *testing.T) (*satSolver, *satSolverIndexSet) {
		s, _, _ := satSolverCheckFixture(t)
		f, e := s.consequenceFrame()
		satOK(t, e)
		satOK(t, f.indexSet(s))
		t.Cleanup(func() { satOK(t, s.consequenceDrop(f, nil)) })
		satOK(t, f.index.insert(17))
		for k := uint32(0); k < 7; k++ {
			satOK(t, s.antecedents.insert(k, &f.index))
		}
		return s, &f.index
	}
	horizon := int64(0)
	t.Run("calibrate", func(t *testing.T) {
		s, set := fixture(t)
		const start = int64(1 << 50)
		s.budget.FailAfter(start)
		satOK(t, s.antecedents.insert(0, set))
		horizon = start - reflect.ValueOf(s.budget).Elem().FieldByName("remaining").Int()
		satFullSolverRuntimeAudit(t, s)
		s.budget.FailAfter(-1)
	})
	failures, success := 0, 0
	for fault := int64(0); fault <= horizon; fault++ {
		t.Run(fmt.Sprint(fault), func(t *testing.T) {
			s, set := fixture(t)
			s.budget.FailAfter(fault)
			e := s.antecedents.insert(0, set)
			satFullSolverRuntimeAudit(t, s)
			if e != nil {
				satProbingExpectedRefusal(t, s.propagation, e)
				failures++
			} else {
				success++
				if s.antecedents.capacity != 16 || s.antecedents.size != 7 {
					t.Fatal("wrong completed source map", s.antecedents.capacity, s.antecedents.size)
				}
			}
			s.budget.FailAfter(-1)
		})
	}
	if int64(failures) != horizon || success != 1 {
		t.Fatal(horizon, failures, success)
	}
	t.Log("actual nested map growth refusal horizon", horizon)
}

func TestSATSolverConsequenceMapResetShrinksAfterFreeBeforeAllocation(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	f, e := s.consequenceFrame()
	satOK(t, e)
	satOK(t, f.indexSet(s))
	t.Cleanup(func() { satOK(t, s.consequenceDrop(f, nil)) })
	satOK(t, f.index.insert(17))
	for k := uint32(0); k < 22; k++ {
		satOK(t, s.antecedents.insert(k, &f.index))
	}
	m := s.antecedents
	if m.capacity != 32 {
		t.Fatal(m.capacity)
	}
	satOK(t, m.reset())
	if m.capacity != 32 {
		t.Fatal("non-sparse old source table unexpectedly shrank")
	}
	satOK(t, m.insert(1, &f.index))
	old := m.raw
	oldInner := make([]memory.Block, m.capacity)
	for i := range oldInner {
		oldInner[i] = m.entry(uint32(i)).raw
	}
	s.budget.FailAfter(0)
	e = m.reset()
	satProbingExpectedRefusal(t, s.propagation, e)
	if m.raw != (memory.Block{}) || m.capacity != 16 || m.size != 1 || m.closeCursor != 32 {
		t.Fatal("source reset refusal prefix", m.raw, m.capacity, m.size, m.closeCursor)
	}
	for _, b := range append(oldInner, old) {
		if _, er := s.runtime.memory.Read(b, 0); er == nil {
			t.Fatal("old source owner survived before replacement refusal", b)
		}
	}
	satFullSolverRuntimeAudit(t, s)
	s.budget.FailAfter(-1)
	satOK(t, m.Close())
}

func TestSATSolverConsequenceNestedIndexSetReachedPointerWordsGuard(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	f, e := s.consequenceFrame()
	satOK(t, e)
	satOK(t, f.indexSet(s))
	t.Cleanup(func() { satOK(t, s.consequenceDrop(f, nil)) })
	satOK(t, f.index.insert(17))
	satOK(t, s.antecedents.insert(7, &f.index))
	row, e := s.antecedents.find(7)
	satOK(t, e)
	old := row.raw.Slot()
	satOK(t, s.runtime.memory.WriteUint32(row.parent, row.base, 0))
	_, e = row.contains(17)
	if !errors.Is(e, ErrSATOwner) {
		t.Error("reached source pointer must not silently use cold counterpart", e)
	}
	satOK(t, s.runtime.memory.WriteUint32(row.parent, row.base, old))
	satFullSolverRuntimeAudit(t, s)
}

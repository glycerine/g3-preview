// Source: src/sat/sat_solver.h:483; src/util/map.h:73; src/util/hashtable.h:244–253; src/util/memory_manager.h:104–120.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package sat

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestSATSolverAntecedentMapEagerNestedConstruction(t *testing.T) {
	b := arena.NewBudget(^uint64(0))
	r, e := util.NewRuntime(util.RuntimeConfig{Budget: b})
	satOK(t, e)
	baseline := r.Stats().LiveBlocks
	m, e := newSATSolverIndexMap(r, b)
	satOK(t, e)
	if !m.constructed || m.constructedEntries != 8 || r.Stats().LiveBlocks-baseline != 9 {
		t.Fatal(m.constructedEntries, r.Stats())
	}
	for i := uint32(0); i < 8; i++ {
		state, e := r.ReadUint32(m.raw, uint64(i)*40+4)
		satOK(t, e)
		if state != 0 {
			t.Fatal(state)
		}
		for j := uint32(0); j < 8; j++ {
			state, e = r.ReadUint32(m.entries[i].raw, uint64(j)*12+4)
			satOK(t, e)
			if state != 0 {
				t.Fatal(state)
			}
		}
	}
	copied := *m
	if e = copied.Close(); !errors.Is(e, ErrSATOwner) {
		t.Fatal(e)
	}
	satOK(t, m.Close())
	if r.Stats().LiveBlocks != baseline {
		t.Fatal("unnamed table root", r.Stats())
	}
	satOK(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
func TestSATSolverAntecedentMapAllConstructorRefusalPrefixes(t *testing.T) {
	successes, failed := 0, 0
	for fail := int64(0); fail < 96; fail++ {
		t.Run(string(rune('A'+fail)), func(t *testing.T) {
			b := arena.NewBudget(^uint64(0))
			r, e := util.NewRuntime(util.RuntimeConfig{Budget: b})
			satOK(t, e)
			b.FailAfter(fail)
			m, e := newSATSolverIndexMap(r, b)
			if e != nil {
				failed++
				var f util.Failure
				if !errors.Is(e, arena.ErrInjectedFailure) && !(errors.As(e, &f) && f.Kind == util.FailureMemout && r.State().OutOfMemory) {
					t.Fatal(e)
				}
			} else {
				successes++
			}
			// Only actual named scalar roots establish ownership; the Runtime registry
			// is used below solely as an independent coverage/bijection check.
			named := []util.Block{}
			if m != nil {
				if m.raw != (util.Block{}) {
					named = append(named, m.raw)
				}
				if m.pending != (util.Block{}) {
					named = append(named, m.pending)
				}
				for i := range m.entries {
					if m.entries[i].raw != (util.Block{}) {
						named = append(named, m.entries[i].raw)
					}
				}
			}
			seen := map[util.Block]bool{}
			for _, ref := range named {
				if seen[ref] {
					t.Fatal("duplicate owner", ref)
				}
				seen[ref] = true
				if _, e := r.Size(ref); e != nil {
					t.Fatal(e)
				}
			}
			if uint32(len(named)) != r.Stats().LiveBlocks {
				t.Fatal("unnamed construction root", len(named), r.Stats())
			}
			b.FailAfter(-1)
			if m != nil {
				satOK(t, m.Close())
			}
			satOK(t, r.Close())
			if b.Used() != 0 {
				t.Fatal(b.Used())
			}
		})
		if successes > 0 {
			break
		}
	}
	if successes != 1 || failed == 0 {
		t.Fatal(successes, failed)
	}
}

func TestSATSolverAntecedentMapAscendingDestructionGuardPrefix(t *testing.T) {
	b := arena.NewBudget(^uint64(0))
	r, e := util.NewRuntime(util.RuntimeConfig{Budget: b})
	satOK(t, e)
	m, e := newSATSolverIndexMap(r, b)
	satOK(t, e)
	// Explicit source-undefined freed-pointer diagnostic domain. An actual stale
	// inner table must stop ascending entry destruction at2, after entries0/1,
	// before later tables or the outer array. No injected free callback exists.
	stale := m.entries[2].raw
	satOK(t, r.Deallocate(&stale))
	if e = m.Close(); !errors.Is(e, util.ErrRuntimeOwner) {
		t.Fatal(e)
	}
	if m.closeCursor != 2 || m.closed || m.entries[0].raw != (util.Block{}) || m.entries[1].raw != (util.Block{}) || m.raw == (util.Block{}) {
		t.Fatal("wrong source destruction prefix", m.closeCursor)
	}
	for i := 3; i < 8; i++ {
		if _, e = r.Size(m.entries[i].raw); e != nil {
			t.Fatal(i, e)
		}
	}
	// Separate caller acknowledgment of an already freed scalar, solely at the
	// owning-abort boundary; it does not reconstruct data or rewind counters.
	m.entries[2].raw = util.Block{}
	satOK(t, m.Close())
	satOK(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

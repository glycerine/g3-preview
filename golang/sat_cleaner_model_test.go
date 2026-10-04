// Source: src/sat/sat_cleaner.cpp and .h. Licensed under the MIT license.
package g3

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/golang/arena"
	"reflect"
	"testing"
)

// Actual named source owners: completed compacted prefix + unread suffix.
// Deleted/duplicate pointer slots in interrupted source storage are NOT owners.
func satCleanerAudit(t *testing.T, c *satCleaner, extra ...satStoredClauseRef) {
	t.Helper()
	l := c.lifecycle
	p := l.propagation
	v := p.variables
	a := p.clauses
	refs := append([]satStoredClauseRef(nil), extra...)
	addVector := func(x []satStoredClauseRef, learned bool) {
		if c.interrupted && c.learned == learned {
			refs = append(refs, x[c.drainPrefix:c.out]...)
			start := c.cursor
			if c.drainSuffix > start {
				start = c.drainSuffix
			}
			refs = append(refs, x[start:c.end]...)
		} else {
			refs = append(refs, x...)
		}
	}
	addVector(l.original, false)
	addVector(l.learned, true)
	if l.pendingCandidate != (satStoredClauseRef{}) {
		refs = append(refs, l.pendingCandidate)
	}
	for i, r := range refs {
		if !a.IsLive(r) {
			t.Fatal("named source owner stale", r)
		}
		for j := 0; j < i; j++ {
			if refs[j] == r {
				t.Fatal("duplicate scalar owner", r)
			}
		}
	}
	small := a.storage.Stats()
	if small.Live != uint32(len(refs)) {
		t.Fatal("unnamed physical clause", small.Live, len(refs))
	}
	want := c.ownerBytes + l.ownerBytes + uint64(cap(l.original)+cap(l.learned))*uint64(reflect.TypeOf(satStoredClauseRef{}).Size()) + uint64(cap(l.reinit))*uint64(reflect.TypeOf(satClauseWrapper{}).Size()) + uint64(cap(l.scopes))*uint64(reflect.TypeOf(satSolverScope{}).Size())
	want += p.ownerBytes + uint64(cap(p.diffLevels.data)) + v.Stats().TotalBytes
	todo := v.elimTodo
	want += todo.ownerBytes + uint64(cap(todo.inSet)) + 4*uint64(cap(todo.values))
	want += a.ownerBytes + 4*uint64(cap(a.ids.free)) + a.largeBytes + a.largeDirectory + small.OwnerBytes + small.PayloadBytes + small.MetadataBytes + small.DirectoryBytes
	mc := p.converter
	want += mc.ownerBytes + mc.entryBytes + mc.pendingBytes + mc.store.ownerBytes + mc.store.directoryBytes + satWatchGraphBytes(mc.store.words.Stats())
	r := p.limit
	want += uint64(reflect.TypeOf(utilReslimit{}).Size()) + 8*uint64(cap(r.limits)) + uint64(reflect.TypeOf((*utilReslimit)(nil)).Size())*uint64(cap(r.children)) + p.memory.Stats().TotalBytes
	if want != l.budget.Used() {
		t.Fatal("whole named once-only physical graph", want, l.budget.Used())
	}
	satScalarAuditRoots(t, v)
}
func TestSATCleanerDeletionThenAttachRefusalRetainsRanges(t *testing.T) {
	c, l := satCleanerFixture(t, 4)
	p := l.propagation
	a := p.clauses
	dead := satCleanerClause(t, l, []satLiteral{0, 4, 6}, false, false)
	kept := satCleanerClause(t, l, []satLiteral{1, 2, 4, 6}, false, false)
	satOK(t, a.ids.reserveFreeIDs(2))
	satOK(t, p.assign(0, satJustification{}))
	satOK(t, c.cleanupWatches())
	l.budget.FailAfter(0)
	e := c.cleanupClauses(false)
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	if a.IsLive(dead) || !a.IsLive(kept) || c.cursor != 2 || c.out != 1 || !c.interrupted || l.cleanerContinuation != c || l.deleted != 1 {
		t.Fatal("source completed delete/shrink/front-copy before attach", c.cursor, c.out, l.deleted)
	}
	sz, e := a.Size(kept)
	satOK(t, e)
	if sz != 3 || len(l.original) != 2 || l.original[0] != kept || l.original[1] != kept {
		t.Fatal("actual source end still unpublished")
	}
	satCleanerAudit(t, c)
	if _, e = c.run(true); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	l.budget.FailAfter(-1)
	// Source cleaner is NOT retried; actual owning lifecycle closes the ledger.
	satOK(t, l.Close())
	if c.interrupted || a.IsLive(kept) {
		t.Fatal("owning continuation not drained")
	}
	satOK(t, c.Close())
}
func TestSATCleanerEveryAllocationPrefixAndOutputs(t *testing.T) {
	failures, endpoint := 0, int64(-1)
	for fault := int64(0); fault < 128; fault++ {
		t.Run(fmt.Sprint(fault), func(t *testing.T) {
			c, l := satCleanerFixture(t, 5)
			p := l.propagation
			satCleanerClause(t, l, []satLiteral{0, 4, 6}, false, false)
			kept := satCleanerClause(t, l, []satLiteral{1, 2, 4, 6}, false, false)
			satCleanerClause(t, l, []satLiteral{1, 6, 8}, true, false)
			satOK(t, p.assign(0, satJustification{}))
			l.budget.FailAfter(fault)
			done, e := c.run(true)
			if e != nil {
				failures++
				satProbingExpectedRefusal(t, p, e)
			} else {
				endpoint = fault
				if !done || len(l.original) != 1 || l.original[0] != kept || len(l.learned) != 0 || c.elimClauses != 1 || c.elimLiterals != 2 || l.deleted != 2 {
					t.Fatal("successful source result", done, len(l.original), len(l.learned), c.elimClauses, c.elimLiterals, l.deleted)
				}
				sz, e := p.clauses.Size(kept)
				satOK(t, e)
				if sz != 3 {
					t.Fatal(sz)
				}
			}
			satCleanerAudit(t, c) // exact prefix BEFORE disabling real refusal
			l.budget.FailAfter(-1)
		})
		if endpoint >= 0 {
			break
		}
	}
	if failures < 3 || endpoint < 0 {
		t.Fatal(failures, endpoint)
	}
	t.Log("actual refusal prefixes", failures, "first success", endpoint)
}
func TestSATCleanerFiniteWatchMatrix(t *testing.T) {
	for first := -1; first <= 1; first++ {
		for second := -1; second <= 1; second++ {
			t.Run(fmt.Sprintf("%d/%d", first, second), func(t *testing.T) {
				c, l := satCleanerFixture(t, 3)
				p := l.propagation
				// Literal0 list: undefined first keeps undefined binary second and EXT;
				// true/false first destroys the entire backing without examining entries.
				raw := satCleanerClause(t, l, []satLiteral{0, 2, 4}, false, false)
				off, e := p.clauses.Offset(raw)
				satOK(t, e)
				list, e := p.variables.watches.Get(0)
				satOK(t, e)
				w, e := newSATBinaryWatch(2, false)
				satOK(t, e)
				satOK(t, list.Push(w))
				w, e = newSATTernaryWatch(2, 4)
				satOK(t, e)
				satOK(t, list.Push(w))
				w, e = newSATClauseWatch(2, off)
				satOK(t, e)
				satOK(t, list.Push(w))
				w, e = newSATExternalWatch(3)
				satOK(t, e)
				satOK(t, list.Push(w))
				for v, state := range []int{first, second} {
					if state != 0 {
						lit := satLiteral(2 * v)
						if state < 0 {
							lit = lit.negated()
						}
						satOK(t, p.assign(lit, satJustification{}))
					}
				}
				e = c.cleanupWatches()
				if satStorageSourceDebug && first == 0 && second < 0 {
					if !errors.Is(e, ErrSATAssertion) {
						t.Fatal(e)
					}
					satCleanerAudit(t, c)
					return
				}
				satOK(t, e)
				n, e := list.Size()
				satOK(t, e)
				want := uint32(0)
				if first == 0 {
					want = 1
					if second == 0 {
						want = 2
					}
				}
				if n != want {
					t.Fatal(first, second, n, want)
				}
				if want == 2 {
					w, e := list.Read(0)
					satOK(t, e)
					if w.Kind() != 0 {
						t.Fatal(w)
					}
				}
				satCleanerAudit(t, c)
			})
		}
	}
}
func TestSATCleanerWrapperSourceGuards(t *testing.T) {
	c, l := satCleanerFixture(t, 1)
	p := l.propagation
	satOK(t, l.push())
	p.extensionPresent = true
	satOK(t, c.cleanup())
	p.extensionPresent = false
	satOK(t, l.pop(1))
	satOK(t, p.assign(0, satJustification{}))
	p.extensionPresent = true
	// Actual propagate visits EXT even with no attached constraints, so complete
	// that source action first while null; then cleanup reaches modified dispatch.
	p.extensionPresent = false
	_, e := p.propagateCore(false)
	satOK(t, e)
	p.extensionPresent = true
	e = c.cleanup()
	if !errors.Is(e, ErrSATMissingSourceExtBinding) || c.lastNumUnits != 1 {
		t.Fatal(e, c.lastNumUnits)
	}
	p.extensionPresent = false
}

func TestSATCleanerCapturedTrailBeforePropagationAndWarm(t *testing.T) {
	c, l := satCleanerFixture(t, 2)
	p := l.propagation
	satOK(t, l.mkBinClause(1, 2, false))
	satOK(t, p.assign(0, satJustification{}))
	done, e := c.run(false)
	satOK(t, e)
	n, e := p.variables.trail.Size(p.variables.trailRef)
	satOK(t, e)
	if !done || n != 2 || c.lastNumUnits != 1 {
		t.Fatal("source captures size before propagation", done, n, c.lastNumUnits)
	}
	done, e = c.run(false)
	satOK(t, e)
	if !done || c.lastNumUnits != 2 {
		t.Fatal("new unit recognized on next source call")
	}
	used := l.budget.Used()
	physical := p.memory.Stats().TotalBytes
	l.budget.FailAfter(0)
	var warmErr error
	alloc := testing.AllocsPerRun(100, func() { _, warmErr = c.run(true) })
	if warmErr != nil || alloc != 0 || l.budget.Used() != used || p.memory.Stats().TotalBytes != physical {
		t.Fatal("warm early exit graph", warmErr, alloc)
	}
	satCleanerAudit(t, c)
	l.budget.FailAfter(-1)
}
func TestSATCleanerFiniteFormulaOracle(t *testing.T) {
	for code := 0; code < 27; code++ {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			c, l := satCleanerFixture(t, 3)
			p := l.propagation
			state := [3]int{}
			q := code
			for i := range state {
				state[i] = q%3 - 1
				q /= 3
			}
			satCleanerClause(t, l, []satLiteral{0, 2, 4}, true, true)
			for i, a := range state {
				if a != 0 {
					lit := satLiteral(2 * i)
					if a < 0 {
						lit = lit.negated()
					}
					satOK(t, p.assign(lit, satJustification{}))
				}
			}
			satOK(t, c.cleanupWatches())
			satOK(t, c.cleanupClauses(true))
			val := func(lit satLiteral, mask int) bool {
				x := mask&(1<<lit.variable()) != 0
				if lit.sign() {
					return !x
				}
				return x
			}
			for mask := 0; mask < 8; mask++ {
				domain := true
				for i, a := range state {
					if a != 0 && ((mask&(1<<i) != 0) != (a > 0)) {
						domain = false
					}
				}
				if !domain {
					continue
				}
				original := mask != 0
				actual := !p.inconsistent
				for i := uint32(0); i < 3; i++ {
					a, e := p.variables.valueVar(satBoolVar(i))
					satOK(t, e)
					if a != lUndef && ((mask&(1<<i) != 0) != (a == lTrue)) {
						actual = false
					}
				}
				for _, r := range l.learned {
					n, e := p.clauses.Size(r)
					satOK(t, e)
					yes := false
					for i := uint32(0); i < n; i++ {
						lit, e := p.clauses.Literal(r, i)
						satOK(t, e)
						yes = yes || val(lit, mask)
					}
					actual = actual && yes
				}
				for idx := uint32(0); idx < 6; idx++ {
					list, e := p.variables.watches.Get(idx)
					satOK(t, e)
					n, e := list.Size()
					satOK(t, e)
					for j := uint32(0); j < n; j++ {
						w, e := list.Read(j)
						satOK(t, e)
						if w.Kind() == 0 {
							lit, e := w.Literal()
							satOK(t, e)
							actual = actual && (val(satLiteral(idx).negated(), mask) || val(lit, mask))
						}
					}
				}
				if original != actual {
					t.Fatal("independent conditional truth table", state, mask, original, actual)
				}
			}
			satCleanerAudit(t, c)
		})
	}
}

func TestSATCleanerStoppedLedgerRejectsNewOwningPublication(t *testing.T) {
	c, l := satCleanerFixture(t, 4)
	p := l.propagation
	satCleanerClause(t, l, []satLiteral{0, 4, 6}, false, false)
	satCleanerClause(t, l, []satLiteral{1, 2, 4, 6}, false, false)
	satOK(t, p.clauses.ids.reserveFreeIDs(4))
	satOK(t, p.assign(0, satJustification{}))
	l.budget.FailAfter(0)
	e := c.cleanupClauses(false)
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	satCleanerAudit(t, c)
	l.budget.FailAfter(-1)
	// Caller owns the freshly constructed object until a real vector publication.
	extra, e := p.clauses.MkClause([]satLiteral{2, 4, 6}, false)
	satOK(t, e)
	before := len(l.original)
	e = l.pushClause(false, extra)
	if !errors.Is(e, arena.ErrClosed) || len(l.original) != before {
		t.Fatal("stopped captured-end cohort accepted append", e, len(l.original), before)
	}
	satCleanerAudit(t, c, extra)
	satOK(t, p.clauses.Delete(extra))
	satOK(t, c.Close())
	satLifecycleAssertOwned(t, l)
}

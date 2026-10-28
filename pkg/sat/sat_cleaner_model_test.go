package sat

// Source: src/sat/sat_cleaner.cpp and .h.

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/lbool"
)

// Actual named source owners: completed compacted prefix + unread suffix.
// Deleted/duplicate pointer slots in interrupted source storage are NOT owners.
func satCleanerAudit(t *testing.T, c *satCleaner, extra ...StoredClauseRef) {
	t.Helper()
	l := c.lifecycle
	p := l.propagation
	v := p.variables
	a := p.clauses
	refs := append([]StoredClauseRef(nil), extra...)
	addVector := func(x []StoredClauseRef, learned bool) {
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
	addVector(l.Original, false)
	addVector(l.learned, true)
	if l.pendingCandidate != (StoredClauseRef{}) {
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
	want := c.ownerBytes + l.ownerBytes
	want += p.ownerBytes + v.Stats().TotalBytes
	todo := v.elimTodo
	want += todo.ownerBytes
	want += a.ownerBytes + a.largeBytes + a.largeDirectory + small.OwnerBytes + small.PayloadBytes + small.MetadataBytes + small.DirectoryBytes
	mc := p.converter
	want += mc.ownerBytes
	r := p.limit
	want += uint64(reflect.TypeOf(control.Reslimit{}).Size()) + 8*uint64(cap(r.Limits)) + uint64(reflect.TypeOf((*control.Reslimit)(nil)).Size())*uint64(cap(r.Children)) + p.memory.Stats().TotalBytes
	if want != l.budget.Used() {
		t.Fatal("whole named once-only physical graph", want, l.budget.Used())
	}
	satScalarAuditRoots(t, v)
}
func TestSATCleanerNativeRecycleUnderRefusedArena(t *testing.T) {
	c, l := satCleanerFixture(t, 4)
	p := l.propagation
	dead := satCleanerClause(t, l, []Literal{0, 4, 6}, false, false)
	kept := satCleanerClause(t, l, []Literal{1, 2, 4, 6}, false, false)
	satOK(t, p.assign(0, satJustification{}))
	satOK(t, c.cleanupWatches())
	l.budget.FailAfter(0)
	satOK(t, c.cleanupClauses(false))
	l.budget.FailAfter(-1)
	if p.clauses.IsLive(dead) || !p.clauses.IsLive(kept) || len(l.Original) != 1 || l.Original[0] != kept || c.interrupted {
		t.Fatal("native recycled ID changed cleaner ownership", l.Original, c.interrupted)
	}
	satCleanerAudit(t, c)
	satOK(t, c.Close())
	satOK(t, l.Close())
}
func TestSATCleanerEveryAllocationPrefixAndOutputs(t *testing.T) {
	failures, endpoint := 0, int64(-1)
	for fault := int64(0); fault < 128; fault++ {
		t.Run(fmt.Sprint(fault), func(t *testing.T) {
			c, l := satCleanerFixture(t, 5)
			p := l.propagation
			satCleanerClause(t, l, []Literal{0, 4, 6}, false, false)
			kept := satCleanerClause(t, l, []Literal{1, 2, 4, 6}, false, false)
			satCleanerClause(t, l, []Literal{1, 6, 8}, true, false)
			satOK(t, p.assign(0, satJustification{}))
			l.budget.FailAfter(fault)
			done, e := c.run(true)
			if e != nil {
				failures++
				satProbingExpectedRefusal(t, p, e)
			} else {
				endpoint = fault
				if !done || len(l.Original) != 1 || l.Original[0] != kept || len(l.learned) != 0 || c.elimClauses != 1 || c.elimLiterals != 2 || l.deleted != 2 {
					t.Fatal("successful source result", done, len(l.Original), len(l.learned), c.elimClauses, c.elimLiterals, l.deleted)
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
	if endpoint < 0 {
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
				raw := satCleanerClause(t, l, []Literal{0, 2, 4}, false, false)
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
						lit := Literal(2 * v)
						if state < 0 {
							lit = lit.Negated()
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
			satCleanerClause(t, l, []Literal{0, 2, 4}, true, true)
			for i, a := range state {
				if a != 0 {
					lit := Literal(2 * i)
					if a < 0 {
						lit = lit.Negated()
					}
					satOK(t, p.assign(lit, satJustification{}))
				}
			}
			satOK(t, c.cleanupWatches())
			satOK(t, c.cleanupClauses(true))
			val := func(lit Literal, mask int) bool {
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
					a, e := p.variables.valueVar(BoolVar(i))
					satOK(t, e)
					if a != lbool.Undef && ((mask&(1<<i) != 0) != (a == lbool.True)) {
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
							actual = actual && (val(Literal(idx).Negated(), mask) || val(lit, mask))
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

func TestSATCleanerNativeRecyclePermitsNextPublication(t *testing.T) {
	c, l := satCleanerFixture(t, 4)
	p := l.propagation
	satCleanerClause(t, l, []Literal{0, 4, 6}, false, false)
	satCleanerClause(t, l, []Literal{1, 2, 4, 6}, false, false)
	satOK(t, p.assign(0, satJustification{}))
	l.budget.FailAfter(0)
	satOK(t, c.cleanupClauses(false))
	l.budget.FailAfter(-1)
	extra, e := p.clauses.MkClause([]Literal{2, 4, 6}, false)
	satOK(t, e)
	before := len(l.Original)
	satOK(t, l.pushClause(false, extra))
	if len(l.Original) != before+1 || l.Original[before] != extra {
		t.Fatal("native recycle blocked publication")
	}
	satCleanerAudit(t, c)
	satOK(t, c.Close())
	satOK(t, l.Close())
}

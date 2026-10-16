// Source: src/sat/sat_cleaner.cpp and .h; src/sat/sat_solver.cpp:3082–3087.
package sat

import (
	"errors"
	"github.com/glycerine/g3/pkg/util/lbool"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func satCleanerFixture(t *testing.T, n uint32) (*satCleaner, *satSolverClauseLifecycle) {
	t.Helper()
	l, _, _, b := satLifecycleFixture(t, n)
	c, e := newSATCleaner(l, b)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, c.Close()) })
	return c, l
}
func satCleanerClause(t *testing.T, l *satSolverClauseLifecycle, x []Literal, learned, frozen bool) StoredClauseRef {
	t.Helper()
	c, e := l.propagation.clauses.MkClause(x, learned)
	satOK(t, e)
	satOK(t, l.pushClause(learned, c))
	if frozen {
		satOK(t, l.propagation.clauses.Freeze(c))
	}
	return c
}
func TestSATCleanerSourceExitsAndSoleCost(t *testing.T) {
	c, l := satCleanerFixture(t, 2)
	p := l.propagation
	done, e := c.run(true)
	satOK(t, e)
	if done {
		t.Fatal("force still observes unchanged trail exit")
	}
	satOK(t, p.assign(0, satJustification{}))
	p.cleanerCounter = 5
	done, e = c.run(false)
	satOK(t, e)
	if done || p.cleanerCounter != 4 || c.lastNumUnits != 0 {
		t.Fatal(done, p.cleanerCounter, c.lastNumUnits)
	}
	done, e = c.run(true)
	satOK(t, e)
	if !done || c.lastNumUnits != 1 || p.cleanerCounter != 0 {
		t.Fatal(done, c.lastNumUnits, p.cleanerCounter)
	}
	p.inconsistent = true
	done, e = c.run(true)
	satOK(t, e)
	if done {
		t.Fatal("inconsistent exit")
	}
	p.inconsistent = false
	copy := *c
	if e = copy.Close(); !errors.Is(e, ErrSATOwner) {
		t.Fatal(e)
	}
	satOK(t, c.Close())
	if _, e = c.run(false); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}
func TestSATCleanerFrozenAndActiveClauseRoutes(t *testing.T) {
	for n := 0; n <= 5; n++ {
		t.Run(string(rune('0'+n)), func(t *testing.T) {
			c, l := satCleanerFixture(t, 6)
			p := l.propagation
			a := p.clauses
			// The false prefix is removed; every retained undefined literal is checked independently.
			x := []Literal{1}
			for i := 0; i < n; i++ {
				x = append(x, Literal(2*(i+1)))
			}
			if n == 0 {
				x = []Literal{1, 3}
			}
			r := satCleanerClause(t, l, x, n < 2, n < 2)
			satOK(t, p.assign(0, satJustification{}))
			if n == 0 {
				satOK(t, p.assign(2, satJustification{}))
			}
			satOK(t, c.cleanupWatches())
			satOK(t, c.cleanupClauses(n < 2))
			if n < 3 {
				if len(l.Original)+len(l.learned) != 0 || a.IsLive(r) {
					t.Fatal("deleted source route", n)
				}
			} else {
				if len(l.Original) != 1 {
					t.Fatal(n)
				}
				sz, e := a.Size(r)
				satOK(t, e)
				if sz != uint32(n) {
					t.Fatal(sz)
				}
				for i := uint32(0); i < sz; i++ {
					v, e := a.Literal(r, i)
					satOK(t, e)
					if v != Literal(2*(i+1)) {
						t.Fatal(v)
					}
				}
			}
			if n == 0 && !p.inconsistent {
				t.Fatal("empty frozen conflict")
			}
			if n == 1 {
				v, e := p.variables.valueLiteral(2)
				satOK(t, e)
				if v != lbool.True {
					t.Fatal(v)
				}
			}
			removed, cost := uint32(1), int32(n+1)
			if n == 0 {
				removed = 2
				cost = 2
			}
			if c.elimLiterals != removed || p.cleanerCounter != cost {
				t.Fatal(c.elimLiterals, p.cleanerCounter)
			}
		})
	}
}
func TestSATCleanerSatisfiedOrderingAndCostWrap(t *testing.T) {
	c, l := satCleanerFixture(t, 4)
	p := l.propagation
	a := satCleanerClause(t, l, []Literal{0, 2, 4}, false, false)
	b := satCleanerClause(t, l, []Literal{0, 3, 5}, true, false)
	satOK(t, p.assign(0, satJustification{}))
	p.cleanerCounter = 0x7ffffffe
	satOK(t, c.cleanupClauses(false))
	if p.cleanerCounter != -2147483647 || len(l.Original) != 0 || p.clauses.IsLive(a) {
		t.Fatal(p.cleanerCounter)
	}
	satOK(t, c.cleanupClauses(true))
	if len(l.learned) != 0 || p.clauses.IsLive(b) || c.elimClauses != 2 || l.deleted != 2 || l.nonLearnedGeneration != 1 {
		t.Fatal("source deletion stats")
	}
	c.resetStatistics()
	if c.elimClauses != 0 || c.elimLiterals != 0 || p.cleanerCounter != -2147483644 {
		t.Fatal("reset only stats")
	}
}

func TestSATCleanerConstructorAndStatistics(t *testing.T) {
	l, _, _, b := satLifecycleFixture(t, 1)
	before := b.Used()
	b.FailAfter(0)
	c, e := newSATCleaner(l, b)
	if c != nil || !errors.Is(e, arena.ErrInjectedFailure) || b.Used() != before {
		t.Fatal(c, e, b.Used(), before)
	}
	b.FailAfter(-1)
	foreign := arena.NewBudget(^uint64(0))
	foreign.FailAfter(0)
	if c, e = newSATCleaner(l, foreign); c != nil || !errors.Is(e, ErrSATOwner) || foreign.Used() != 0 {
		t.Fatal(c, e)
	}
	c, e = newSATCleaner(l, b)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, c.Close()) })
	if c.ownerBytes != uint64(reflect.TypeOf(satCleaner{}).Size()) || b.Used() != before+c.ownerBytes {
		t.Fatal("real cold allocation")
	}
	c.elimClauses = 7
	c.elimLiterals = 9
	s := statFixture(t)
	ck := statKey("elim clauses")
	lk := statKey("elim literals")
	satOK(t, c.collectStatistics(s, ck, lk))
	if s.Size() != 2 {
		t.Fatal(s.Size())
	}
	x := s.UintValue(0)
	y := s.UintValue(1)
	if x != 7 || y != 9 {
		t.Fatal(x, y)
	}
	c.resetStatistics()
	if c.elimClauses != 0 || c.elimLiterals != 0 {
		t.Fatal("source statistics reset")
	}
}

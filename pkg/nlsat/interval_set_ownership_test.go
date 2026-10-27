package nlsat

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/sat"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestIntervalReferenceOwnershipAndRefusalPrefixes(t *testing.T) {
	for _, phase := range []string{"mk", "union"} {
		for _, stop := range []int64{0, 1, 2, 3, 4, 8, 16, 32, 64, 128, -1} {
			t.Run(fmt.Sprintf("%s/%d", phase, stop), func(t *testing.T) {
				m := intervalFixture(t)
				pool := intervalPool(t, m)
				a := intervalMk(t, m, pool, intervalSpec{lower: -2, upper: 1, why: sat.MakeLiteral(1, false)})
				b := intervalMk(t, m, pool, intervalSpec{lower: 0, upper: 2, why: sat.MakeLiteral(2, false)})
				defer func() { assignmentMust(t, m.DecRef(b)); assignmentMust(t, m.DecRef(a)) }()
				m.budget.FailAfter(stop)
				var s *IntervalSet
				var err error
				if phase == "mk" {
					s, err = m.Mk(false, false, pool[-1], false, false, pool[3], sat.MakeLiteral(3, false))
				} else {
					s, err = m.MkUnion(a, b)
				}
				m.budget.FailAfter(-1)
				assignmentMust(t, m.am.CleanupRetained())
				assignmentMust(t, m.Cleanup())
				var physical umemory.Failure
				if err != nil && !errors.Is(err, arena.ErrInjectedFailure) && !(errors.As(err, &physical) && physical.Kind == umemory.FailureMemout && physical.ErrorCode() == 101) {
					t.Fatal("unexpected allocation refusal", err)
				}
				if s != nil {
					assignmentMust(t, m.IncRef(s))
					assignmentMust(t, m.DecRef(s))
				}
				if len(m.live) != 2 || len(m.pending) != 0 || len(m.retired) != 0 {
					t.Fatal("new owner or buffer retained", len(m.live), len(m.pending), len(m.retired))
				}
				if a.refs != 1 || b.refs != 1 {
					t.Fatal("failed creation changed input ownership")
				}
			})
		}
	}
}

func TestIntervalManagerBorrowingAndCopiedOwnerGuards(t *testing.T) {
	m := intervalFixture(t)
	pool := intervalPool(t, m)
	s := intervalMk(t, m, pool, intervalSpec{lower: 0, upper: 1, why: sat.MakeLiteral(1, false)})
	if err := m.Close(); !errors.Is(err, ErrIntervalLive) {
		t.Fatal("manager accepted live caller set", err)
	}
	copy := *m
	if _, err := copy.IsFull(s); err == nil {
		t.Fatal("copied manager accepted")
	}
	setCopy := *s
	if err := m.IncRef(&setCopy); err == nil {
		t.Fatal("copied set accepted")
	}
	s.refs = 0x7fffffff
	assignmentMust(t, m.IncRef(s))
	if s.refs != 0 {
		t.Fatal("31-bit reference counter did not wrap")
	}
	s.refs = 1
	assignmentMust(t, m.DecRef(s))
	if err := m.IncRef(s); err == nil {
		t.Fatal("retired set accepted")
	}
	assignmentMust(t, m.Close())
	assignmentMust(t, m.Close())
	if _, err := m.NumIntervals(nil); err == nil {
		t.Fatal("closed manager accepted query")
	}
}

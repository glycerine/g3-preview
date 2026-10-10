package sat

import (
	"errors"
	"fmt"
	"testing"
)

func TestSATCaseSplitHeapIndependentFiniteActivityOracle(t *testing.T) {
	for code := 0; code < 81; code++ {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			s, _, _ := satSolverCheckFixture(t)
			q := s.Variables.queue
			var weight [4]uint32
			var active [4]bool
			n := code
			for i := 0; i < 4; i++ {
				_, e := s.MkVar(false, true)
				satOK(t, e)
				weight[i] = uint32(n % 3)
				n /= 3
				active[i] = true
				satOK(t, s.Variables.activity.Set(uint32(i), weight[i]))
				satOK(t, q.ActivityIncreased(BoolVar(i)))
			}
			for left := 4; left > 0; left-- {
				v, e := q.NextVar()
				satOK(t, e)
				if uint32(v) >= 4 || !active[v] {
					t.Fatal(v, active)
				}
				for i := 0; i < 4; i++ {
					if active[i] && weight[i] > weight[v] {
						t.Fatal("not maximal actual activity", v, weight, active)
					}
				}
				active[v] = false
				present, e := q.containsChecked(v)
				satOK(t, e)
				if present {
					t.Fatal("not erased", v)
				}
				satFullSolverRuntimeAudit(t, s)
			}
			empty, e := q.emptyChecked()
			satOK(t, e)
			if !empty {
				t.Fatal(empty)
			}
			satOK(t, q.UnassignVar(2))
			satOK(t, q.UnassignVar(0))
			satOK(t, q.DelVar(2))
			present, e := q.containsChecked(2)
			satOK(t, e)
			if present {
				t.Fatal(present)
			}
			satOK(t, q.Reset())
			empty, e = q.emptyChecked()
			satOK(t, e)
			if !empty {
				t.Fatal(empty)
			}
			satFullSolverRuntimeAudit(t, s)
		})
	}
}

// Source insert publishes the reverse index before a refusing values.push.
func TestSATCaseSplitHeapWarmPhysicalPlateauAndOwnerGuards(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	for i := 0; i < 4; i++ {
		_, e := s.MkVar(false, true)
		satOK(t, e)
	}
	q := s.Variables.queue
	copyOwner := *q
	if _, e := copyOwner.emptyChecked(); !errors.Is(e, ErrSATOwner) {
		t.Fatal(e)
	}
	if e := copyOwner.Close(); !errors.Is(e, ErrSATOwner) {
		t.Fatal(e)
	}
	// Warm resets reuse actual existing member capacities. No source allocator
	// event or Go physical allocation may be fabricated on this hot route.
	run := func() {
		satOK(t, q.Reset())
		for i := 0; i < 4; i++ {
			satOK(t, q.UnassignVar(BoolVar(i)))
		}
		for i := 0; i < 4; i++ {
			_, e := q.NextVar()
			satOK(t, e)
		}
	}
	run()
	used := s.budget.Used()
	before := s.runtime.memory.Stats()
	s.budget.FailAfter(0)
	if allocations := testing.AllocsPerRun(20, run); allocations != 0 {
		t.Fatal("warmed heap host allocation", allocations)
	}
	satFullSolverRuntimeAudit(t, s)
	if after := s.runtime.memory.Stats(); after != before || s.budget.Used() != used {
		t.Fatal("warmed actual graph growth", before, after, used, s.budget.Used())
	}
	s.budget.FailAfter(-1)
}

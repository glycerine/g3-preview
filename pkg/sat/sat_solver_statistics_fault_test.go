// Source: src/sat/sat_solver.cpp:2781-2801,3716-3751; src/util/statistics.cpp.
package sat

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func satSolverStatisticsAudit(t *testing.T, s *Solver, st *util.Statistics, b *arena.Budget) {
	satFullSolverRuntimeAudit(t, s)
	q := st.Stats()
	sum := q.OwnerBytes + q.UintBytes + q.DoubleBytes + q.BindingBytes
	if sum != b.Used() {
		t.Fatal("actual Statistics owner graph", sum, b.Used())
	}
	for i := uint32(0); i < st.Size(); i++ {
		v, e := st.UintValue(i)
		satOK(t, e)
		if v != i+1 {
			t.Fatal("completed source update prefix", i, v)
		}
	}
}
func TestSATSolverStatisticsEveryActualOutputReservationPrefix(t *testing.T) {
	horizon := int64(0)
	t.Run("calibrate", func(t *testing.T) {
		s, _, _ := satSolverCheckFixture(t)
		b := arena.NewBudget(^uint64(0))
		st, k := satSolverStatisticsFixture(t, b)
		satSolverSeedStatistics(s)
		const n = int64(1 << 50)
		b.FailAfter(n)
		satOK(t, s.collectStatistics(st, k))
		horizon = n - testFieldBySourceName(reflect.ValueOf(b).Elem(), "remaining").Int()
		satSolverStatisticsAudit(t, s, st, b)
		b.FailAfter(-1)
	})
	failures, success := int64(0), 0
	for fault := int64(0); fault <= horizon; fault++ {
		t.Run(fmt.Sprint(fault), func(t *testing.T) {
			s, _, _ := satSolverCheckFixture(t)
			b := arena.NewBudget(^uint64(0))
			st, k := satSolverStatisticsFixture(t, b)
			satSolverSeedStatistics(s)
			before := s.budget.Used()
			blocks := s.runtime.memory.Stats().LiveBlocks
			b.FailAfter(fault)
			e := s.collectStatistics(st, k)
			satSolverStatisticsAudit(t, s, st, b)
			if before != s.budget.Used() || blocks != s.runtime.memory.Stats().LiveBlocks {
				t.Fatal("borrowed solver storage changed")
			}
			if e != nil {
				if !errors.Is(e, arena.ErrInjectedFailure) {
					t.Fatal(e)
				}
				failures++
			} else {
				if st.Size() != 25 {
					t.Fatal(st.Size())
				}
				success++
			}
			if s.Variables.mkVarCount != 1 || s.scc.numElim != 23 || s.probing.numAssigned != 25 {
				t.Fatal("collect changed source counters")
			}
			b.FailAfter(-1)
		})
	}
	if failures != horizon || success != 1 {
		t.Fatal(horizon, failures, success)
	}
	t.Log("actual output reserve horizon", horizon)
}

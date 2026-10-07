// Source: src/sat/sat_solver.cpp:3753-3768; src/sat/sat_types.h:289-293.
package sat

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestSATSolverStatDisplaySourceConditionalFields(t *testing.T) {
	s := satInspectionFixture(t)
	o := satInspectionStream(t, arena.NewBudget(math.MaxUint64))
	_, e := s.Lifecycle.MkClauseCore(4, []Literal{0, 2, 4, 6}, true)
	satOK(t, e)
	s.gc.numFrozen = 2
	t.Cleanup(func() { s.gc.numFrozen = 0 })
	s.gc.gcClauseStat = 9
	bytes, e := s.runtime.memory.AllocationSize()
	satOK(t, e)
	satOK(t, s.displayCompactStat(o))
	satInspectionText(t, o, fmt.Sprintf(" :clauses 1 :learned 4294967295 :frozen 2 :gc-clause 9 :memory %.2f", float64(bytes)/(1024*1024)))
	if o.State().FloatMode != util.NumericFixedFloat || o.State().Precision != 2 {
		t.Fatal(o.State())
	}
	satOK(t, o.ResetText())
	satOK(t, s.displayMemoryStat(o))
	satInspectionText(t, o, fmt.Sprintf(" :memory %.2f", float64(bytes)/(1024*1024)))
	satFullSolverRuntimeAudit(t, s)
}
func TestSATSolverStatDisplayEmptySourceMembers(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	o := satInspectionStream(t, arena.NewBudget(math.MaxUint64))
	s.gc.numFrozen = 8
	t.Cleanup(func() { s.gc.numFrozen = 0 })
	satOK(t, s.displayCompactStat(o))
	x, e := o.String()
	satOK(t, e)
	if !strings.HasPrefix(x, " :gc-clause 0 :memory ") {
		t.Fatal(x)
	}
	satFullSolverRuntimeAudit(t, s)
}
func TestSATSolverStatDisplayEveryReservationPrefix(t *testing.T) {
	var h int64
	var baseline string
	t.Run("calibrate", func(t *testing.T) {
		s := satInspectionFixture(t)
		o := satInspectionStream(t, arena.NewBudget(math.MaxUint64))
		const n = int64(1 << 50)
		o.Budget.FailAfter(n)
		satOK(t, s.displayCompactStat(o))
		h = n - testFieldBySourceName(reflect.ValueOf(o.Budget).Elem(), "remaining").Int()
		baseline, _ = o.String()
		satInspectionStreamAudit(t, s, o, o.Budget)
		o.Budget.FailAfter(-1)
	})
	var failures int64
	for i := int64(0); i <= h; i++ {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			s := satInspectionFixture(t)
			o := satInspectionStream(t, arena.NewBudget(math.MaxUint64))
			o.Budget.FailAfter(i)
			e := s.displayCompactStat(o)
			satInspectionStreamAudit(t, s, o, o.Budget)
			x, z := o.String()
			satOK(t, z)
			if !strings.HasPrefix(baseline, x) {
				t.Fatal(x, baseline)
			}
			if e != nil {
				if !errors.Is(e, arena.ErrInjectedFailure) {
					t.Fatal(e)
				}
				failures++
			} else if x != baseline {
				t.Fatal(x, baseline)
			}
			o.Budget.FailAfter(-1)
		})
	}
	if failures != h {
		t.Fatal(failures, h)
	}
	t.Log("actual stream horizon", h)
}

func TestSATSolverStatDisplayWarmAndOwnerGuards(t *testing.T) {
	s := satInspectionFixture(t)
	o := satInspectionStream(t, arena.NewBudget(math.MaxUint64))
	satOK(t, s.displayCompactStat(o))
	warm := o.Budget.Used()
	for i := 0; i < 20; i++ {
		satOK(t, o.ResetText())
		o.Budget.FailAfter(0)
		satOK(t, s.displayCompactStat(o))
		satInspectionStreamAudit(t, s, o, o.Budget)
		if o.Budget.Used() != warm {
			t.Fatal(warm, o.Budget.Used())
		}
		o.Budget.FailAfter(-1)
	}
	copy := *s
	if e := copy.displayCompactStat(o); e == nil {
		t.Fatal("copied owner accepted")
	}
	satOK(t, s.Close())
	if e := s.displayCompactStat(o); e == nil {
		t.Fatal("closed owner accepted")
	}
}

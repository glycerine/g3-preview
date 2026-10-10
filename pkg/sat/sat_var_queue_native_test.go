package sat

import (
	"errors"
	"reflect"
	"sort"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestSATNativeCaseSplitQueueGrowthIndependentPriorityAndColdBudget(t *testing.T) {
	header := uint64(reflect.TypeOf(satVarQueue{}).Size())
	activityHeader := uint64(reflect.TypeOf(satSolverActivity{}).Size())
	b := arena.NewBudget(header + activityHeader)
	a, e := newSATSolverActivity(b)
	satOK(t, e)
	b.FailAfter(0)
	if q, e := newSATSolverVarQueue(a, b); q != nil || !errors.Is(e, arena.ErrInjectedFailure) || b.Used() != activityHeader {
		t.Fatal("cold queue refusal leaked", q, e, b.Used())
	}
	b.FailAfter(1)
	q, e := newSATSolverVarQueue(a, b)
	satOK(t, e)
	if len(q.values) != 1 || q.values[0] != -1 || len(q.indices) != 128 || !q.Empty() {
		t.Fatal("source heap initial logical values", q.values, len(q.indices))
	}
	const count = 4097
	weights := make([]uint32, count)
	active := make([]BoolVar, 0, count)
	// The exact two cold headers exhaust Budget. Native backing and heap
	// operations must succeed without any additional reservation.
	for i := 0; i < count; i++ {
		weights[i] = uint32(i) * 2654435761
		satOK(t, a.Push(weights[i]))
		satOK(t, q.MkVar(BoolVar(i)))
	}
	weights[0] = ^uint32(0)
	satOK(t, a.Set(0, weights[0]))
	satOK(t, q.ActivityIncreased(0))
	for i := 0; i < count; i++ {
		if i%7 == 0 && i != 0 {
			satOK(t, q.DelVar(BoolVar(i)))
		} else {
			active = append(active, BoolVar(i))
		}
	}
	// Independent sorting supplies the expected sequence; every weight is
	// distinct, so this also verifies the current borrowed comparator.
	sort.Slice(active, func(i, j int) bool { return weights[active[i]] > weights[active[j]] })
	for i, want := range active {
		got, e := q.NextVar()
		satOK(t, e)
		if got != want || q.Contains(got) || q.Size() != uint32(len(active)-i-1) {
			t.Fatal("priority/reverse index", i, got, want, q.Size())
		}
	}
	if !q.Empty() || b.Used() != header+activityHeader {
		t.Fatal("native heap changed cold budget", b.Used())
	}
	satOK(t, q.UnassignVar(count-1))
	satOK(t, q.UnassignVar(3))
	satOK(t, q.DelVar(count-1))
	v, e := q.NextVar()
	satOK(t, e)
	if v != 3 {
		t.Fatal("native reinsert/delete", v)
	}
	satOK(t, q.Reset())
	satOK(t, q.Close())
	satOK(t, q.Close())
	if q.values != nil || q.indices != nil || b.Used() != activityHeader {
		t.Fatal("native close/refund", b.Used())
	}
	if _, e = q.NextVar(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("closed queue", e)
	}
	satOK(t, a.Close())
	if b.Used() != 0 {
		t.Fatal("cold owner graph refund", b.Used())
	}
}

func TestSATNativeCaseSplitQueueOwnerBudgetAndIndexGuards(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	foreign := arena.NewBudget(^uint64(0))
	foreign.FailAfter(0)
	if q, e := newSATSolverVarQueue(s.Variables.activity, foreign); q != nil || !errors.Is(e, ErrSATOwner) || foreign.Used() != 0 {
		t.Fatal("borrowed activity cohort before reserve", q, e, foreign.Used())
	}
	q := s.Variables.queue
	for _, v := range []BoolVar{1 << 31, ^BoolVar(0)} {
		for _, op := range []func(BoolVar) error{q.ActivityIncreased, q.DelVar, q.UnassignVar, q.MkVar} {
			if e := op(v); !errors.Is(e, arena.ErrBounds) {
				t.Fatal("signed source index guard", v, e)
			}
		}
	}
	for _, v := range []BoolVar{128, 1<<31 - 1} {
		satOK(t, q.ActivityIncreased(v))
		satOK(t, q.DelVar(v))
	}
}

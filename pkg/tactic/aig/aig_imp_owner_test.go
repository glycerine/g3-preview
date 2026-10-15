package aig

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestAIGManagerSeparateImpChargesAndConstructorPrefixes(t *testing.T) {
	m, budget := aigFixture(t)
	warm, e := NewManager(m)
	aigMust(t, e)
	aigMust(t, warm.Close())
	baseline := budget.Used()
	reached := false
	for prefix := int64(0); prefix < 16; prefix++ {
		budget.FailAfter(prefix)
		g, e := NewManager(m)
		budget.FailAfter(-1)
		if e == nil {
			if g == nil || g.managerImp == nil || g.header != uint64(reflect.TypeOf(Manager{}).Size()) ||
				g.impHeader != uint64(reflect.TypeOf(managerImp{}).Size()) {
				t.Fatalf("separate source owners: %#v", g)
			}
			aigMust(t, g.Close())
			reached = true
			break
		}
		if g != nil || !errors.Is(e, arena.ErrInjectedFailure) {
			t.Fatalf("constructor prefix %d: %v %v", prefix, g, e)
		}
		if used := budget.Used(); used != baseline {
			t.Fatalf("constructor prefix %d retained %d bytes", prefix, used-baseline)
		}
	}
	if !reached || budget.Used() != baseline {
		t.Fatal("AIG constructor never completed without retaining its owner charges")
	}
}

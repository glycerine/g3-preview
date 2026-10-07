package util

import (
	"fmt"
	"reflect"
	"testing"
)

// Measure the successful operation, then refuse every actual cold reservation.
// Roots are audited before cleanup, including an incomplete graph constructor.
func TestNegatedLiteralCliquesEveryReservationOwnsAndCloses(t *testing.T) {
	const maximum = int64(10000)
	horizon := int64(0)
	run := func(t *testing.T, fault int64, measure bool) {
		m, b := memoryRuntimeFixture(t, false, 64)
		b.FailAfter(fault)
		var ps RuntimeUintColumn
		ps.Bind(m)
		var out *RuntimeUnsignedRows
		g, e := NewNegatedLiteralCliques(m, b)
		if e == nil {
			for _, edge := range [][2]uint32{{1, 2}, {3, 5}, {1, 5}} {
				e = g.AddEdge(edge[0], edge[1])
				if e != nil {
					break
				}
			}
		}
		if e == nil {
			out, e = NewRuntimeUnsignedRows(m, b)
		}
		if e == nil {
			for _, v := range []uint32{0, 3, 4} {
				e = ps.Push(v)
				if e != nil {
					break
				}
			}
		}
		if e == nil {
			e = g.Cliques(&ps, out)
		}
		cliqueTestAudit(t, m, g, &ps, out)
		if measure {
			memoryRuntimeRequire(t, e)
			horizon = maximum - testFieldBySourceName(reflect.ValueOf(b).Elem(), "remaining").Int()
		} else if fault < horizon {
			cliqueTestRefusal(t, e)
		} else {
			memoryRuntimeRequire(t, e)
		}
		b.FailAfter(-1)
		if out != nil {
			memoryRuntimeRequire(t, out.CloseChecked())
		}
		memoryRuntimeRequire(t, ps.Finalize())
		if g != nil {
			memoryRuntimeRequire(t, g.Close())
		}
		if m.Stats().LiveBlocks != 0 {
			t.Fatal("orphaned graph roots", m.Stats())
		}
	}
	t.Run("measure", func(t *testing.T) { run(t, maximum, true) })
	if horizon == 0 {
		t.Fatal("did not measure reservations")
	}
	for fault := int64(0); fault <= horizon; fault++ {
		t.Run(fmt.Sprint(fault), func(t *testing.T) { run(t, fault, false) })
	}
}

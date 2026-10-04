// Source: src/math/polynomial/polynomial.cpp:2195-2198; src/util/vector.h.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
)

func TestPolynomialSOMVectorWrappedIndexReachesChildWithoutReservation(t *testing.T) {
	for _, populated := range []bool{false, true} {
		t.Run(map[bool]string{false: "empty", true: "populated"}[populated], func(t *testing.T) {
			m := polynomialCellFixture(t)
			v, e := NewPolynomialSOMVector(m)
			serviceMust(t, e)
			serviceMust(t, v.SetOwner(m))
			t.Cleanup(func() { serviceMust(t, v.CloseChecked()) })
			if populated {
				_, e = v.At(1)
				serviceMust(t, e)
			}
			n, e := v.data.size(m)
			serviceMust(t, e)
			oldRaw, oldUsed := v.data.raw, m.integer.budget.Used()
			m.integer.budget.FailAfter(0)
			_, e = v.At(^uint32(0))
			want := error(arena.ErrBounds)
			if modularSourceDebug {
				want = ErrPolynomialAssertion
			} else if !populated {
				want = memory.ErrRuntimeOwner // actual reached NULL payload read.
			}
			if !errors.Is(e, want) {
				t.Errorf("source idx+1 wraps to0; reached child guard precedes allocation: got%v want%v", e, want)
			}
			got, sizeErr := v.data.size(m)
			serviceMust(t, sizeErr)
			if got != n || v.data.raw != oldRaw || v.pending != nil || m.integer.budget.Used() != oldUsed {
				t.Fatal("wrapped capacity request changed actual class/vector prefix")
			}
			polynomialSOMPhysicalAudit(t, m, v, nil)
			m.integer.budget.FailAfter(-1)
		})
	}
}

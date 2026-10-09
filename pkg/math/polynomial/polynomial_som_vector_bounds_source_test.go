// Source: src/math/polynomial/polynomial.cpp:2195-2198; src/util/vector.h.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
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
			n, oldUsed := len(v.children), m.integer.Budget().Used()
			m.integer.Budget().FailAfter(0)
			_, e = v.At(^uint32(0))
			want := error(arena.ErrBounds)
			if polynomialSourceDebug {
				want = ErrPolynomialAssertion

			}
			if !errors.Is(e, want) {
				t.Errorf("source idx+1 wraps to0; reached child guard precedes allocation: got%v want%v", e, want)
			}
			if len(v.children) != n || v.pending != nil || m.integer.Budget().Used() != oldUsed {
				t.Fatal("wrapped capacity request changed actual class/vector prefix")
			}
			polynomialSOMPhysicalAudit(t, m, v, nil)
			m.integer.Budget().FailAfter(-1)
		})
	}
}

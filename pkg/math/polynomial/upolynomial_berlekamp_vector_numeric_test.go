package polynomial

import (
	"fmt"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestUPolynomialBerlekampVectorHelperActualAllocationPrefixes(t *testing.T) {
	for _, divide := range []bool{false, true} {
		complete, retained := false, false
		for ordinal := int64(0); ordinal < 256 && !complete; ordinal++ {
			t.Run(fmt.Sprintf("divide=%v/%d", divide, ordinal), func(t *testing.T) {
				q, s := preparationPhysicalFixture(t)
				m, b := s.manager, q.Budget()
				var big unumeral.Int
				serviceMust(t, q.Integers().SetLimbs(&big, []uint32{1, 0, 1}))
				t.Cleanup(func() { serviceMust(t, q.Integers().Del(&big)) })
				// A genuine integer-mode caller polynomial supplies a BIG coefficient.
				// After switching modes the helpers' actual SET/normalization is reached;
				// this qualifies allocation-prefix evidence, not field-validity of A.
				a := upolySourceNew(t, m, nil)
				serviceMust(t, m.SetInts(a, []unumeral.Int{big, unumeral.NewInt(0), unumeral.NewInt(1)}))
				divisor := upolySourceNew(t, m, []int32{1, 1})
				driver := upolySourceNew(t, m, []int32{-1, 0, 1})
				out := upolySourceNew(t, m, nil)
				expected := []int32{1, 1}
				if divide {
					expected = []int32{-1, 1}
				}
				want := upolySourceNew(t, m, expected)
				three := unumeral.NewInt(3)
				serviceMust(t, m.SetZP(&three))
				matrix := upolyBerlekampMatrix{m: m}
				serviceMust(t, matrix.init(m, mustUPolyView(t, m, driver)))
				av, bv := mustUPolyView(t, m, a), mustUPolyView(t, m, divisor)
				target, e := m.descriptor(out)
				serviceMust(t, e)
				t.Cleanup(func() { b.FailAfter(-1); serviceMust(t, m.CleanupBerlekampMatrix()); matrix.close() })
				b.FailAfter(ordinal)
				body := matrix.vectorNumericCall(divide, av, bv, &target.state)
				b.FailAfter(-1)
				if matrix.numeric.pending() {
					retained = true
					if m.pendingBerlekamp != &matrix {
						t.Fatal("helper acquisition lost exact matrix owner", divide, ordinal, body)
					}
				}
				if body != nil && !numericPhysicalReservationRefusal(body) {
					t.Fatal("unexpected actual vector helper prefix", divide, ordinal, body)
				}
				serviceMust(t, m.CleanupBerlekampMatrix())
				if body == nil {
					same, e := m.Equal(out, want)
					serviceMust(t, e)
					if !same {
						t.Fatal("independent normalized GCD/quotient result", divide)
					}
					complete = true
				}
				matrix.close()
				if matrix.numeric.pending() || m.pendingBerlekamp != nil {
					t.Fatal("helper cleanup retained acquired numerical work")
				}
			})
		}
		if !complete || !retained {
			t.Fatal("helper completion/actual acquisition not reached", divide, complete, retained)
		}
	}
}

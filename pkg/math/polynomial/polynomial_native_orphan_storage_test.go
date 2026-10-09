package polynomial

import (
	"testing"
)

func TestPolynomialNativeDerivativeOrphanGrowthAndTransfer(t *testing.T) {
	m := polynomialPSCFullFixtureWithLimit(t, 1<<25)
	c := m.core
	const count = 257
	inputs := make([]Polynomial, count)
	zeros := make([]Polynomial, count)
	receipts := make([]polynomialDerivativeInvocationReceipt, count)
	for i := range inputs {
		inputs[i] = polynomialQueryTerms(t, c, []int32{1}, [][]MonomialPower{{{0, uint32(2 * (i + 1))}}})
	}
	serviceMust(t, c.modular.SetZPUint64(2))
	moves := 0
	for i, p := range inputs {
		old := m.derivative.orphans
		var e error
		zeros[i], e = m.derivativeWithReceipt(p, 0, &receipts[i])
		serviceMust(t, e)
		polynomialCaller(t, c, zeros[i])
		yes, e := c.IsZero(zeros[i])
		serviceMust(t, e)
		if !yes || len(m.derivative.orphans) != i+1 {
			t.Fatal("GF2 derivative must retain one created zero term", i)
		}
		if len(old) != 0 && &old[0] != &m.derivative.orphans[0] {
			moves++
			for _, a := range old {
				if a != (Monomial{}) {
					t.Fatal("native relocation left duplicate owning record")
				}
			}
		}
	}
	if moves < 5 {
		t.Fatal("native cohort relocation not exercised", moves)
	}
	for i, a := range m.derivative.orphans {
		power, e := c.monomials.ReadPower(a, 0)
		serviceMust(t, e)
		rc, e := c.monomialRefCount(a)
		serviceMust(t, e)
		if power != (MonomialPower{0, uint32(2*i + 1)}) || rc != 0 || m.derivative.orphanInvocations[i] != receipts[i].invocation {
			t.Fatal("native cohort changed actual root or invocation", i, power, rc)
		}
	}
	polynomialImpPhysicalAuditNamed(t, m, append(append([]Polynomial{}, inputs...), zeros...), nil, 0)
	serviceMust(t, c.modular.SetZPUint64(3))
	for i, p := range inputs {
		result, e := m.Derivative(p, 0)
		serviceMust(t, e)
		polynomialCaller(t, c, result)
		k := i + 1
		want := map[[3]uint32]int64{}
		if k%3 != 0 {
			coefficient := int64((2 * k) % 3)
			if coefficient == 2 {
				coefficient = -1
			}
			want[[3]uint32{uint32(2*k - 1), 0, 0}] = coefficient
			if m.derivative.orphans[i] != (Monomial{}) || m.derivative.orphanInvocations[i] != 0 {
				t.Fatal("actual nonzero INC did not transfer prior cohort", i)
			}
		} else if m.derivative.orphans[i] == (Monomial{}) {
			t.Fatal("zero derivative consumed unreferenced root", i)
		}
		polynomialQueryExpect(t, c, result, want)
	}
	for i := range receipts {
		serviceMust(t, m.cleanupDerivativeInvocation(&receipts[i], true))
	}
	if m.derivativeHasOrphans() {
		t.Fatal("matching invocation cleanup retained actual old roots")
	}
	serviceMust(t, m.CleanupDerivative())
}

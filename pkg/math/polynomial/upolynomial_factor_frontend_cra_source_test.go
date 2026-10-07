// Source: src/math/polynomial/upolynomial.cpp:663-721,3042-3118.
package polynomial

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestUPolynomialFactorFrontendActualCRAConstructorFrontiers(t *testing.T) {
	z, s := preparationPhysicalFixture(t)
	m := s.manager
	fs, e := m.NewFactors()
	serviceMust(t, e)
	var input upolyVectorState
	var square, linear util.Int
	serviceMust(t, z.Integers().SetUint64(&square, 9000000006000000001))
	serviceMust(t, z.Integers().SetUint64(&linear, 6000000002))
	serviceMust(t, m.set(&input, []util.Int{square, linear, util.NewInt(1)}))
	defer func() {
		z.Budget().FailAfter(-1)
		serviceMust(t, s.CleanupRetained())
		serviceMust(t, fs.Close())
		serviceMust(t, m.reset(&input))
		m.freeBacking(&input)
		serviceMust(t, z.Integers().Del(&linear))
		serviceMust(t, z.Integers().Del(&square))
	}()
	_, e = s.Factor(m.stateView(&input), fs, DefaultUPolynomialFactorParams())
	serviceMust(t, e)
	q := &s.frontend.cra
	if polynomialSourceDebug && s.frontend.verification.Invocation == 0 {
		t.Fatal("actual positive CRA division private verification child was not captured")
	}
	if s.frontend.primitive.invocation == 0 {
		t.Fatal("actual modular candidate primitive scoped cont not bound")
	}
	if s.frontend.divisibility.invocation == 0 {
		t.Fatal("actual candidate divisibility scoped ratio not bound")
	}
	if q.invocation == 0 || q.peak != 12 || q.constructed != 0 {
		t.Fatal("actual modular reconstruction did not bind literal reached CRA NUM frontiers", q.invocation, q.peak, q.constructed)
	}
	var caller []*util.Int
	caller = append(caller, &square, &linear)
	for i := range m.values(&input) {
		caller = append(caller, &m.values(&input)[i])
	}
	primeTrialPhysicalAuditOwners(t, z, s, []*UPolynomialFactors{fs}, caller...)
}

func TestUPolynomialFactorFrontendActualCRAEveryBIGPrefix(t *testing.T) {
	completed := false
	for ordinal := int64(0); ordinal < 256 && !completed; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			z, s := preparationPhysicalFixture(t)
			m := s.manager
			fs, e := m.NewFactors()
			serviceMust(t, e)
			var input upolyVectorState
			var square, linear util.Int
			serviceMust(t, z.Integers().SetUint64(&square, 9000000006000000001))
			serviceMust(t, z.Integers().SetUint64(&linear, 6000000002))
			serviceMust(t, m.set(&input, []util.Int{square, linear, util.NewInt(1)}))
			defer func() {
				z.Budget().FailAfter(-1)
				serviceMust(t, s.CleanupRetained())
				serviceMust(t, fs.Close())
				serviceMust(t, m.reset(&input))
				m.freeBacking(&input)
				serviceMust(t, z.Integers().Del(&linear))
				serviceMust(t, z.Integers().Del(&square))
			}()
			var caller []*util.Int
			caller = append(caller, &square, &linear)
			for i := range m.values(&input) {
				caller = append(caller, &m.values(&input)[i])
			}
			z.Budget().FailAfter(ordinal)
			_, body := s.Factor(m.stateView(&input), fs, DefaultUPolynomialFactorParams())
			primeTrialPhysicalAuditOwners(t, z, s, []*UPolynomialFactors{fs}, caller...)
			if body == nil {
				completed = true
				return
			}
			receipt := s.frontend.receipt
			firstBody, firstSource := s.frontend.bodyCause, s.frontend.sourceCause
			if receipt.invocation != 0 && s.frontend.active {
				e = s.cleanupFactorFrontendInvocation(&receipt, false)
				if e != nil && !numericPhysicalReservationRefusal(e) {
					t.Fatal("before-lift false child phase", e)
				}
				primeTrialPhysicalAuditOwners(t, z, s, []*UPolynomialFactors{fs}, caller...)
				if e == nil {
					e = s.cleanupFactorFrontendInvocation(&receipt, true)
				}
				if e != nil && !numericPhysicalReservationRefusal(e) {
					t.Fatal("before-lift true unjoined phase", e)
				}
				primeTrialPhysicalAuditOwners(t, z, s, []*UPolynomialFactors{fs}, caller...)
				if s.frontend.bodyCause != firstBody || s.frontend.sourceCause != firstSource {
					t.Fatal("own phases erased actual first causes")
				}
			}
		})
	}
	if !completed {
		t.Fatal("real CRA source prefix horizon did not reach success")
	}
}

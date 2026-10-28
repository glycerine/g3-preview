package polynomial

// Source: src/math/polynomial/upolynomial.cpp:2967-3011,3042-3118.

import (
	"fmt"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestUPolynomialFactorFrontendActualQuadraticFrontiers(t *testing.T) {
	z, s := preparationPhysicalFixture(t)
	m := s.manager
	fs, e := m.NewFactors()
	serviceMust(t, e)
	var input upolyVectorState
	serviceMust(t, m.set(&input, []unumeral.Int{unumeral.NewInt(-1), unumeral.NewInt(0), unumeral.NewInt(1)}))
	defer func() {
		z.Budget().FailAfter(-1)
		serviceMust(t, s.CleanupRetained())
		serviceMust(t, fs.Close())
		serviceMust(t, m.reset(&input))
		m.freeBacking(&input)
	}()
	_, e = s.Factor(m.stateView(&input), fs, DefaultUPolynomialFactorParams())
	serviceMust(t, e)
	if s.frontend.modularPower.invocation == 0 {
		t.Fatal("actual quadratic Power plain RAW child was not bound")
	}
	if s.frontend.vectorNormalize.invocation != 2 || s.frontend.vectorNormalize.constructed != 0 {
		t.Fatal("actual f1/f2 normalize scoped g/DEBUG temporaries not bound", s.frontend.vectorNormalize.invocation)
	}
	if s.frontend.perfectSquare.Frame != &z.Integers().PerfectSquare || s.frontend.perfectSquare.Invocation == 0 {
		t.Fatal("actual quadratic Integer six-RAW perfectSquare child was not captured")
	}
	q := &s.frontend.quadratic
	peak := uint8(6)
	if polynomialSourceDebug {
		peak = 7
	}
	if q.invocation == 0 || q.peak != peak || q.constructed != 0 {
		t.Fatal("actual quadratic source NUM/vector declarations were not bound", q.invocation, q.peak, q.constructed)
	}
	if fs.count != 2 {
		t.Fatal("actual ordered factor pushes", fs.count)
	}
	roots := []*unumeral.Int{}
	for i := range m.values(&input) {
		roots = append(roots, &m.values(&input)[i])
	}
	primeTrialPhysicalAuditOwners(t, z, s, []*UPolynomialFactors{fs}, roots...)
}

func TestUPolynomialFactorFrontendActualQuadraticPowerFailedBodyRetainsRAW(t *testing.T) {
	reached := false
	for ordinal := int64(0); ordinal < 256 && !reached; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			z, s := preparationPhysicalFixture(t)
			m := s.manager
			fs, e := m.NewFactors()
			serviceMust(t, e)
			var input upolyVectorState
			var b unumeral.Int
			serviceMust(t, z.Integers().SetLimbs(&b, []uint32{0xffffffff, 0xffffffff, 0xffffffff, 0x7fffffff}))
			serviceMust(t, m.set(&input, []unumeral.Int{unumeral.NewInt(1), b, unumeral.NewInt(1)}))
			defer func() {
				z.Budget().FailAfter(-1)
				serviceMust(t, s.CleanupRetained())
				serviceMust(t, fs.Close())
				serviceMust(t, m.reset(&input))
				m.freeBacking(&input)
				serviceMust(t, z.Integers().Del(&b))
			}()
			roots := []*unumeral.Int{&b}
			for i := range m.values(&input) {
				roots = append(roots, &m.values(&input)[i])
			}
			z.Budget().FailAfter(ordinal)
			_, body := s.Factor(m.stateView(&input), fs, DefaultUPolynomialFactorParams())
			primeTrialPhysicalAuditOwners(t, z, s, []*UPolynomialFactors{fs}, roots...)
			q := &s.frontend
			c := &q.modularPower
			if body == nil || !c.active || c.endEntered || (c.power).Cell == 0 {
				return
			}
			reached = true
			if !q.active || c.bodyCause == nil || c.sourceCause != nil || q.quadratic.constructed != 0 || q.constructed != 0 {
				t.Fatal("failed RAW body blocked literal older NUM/vector DT or lost retained authority", ordinal, body, c.bodyCause, c.sourceCause, q.quadratic.constructed, q.constructed)
			}
			receipt := q.receipt
			original := c.power
			firstBody, firstSource := q.bodyCause, q.sourceCause
			e = s.cleanupFactorFrontendInvocation(&receipt, false)
			if e != nil {
				t.Fatal("false completed child", e)
			}
			if !c.active || c.power != original {
				t.Fatal("false ACK deleted unjoined source RAW")
			}
			primeTrialPhysicalAuditOwners(t, z, s, []*UPolynomialFactors{fs}, roots...)
			e = s.cleanupFactorFrontendInvocation(&receipt, true)
			if e != nil && !numericPhysicalReservationRefusal(e) {
				t.Fatal("true exact RAW abort", e)
			}
			primeTrialPhysicalAuditOwners(t, z, s, []*UPolynomialFactors{fs}, roots...)
			if q.bodyCause != firstBody || q.sourceCause != firstSource {
				t.Fatal("own ACK erased original causes")
			}
		})
	}
	if !reached {
		t.Fatal("actual retained Power RAW prefix not reached")
	}
}

func TestUPolynomialFactorFrontendActualQuadraticPerfectSquareBIGPrefixes(t *testing.T) {
	complete, owned := false, false
	for ordinal := int64(0); ordinal < 512 && !complete; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			z, s := preparationPhysicalFixture(t)
			m := s.manager
			fs, e := m.NewFactors()
			serviceMust(t, e)
			var input upolyVectorState
			serviceMust(t, m.set(&input, []unumeral.Int{unumeral.NewInt(1), unumeral.NewInt(50000), unumeral.NewInt(1)}))
			defer func() {
				z.Budget().FailAfter(-1)
				serviceMust(t, s.CleanupRetained())
				serviceMust(t, fs.Close())
				serviceMust(t, m.reset(&input))
				m.freeBacking(&input)
			}()
			roots := []*unumeral.Int{}
			for i := range m.values(&input) {
				roots = append(roots, &m.values(&input)[i])
			}
			z.Budget().FailAfter(ordinal)
			_, body := s.Factor(m.stateView(&input), fs, DefaultUPolynomialFactorParams())
			primeTrialPhysicalAuditOwners(t, z, s, []*UPolynomialFactors{fs}, roots...)
			if body == nil {
				complete = true
				return
			}
			q := &s.frontend
			r := q.receipt
			firstBody, firstSource := q.bodyCause, q.sourceCause
			if z.Integers().MatchesPerfectSquareInvocation(&q.perfectSquare) && unumeral.NumericAllocationPending(&q.perfectSquare.Frame.Numeric) {
				owned = true
				if q.perfectSquare.Frame.Executing || !q.active {
					t.Fatal("actual acquired RAW leaf lost matched completed-body parent", ordinal)
				}
			}
			if q.active {
				e = s.cleanupFactorFrontendInvocation(&r, false)
				if e == nil && ordinal%2 == 0 && z.Integers().MatchesPerfectSquareInvocation(&q.perfectSquare) {
					e = z.Integers().CleanupPerfectSquare()
				}
				if e == nil {
					e = s.cleanupFactorFrontendInvocation(&r, true)
				}
				if e != nil && !numericPhysicalReservationRefusal(e) {
					t.Fatal("before-lift exact square owning phases", ordinal, e)
				}
				primeTrialPhysicalAuditOwners(t, z, s, []*UPolynomialFactors{fs}, roots...)
				if q.bodyCause != firstBody || q.sourceCause != firstSource {
					t.Fatal("own erased actual causes")
				}
			}
		})
	}
	if !complete || !owned {
		t.Fatal("actual square BIG prefix did not prove direct pending leaf and success", complete, owned)
	}
}

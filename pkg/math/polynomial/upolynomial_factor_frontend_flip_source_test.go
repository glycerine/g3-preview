// Source: src/math/polynomial/upolynomial.cpp:2960-2965,3030-3118.
package polynomial

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestUPolynomialFactorFrontendActualFlipNumeral(t *testing.T) {
	z, s := preparationPhysicalFixture(t)
	m := s.manager
	fs, e := m.NewFactors()
	serviceMust(t, e)
	var input upolyVectorState
	serviceMust(t, m.set(&input, []util.Int{util.NewInt(-1), util.NewInt(0), util.NewInt(0), util.NewInt(-1)}))
	defer func() {
		z.Budget().FailAfter(-1)
		serviceMust(t, s.CleanupRetained())
		serviceMust(t, fs.Close())
		serviceMust(t, m.reset(&input))
		m.freeBacking(&input)
	}()
	_, e = s.Factor(m.stateView(&input), fs, DefaultUPolynomialFactorParams())
	serviceMust(t, e)
	q := &s.frontend.flip
	if q.invocation == 0 || q.constructed || q.executing {
		t.Fatal("actual odd negative sign flip scoped NUM was not bound", q.invocation, q.constructed, q.executing)
	}
	var roots []*util.Int
	for i := range m.values(&input) {
		roots = append(roots, &m.values(&input)[i])
	}
	primeTrialPhysicalAuditOwners(t, z, s, []*UPolynomialFactors{fs}, roots...)
}

func TestUPolynomialFactorFrontendActualFlipBIGPrefixes(t *testing.T) {
	complete := false
	for ordinal := int64(0); ordinal < 128 && !complete; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			z, s := preparationPhysicalFixture(t)
			m := s.manager
			fs, e := m.NewFactors()
			serviceMust(t, e)
			var input upolyVectorState
			var big util.Int
			serviceMust(t, z.Integers().SetUint64(&big, 1099511627899))
			serviceMust(t, z.Integers().Neg(&big))
			serviceMust(t, m.set(&input, []util.Int{big, big}))
			defer func() {
				z.Budget().FailAfter(-1)
				serviceMust(t, s.CleanupRetained())
				serviceMust(t, fs.Close())
				serviceMust(t, m.reset(&input))
				m.freeBacking(&input)
				serviceMust(t, z.Integers().Del(&big))
			}()
			roots := []*util.Int{&big}
			for i := range m.values(&input) {
				roots = append(roots, &m.values(&input)[i])
			}
			z.Budget().FailAfter(ordinal)
			_, body := s.Factor(m.stateView(&input), fs, DefaultUPolynomialFactorParams())
			primeTrialPhysicalAuditOwners(t, z, s, []*UPolynomialFactors{fs}, roots...)
			if body == nil {
				if s.frontend.flip.invocation == 0 {
					t.Fatal("actual BIG flip was not reached")
				}
				complete = true
				return
			}
			q := &s.frontend
			r := q.receipt
			cause, source := q.bodyCause, q.sourceCause
			if q.active {
				e = s.cleanupFactorFrontendInvocation(&r, false)
				if e == nil {
					e = s.cleanupFactorFrontendInvocation(&r, true)
				}
				if e != nil && !numericPhysicalReservationRefusal(e) {
					t.Fatal("exact own phase", e)
				}
				primeTrialPhysicalAuditOwners(t, z, s, []*UPolynomialFactors{fs}, roots...)
				if q.bodyCause != cause || q.sourceCause != source {
					t.Fatal("own erased stopped causes")
				}
			}
		})
	}
	if !complete {
		t.Fatal("BIG flip prefixes did not reach success")
	}
}
func TestUPolynomialFactorFrontendForeignFactorsClose(t *testing.T) {
	z, s := preparationPhysicalFixture(t)
	m := s.manager
	fs, e := m.NewFactors()
	serviceMust(t, e)
	foreign, e := m.NewFactors()
	serviceMust(t, e)
	var input upolyVectorState
	serviceMust(t, m.set(&input, []util.Int{util.NewInt(1), util.NewInt(1), util.NewInt(0), util.NewInt(1)}))
	reached := false
	m.config.Checkpoint = func() bool {
		if !reached && s.frontend.executing {
			reached = true
			before := *fs
			header := foreign.header
			bytes := z.Budget().Used()
			serviceMust(t, foreign.Close())
			if !foreign.closed || bytes-z.Budget().Used() != header || !reflect.DeepEqual(before, *fs) {
				t.Fatal("foreign factors cleanup touched actual output")
			}
		}
		return true
	}
	defer func() {
		z.Budget().FailAfter(-1)
		serviceMust(t, s.CleanupRetained())
		serviceMust(t, fs.Close())
		if !foreign.closed {
			serviceMust(t, foreign.Close())
		}
		serviceMust(t, m.reset(&input))
		m.freeBacking(&input)
	}()
	_, e = s.Factor(m.stateView(&input), fs, DefaultUPolynomialFactorParams())
	serviceMust(t, e)
	if !reached {
		t.Fatal("actual caller checkpoint not reached")
	}
	roots := []*util.Int{}
	for i := range m.values(&input) {
		roots = append(roots, &m.values(&input)[i])
	}
	primeTrialPhysicalAuditOwners(t, z, s, []*UPolynomialFactors{fs}, roots...)
}

package polynomial

import (
	"fmt"
	"testing"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

// The actual public frontend reaches the all-primes-bad path for (1+P*x)^2,
// where P is the exact product of every retained modular-GCD trial prime.
func TestUPolynomialFactorFrontendActualAllPrimeFallbackContext(t *testing.T) {
	completed := false
	for ordinal := int64(0); ordinal < 48 && !completed; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			q, s := preparationPhysicalFixture(t)
			m := s.manager
			z := q.Integers()
			var p, p2, twoP unumeral.Int
			serviceMust(t, z.SetInt64(&p, 1))
			for _, prime := range upolyBigPrimes {
				n := unumeral.NewInt(int32(prime))
				serviceMust(t, z.Mul(&p, &n, &p))
			}
			serviceMust(t, z.Mul(&p, &p, &p2))
			two := unumeral.NewInt(2)
			serviceMust(t, z.Mul(&p, &two, &twoP))
			var input upolyVectorState
			serviceMust(t, m.set(&input, []unumeral.Int{unumeral.NewInt(1), twoP, p2}))
			fs, e := m.NewFactors()
			serviceMust(t, e)
			armed, afterFallback := false, false
			published := false
			m.config.Checkpoint = func() bool {
				finite, e := m.modular.Modular()
				if e != nil {
					t.Fatal(e)
				}
				f := &s.frontend
				if !armed && f.active && f.executing && f.modularGCD.constructed == 6 && !f.image.classConstructed && !finite {
					if f.image.invocation != uint64(len(upolyBigPrimes)) {
						t.Fatal("fallback reached before every actual bad-prime image", f.image.invocation)
					}
					armed = true
					q.Budget().FailAfter(ordinal)
				} else if armed && f.modularGCD.constructed == 0 {
					afterFallback = true
					if f.B.size == 2 {
						one, e := z.IsOne(&m.values(&f.B)[0])
						serviceMust(t, e)
						same, e := z.Equal(&m.values(&f.B)[1], &p)
						serviceMust(t, e)
						published = one && same
					}
					return false // Stop the real next frontend iteration after GCD publication.
				}
				return true
			}
			defer func() {
				q.Budget().FailAfter(-1)
				serviceMust(t, s.CleanupRetained())
				serviceMust(t, fs.Close())
				serviceMust(t, m.reset(&input))
				m.freeBacking(&input)
				serviceMust(t, z.Del(&twoP))
				serviceMust(t, z.Del(&p2))
				serviceMust(t, z.Del(&p))
			}()
			_, body := s.Factor(m.stateView(&input), fs, DefaultUPolynomialFactorParams())
			q.Budget().FailAfter(-1)
			if !armed {
				t.Fatal("public frontend did not reach real all-primes-bad fallback", body)
			}
			f := &s.frontend
			caller := []*unumeral.Int{&p, &p2, &twoP}
			for i := range m.values(&input) {
				caller = append(caller, &m.values(&input)[i])
			}
			primeTrialPhysicalAuditOwners(t, q, s, []*UPolynomialFactors{fs}, caller...)
			if afterFallback {
				completed = true
				if body == nil || f.vectorNormalize.invocation == 0 || !published {
					t.Fatal("fallback normalization/known GCD 1+P*x publication missing", body, f.vectorNormalize.invocation, published)
				}
			} else if body == nil {
				t.Fatal("armed prefix did not stop or complete fallback")
			}
			firstBody, firstSource := f.bodyCause, f.sourceCause
			serviceMust(t, s.CleanupRetained())
			if f.bodyCause != firstBody || f.sourceCause != firstSource {
				t.Fatal("owning fallback cleanup rewrote source causes")
			}
		})
	}
	if !completed {
		t.Fatal("fallback reservation prefix horizon did not reach publication")
	}
}

// Source: src/math/polynomial/polynomial.cpp:2757-2768;
// src/util/mpzzp.h:167-185,195-208; src/util/mpz.h:468-485;
// src/util/mpz.cpp:915-988.
package polynomial

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestScalarDivDirectBigActualAllocationPrefixes(t *testing.T) { scalarDivDirectPrefixes(t, false) }
func TestScalarDivDirectCopiedRawChildRefusesWithoutMutation(t *testing.T) {
	scalarDivDirectPrefixes(t, true)
}
func scalarDivDirectPrefixes(t *testing.T, copiedProbe bool) {
	for route := 0; route < 3; route++ {
		success, retained := false, false
		var copiedGuardFailure string
		for prefix := int64(0); prefix < 256 && !success; prefix++ {
			m := polynomialPSCFullFixture(t)
			c := m.core
			_, e := c.monomials.MkVar()
			serviceMust(t, e)
			var coefficient, divisor, modulus util.Int
			digits := make([]uint32, 64)
			digits[0] = 3
			digits[63] = 1
			if route == 2 {
				prime := make([]uint32, 17)
				for i := range prime {
					prime[i] = ^uint32(0)
				}
				prime[16] = 511
				serviceMust(t, c.integer.SetLimbs(&modulus, prime))
				serviceMust(t, c.modular.SetZP(&modulus))
				digits = make([]uint32, 16)
				digits[0] = 3
				digits[15] = 0x80000000
			}
			serviceMust(t, c.integer.SetLimbs(&coefficient, digits))
			if route == 1 {
				serviceMust(t, c.integer.Neg(&coefficient))
			}
			dd := make([]uint32, 33)
			dd[32] = 1
			if route == 2 {
				dd = make([]uint32, 8)
				dd[0] = 1
				dd[7] = 0x80000000
			}
			serviceMust(t, c.integer.SetLimbs(&divisor, dd))
			term, e := c.monomials.MkVariable(0, 1)
			serviceMust(t, e)
			p, e := c.MkTerms([]util.Int{coefficient}, []Monomial{term})
			serviceMust(t, e)
			p = polynomialCaller(t, c, p)
			// A real old polynomial deletion warms its source owner ID recycle vector.
			seventeen := util.NewInt(17)
			warm, e := c.MkConstantTakeCopy(&seventeen)
			serviceMust(t, e)
			serviceMust(t, c.IncRef(warm))
			serviceMust(t, c.DecRef(warm))
			var receipt polynomialScalarDivInvocationReceipt
			var appendReceipt polynomialCheapAppendInvocationReceipt
			var factory polynomialFactoryInvocationReceipt
			c.integer.Budget().FailAfter(prefix)
			result, callErr := c.scalarDivWithParentReceipts(p, &divisor, &receipt, &appendReceipt, &factory)
			if callErr == nil && result != (Polynomial{}) {
				serviceMust(t, c.IncRef(result))
				serviceMust(t, c.transferFactoryResult(&factory, result))
			}
			s := &c.scalarDivision
			polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, result, s.returned}, []*util.Int{&coefficient, &divisor, &modulus, &s.a}, 0)
			if callErr != nil && !numericPhysicalReservationRefusal(callErr) {
				t.Fatal("literal refusal", route, prefix, callErr)
			}
			raw := &s.division
			if route == 2 {
				raw = &s.extended
			}
			if callErr != nil && raw.ActiveField && !raw.BodySucceeded {
				for i := uint8(0); i < raw.Count; i++ {
					if raw.Values[i].Cell != 0 {
						retained = true
					}
				}
			}
			if copiedProbe && route == 2 && retained && raw.ActiveField && !raw.BodySucceeded && copiedGuardFailure == "" {
				before := *raw
				copied := before
				if err := c.integer.FinishScalarRawSource(&copied); !errors.Is(err, ErrPolynomialOwner) {
					copiedGuardFailure = "copied child source finish accepted foreign frame"
				} else {
					if err := c.integer.CleanupScalarRaw(&copied, &s.numeric.hooks, true); !errors.Is(err, ErrPolynomialOwner) {
						copiedGuardFailure = "copied child owning cleanup accepted foreign frame"
					}
					if err := c.integer.BeginScalarRaw(&copied, &s.numeric.hooks, 6); !errors.Is(err, ErrPolynomialOwner) {
						copiedGuardFailure = "copied child begin accepted foreign frame"
					}
				}
				if !reflect.DeepEqual(before, *raw) || !reflect.DeepEqual(before, copied) {
					copiedGuardFailure = "copied child guard mutated actual or copied scope"
				}
				polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, result, s.returned}, []*util.Int{&coefficient, &divisor, &modulus, &s.a}, 0)
			}
			// Child RAW end-only DEL does not run on body refusal; older caller NUM
			// completes before true owning abort may drain those unjoined values.
			serviceMust(t, c.cleanupScalarDivInvocation(&receipt, false))
			polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, result, s.returned}, []*util.Int{&coefficient, &divisor, &modulus, &s.a}, 0)
			serviceMust(t, c.cleanupScalarDivInvocation(&receipt, true))
			serviceMust(t, c.cleanupFactoryInvocation(&factory, true))
			serviceMust(t, c.cheap.cleanupAppendInvocation(&appendReceipt, true))
			polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, result}, []*util.Int{&coefficient, &divisor, &modulus}, 0)
			c.integer.Budget().FailAfter(-1)

			if callErr == nil {
				success = true
				if route != 2 {
					expected := make([]uint32, 32)
					expected[31] = 1
					if route == 1 {
						expected[0] = 1
					}
					var want util.Int
					serviceMust(t, c.integer.SetLimbs(&want, expected))
					if route == 1 {
						serviceMust(t, c.integer.Neg(&want))
					}
					got, e := c.Coefficient(result, 0)
					serviceMust(t, e)
					cmp, e := c.integer.Compare(&got, &want)
					serviceMust(t, e)
					if cmp != 0 {
						t.Fatal("independent power-of-two Euclidean quotient", route, prefix)
					}
					serviceMust(t, c.integer.Del(&want))
				}
				t.Log("actual ScalarDiv direct first success", route, prefix)
			}
			if result != (Polynomial{}) {
				serviceMust(t, c.DecRef(result))
			}
			for _, v := range []*util.Int{&coefficient, &divisor, &modulus} {
				serviceMust(t, c.integer.Del(v))
			}
		}
		if copiedGuardFailure != "" {
			t.Fatal(copiedGuardFailure)
		}
		if !success {
			t.Fatal("actual direct first success missing", route)
		}
		if route == 2 && !retained {
			t.Fatal("actual ExtendedGCD RAW body refusal missing")
		}
	}
}

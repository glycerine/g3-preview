package polynomial

// Source: src/util/mpzzp.h:46-56;
// src/math/polynomial/upolynomial_factorization.cpp:676-843.

import (
	"fmt"
	"reflect"
	"testing"
)

func TestUPolynomialHenselActualLowerCopyFrontierDiagnostic(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("actual direct source allocator profile")
	}
	found := false
	for ordinal := int64(0); ordinal < 512; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m, prime, lift, donor, pf, lf, service, f, _, b := upolyHenselPhysicalFixture(t)
			cores := []*UPolynomialManager{m, prime, lift, donor}
			owners := []*UPolynomialFactors{pf, lf}
			b.FailAfter(ordinal)
			body := service.HenselLiftFactors(f, pf, 16, lf)
			upolyHenselAuditPhysical(t, cores, owners)
			failed := testFieldBySourceName(reflect.ValueOf(m.IntegerManager()).Elem(), "failedNullRef").Uint()
			modular := testFieldBySourceName(reflect.ValueOf(lift).Elem(), "modular").Elem()
			if testFieldBySourceName(testFieldBySourceName(modular, "lower"), "cell").Uint() == failed && testFieldBySourceName(testFieldBySourceName(modular, "p"), "cell").Uint() != failed && testFieldBySourceName(testFieldBySourceName(modular, "upper"), "cell").Uint() != failed {
				found = true
				t.Log("actual exact lower Copy frontier", ordinal, body)
			}
			b.FailAfter(-1)
			upolyHenselCallerSetZForDeadBounds(t, cores, owners)
		})
		if found {
			break
		}
	}
	if !found {
		t.Fatal("actual source lower Copy frontier absent")
	}
}

// Source: src/util/mpzzp.h:46-56;
// src/math/polynomial/upolynomial_factorization.cpp:676-843.
// Licensed under the MIT license. Source author: Leonardo de Moura.
// Source author: Dejan (t-dejanj), 2011-11-15.
package g3

import (
	"fmt"
	"github.com/glycerine/g3/golang/mpnum"
	"reflect"
	"testing"
)

func TestUPolynomialHenselActualLowerCopyFrontierDiagnostic(t *testing.T) {
	if !cooperationSourceDebug {
		t.Skip("actual direct source allocator profile")
	}
	found := false
	for ordinal := int64(0); ordinal < 512; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m, prime, lift, donor, pf, lf, service, f, _, b := upolyHenselPhysicalFixture(t)
			cores := []*mpnum.UPolynomialManager{m, prime, lift, donor}
			owners := []*mpnum.UPolynomialFactors{pf, lf}
			b.FailAfter(ordinal)
			body := service.HenselLiftFactors(f, pf, 16, lf)
			upolyHenselAuditPhysical(t, cores, owners)
			failed := reflect.ValueOf(m.IntegerManager()).Elem().FieldByName("failedNullRef").Uint()
			modular := reflect.ValueOf(lift).Elem().FieldByName("modular").Elem()
			if modular.FieldByName("lower").FieldByName("cell").Uint() == failed && modular.FieldByName("p").FieldByName("cell").Uint() != failed && modular.FieldByName("upper").FieldByName("cell").Uint() != failed {
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

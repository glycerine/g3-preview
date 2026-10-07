// Source: src/math/polynomial/upolynomial.cpp:2972-2989,3042-3118;
// src/util/mpzzp.h:254-257; src/util/mpz.cpp:1871-1929.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package polynomial

import (
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

// Invalid typed-owner guard fixture, not source reservation credit. The real
// guarded frontend member is used; constructed mirrors its source NUM frontier.
func TestUPolynomialFactorFrontendForeignPerfectSquareReceiver(t *testing.T) {
	z, s := preparationPhysicalFixture(t)
	m := s.manager
	foreign, e := util.NewIntegerModularManager(z.Integers())
	serviceMust(t, e)
	fs, e := m.NewFactors()
	serviceMust(t, e)
	defer func() {
		serviceMust(t, s.CleanupRetained())
		serviceMust(t, fs.Close())
		serviceMust(t, foreign.Close())
	}()
	q, e := s.beginFactorFrontend()
	serviceMust(t, e)
	q.result = fs
	q.constructed = 2
	defer func() { q.bodyReturned(); serviceMust(t, q.finish(nil)) }()
	context := numericIntegerModularManager_factorFrontendAllocationContext(m.modular, q)
	if context.factorFrontend == nil {
		t.Fatal("actual typed context absent")
	}
	old := q.perfectSquare
	destination, a := util.NewInt(77), util.NewInt(4)
	_, refusal := foreign.WithAllocationContext((context).utility(foreign.Integers())).IsPerfectSquare(&a, &destination)
	if refusal == nil || destination != util.NewInt(77) || !reflect.DeepEqual(old, q.perfectSquare) {
		t.Fatal("foreign modular receiver acquired actual frontend child", refusal, destination, q.perfectSquare)
	}
	yes, e := foreign.IsPerfectSquare(&a, &destination)
	serviceMust(t, e)
	if !yes || destination != util.NewInt(2) {
		t.Fatal("independent nil receiver route", yes, destination)
	}
}

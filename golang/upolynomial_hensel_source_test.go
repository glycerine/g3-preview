// Licensed under the MIT license.
package g3

import (
	"github.com/glycerine/g3/golang/mpnum"
	"testing"
)

func TestUPolynomialHenselPairExactSourceActions(t *testing.T) {
	m, _, b := upolySourceOwner(t)
	primes, e := mpnum.NewPrimeGenerator(b)
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if e := primes.Close(); e != nil {
			t.Error(e)
		}
	}()
	service, e := mpnum.NewUPolynomialFactorizationService(m, mpnum.UPolynomialFactorizationConfig{Primes: primes})
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if e := service.Close(); e != nil {
			t.Error(e)
		}
	}()
	// Over F5: A=x-1, B=x+1, U=2, V=-2 satisfy AU+BV=1.
	// C=x²-6, so (C-AB)/5=-1; the actual source remainder path
	// yields S=2,T=-2 and publishes A1=x+9, B1=x-9.
	u := upolySourceNew(t, m, []int32{2})
	a := upolySourceNew(t, m, []int32{-1, 1})
	v := upolySourceNew(t, m, []int32{-2})
	bb := upolySourceNew(t, m, []int32{1, 1})
	c := upolySourceNew(t, m, []int32{-6, 0, 1})
	outA := upolySourceNew(t, m, nil)
	outB := upolySourceNew(t, m, nil)
	five := mpnum.NewInt(5)
	if e = service.HenselLiftPair(&five, &five, &five, mustUPolyView(t, m, u), mustUPolyView(t, m, a), mustUPolyView(t, m, v), mustUPolyView(t, m, bb), mustUPolyView(t, m, c), outA, outB); e != nil {
		t.Fatal(e)
	}
	ea := upolySourceNew(t, m, []int32{9, 1})
	eb := upolySourceNew(t, m, []int32{-9, 1})
	if same, e := m.Equal(outA, ea); e != nil || !same {
		t.Fatal("first lifted prefix", same, e)
	}
	if same, e := m.Equal(outB, eb); e != nil || !same {
		t.Fatal("second lifted prefix", same, e)
	}
}
func TestUPolynomialHenselMultipleFactorsAndActualModulus(t *testing.T) {
	m, _, b := upolySourceOwner(t)
	primeCore, e := mpnum.NewRelatedUPolynomialManager(m)
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if e := primeCore.Close(); e != nil {
			t.Error(e)
		}
	}()
	liftedCore, e := mpnum.NewRelatedUPolynomialManager(m)
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if e := liftedCore.Close(); e != nil {
			t.Error(e)
		}
	}()
	if e = primeCore.SetZPUint64(5); e != nil {
		t.Fatal(e)
	}
	primes, e := mpnum.NewPrimeGenerator(b)
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if e := primes.Close(); e != nil {
			t.Error(e)
		}
	}()
	service, e := mpnum.NewUPolynomialFactorizationService(m, mpnum.UPolynomialFactorizationConfig{Primes: primes})
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if e := service.Close(); e != nil {
			t.Error(e)
		}
	}()
	pf, e := primeCore.NewFactors()
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if e := pf.Close(); e != nil {
			t.Error(e)
		}
	}()
	lf, e := liftedCore.NewFactors()
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if e := lf.Close(); e != nil {
			t.Error(e)
		}
	}()
	// (x-1)(x+1)(x+2) is square-free mod5 and has three actual factors.
	f := upolySourceNew(t, m, []int32{-2, -1, 2, 1})
	for _, coeff := range [][]int32{{-1, 1}, {1, 1}, {2, 1}} {
		p := upolySourceNew(t, m, coeff)
		if e = pf.PushCopy(mustUPolyView(t, m, p), 1); e != nil {
			t.Fatal(e)
		}
	}
	if e = service.HenselLiftFactors(mustUPolyView(t, m, f), pf, 3, lf); e != nil {
		t.Fatal(e)
	}
	if n, e := lf.DistinctFactors(); e != nil || n != 3 {
		t.Fatal(n, e)
	}
	nm, e := liftedCore.NumeralManager()
	if e != nil {
		t.Fatal(e)
	}
	var modulus mpnum.Int
	if e = nm.Modulus(&modulus); e != nil {
		t.Fatal(e)
	}
	if n, e := m.IntegerManager().Uint64(&modulus); e != nil || n != 625 {
		t.Fatal("quadratic doubling to p^4", n, e)
	}
	if e = m.IntegerManager().Del(&modulus); e != nil {
		t.Fatal(e)
	}
	out, e := liftedCore.NewPolynomial()
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if e := liftedCore.DelPolynomial(&out); e != nil {
			t.Error(e)
		}
	}()
	if e = lf.Multiply(out); e != nil {
		t.Fatal(e)
	}
	// Inputs are unchanged integers: all roots fit the final centered modulus.
	view := mustUPolyView(t, liftedCore, out)
	copy := upolySourceNew(t, m, nil)
	if e = m.SetView(copy, view); e != nil {
		t.Fatal(e)
	}
	if same, e := m.Equal(copy, f); e != nil || !same {
		t.Fatal(same, e)
	}
}

func TestUPolynomialHenselIndependentStoresActualSourceTransfers(t *testing.T) {
	m, q, b, limit, cooperation := upolySourceOwnerServices(t)
	config := mpnum.UPolynomialConfig{Integer: q.Integers(), Checkpoint: limit.inc, Cooperation: func() error { return cooperation.Cooperate("upolynomial") }}
	prime, e := mpnum.NewUPolynomialManager(config)
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if e := prime.Close(); e != nil {
			t.Error(e)
		}
	}()
	lift, e := mpnum.NewUPolynomialManager(config)
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if e := lift.Close(); e != nil {
			t.Error(e)
		}
	}()
	store, e := m.CoefficientStore()
	if e != nil {
		t.Fatal(e)
	}
	if e = prime.BorrowCoefficientStore(store); e != nil {
		t.Fatal(e)
	}
	if e = prime.SetZPUint64(5); e != nil {
		t.Fatal(e)
	}
	pgen, e := mpnum.NewPrimeGenerator(b)
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if e := pgen.Close(); e != nil {
			t.Error(e)
		}
	}()
	service, e := mpnum.NewUPolynomialFactorizationService(m, mpnum.UPolynomialFactorizationConfig{Primes: pgen})
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if e := service.Close(); e != nil {
			t.Error(e)
		}
	}()
	pf, e := prime.NewFactors()
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if e := pf.Close(); e != nil {
			t.Error(e)
		}
	}()
	lf, e := lift.NewFactors()
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if e := lf.Close(); e != nil {
			t.Error(e)
		}
	}()
	f := upolySourceNew(t, m, []int32{-2, -1, 2, 1})
	for _, cs := range [][]int32{{-1, 1}, {1, 1}, {2, 1}} {
		p := upolySourceNew(t, m, cs)
		if e = pf.PushCopy(mustUPolyView(t, m, p), 1); e != nil {
			t.Fatal(e)
		}
	}
	if e = service.HenselLiftFactors(mustUPolyView(t, m, f), pf, 3, lf); e != nil {
		t.Fatal("actual independent store Hensel", e)
	}
	if n, e := lf.DistinctFactors(); e != nil || n != 3 {
		t.Fatal(n, e)
	}
}

func TestUPolynomialHenselActualTaggedDiagnostics(t *testing.T) {
	m, _, b := upolySourceOwner(t)
	prime, e := mpnum.NewRelatedUPolynomialManager(m)
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if e := prime.Close(); e != nil {
			t.Error(e)
		}
	}()
	lift, e := mpnum.NewRelatedUPolynomialManager(m)
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if e := lift.Close(); e != nil {
			t.Error(e)
		}
	}()
	if e = prime.SetZPUint64(5); e != nil {
		t.Fatal(e)
	}
	pgen, e := mpnum.NewPrimeGenerator(b)
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if e := pgen.Close(); e != nil {
			t.Error(e)
		}
	}()
	service, e := mpnum.NewUPolynomialFactorizationService(m, mpnum.UPolynomialFactorizationConfig{Primes: pgen, FactorizationDebugTag: true})
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if e := service.Close(); e != nil {
			t.Error(e)
		}
	}()
	pf, e := prime.NewFactors()
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if e := pf.Close(); e != nil {
			t.Error(e)
		}
	}()
	lf, e := lift.NewFactors()
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if e := lf.Close(); e != nil {
			t.Error(e)
		}
	}()
	f := upolySourceNew(t, m, []int32{-6, 0, 1})
	for _, cs := range [][]int32{{-1, 1}, {1, 1}} {
		p := upolySourceNew(t, m, cs)
		if e = pf.PushCopy(mustUPolyView(t, m, p), 1); e != nil {
			t.Fatal(e)
		}
	}
	if e = service.HenselLiftFactors(mustUPolyView(t, m, f), pf, 16, lf); e != nil {
		t.Fatal("actual conditional helper arithmetic", e)
	}
	var modulus mpnum.Int
	nm, e := lift.NumeralManager()
	if e != nil {
		t.Fatal(e)
	}
	if e = nm.Modulus(&modulus); e != nil {
		t.Fatal(e)
	}
	if n, e := m.IntegerManager().Uint64(&modulus); e != nil || n != 152587890625 {
		t.Fatal(n, e)
	}
	if e = m.IntegerManager().Del(&modulus); e != nil {
		t.Fatal(e)
	}
}

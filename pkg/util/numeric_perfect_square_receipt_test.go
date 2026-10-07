// Source: src/util/mpz.cpp:1871-1929; src/util/mpzzp.h.
package util

import "testing"

func TestPerfectSquareIssuedReceipt(t *testing.T) {
	m, z, _ := modularManager(t, 0)
	a, out := NewInt(49), Int{}
	var r integerPerfectSquareInvocationReceipt
	exact, e := m.IsPerfectSquareWithReceipt(&a, &out, &r)
	serviceMust(t, e)
	if !exact || !z.MatchesPerfectSquareHistory(&r) {
		t.Fatal("missing actual square invocation", exact)
	}
	v, e := z.Int64(&out)
	serviceMust(t, e)
	if v != 7 {
		t.Fatal("root", v)
	}
	active, end, stopped := r.State(z)
	if active || !end || stopped {
		t.Fatal("completed source scope", active, end, stopped)
	}
	saved := r
	for _, n := range []int32{-1, 0, 1} {
		a = NewInt(n)
		_, e = m.IsPerfectSquareWithReceipt(&a, &out, &r)
		serviceMust(t, e)
		if r != saved {
			t.Fatal("erased RAW constructor changed receipt", n)
		}
	}
	serviceMust(t, z.Del(&out))
}

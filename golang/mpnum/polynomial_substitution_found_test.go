// Source: src/math/polynomial/polynomial.cpp:3185,6253-6298; src/util/vector.h.
// Licensed under the MIT license.
// Author: Leonardo de Moura.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestPolynomialFoundVariablesGrowthAndRetainedBits(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b})
	serviceMust(t, e)
	var v polynomialFoundVariables
	v.runtime = r
	serviceMust(t, v.reserve(3))
	serviceMust(t, v.put(1, true))
	serviceMust(t, v.reserve(9))
	n, e := v.size()
	serviceMust(t, e)
	if n != 9 {
		t.Fatal(n)
	}
	for i := uint32(0); i < 9; i++ {
		a, e := v.at(i)
		serviceMust(t, e)
		if a != (i == 1) {
			t.Fatal(i, a)
		}
	}
	serviceMust(t, v.reserve(2))
	n, e = v.size()
	serviceMust(t, e)
	if n != 9 {
		t.Fatal("source reserve does not shrink", n)
	}
	serviceMust(t, v.sourceFinalize())
	serviceMust(t, v.close())
	serviceMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
func TestPolynomialFoundVariablesEveryGrowthOldSize(t *testing.T) {
	success := false
	for fault := int64(0); fault < 32 && !success; fault++ {
		b := arena.NewBudget(1 << 20)
		r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b})
		serviceMust(t, e)
		var v polynomialFoundVariables
		v.runtime = r
		b.FailAfter(fault)
		e = v.reserve(19)
		n, ne := v.size()
		serviceMust(t, ne)
		if e != nil {
			if !numericPhysicalReservationRefusal(e) || n != 0 {
				t.Fatal(fault, e, n)
			}
		} else {
			if n != 19 {
				t.Fatal(n)
			}
			success = true
		}
		st := r.Stats()
		if st.TotalBytes != b.Used() {
			t.Fatal("named sole raw/pending Runtime owner", st, b.Used())
		}
		if e != nil && v.raw == (memory.Block{}) && v.pending == (memory.Block{}) && st.LiveBlocks != 0 {
			t.Fatal("unnamed actual root")
		}
		b.FailAfter(-1)
		serviceMust(t, v.close())
		serviceMust(t, r.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	}
	if !success {
		t.Fatal("no complete mathematical success")
	}
}

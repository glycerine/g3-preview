// Licensed under the MIT license.
package polynomial

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func upolyDivisionValues(t *testing.T, m *UPolynomialManager, p UPolynomial) []int64 {
	t.Helper()
	n, e := m.Size(p)
	if e != nil {
		t.Fatal(e)
	}
	out := make([]int64, n)
	for i := uint32(0); i < n; i++ {
		var v util.Int
		if e = m.ReadCoefficient(p, i, &v); e != nil {
			t.Fatal(e)
		}
		out[i], e = m.IntegerManager().Int64(&v)
		if e != nil {
			t.Fatal(e)
		}
		if e = m.IntegerManager().Del(&v); e != nil {
			t.Fatal(e)
		}
	}
	return out
}
func TestUPolynomialExactDivisionSourceThirtyENSURE(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	// Expanded source factors are an explicit multivariate-conversion input
	// adapter. Each case executes all three actual dense ENSURE routes.
	cases := []struct {
		a, b, q []int32
		ok      bool
	}{
		{[]int32{-6, 11, -6, 1}, []int32{2, -3, 1}, []int32{-3, 1}, true},
		{[]int32{-12, 22, -12, 2}, []int32{2, -3, 1}, []int32{-6, 2}, true},
		{[]int32{-12, 22, -12, 2}, []int32{-8, 14, -7, 1}, nil, false},
		{[]int32{-3, 1}, []int32{-1, 1}, nil, false},
		{nil, []int32{-8, 14, -7, 1}, nil, true},
		{[]int32{-8, 14, -7, 1}, nil, nil, false},
		{nil, nil, nil, false},
		{[]int32{-2, 2}, []int32{-1, 1}, []int32{2}, true},
		{[]int32{-2, 2}, []int32{-4, 4}, nil, false},
		{[]int32{-4, 6}, []int32{2}, []int32{-2, 3}, true},
	}
	assertions := 0
	for i, c := range cases {
		a := upolySourceNew(t, m, c.a)
		b := upolySourceNew(t, m, c.b)
		q := upolySourceNew(t, m, nil)
		expected := upolySourceNew(t, m, c.q)
		got, e := m.ExactDiv(a, b, q)
		if e != nil {
			t.Fatal(i, e)
		}
		assertions++
		if got != c.ok {
			t.Fatal(i, got, c.ok)
		}
		divides, e := m.Divides(a, b)
		if e != nil {
			t.Fatal(i, e)
		}
		assertions++
		if divides != c.ok {
			t.Fatal(i, divides, c.ok)
		}
		assertions++
		if c.ok {
			equal, e := m.Equal(q, expected)
			if e != nil || !equal {
				t.Fatal(i, e, equal)
			}
		}
	}
	if assertions != 30 {
		t.Fatal(assertions)
	}
}
func TestUPolynomialFieldDivisionIndependentFinite(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	for _, prime := range []uint64{5, 7, 13} {
		if e := m.SetZPUint64(prime); e != nil {
			t.Fatal(e)
		}
		// Independent scalar convolution: A = B*Q + R. Centered manager
		// residues are compared modulo p, without using its polynomial Mul/Add.
		for c := int32(1); c <= 3; c++ {
			bv := []int32{1, c, 1}
			qv := []int32{2, 1}
			rv := []int32{c}
			av := make([]int32, 4)
			for i, x := range bv {
				for j, y := range qv {
					av[i+j] += x * y
				}
			}
			av[0] += c
			a := upolySourceNew(t, m, av)
			b := upolySourceNew(t, m, bv)
			q := upolySourceNew(t, m, nil)
			r := upolySourceNew(t, m, nil)
			d := uint32(99)
			if e := m.DivRem(a, b, q, r, &d); e != nil {
				t.Fatal(prime, c, e)
			}
			if d != 0 {
				t.Fatal("field exponent", d)
			}
			for _, f := range []struct {
				p    UPolynomial
				want []int32
			}{{q, qv}, {r, rv}} {
				got := upolyDivisionValues(t, m, f.p)
				if len(got) != len(f.want) {
					t.Fatal(prime, c, got, f.want)
				}
				for i, v := range got {
					if (v-int64(f.want[i]))%int64(prime) != 0 {
						t.Fatal(prime, c, got, f.want)
					}
				}
			}
		}
	}
	// All retained coefficients must return to Z before cleanup under a new
	// modulus could fail source debug normalized assertions.
	if e := m.SetZ(); e != nil {
		t.Fatal(e)
	}
}

func TestUPolynomialDivisionPublishedPrefixes(t *testing.T) {
	m, _, _, limit, _ := upolySourceOwnerServices(t)
	a := upolySourceNew(t, m, []int32{1, 1, 1})
	b := upolySourceNew(t, m, []int32{1, 2})
	q := upolySourceNew(t, m, []int32{91})
	r := upolySourceNew(t, m, []int32{92})
	// Source wrapper swaps only after core success. Cancellation occurs after
	// scratch setup but before the first quotient arithmetic/counter increment.
	limit.Cancel()
	d := uint32(77)
	if e := m.DivRem(a, b, q, r, &d); e != ErrUPolynomialCanceled {
		t.Fatal(e)
	}
	if d != 0 {
		t.Fatal(d)
	}
	if got := upolyDivisionValues(t, m, q); len(got) != 1 || got[0] != 91 {
		t.Fatal(got)
	}
	if got := upolyDivisionValues(t, m, r); len(got) != 1 || got[0] != 92 {
		t.Fatal(got)
	}
	limit.ResetCancel()
	// ExactDiv initial rejection leaves old q; later rejection resets q.
	noInitial := upolySourceNew(t, m, []int32{1, 1})
	large := upolySourceNew(t, m, []int32{1, 1, 1})
	if ok, e := m.ExactDiv(noInitial, large, q); e != nil || ok {
		t.Fatal(ok, e)
	}
	if got := upolyDivisionValues(t, m, q); len(got) != 1 || got[0] != 91 {
		t.Fatal(got)
	}
	if ok, e := m.ExactDiv(a, b, q); e != nil || ok {
		t.Fatal(ok, e)
	} // LC 2 does not divide LC 1: still initial.
	if got := upolyDivisionValues(t, m, q); len(got) != 1 || got[0] != 91 {
		t.Fatal(got)
	}
	unit := upolySourceNew(t, m, []int32{1, 1})
	if ok, e := m.ExactDiv(a, unit, q); e != nil || ok {
		t.Fatal(ok, e)
	}
	if got := upolyDivisionValues(t, m, q); len(got) != 0 {
		t.Fatal("later reject", got)
	}
	constant := upolySourceNew(t, m, []int32{2})
	if e := m.DivRem(a, constant, q, r, &d); e != nil {
		t.Fatal(e)
	}
	// Source non-field constant division copies A, despite the equation in the
	// header comment requiring division by the constant. Freeze actual result.
	if got := upolyDivisionValues(t, m, q); len(got) != 3 || got[0] != 1 || got[1] != 1 || got[2] != 1 {
		t.Fatal(got)
	}
	if got := upolyDivisionValues(t, m, r); len(got) != 0 {
		t.Fatal(got)
	}
}

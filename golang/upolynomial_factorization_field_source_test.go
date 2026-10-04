// Licensed under the MIT license.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/mpnum"
	"testing"
)

// Independent finite-field factor counts and repeated characteristic powers;
// this is additional coverage, not registered upolynomial.cpp ENSURE claims.
func TestUPolynomialFactorFiniteFieldSquareFreeAndBerlekamp(t *testing.T) {
	for _, tc := range []struct {
		prime        uint64
		coefficients []int32
		factors      uint32
		multiplicity uint32
	}{
		{2, []int32{1, 0, 0, 0, 1}, 1, 4},
		{3, []int32{1, 0, 1}, 1, 1},
		{5, []int32{-1, 0, 1}, 2, 1},
		{5, []int32{-1, 0, 0, 0, 1}, 4, 1},
		{3, []int32{1, 0, 0, 0, 0, 0, 1}, 1, 3},
	} {
		m, _, _ := upolySourceOwner(t)
		if e := m.SetZPUint64(tc.prime); e != nil {
			t.Fatal(e)
		}
		p := upolySourceNew(t, m, tc.coefficients)
		// Numeric input constructors normalize through the actual modular manager.
		fs, e := m.NewFactors()
		if e != nil {
			t.Fatal(e)
		}
		_, e = m.FactorZP(mustUPolyView(t, m, p), fs)
		if e != nil {
			t.Fatal(tc, e)
		}
		n, e := fs.DistinctFactors()
		if e != nil || n != tc.factors {
			t.Fatal(tc, n, e)
		}
		for i := uint32(0); i < n; i++ {
			k, e := fs.Multiplicity(i)
			if e != nil || k != tc.multiplicity {
				t.Fatal(tc, i, k, e)
			}
		}
		out := upolySourceNew(t, m, nil)
		if e = fs.Multiply(out); e != nil {
			t.Fatal(e)
		}
		same, e := m.Equal(p, out)
		if e != nil || !same {
			t.Fatal("finite reconstruction", tc, same, e)
		}
		if e = fs.Close(); e != nil {
			t.Fatal(e)
		}
	}
}
func TestUPolynomialFactorBerlekampAppendAndIrreducibleReturn(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	if e := m.SetZPUint64(5); e != nil {
		t.Fatal(e)
	}
	fs, e := m.NewFactors()
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if e := fs.Close(); e != nil {
			t.Error(e)
		}
	}()
	initial := upolySourceNew(t, m, []int32{1, 1})
	if e = fs.PushCopy(mustUPolyView(t, m, initial), 2); e != nil {
		t.Fatal(e)
	}
	p := upolySourceNew(t, m, []int32{-1, 0, 0, 0, 1})
	split, e := m.FactorZPSquareFreeBerlekamp(mustUPolyView(t, m, p), fs, false)
	if e != nil || !split {
		t.Fatal(split, e)
	}
	n, e := fs.DistinctFactors()
	if e != nil || n != 5 {
		t.Fatal(n, e)
	}
	if k, e := fs.Multiplicity(0); e != nil || k != 2 {
		t.Fatal("preexisting prefix", k, e)
	}
	if e = m.SetZPUint64(3); e != nil {
		t.Fatal(e)
	}
	irreducible := upolySourceNew(t, m, []int32{1, 0, 1})
	split, e = m.FactorZPSquareFree(mustUPolyView(t, m, irreducible), fs)
	if e != nil || split {
		t.Fatal(split, e)
	}
	n, e = fs.DistinctFactors()
	if e != nil || n != 6 {
		t.Fatal(n, e)
	}
}

func TestUPolynomialFactorBerlekampEmptyReleaseAndDebug(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	if e := m.SetZPUint64(5); e != nil {
		t.Fatal(e)
	}
	fs, e := m.NewFactors()
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if e := fs.Close(); e != nil {
			t.Error(e)
		}
	}()
	empty := upolySourceNew(t, m, nil)
	yes, err := m.FactorZPSquareFree(mustUPolyView(t, m, empty), fs)
	if astSourceDebug {
		if !errors.Is(err, mpnum.ErrUPolynomialAssertion) {
			t.Fatal("actual source entry assertion", yes, err)
		}
		if n, e := fs.DistinctFactors(); e != nil || n != 0 {
			t.Fatal("preassertion prefix", n, e)
		}
	} else {
		if err != nil || !yes {
			t.Fatal("finite source release empty continuation", yes, err)
		}
		if n, e := fs.DistinctFactors(); e != nil || n != 1 {
			t.Fatal("source appends empty factor before final disabled assertion", n, e)
		}
	}
}

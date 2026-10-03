// Copyright (c) 2011 Microsoft Corporation. Licensed under the MIT license.
package g3

import (
	"github.com/glycerine/g3/golang/mpnum"
	"testing"
)

func TestUPolynomialFactorDegreeSetEmptyConstructorAndIntersections(t *testing.T) {
	m, _, b := upolySourceOwner(t)
	empty, e := m.NewFactors()
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if e := empty.Close(); e != nil {
			t.Error(e)
		}
	}()
	ds, e := mpnum.NewUPolynomialDegreeSet(empty)
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if e := ds.Close(); e != nil {
			t.Error(e)
		}
	}()
	if n, e := ds.MaxDegree(); e != nil || n != 0 {
		t.Fatal(n, e)
	}
	if yes, e := ds.InSet(0); e != nil || !yes {
		t.Fatal(yes, e)
	}
	if ds.Stats().WordBytes != 8 {
		t.Fatal("source push defaultcapacity2words", ds.Stats())
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
	p := upolySourceNew(t, m, []int32{1, 0, 1})
	if e = fs.PushCopy(mustUPolyView(t, m, p), 2); e != nil {
		t.Fatal(e)
	}
	other, e := mpnum.NewUPolynomialDegreeSet(fs)
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if e := other.Close(); e != nil {
			t.Error(e)
		}
	}()
	for i := uint32(0); i <= 4; i++ {
		yes, e := other.InSet(i)
		if e != nil || yes != (i%2 == 0) {
			t.Fatal(i, yes, e)
		}
	}
	// Actual first swap of initialized empty {0} with the first factor image.
	b.FailAfter(0)
	if e = ds.Swap(other); e != nil {
		t.Fatal(e)
	}
	if e = ds.Intersect(other); e != nil {
		t.Fatal(e)
	}
	b.FailAfter(-1)
	if n, e := ds.MaxDegree(); e != nil || n != 4 {
		t.Fatal("source &= retains left bit width", n, e)
	}
	for i := uint32(0); i <= 4; i++ {
		yes, e := ds.InSet(i)
		if e != nil || yes != (i == 0) {
			t.Fatal(i, yes, e)
		}
	}
	if trivial, e := ds.IsTrivial(); e != nil || !trivial {
		t.Fatal(trivial, e)
	}
}

func TestUPolynomialFactorCombinationExactOrderRemovalAndProducts(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	fs, e := m.NewFactors()
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if e := fs.Close(); e != nil {
			t.Error(e)
		}
	}()
	for c := int32(1); c <= 4; c++ {
		p := upolySourceNew(t, m, []int32{c, 1})
		if e = fs.PushCopy(mustUPolyView(t, m, p), 1); e != nil {
			t.Fatal(e)
		}
	}
	ds, e := mpnum.NewUPolynomialDegreeSet(fs)
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if e := ds.Close(); e != nil {
			t.Error(e)
		}
	}()
	it, e := mpnum.NewUPolynomialFactorCombination(fs, ds)
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if e := it.Close(); e != nil {
			t.Error(e)
		}
	}()
	left := upolySourceNew(t, m, nil)
	right := upolySourceNew(t, m, nil)
	expected := [][]int32{{1, 1}, {2, 1}, {3, 1}, {4, 1}, {2, 3, 1}, {3, 4, 1}, {4, 5, 1}, {6, 5, 1}, {8, 6, 1}, {12, 7, 1}}
	for i, coeff := range expected {
		yes, e := it.Next(false)
		if e != nil || !yes {
			t.Fatal(i, yes, e)
		}
		if e = it.Left(left); e != nil {
			t.Fatal(e)
		}
		want := upolySourceNew(t, m, coeff)
		if same, e := m.Equal(left, want); e != nil || !same {
			t.Fatal(i, same, e)
		}
		if e = it.Right(right); e != nil {
			t.Fatal(e)
		}
		product := upolySourceNew(t, m, nil)
		if e = m.Mul(left, right, product); e != nil {
			t.Fatal(e)
		}
		all := upolySourceNew(t, m, []int32{24, 50, 35, 10, 1})
		if same, e := m.Equal(product, all); e != nil || !same {
			t.Fatal(i, same, e)
		}
	}
	if yes, e := it.Next(false); e != nil || yes {
		t.Fatal(yes, e)
	}
	removal, e := mpnum.NewUPolynomialFactorCombination(fs, ds)
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if e := removal.Close(); e != nil {
			t.Error(e)
		}
	}()
	if yes, e := removal.Next(false); e != nil || !yes {
		t.Fatal(yes, e)
	}
	if yes, e := removal.Next(true); e != nil || !yes {
		t.Fatal(yes, e)
	}
	if n, e := removal.LeftSize(); e != nil || n != 1 {
		t.Fatal(n, e)
	}
	if n, e := removal.RightSize(); e != nil || n != 2 {
		t.Fatal(n, e)
	}
	if e = removal.Left(left); e != nil {
		t.Fatal(e)
	}
	want := upolySourceNew(t, m, []int32{2, 1})
	if same, e := m.Equal(left, want); e != nil || !same {
		t.Fatal(same, e)
	}
}

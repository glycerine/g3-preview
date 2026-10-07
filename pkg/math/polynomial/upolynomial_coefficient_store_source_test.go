// Licensed under the MIT license.
// Tests FIRST: real sibling cores share physical storage, not arithmetic modes.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestUPolynomialRelatedCoreActualCrossCoreReadAndMode(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	p := upolySourceNew(t, m, []int32{-8, 9, 1})
	v := mustUPolyView(t, m, p)
	child, e := NewRelatedUPolynomialManager(m)
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if e := child.Close(); e != nil {
			t.Error(e)
		}
	}()
	if e = child.SetZPUint64(7); e != nil {
		t.Fatal(e)
	}
	out, e := child.NewPolynomial()
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if e := child.DelPolynomial(&out); e != nil {
			t.Error(e)
		}
	}()
	if e = child.SetView(out, v); e != nil {
		t.Fatal("source cross-core borrowed coefficient copy", e)
	}
	expected, e := child.NewPolynomial()
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if e := child.DelPolynomial(&expected); e != nil {
			t.Error(e)
		}
	}()
	if e = child.SetInts(expected, []util.Int{util.NewInt(-1), util.NewInt(2), util.NewInt(1)}); e != nil {
		t.Fatal(e)
	}
	same, e := child.Equal(out, expected)
	if e != nil || !same {
		t.Fatal(same, e)
	}
	modular, e := m.Modular()
	if e != nil || modular {
		t.Fatal("parent mode changed", modular, e)
	}
	store, e := m.CoefficientStore()
	if e != nil {
		t.Fatal(e)
	}
	other, e := child.CoefficientStore()
	if e != nil || other != store {
		t.Fatal("actual shared cold store", e)
	}
	stats, e := store.Stats()
	if e != nil || stats.Owners != 2 || stats.LiveBackings < 3 {
		t.Fatal(stats, e)
	}
}
func TestUPolynomialRelatedCoreStoreLifetimeAndScalarBorrow(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	child, e := NewRelatedUPolynomialManager(m)
	if e != nil {
		t.Fatal(e)
	}
	p, e := child.NewPolynomial()
	if e != nil {
		t.Fatal(e)
	}
	if e = child.SetInts(p, []util.Int{util.NewInt(1), util.NewInt(2)}); e != nil {
		t.Fatal(e)
	}
	v, e := child.View(p, 0, 2)
	if e != nil {
		t.Fatal(e)
	}
	store, e := m.CoefficientStore()
	if e != nil {
		t.Fatal(e)
	}
	if e = m.Close(); e != nil {
		t.Fatal("parent source buffers only", e)
	}
	same, e := child.EqualViews(v, v)
	if e != nil || !same {
		t.Fatal("child shared backing survived parent teardown", same, e)
	}
	stats, e := store.Stats()
	if e != nil || stats.Owners != 1 {
		t.Fatal(stats, e)
	}
	if e = child.DelPolynomial(&p); e != nil {
		t.Fatal(e)
	}
	if _, e = child.EqualViews(v, v); !errors.Is(e, ErrUPolynomialOwner) {
		t.Fatal("actual released physical borrow", e)
	}
	if e = child.Close(); e != nil {
		t.Fatal(e)
	}
	if _, e = store.Stats(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("last actual owner releases store", e)
	}
}

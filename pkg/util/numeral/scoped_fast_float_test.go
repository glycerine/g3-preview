// Concrete owner/domain regressions.
package numeral

import (
	"errors"
	"math/big"
	"testing"
)

func TestFastFloatOwnerProvenance(t *testing.T) {
	m := fastManager(t, 3)
	other := fastManager(t, 3)
	a, e := NewScopedFastFloat(m)
	serviceMust(t, e)
	defer a.Close()
	b, e := NewScopedFastFloat(other)
	serviceMust(t, e)
	defer b.Close()
	serviceMust(t, a.SetInt32(-7))
	serviceMust(t, b.SetInt32(3))
	before := m.config.Storage.Descriptors.Budget.Used()
	if !errors.Is(a.Assign(b), ErrScopedOwner) {
		t.Fatal("foreign assignment accepted")
	}
	if _, e = a.Add(b); !errors.Is(e, ErrScopedOwner) {
		t.Fatal("foreign expression accepted", e)
	}
	if m.config.Storage.Descriptors.Budget.Used() != before {
		t.Fatal("provenance checked after numeric work")
	}
	shallow := *a
	if _, e = shallow.Get(); !errors.Is(e, ErrScopedCopy) {
		t.Fatal("copied cold owner", e)
	}
	fastAudit(t, m, &a.num)
}
func TestFastFloatNeighborAndThinRoutes(t *testing.T) {
	m := fastManager(t, 5)
	var a, b FastFloat
	defer m.Del(&a)
	defer m.Del(&b)
	serviceMust(t, m.SetInt32(&a, -7))
	serviceMust(t, m.Copy(&b, &a))
	serviceMust(t, m.Abs(&b))
	serviceMust(t, m.Swap(&a, &b))
	serviceMust(t, m.Neg(&a))
	ok, e := m.Equal(&a, &b)
	fastEnsure(t, ok, e)
	serviceMust(t, m.Inc(&a))
	serviceMust(t, m.Dec(&a))
	ok, e = m.Equal(&a, &b)
	fastEnsure(t, ok, e)
	serviceMust(t, m.SetInt32(&a, 2))
	serviceMust(t, m.Inv(&a, &b))
	if fastExact(m, &b).Cmp(big.NewRat(1, 2)) != 0 {
		t.Fatal("inverse")
	}
	serviceMust(t, m.InvInPlace(&b))
	ok, e = m.Equal(&a, &b)
	fastEnsure(t, ok, e)
	serviceMust(t, m.SetFraction32(&a, -7, 3))
	serviceMust(t, m.Floor(&a, &b))
	if fastExact(m, &b).Cmp(big.NewRat(-3, 1)) != 0 {
		t.Fatal("floor")
	}
	serviceMust(t, m.Ceil(&a, &b))
	if fastExact(m, &b).Cmp(big.NewRat(-2, 1)) != 0 {
		t.Fatal("ceil")
	}
	serviceMust(t, m.SetMax(&a))
	e = m.Next(&a)
	if !errors.Is(e, ErrFastFloatOverflow) || a.exponent != 2147483647 || !m.minSig(&a) {
		t.Fatal("source next overflow prefix", e, a)
	}
	fastAudit(t, m, &a, &b)
	serviceMust(t, m.SetInt32(&a, 0))
	serviceMust(t, m.SetExponent(&a, 123))
	if a.exponent != 0 {
		t.Fatal("zero exponent noop")
	}
	if e = m.ToInteger(&b, nil, nil); e == nil {
		t.Fatal("invalid integer owner")
	}
}

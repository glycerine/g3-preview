package polynomial

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestPolynomialNewtonDisabledOuterAssertionSkipsPolynomialReads(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(fmt.Sprint(enabled), func(t *testing.T) {
			m := polynomialCellFixture(t)
			n, e := NewPolynomialNewtonInterpolator(m)
			serviceMust(t, e)
			defer func() { serviceMust(t, n.Close()) }()
			polynomialThrowAssertionFixture(t)
			if polynomialSourceDebug {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(enabled))
			}
			unread := Polynomial{owner: m.origin, slot: ^uint32(0), generation: 1}
			e = n.assertPolynomialNormalized(unread, 2863, "pm.is_p_normalized(temp)")
			if polynomialSourceDebug && enabled {
				if e == nil {
					t.Fatal("enabled assertion omitted the polynomial read")
				}
			} else {
				serviceMust(t, e)
			}
		})
	}
}

func TestPolynomialNewtonEntryAssertionActions(t *testing.T) {
	for _, action := range []string{"T", "C"} {
		t.Run("integer-"+action, func(t *testing.T) {
			m := polynomialCellFixture(t)
			n, e := NewPolynomialNewtonInterpolator(m)
			serviceMust(t, e)
			polynomialAssertionActionsFixture(t, strings.Repeat(action, 16))
			zero := unumeral.NewInt(0)
			e = n.Add(&zero, m.unit)
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(e, ErrPolynomialAssertion) || len(n.inputs) != 0 {
					t.Fatal("modular assertion did not stop before source publication", e, len(n.inputs))
				}
			} else {
				serviceMust(t, e)
				if len(n.inputs) != 1 {
					t.Fatal("Continue/release omitted first sample", len(n.inputs))
				}
			}
			serviceMust(t, n.Close())
		})
		t.Run("duplicate-"+action, func(t *testing.T) {
			m := polynomialCellFixture(t)
			serviceMust(t, m.SetZPUint64(97))
			n, e := NewPolynomialNewtonInterpolator(m)
			serviceMust(t, e)
			zero := unumeral.NewInt(0)
			serviceMust(t, n.Add(&zero, m.unit))
			polynomialAssertionActionsFixture(t, strings.Repeat(action, 16))
			e = n.Add(&zero, m.unit)
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(e, ErrPolynomialAssertion) || len(n.inputs) != 1 {
					t.Fatal("duplicate assertion did not stop before subtraction", e, len(n.inputs))
				}
			} else {
				serviceMust(t, e)
				if len(n.inputs) != 2 {
					t.Fatal("Continue/release omitted duplicate sample", len(n.inputs))
				}
			}
			serviceMust(t, n.Close())
		})
	}
}

func TestPolynomialNewtonVectorOwnerAssertionActions(t *testing.T) {
	for _, action := range []string{"T", "C"} {
		t.Run(action, func(t *testing.T) {
			m := polynomialCellFixture(t)
			v, e := NewPolynomialNewtonVector(m)
			serviceMust(t, e)
			polynomialAssertionActionsFixture(t, action)
			_, e = v.At(0)
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(e, ErrPolynomialAssertion) {
					t.Fatal("null-owner assertion did not throw", e)
				}
			} else if !errors.Is(e, ErrPolynomialOwner) {
				t.Fatal("Continue/release did not reach safe null dereference boundary", e)
			}
			serviceMust(t, v.SetOwner(m))
			serviceMust(t, v.Close())
		})
	}
}

func TestPolynomialNewtonVectorSetOwnerContinue(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source SASSERT erased")
	}
	m := polynomialCellFixture(t)
	foreign := polynomialCellFixture(t)
	v, e := NewPolynomialNewtonVector(m)
	serviceMust(t, e)
	serviceMust(t, v.SetOwner(m))
	polynomialAssertionActionsFixture(t, "CC")
	serviceMust(t, v.SetOwner(foreign))
	if v.owner != foreign {
		t.Fatal("Continue did not perform following owner assignment")
	}
	serviceMust(t, v.SetOwner(m))
	serviceMust(t, v.Close())
}

func TestPolynomialNewtonMkEmptyAssertionActions(t *testing.T) {
	for _, action := range []string{"T", "C"} {
		t.Run(action, func(t *testing.T) {
			m := polynomialCellFixture(t)
			serviceMust(t, m.SetZPUint64(97))
			n, e := NewPolynomialNewtonInterpolator(m)
			serviceMust(t, e)
			r, e := NewPolynomialRef(m)
			serviceMust(t, e)
			polynomialAssertionActionsFixture(t, action)
			e = n.Mk(0, r)
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(e, ErrPolynomialAssertion) {
					t.Fatal("empty interpolation assertion did not throw", e)
				}
			} else if polynomialSourceDebug {
				if !errors.Is(e, ErrPolynomialAssertion) {
					t.Fatal("Continue did not reach the following checked vector read", e)
				}
			} else if !errors.Is(e, arena.ErrBounds) {
				t.Fatal("release did not reach the following native vector read", e)
			}
			serviceMust(t, r.Close())
			serviceMust(t, n.Close())
		})
	}
}

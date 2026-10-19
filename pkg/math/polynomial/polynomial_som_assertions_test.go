package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func polynomialSOMMismatchedBuffer(t *testing.T) (*MultivariatePolynomialCore, *PolynomialSOMBuffer) {
	t.Helper()
	m := polynomialCellFixture(t)
	b, e := NewPolynomialSOMBuffer(m)
	serviceMust(t, e)
	serviceMust(t, b.SetOwner(m))
	u, e := m.monomials.Unit()
	serviceMust(t, e)
	serviceMust(t, m.monomials.IncRef(u))
	serviceMust(t, b.buffer.ms.appendMonomial(m, u))
	serviceMust(t, b.buffer.positions.Set(u, 0))
	return m, b
}

func TestPolynomialSOMResetSizeAssertionActions(t *testing.T) {
	for _, action := range []string{"T", "C"} {
		t.Run(action, func(t *testing.T) {
			_, b := polynomialSOMMismatchedBuffer(t)
			polynomialAssertionActionsFixture(t, action)
			e := b.Reset()
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(e, ErrPolynomialAssertion) {
					t.Fatal("source reset size assertion did not throw", e)
				}
			} else if !errors.Is(e, arena.ErrBounds) {
				t.Fatal("Continue/release did not reach the next source vector read", e)
			}
			serviceMust(t, b.CloseChecked())
		})
	}
}

func TestPolynomialSOMRemoveZerosSizeAssertionActions(t *testing.T) {
	for _, action := range []string{"T", "C"} {
		t.Run(action, func(t *testing.T) {
			_, b := polynomialSOMMismatchedBuffer(t)
			polynomialAssertionActionsFixture(t, action)
			_, e := b.Mk(false)
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(e, ErrPolynomialAssertion) {
					t.Fatal("source remove_zeros size assertion did not throw", e)
				}
			} else if !errors.Is(e, arena.ErrBounds) {
				t.Fatal("Continue/release did not reach the next source vector read", e)
			}
			serviceMust(t, b.CloseChecked())
		})
	}
}

func TestPolynomialSOMRemoveZerosDuplicateAssertionActions(t *testing.T) {
	for _, action := range []string{"T", "C"} {
		t.Run(action, func(t *testing.T) {
			m := polynomialCellFixture(t)
			b, e := NewPolynomialSOMBuffer(m)
			serviceMust(t, e)
			serviceMust(t, b.SetOwner(m))
			u, e := m.monomials.Unit()
			serviceMust(t, e)
			for _, a := range []unumeral.Int{unumeral.NewInt(0), unumeral.NewInt(5)} {
				serviceMust(t, m.monomials.IncRef(u))
				serviceMust(t, b.buffer.as.appendCoefficient(m, a))
				serviceMust(t, b.buffer.ms.appendMonomial(m, u))
			}
			serviceMust(t, b.buffer.positions.Set(u, 1))
			polynomialAssertionActionsFixture(t, action)
			e = b.buffer.removeZeros(false)
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(e, ErrPolynomialAssertion) {
					t.Fatal("source duplicate assertion did not throw", e)
				}
			} else {
				serviceMust(t, e)
				n, e := b.Size()
				serviceMust(t, e)
				if n != 1 {
					t.Fatal("Continue/release did not finish source compaction", n)
				}
				a, e := b.Coefficient(0)
				serviceMust(t, e)
				v, e := m.integer.Int64(&a)
				serviceMust(t, e)
				if v != 5 {
					t.Fatal("Continue/release changed retained coefficient", v)
				}
			}
			serviceMust(t, b.CloseChecked())
			polynomialPhysicalAudit(t, m)
		})
	}
}

func TestPolynomialSOMDisplaySizeAssertionActions(t *testing.T) {
	for _, action := range []string{"T", "C"} {
		t.Run(action, func(t *testing.T) {
			m := polynomialCellFixture(t)
			b, e := NewPolynomialSOMBuffer(m)
			serviceMust(t, e)
			serviceMust(t, b.SetOwner(m))
			serviceMust(t, b.buffer.as.appendCoefficient(m, unumeral.NewInt(1)))
			polynomialAssertionActionsFixture(t, action)
			out := &monomialTestSink{}
			e = b.DisplayInto(out)
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(e, ErrPolynomialAssertion) || out.text.Len() != 0 {
					t.Fatal("source display size assertion did not stop before output", e, out.text.String())
				}
			} else if !errors.Is(e, arena.ErrBounds) || out.text.String() != "1*" {
				t.Fatal("Continue/release did not reach the source monomial read", e, out.text.String())
			}
			serviceMust(t, b.CloseChecked())
		})
	}
}

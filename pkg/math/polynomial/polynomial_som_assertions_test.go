package polynomial

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestPolynomialSOMDisabledCompactionAssertionsSkipSlotReads(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(fmt.Sprint(enabled), func(t *testing.T) {
			m := polynomialCellFixture(t)
			polynomialThrowAssertionFixture(t)
			if polynomialSourceDebug {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(enabled))
			}
			for _, e := range []error{
				m.som.assertCompactionMonomial(17, m.monomials.unit),
				m.som.assertDiscardedCoefficientZero(17),
			} {
				if polynomialSourceDebug && enabled {
					if e == nil {
						t.Fatal("enabled assertion omitted unreadable slot")
					}
				} else {
					serviceMust(t, e)
				}
			}
		})
	}
}

func TestPolynomialCheapAppendDuplicateAssertionActions(t *testing.T) {
	for _, mode := range []string{"copy", "take", "multiply"} {
		for _, action := range []string{"T", "C", "disabled"} {
			t.Run(fmt.Sprintf("%s/%s", mode, action), func(t *testing.T) {
				m := polynomialCellFixture(t)
				a := unumeral.NewInt(3)
				serviceMust(t, m.cheap.add(&a, m.monomials.unit, false))
				polynomialAssertionActionsFixture(t, action)
				if polynomialSourceDebug && action == "disabled" {
					serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
				}
				var e error
				if mode == "multiply" {
					e = m.cheap.addMul(&a, m.monomials.unit, m.unit)
				} else {
					e = m.cheap.add(&a, m.monomials.unit, mode == "take")
				}
				want := uint32(2)
				if polynomialSourceDebug && action == "T" {
					if !errors.Is(e, ErrPolynomialAssertion) {
						t.Fatal("duplicate assertion did not throw", e)
					}
					want = 1
				} else {
					serviceMust(t, e)
				}
				count, e := m.cheap.ms.size(m)
				serviceMust(t, e)
				if count != want {
					t.Fatal("duplicate append publication", count, want)
				}
				serviceMust(t, m.cheap.reset())
			})
		}
	}
}

func TestPolynomialSOMNormalizePositiveGCDAssertionActions(t *testing.T) {
	for _, action := range []string{"T", "C", "disabled"} {
		t.Run(action, func(t *testing.T) {
			m := polynomialCellFixture(t)
			// Exercise normalize_numerals directly, before a SOM removes zeros.
			m.cheap.as.values = []unumeral.Int{{}}
			defer func() { m.cheap.as.values = nil }()
			polynomialAssertionActionsFixture(t, action)
			if polynomialSourceDebug && action == "disabled" {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			e := m.cheap.normalize()
			want := unumeral.ErrDivideByZero
			if polynomialSourceDebug && action == "T" {
				want = ErrPolynomialAssertion
			}
			if !errors.Is(e, want) {
				t.Fatal("positive GCD assertion/division frontier", e, want)
			}
			if m.cheap.gcdActive || m.cheap.normalizeScope.active {
				t.Fatal("completed source NUM destructor retained the invocation")
			}
		})
	}
}

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

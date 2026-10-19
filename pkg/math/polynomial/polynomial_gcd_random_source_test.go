// Source: src/math/polynomial/polynomial.cpp:4053-4068.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"

	"github.com/glycerine/g3/pkg/clibrand"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestPolynomialFreshValueUsesSharedLibcStreamAndRejectsUsedInputs(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	serviceMust(t, c.modular.SetZPUint64(7))
	serviceMust(t, full.newton.SetOwner(c))
	n, e := full.newton.At(0)
	serviceMust(t, e)
	one, four := unumeral.NewInt(1), unumeral.NewInt(4)
	serviceMust(t, c.modular.SetInt32(&four, 4))
	serviceMust(t, n.Add(&one, c.unit))
	serviceMust(t, n.Add(&four, c.unit))
	clibrand.Srand(1)
	t.Cleanup(func() { clibrand.Srand(1) })
	out := unumeral.NewInt(99)
	serviceMust(t, full.peekFresh(n.Inputs(), 7, &out))
	// Captured glibc seed-1 draws: 1804289383%7=1,
	// 846930886%7=4, 1681692777%7=2. The first two are already used.
	if out != unumeral.NewInt(2) {
		t.Fatal("source rejection/candidate", out)
	}
	if next := clibrand.Rand(); next != 1714636915 {
		t.Fatal("source draw count/shared process stream", next)
	}
	polynomialImpPhysicalAudit(t, full)
}

// The source modulo-zero expression has no defined C++ value. This fixture
// proves the reached Go safety boundary and the DEBUG assertion/read order;
// it does not reinterpret that domain as valid finite-field arithmetic.
func TestPolynomialFreshValueZeroModulusSourceProfile(t *testing.T) {
	polynomialThrowAssertionFixture(t)
	full := polynomialPSCFullFixture(t)
	c := full.core
	serviceMust(t, full.newton.SetOwner(c))
	n, e := full.newton.At(0)
	serviceMust(t, e)
	clibrand.Srand(1)
	t.Cleanup(func() { clibrand.Srand(1) })
	out := unumeral.NewInt(99)
	e = full.peekFresh(n.Inputs(), 0, &out)
	want := unumeral.ErrDivideByZero
	next := uint32(846930886)
	if polynomialSourceDebug {
		want = ErrPolynomialAssertion
		next = 1804289383
	}
	if !errors.Is(e, want) || out != unumeral.NewInt(99) {
		t.Fatal("reached zero-modulus guard before SET", e, out)
	}
	if draw := clibrand.Rand(); draw != next {
		t.Fatal("DEBUG-before-draw versus release draw-before-modulo", draw, next)
	}
	polynomialImpPhysicalAudit(t, full)
}

// Throw stops before the shared draw; Continue and disabled assertions reach
// the source modulo expression and its Go zero-divisor boundary.
func TestPolynomialFreshValueAssertionActions(t *testing.T) {
	for _, action := range []string{"T", "C", "disabled"} {
		t.Run(action, func(t *testing.T) {
			full := polynomialPSCFullFixture(t)
			c := full.core
			serviceMust(t, full.newton.SetOwner(c))
			n, e := full.newton.At(0)
			serviceMust(t, e)
			polynomialAssertionActionsFixture(t, action)
			if polynomialSourceDebug && action == "disabled" {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			clibrand.Srand(1)
			t.Cleanup(func() { clibrand.Srand(1) })
			out := unumeral.NewInt(99)
			e = full.peekFresh(n.Inputs(), 0, &out)
			want := unumeral.ErrDivideByZero
			next := uint32(846930886)
			if polynomialSourceDebug && action == "T" {
				want = ErrPolynomialAssertion
				next = 1804289383
			}
			if !errors.Is(e, want) || out != unumeral.NewInt(99) {
				t.Fatal("assertion action before numeral SET", e, out)
			}
			if got := clibrand.Rand(); got != next {
				t.Fatal("assertion action shared-stream frontier", got, next)
			}
		})
	}
}

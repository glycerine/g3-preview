package polynomial

// Source: algebraic_numbers.cpp:235-240,427,445-446,509-539.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestAlgebraicCellCopyRootToBasicDeletesBeforeConstCopy(t *testing.T) {
	c, _, _ := algebraicCompareFixtureProfile(t, true)
	q := unumeral.NewRational()
	serviceMust(t, c.config.Rationals.SetString(&q, "1267650600228229401496703205377"))
	var src Algebraic
	serviceMust(t, c.SetRational(&src, &q))
	serviceMust(t, c.config.Rationals.Del(&q))
	dst := algebraicTinyBinaryBoundsRoot(t, c)
	old := dst
	l, e := c.location(&dst)
	serviceMust(t, e)
	record, e := c.load(l)
	serviceMust(t, e)
	oldPoly := record.poly
	z := c.config.Rationals.Integers()
	z.Budget().FailAfter(0)
	body := c.Copy(&dst, &src)
	z.Budget().FailAfter(-1)
	if !numericPhysicalReservationRefusal(body) || dst != (Algebraic{}) || c.constQ.owner != c {
		t.Fatal("root-to-basic copy constructed temporary before deleting destination", body, dst)
	}
	if _, e := c.locationCleanup(&old); e == nil {
		t.Fatal("old root survived const temporary refusal")
	}
	if _, e := c.config.Polynomials.Size(oldPoly); e == nil {
		t.Fatal("old root polynomial survived source deletion")
	}
	serviceMust(t, c.CleanupRetained())
	serviceMust(t, c.Copy(&dst, &src))
	// Source is unchanged and the new basic result has an independent numeral.
	srcLoc, e := c.location(&src)
	serviceMust(t, e)
	dstLoc, e := c.location(&dst)
	serviceMust(t, e)
	sr, e := c.load(srcLoc)
	serviceMust(t, e)
	dr, e := c.load(dstLoc)
	serviceMust(t, e)
	equal, e := c.config.Rationals.Equal(&sr.value, &dr.value)
	serviceMust(t, e)
	if !equal || sr.value.Num.Cell == dr.value.Num.Cell {
		t.Fatal("basic copy lost value or aliased source numerator")
	}
	serviceMust(t, c.Del(&dst))
	serviceMust(t, c.Del(&src))
}

func TestAlgebraicCellCopyBranchSwitchBeforeDestruction(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source assertions erased")
	}
	for _, targetRoot := range []bool{false, true} {
		for _, sourceRoot := range []bool{false, true} {
			t.Run(map[bool]string{false: "basic", true: "root"}[targetRoot]+"/"+map[bool]string{false: "basic", true: "root"}[sourceRoot], func(t *testing.T) {
				c, _, _ := algebraicCompareFixtureProfile(t, true)
				var src, dst Algebraic
				if targetRoot {
					dst = algebraicTinyBinaryBoundsRoot(t, c)
				}
				if sourceRoot {
					src = algebraicTinyBinaryBoundsRoot(t, c)
				}
				polynomialAssertionActionsFixture(t, "")
				serviceMust(t, diagnostic.BorrowProcessDebugContext().Close())
				beforeSrc, beforeDst, callers := src, dst, c.callers
				z := c.config.Rationals.Integers()
				used, stats := z.Budget().Used(), z.RuntimeOwner().Stats()
				if e := c.Copy(&dst, &src); !errors.Is(e, arena.ErrClosed) {
					t.Error("copy read payload before selected branch assertion", e)
				}
				if src != beforeSrc || dst != beforeDst || c.callers != callers || used != z.Budget().Used() || stats != z.RuntimeOwner().Stats() {
					t.Error("branch switch failure followed destination mutation")
				}
				serviceMust(t, c.CleanupRetained())
				serviceMust(t, c.CleanupValue(&dst))
				serviceMust(t, c.CleanupValue(&src))
			})
		}
	}
}

func TestAlgebraicCellCopySelfSkipsAssertionsAndPayload(t *testing.T) {
	c, _, _ := algebraicCompareFixtureProfile(t, true)
	a := algebraicTinyBinaryBoundsRoot(t, c)
	l, e := c.location(&a)
	serviceMust(t, e)
	// An allocated-object prefix is unread by source self-assignment. This is
	// an operand-erasure vector, not a completed C++ construction claim.
	initialized := l.initialized
	l.initialized = false
	if polynomialSourceDebug {
		polynomialAssertionActionsFixture(t, "")
		serviceMust(t, diagnostic.BorrowProcessDebugContext().Close())
	}
	serviceMust(t, c.Copy(&a, &a))
	l.initialized = initialized
	serviceMust(t, c.CleanupValue(&a))
}

func TestAlgebraicCellCopyPolynomialAssertionActions(t *testing.T) {
	for _, field := range []string{"pointer", "size"} {
		for _, action := range []string{"T", "C", "disabled"} {
			t.Run(field+"/"+action, func(t *testing.T) {
				c, _, _ := algebraicCompareFixtureProfile(t, true)
				polynomialAssertionActionsFixture(t, action)
				if polynomialSourceDebug && action == "disabled" {
					serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
				}
				r := algebraicRecord{}
				if field == "pointer" {
					r.poly = UPolynomial{slot: 1}
				} else {
					r.size = 3
				}
				body := c.assertCellCopyPolynomialEmpty(&r)
				if polynomialSourceDebug && action == "T" {
					if !linearThrownAssertion(body) || !errors.Is(body, ErrAlgebraicAssertion) {
						t.Fatal("copy_poly assertion lost source Throw", body)
					}
				} else {
					serviceMust(t, body)
				}
			})
		}
	}
}

func TestAlgebraicCellCopyAssertionDisabledOperandsAndResetOrder(t *testing.T) {
	c, _, _ := algebraicCompareFixtureProfile(t, true)
	polynomialAssertionActionsFixture(t, "")
	if polynomialSourceDebug {
		serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
	}
	serviceMust(t, c.assertCellCopyPolynomialEmpty(nil))
	serviceMust(t, c.assertCellResetZero(nil))
	if !polynomialSourceDebug {
		return
	}
	serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(true))
	a := algebraicTinyBinaryBoundsRoot(t, c)
	serviceMust(t, diagnostic.BorrowProcessDebugContext().Close())
	q := unumeral.NewRational()
	original := a
	if e := c.SetTake(&a, &q); !errors.Is(e, arena.ErrClosed) || a != original {
		t.Fatal("zero setter skipped deletion's established tag switch", e, a)
	}
	serviceMust(t, c.CleanupValue(&a))
	if e := c.SetTake(&a, &q); !errors.Is(e, arena.ErrClosed) || a != (Algebraic{}) {
		t.Fatal("zero setter omitted the post-reset assertion switch", e, a)
	}
}

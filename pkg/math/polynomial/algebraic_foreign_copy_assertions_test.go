package polynomial

// Source: algebraic_numbers.cpp:235-240,427,509-539; algebraic_numbers.h:372-373.

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestAlgebraicCopyFromCoreBranchSwitchBeforePayload(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source assertion erased")
	}
	for sk := 0; sk < 3; sk++ {
		for dk := 0; dk < 3; dk++ {
			t.Run(fmt.Sprintf("%d-to-%d", sk, dk), func(t *testing.T) {
				source, _, _ := algebraicCompareFixtureProfile(t, true)
				target, _, _ := algebraicCompareFixtureProfile(t, true)
				src, dst := algebraicForeignValue(t, source, sk), algebraicForeignValue(t, target, dk)
				polynomialAssertionActionsFixture(t, "")
				serviceMust(t, diagnostic.BorrowProcessDebugContext().Close())
				oldSrc, oldDst, callers := src, dst, target.callers
				z := target.config.Rationals.Integers()
				used, stats := z.Budget().Used(), z.RuntimeOwner().Stats()
				if e := target.CopyFromCore(&dst, source, &src); !errors.Is(e, arena.ErrClosed) {
					t.Error("foreign copy skipped selected branch assertion", e)
				}
				if target.foreignCopy.active || source.foreignCopyBorrowers != 0 {
					t.Error("branch gate failure acquired a source borrow")
				}
				if src != oldSrc || dst != oldDst || target.callers != callers || used != z.Budget().Used() || stats != z.RuntimeOwner().Stats() {
					t.Error("foreign branch gate followed mutation")
				}
				serviceMust(t, target.CleanupRetained())
				serviceMust(t, target.CleanupValue(&dst))
				serviceMust(t, source.CleanupValue(&src))
			})
		}
	}
}

func TestAlgebraicCopyFromCoreSharedZeroSelfSkipsAssertions(t *testing.T) {
	source, _, _ := algebraicCompareFixtureProfile(t, true)
	target, _, _ := algebraicCompareFixtureProfile(t, true)
	if polynomialSourceDebug {
		polynomialAssertionActionsFixture(t, "")
		serviceMust(t, diagnostic.BorrowProcessDebugContext().Close())
	}
	sz, tz := source.config.Rationals.Integers(), target.config.Rationals.Integers()
	sourceUsed, targetUsed := sz.Budget().Used(), tz.Budget().Used()
	sz.Budget().FailAfter(0)
	tz.Budget().FailAfter(0)
	var a Algebraic
	e := target.CopyFromCore(&a, source, &a)
	sz.Budget().FailAfter(-1)
	tz.Budget().FailAfter(-1)
	serviceMust(t, e)
	if a != (Algebraic{}) || sourceUsed != sz.Budget().Used() || targetUsed != tz.Budget().Used() || source.foreignCopyBorrowers != 0 || target.foreignCopy.active {
		t.Fatal("self copy read source body or acquired ownership")
	}
}

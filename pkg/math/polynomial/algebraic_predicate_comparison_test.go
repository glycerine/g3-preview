package polynomial

import (
	"errors"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"strconv"
	"testing"
)

func TestAlgebraicPredicateRationalComparisonAllocationPrefixes(t *testing.T) {
	success, refused := false, false
	for ordinal := int64(0); ordinal < 256 && !success; ordinal++ {
		t.Run(strconv.FormatInt(ordinal, 10), func(t *testing.T) {
			c, _ := algebraicCellSourceFixtureProfile(t, true)
			root := algebraicLargeDyadicPromotionRoot(t, c)
			algebraicPredicateBeforeQ(t, c, &root)
			serviceMust(t, c.setPredicateQParts(&c.predicate.candidate, &c.isRationalTmp))
			l, e := c.location(&root)
			serviceMust(t, e)
			r, e := c.load(l)
			serviceMust(t, e)
			z := c.config.Rationals.Integers()
			z.Budget().FailAfter(ordinal)
			less, body := c.lessPredicateRational(r.interval.BorrowLower())
			z.Budget().FailAfter(-1)
			if body != nil {
				refused = true
				if !numericPhysicalReservationRefusal(body) {
					t.Fatal(body)
				}
			} else {
				success = true
				if !less {
					t.Fatal("1 is not less than (2^63+1)/2^63")
				}
				less, e = c.lessPredicateRational(r.interval.BorrowUpper())
				serviceMust(t, e)
				if less {
					t.Fatal("equal upper bound compares less than candidate")
				}
			}
			_ = c.finishPredicate(c.finishPredicateQ(body))
			serviceMust(t, c.CleanupRetained())
			serviceMust(t, c.Del(&root))
		})
	}
	if !success || !refused {
		t.Fatal("missing rational comparison success/refusal", success, refused)
	}
}

func TestAlgebraicPredicateRationalComparisonOwnsActualShiftCopy(t *testing.T) {
	var c *AlgebraicRootCore
	reached := false
	transport := func(f umemory.Failure) error {
		if c == nil || !c.predicateQ.executing {
			return f
		}
		reached = true
		for _, e := range []error{c.Close(), c.CleanupRetained(), c.finishPredicateQ(nil), c.finishPredicate(nil)} {
			if !errors.Is(e, ErrAlgebraicPending) {
				t.Fatal("comparison callback obtained candidate lifecycle", e)
			}
		}
		return f
	}
	var runtime *umemory.Runtime
	c, _, runtime = algebraicNegTLSFixture(t, transport)
	root := algebraicLargeDyadicPromotionRoot(t, c)
	algebraicPredicateBeforeQ(t, c, &root)
	serviceMust(t, c.setPredicateQParts(&c.predicate.candidate, &c.isRationalTmp))
	z, b := c.config.Rationals.Integers(), c.config.Binary
	l, e := c.location(&root)
	serviceMust(t, e)
	r, e := c.load(l)
	serviceMust(t, e)
	// Execute the first source multiply to warm only its cached destination.
	// The later, independently cached shift destination remains unconstructed.
	serviceMust(t, z.Mul(r.interval.BorrowLower().BorrowNumerator(), c.localQ.BorrowDenominator(), &b.Tmp))
	pad := algebraicCellSourceTLSBoundary(t, c)
	_, body := c.lessPredicateRational(r.interval.BorrowLower())
	s := &c.predicateQ
	if body == nil || !reached || !z.MatchesCopyAllocationInvocation(&s.comparisonCopy) {
		t.Fatal("missing actual comparison shift-copy receipt", body)
	}
	serviceMust(t, runtime.SetMaxSize(0))
	serviceMust(t, runtime.Deallocate(&pad))
	if _, e := c.lessPredicateRational(r.interval.BorrowLower()); !errors.Is(e, ErrAlgebraicPending) {
		t.Fatal("outstanding comparison copy replaced", e)
	}
	_ = c.finishPredicate(c.finishPredicateQ(body))
	serviceMust(t, c.CleanupRetained())
	if z.MatchesCopyAllocationInvocation(&s.comparisonCopy) || s.active {
		t.Fatal("comparison copy acknowledgement lost")
	}
	serviceMust(t, c.Del(&root))
}

func TestAlgebraicPredicateRationalComparisonPreservesOlderCopy(t *testing.T) {
	c, _ := algebraicCellSourceFixtureProfile(t, true)
	root := algebraicLargeDyadicPromotionRoot(t, c)
	algebraicPredicateBeforeQ(t, c, &root)
	serviceMust(t, c.setPredicateQParts(&c.predicate.candidate, &c.isRationalTmp))
	l, e := c.location(&root)
	serviceMust(t, e)
	r, e := c.load(l)
	serviceMust(t, e)
	z := c.config.Rationals.Integers()
	runtime := z.RuntimeOwner()
	var old unumeral.IntegerCopyAllocationReceipt
	var destination unumeral.Int
	pad := algebraicCellSourceTLSBoundary(t, c)
	body := z.CopyFromOwnerWithReceipt(&destination, z, &z.Two64, &old)
	if body == nil || !z.MatchesCopyAllocationInvocation(&old) {
		t.Fatal("older independent copy did not retain allocation", body)
	}
	serviceMust(t, runtime.SetMaxSize(0))
	serviceMust(t, runtime.Deallocate(&pad))
	_, body = c.lessPredicateRational(r.interval.BorrowLower())
	if body == nil || c.predicateQ.comparisonCopy != (unumeral.IntegerCopyAllocationReceipt{}) {
		t.Fatal("refused comparison adopted old copy", body)
	}
	state := z.CopyAllocation
	_ = c.finishPredicate(c.finishPredicateQ(body))
	serviceMust(t, c.CleanupRetained())
	if !z.MatchesCopyAllocationInvocation(&old) || state != z.CopyAllocation {
		t.Fatal("comparison cleanup consumed old allocation")
	}
	serviceMust(t, z.CleanupCopyAllocationInvocation(&old))
	serviceMust(t, z.Del(&destination))
	serviceMust(t, c.Del(&root))
}

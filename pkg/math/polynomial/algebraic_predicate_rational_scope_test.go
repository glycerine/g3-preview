package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"reflect"
	"strconv"
	"testing"
)

// Reach the retained SET phase with the actual source coefficient and scaled
// bounds. No rational payload or child receipt is fabricated by the fixture.
func algebraicPredicateBeforeQ(t *testing.T, c *AlgebraicRootCore, root *Algebraic) {
	t.Helper()
	l, e := c.location(root)
	serviceMust(t, e)
	r, e := c.load(l)
	serviceMust(t, e)
	serviceMust(t, c.beginPredicate())
	serviceMust(t, c.constructPredicateSaved(root, &r))
	v, e := c.sourcePolynomial(&r)
	serviceMust(t, e)
	lead, e := c.config.Polynomials.read(v, r.size-1)
	serviceMust(t, e)
	z := c.config.Rationals.Integers()
	serviceMust(t, z.CopyFromOwnerWithReceipt(&c.isRationalTmp, z, lead, &c.predicate.leadingCopy))
	serviceMust(t, z.Abs(&c.isRationalTmp))
	serviceMust(t, c.scalePredicateBounds(&r))
	serviceMust(t, c.floorPredicateCandidate(&c.predicate.upper))
	serviceMust(t, c.beginPredicateQ())
}
func TestAlgebraicPredicateRationalCandidateAllocationPrefixes(t *testing.T) {
	success, copied := false, false
	for ordinal := int64(0); ordinal < 1024 && !success; ordinal++ {
		t.Run(strconv.FormatInt(ordinal, 10), func(t *testing.T) {
			c, _ := algebraicCellSourceFixtureProfile(t, true)
			root := algebraicLargeDyadicPromotionRoot(t, c)
			algebraicPredicateBeforeQ(t, c, &root)
			q, z := c.config.Rationals, c.config.Rationals.Integers()
			z.Budget().FailAfter(ordinal)
			body := c.setPredicateQParts(&c.predicate.candidate, &c.isRationalTmp)
			z.Budget().FailAfter(-1)
			if body != nil && !numericPhysicalReservationRefusal(body) {
				t.Fatal(body)
			}
			s := &c.predicateQ
			if s.parts.Numerator != (unumeral.IntegerCopyAllocationReceipt{}) || s.parts.Denominator != (unumeral.IntegerCopyAllocationReceipt{}) {
				copied = true
			}
			if body == nil {
				success = true
				want := unumeral.NewRational()
				serviceMust(t, q.SetString(&want, "9223372036854775809/9223372036854775808"))
				same, e := q.Equal(&c.localQ, &want)
				serviceMust(t, e)
				if !same {
					t.Fatal("independent rational candidate")
				}
				serviceMust(t, q.Del(&want))
			}
			cause := c.finishPredicateQ(body)
			_ = c.finishPredicate(cause)
			serviceMust(t, c.CleanupRetained())
			if s.active || c.localActive || c.predicate.active {
				t.Fatal("candidate owner remained active")
			}
			serviceMust(t, c.Del(&root))
		})
	}
	if !success || !copied {
		t.Fatal("missing rational candidate success/actual copy allocation", success, copied)
	}
}

func TestAlgebraicPredicateRationalCandidateDoesNotAdoptOlderCopy(t *testing.T) {
	c, _ := algebraicCellSourceFixtureProfile(t, true)
	root := algebraicLargeDyadicPromotionRoot(t, c)
	algebraicPredicateBeforeQ(t, c, &root)
	z := c.config.Rationals.Integers()
	runtime := z.RuntimeOwner()
	var old unumeral.IntegerCopyAllocationReceipt
	var destination unumeral.Int
	pad := algebraicCellSourceTLSBoundary(t, c)
	body := z.CopyFromOwnerWithReceipt(&destination, z, &z.Two64, &old)
	if body == nil || !z.MatchesCopyAllocationInvocation(&old) {
		t.Fatal("missing older copy", body)
	}
	serviceMust(t, runtime.SetMaxSize(0))
	serviceMust(t, runtime.Deallocate(&pad))
	body = c.setPredicateQParts(&c.predicate.candidate, &c.isRationalTmp)
	if body == nil || c.predicateQ.parts.Numerator != (unumeral.IntegerCopyAllocationReceipt{}) || c.predicateQ.parts.Denominator != (unumeral.IntegerCopyAllocationReceipt{}) {
		t.Fatal("refused SET adopted older copy", body)
	}
	state := z.CopyAllocation
	_ = c.finishPredicate(c.finishPredicateQ(body))
	serviceMust(t, c.CleanupRetained())
	if !z.MatchesCopyAllocationInvocation(&old) || state != z.CopyAllocation {
		t.Fatal("predicate consumed older copy")
	}
	serviceMust(t, z.CleanupCopyAllocationInvocation(&old))
	serviceMust(t, z.Del(&destination))
	serviceMust(t, c.Del(&root))
	if e := c.beginPredicateQ(); !errors.Is(e, ErrAlgebraicPending) {
		t.Fatal("candidate scope accepted without predicate", e)
	}
}

// Qualified malformed requested-size guard: NUM is physically released before
// DEN refuses, preserving both the real refused setter and destructor errors.
func TestAlgebraicPredicateRationalCandidateSourceDestructorStop(t *testing.T) {
	c, _ := algebraicCellSourceFixtureProfile(t, true)
	root := algebraicLargeDyadicPromotionRoot(t, c)
	algebraicPredicateBeforeQ(t, c, &root)
	serviceMust(t, c.setPredicateQParts(&c.predicate.candidate, &c.isRationalTmp))
	z := c.config.Rationals.Integers()
	serviceMust(t, c.restoreSaved())
	z.Budget().FailAfter(0)
	body := c.SetTake(&root, &c.localQ)
	z.Budget().FailAfter(-1)
	if body == nil || !numericPhysicalReservationRefusal(body) {
		t.Fatal("setter did not refuse", body)
	}
	num, den := c.localQ.BorrowNumerator().Cell, c.localQ.BorrowDenominator().Cell
	if num == 0 || den == 0 || z.IsFailureRef(num) || z.IsFailureRef(den) {
		t.Fatal("actual BIG candidate fields absent")
	}
	cell := z.Cell(den)
	requested := cell.Backing.Requested
	cell.Backing.Requested = uint64(^uint32(0)) + 1
	z.Store(den, cell)
	cause := c.finishPredicateQ(body)
	s := &c.predicateQ
	if !errors.Is(cause, body) || !errors.Is(cause, arena.ErrBounds) || s.sourceCursor != 1 || !s.active || z.Cell(den).Released || unumeral.ValidNumericRef(z.Cells, num) {
		t.Fatal("source rational destruction frontier", cause, s.sourceCursor)
	}
	_ = c.finishPredicate(cause)
	if c.predicate.stage != 0 || c.predicate.childStop == nil || !c.predicate.restored {
		t.Fatal("older predicate crossed candidate stop")
	}
	state, used, stats := *s, z.Budget().Used(), z.RuntimeOwner().Stats()
	if e := c.finishPredicateQ(nil); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("source stop lost", e)
	}
	if !reflect.DeepEqual(state, *s) || used != z.Budget().Used() || stats != z.RuntimeOwner().Stats() {
		t.Fatal("source rational destructor replayed")
	}
	if e := c.cleanupPredicate(); !errors.Is(e, ErrAlgebraicPending) {
		t.Fatal("older predicate cleaned before candidate", e)
	}
	cell = z.Cell(den)
	cell.Backing.Requested = requested
	z.Store(den, cell)
	serviceMust(t, c.CleanupRetained())
	if s.active || s.sourceCursor != 1 || s.ownCursor != 2 || s.sourceStop == nil || c.predicate.stage != 0 || c.predicate.active {
		t.Fatal("own cleanup changed source history")
	}
	serviceMust(t, c.Del(&root))
}

func TestAlgebraicPredicateRationalCandidateCopyCallbackGuards(t *testing.T) {
	var c *AlgebraicRootCore
	reached := false
	transport := func(f umemory.Failure) error {
		if c == nil || !c.predicateQ.executing {
			return f
		}
		reached = true
		for _, e := range []error{c.Close(), c.CleanupRetained(), c.cleanupPredicateQ(), c.cleanupPredicate(), c.finishPredicateQ(nil), c.finishPredicate(nil)} {
			if !errors.Is(e, ErrAlgebraicPending) {
				t.Fatal("callback obtained candidate lifecycle", e)
			}
		}
		return f
	}
	c, _, runtime := algebraicNegTLSFixture(t, transport)
	root := algebraicLargeDyadicPromotionRoot(t, c)
	algebraicPredicateBeforeQ(t, c, &root)
	pad := algebraicCellSourceTLSBoundary(t, c)
	body := c.setPredicateQParts(&c.predicate.candidate, &c.isRationalTmp)
	z := c.config.Rationals.Integers()
	if body == nil || !reached || !z.MatchesCopyAllocationInvocation(&c.predicateQ.parts.Numerator) {
		t.Fatal("missing actual candidate copy refusal", body, reached)
	}
	serviceMust(t, runtime.SetMaxSize(0))
	serviceMust(t, runtime.Deallocate(&pad))
	_ = c.finishPredicate(c.finishPredicateQ(body))
	serviceMust(t, c.CleanupRetained())
	serviceMust(t, c.Del(&root))
}

// Real runtime operations reach a signed-host deallocation accounting guard.
// This qualified transport is distinct from ordinary allocation exhaustion.
func TestAlgebraicPredicateRationalCandidateSignedHostDestructor(t *testing.T) {
	c, _, runtime := algebraicNegTLSFixture(t, nil)
	root := algebraicLargeDyadicPromotionRoot(t, c)
	algebraicPredicateBeforeQ(t, c, &root)
	serviceMust(t, c.setPredicateQParts(&c.predicate.candidate, &c.isRationalTmp))
	z := c.config.Rationals.Integers()
	num, den := c.localQ.BorrowNumerator().Cell, c.localQ.BorrowDenominator().Cell
	contextTLSNearMinimum(t, runtime)
	lane, e := runtime.Allocate(8)
	serviceMust(t, e)
	delta := int64(-1000000)
	if e = runtime.Reallocate(&lane, uint64(16)+uint64(delta)-8); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	cause := c.finishPredicateQ(nil)
	s := &c.predicateQ
	if !errors.Is(cause, arena.ErrBounds) || s.sourceCursor != 0 || !z.Cell(num).Released || z.Cell(den).Released {
		t.Fatal("actual NUM release guard", cause, s.sourceCursor)
	}
	_ = c.finishPredicate(cause)
	if c.predicate.stage != 0 || c.predicate.childStop == nil || c.predicate.restored {
		t.Fatal("older predicate crossed NUM stop")
	}
	for _, field := range []*unumeral.Int{&c.predicate.candidate, c.predicate.lower.BorrowNumerator(), c.predicate.upper.BorrowNumerator(), c.predicate.saved.BorrowUpper().BorrowNumerator()} {
		if !unumeral.ValidNumericRef(z.Cells, field.Cell) || z.Cell(field.Cell).Released {
			t.Fatal("older predicate value physically destroyed")
		}
	}
	contextTLSFinish(t, runtime, c.CleanupRetained)
	if s.active || s.sourceCursor != 0 || s.ownCursor != 2 || s.sourceStop == nil || c.predicate.active {
		t.Fatal("source/own rational cursors conflated")
	}
	contextTLSFinish(t, runtime, func() error { return c.Del(&root) })
	contextTLSFinish(t, runtime, func() error {
		if lane == (umemory.Block{}) {
			return nil
		}
		return runtime.Deallocate(&lane)
	})
}

func TestAlgebraicPredicateRationalCandidatePublicNonDyadic(t *testing.T) {
	c, _, _ := algebraicCompareFixture(t)
	// (3*x-1)*(x-2): bisection cannot discover the root 1/3 exactly.
	// The retained rational-root-theorem SET/evaluation/publication path must run.
	root := algebraicCompareRoot(t, c, []int32{2, -7, 3}, 0, 0, 1, 0, false)
	yes, e := c.IsRational(&root)
	serviceMust(t, e)
	if !yes || c.predicateQ.owner != c || c.predicateQ.sourceCursor != 2 || c.predicateQ.active || c.localActive || c.predicate.active {
		t.Fatal("public predicate bypassed/composed candidate scope incorrectly")
	}
	l, e := c.location(&root)
	serviceMust(t, e)
	if l.kind != 0 {
		t.Fatal("candidate was not published as a basic numeral")
	}
	want := unumeral.NewRational()
	serviceMust(t, c.config.Rationals.SetFraction(&want, 1, 3))
	cmp, e := c.CompareRational(&root, &want)
	serviceMust(t, e)
	if cmp != 0 {
		t.Fatal("public rational candidate is not 1/3", cmp)
	}
	serviceMust(t, c.config.Rationals.Del(&want))
	serviceMust(t, c.Del(&root))
}

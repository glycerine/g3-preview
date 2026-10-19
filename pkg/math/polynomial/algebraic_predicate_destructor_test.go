package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"reflect"
	"testing"
)

// Execute the actual retained constructor/refinement/scaled-bound phases. The
// test can then arm a real failure at candidate floor, after diagnostic peaks.
func algebraicPredicateBeforeFloor(t *testing.T, c *AlgebraicRootCore, root *Algebraic) {
	t.Helper()
	l, e := c.location(root)
	serviceMust(t, e)
	r, e := c.load(l)
	serviceMust(t, e)
	serviceMust(t, c.beginPredicate())
	serviceMust(t, c.constructPredicateSaved(root, &r))
	p := &c.predicate
	v, e := c.sourcePolynomial(&r)
	serviceMust(t, e)
	lead, e := c.config.Polynomials.read(v, r.size-1)
	serviceMust(t, e)
	z := c.config.Rationals.Integers()
	serviceMust(t, z.CopyFromOwnerWithReceipt(&c.isRationalTmp, z, lead, &p.leadingCopy))
	serviceMust(t, z.Abs(&c.isRationalTmp))
	k, e := z.Log2(&c.isRationalTmp)
	serviceMust(t, e)
	ok, e := c.refineAtWithReceipt(root, k+1, true, &p.refinement)
	serviceMust(t, e)
	if !ok {
		t.Fatal("fixture unexpectedly found a rational root")
	}
	l, e = c.location(root)
	serviceMust(t, e)
	r, e = c.load(l)
	serviceMust(t, e)
	serviceMust(t, c.scalePredicateBounds(&r))
}

// Qualified returning OOM transport and signed-host accounting guard: the
// original floor-copy failure and younger upper-bound destructor both stop.
func TestAlgebraicPredicateOwnDestructorJoinsBodyAndRetainsOlderValues(t *testing.T) {
	var c *AlgebraicRootCore
	var runtime *umemory.Runtime
	var lane umemory.Block
	reached := false
	transport := func(f umemory.Failure) error {
		if c == nil || !c.predicate.candidateActive || c.predicate.finishing {
			return f
		}
		reached = true
		serviceMust(t, runtime.SetMaxSize(0))
		contextTLSNearMinimum(t, runtime)
		var e error
		lane, e = runtime.Allocate(8)
		serviceMust(t, e)
		delta := int64(-1000000)
		if e = runtime.Reallocate(&lane, uint64(16)+uint64(delta)-8); !errors.Is(e, arena.ErrBounds) {
			t.Fatal(e)
		}
		return f
	}
	c, _, runtime = algebraicNegTLSFixture(t, transport)
	root := algebraicBigBinaryBoundsRootWithMinimal(t, c, false)
	algebraicPredicateBeforeFloor(t, c, &root)
	s := &c.predicate
	if !s.lowerActive || !s.upperActive || s.candidateActive {
		t.Fatal("source bound/candidate declaration order")
	}
	pad := algebraicCellSourceTLSBoundary(t, c)
	body := c.floorPredicateCandidate(&s.upper)
	if !reached || body == nil || !numericPhysicalReservationRefusal(body) {
		t.Fatal("actual floor-copy refusal not reached", body)
	}
	cause := c.finishPredicate(body)
	if !errors.Is(cause, body) || !errors.Is(cause, arena.ErrBounds) || s.sourceStop == nil || !s.active || s.stage != 1 || s.savedSourceCursor != 0 || s.restored {
		t.Fatal("predicate source destructor frontier", cause, s.stage, s.savedSourceCursor)
	}
	z := c.config.Rationals.Integers()
	for _, n := range []*unumeral.Int{s.lower.BorrowNumerator(), s.saved.BorrowLower().BorrowNumerator(), s.saved.BorrowUpper().BorrowNumerator()} {
		if !unumeral.ValidNumericRef(z.Cells, n.Cell) || z.Cell(n.Cell).Released {
			t.Fatal("older predicate value destroyed after upper destructor stop")
		}
	}
	if !z.MatchesCopyAllocationInvocation(&s.candidateCopy) {
		t.Fatal("actual floor-copy allocation receipt lost")
	}
	state, used, stats := *s, z.Budget().Used(), runtime.Stats()
	if e := c.finishPredicate(nil); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("stopped source finalizer did not remain stopped", e)
	}
	if !reflect.DeepEqual(state, *s) || used != z.Budget().Used() || stats != runtime.Stats() {
		t.Fatal("source finalizer replayed consumed release")
	}
	contextTLSFinish(t, runtime, c.CleanupRetained)
	if s.active || s.stage != 1 || s.ownStage != 5 || s.savedSourceCursor != 0 || s.savedOwnCursor != 2 || s.sourceStop == nil {
		t.Fatal("owning cleanup moved source predicate cursor")
	}
	contextTLSFinish(t, runtime, func() error { return c.CleanupValue(&root) })
	for _, block := range []*umemory.Block{&lane, &pad} {
		contextTLSFinish(t, runtime, func() error {
			if *block == (umemory.Block{}) {
				return nil
			}
			return runtime.Deallocate(block)
		})
	}
}

// The deliberate upper requested-size guard is separate from the actual
// refused refinement body. Source saved-interval deletion is lower then upper.
func TestAlgebraicPredicateSavedDestructorCursorAndBodyError(t *testing.T) {
	c, _ := algebraicCellSourceFixtureProfile(t, true)
	root := algebraicBigBinaryBoundsRootWithMinimal(t, c, false)
	l, e := c.location(&root)
	serviceMust(t, e)
	r, e := c.load(l)
	serviceMust(t, e)
	originalLower, originalUpper := r.interval.BorrowLower().BorrowNumerator().Cell, r.interval.BorrowUpper().BorrowNumerator().Cell
	serviceMust(t, c.beginPredicate())
	serviceMust(t, c.constructPredicateSaved(&root, &r))
	z := c.config.Rationals.Integers()
	s := &c.predicate
	v, e := c.sourcePolynomial(&r)
	serviceMust(t, e)
	lead, e := c.config.Polynomials.read(v, r.size-1)
	serviceMust(t, e)
	serviceMust(t, z.CopyFromOwnerWithReceipt(&c.isRationalTmp, z, lead, &s.leadingCopy))
	serviceMust(t, z.Abs(&c.isRationalTmp))
	k, e := z.Log2(&c.isRationalTmp)
	serviceMust(t, e)
	z.Budget().FailAfter(0)
	_, body := c.refineAtWithReceipt(&root, k+1, true, &s.refinement)
	z.Budget().FailAfter(-1)
	if body == nil || !numericPhysicalReservationRefusal(body) || c.refinementSourceStop(&s.refinement) != nil {
		t.Fatal("refinement body refusal did not precede normal source cleanup", body)
	}
	cell := z.Cell(originalUpper)
	requested := cell.Backing.Requested
	cell.Backing.Requested = uint64(^uint32(0)) + 1
	z.Store(originalUpper, cell)
	cause := c.finishPredicate(body)
	if !errors.Is(cause, body) || !errors.Is(cause, arena.ErrBounds) || s.stage != 4 || s.savedSourceCursor != 1 || s.sourceStop == nil || !s.restored || !s.active {
		t.Fatal("saved interval source cursor/body error", cause, s.stage, s.savedSourceCursor)
	}
	if unumeral.ValidNumericRef(z.Cells, originalLower) || !unumeral.ValidNumericRef(z.Cells, originalUpper) {
		t.Fatal("saved lower/upper destruction order")
	}
	before, used, stats := *s, z.Budget().Used(), z.RuntimeOwner().Stats()
	if e := c.finishPredicate(nil); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("stopped saved destructor resumed", e)
	}
	if !reflect.DeepEqual(before, *s) || used != z.Budget().Used() || stats != z.RuntimeOwner().Stats() {
		t.Fatal("saved destructor replayed source release")
	}
	cell = z.Cell(originalUpper)
	cell.Backing.Requested = requested
	z.Store(originalUpper, cell)
	serviceMust(t, c.CleanupRetained())
	if s.active || s.stage != 4 || s.savedSourceCursor != 1 || s.ownStage != 5 || s.savedOwnCursor != 2 {
		t.Fatal("owning cleanup changed saved source cursor")
	}
	serviceMust(t, c.Del(&root))
}

func TestAlgebraicPredicateFloorCopyPreservesOlderAllocation(t *testing.T) {
	c, _, runtime := algebraicNegTLSFixture(t, nil)
	root := algebraicBigBinaryBoundsRootWithMinimal(t, c, false)
	algebraicPredicateBeforeFloor(t, c, &root)
	z := c.config.Rationals.Integers()
	destination := unumeral.NewInt(0)
	var older unumeral.IntegerCopyAllocationReceipt
	pad := algebraicCellSourceTLSBoundary(t, c)
	body := z.CopyFromOwnerWithReceipt(&destination, z, &z.Two64, &older)
	if body == nil || !z.MatchesCopyAllocationInvocation(&older) {
		t.Fatal("actual older copy refusal missing", body)
	}
	serviceMust(t, runtime.SetMaxSize(0))
	serviceMust(t, runtime.Deallocate(&pad))
	body = c.floorPredicateCandidate(&c.predicate.upper)
	if body == nil || z.MatchesCopyAllocationInvocation(&c.predicate.candidateCopy) {
		t.Fatal("floor adopted refused older allocation", body)
	}
	if e := c.finishPredicate(body); !errors.Is(e, body) {
		t.Fatal("source finalizer discarded refused floor body", e)
	}
	state, used, stats := z.CopyAllocation, z.Budget().Used(), runtime.Stats()
	serviceMust(t, c.CleanupRetained())
	if !z.MatchesCopyAllocationInvocation(&older) || state != z.CopyAllocation || used != z.Budget().Used() || stats != runtime.Stats() {
		t.Fatal("predicate owning cleanup consumed older copy")
	}
	serviceMust(t, z.CleanupCopyAllocationInvocation(&older))
	serviceMust(t, z.Del(&destination))
	serviceMust(t, c.Del(&root))
}

package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

// (2^64*x-(2^64+1))*(x-2), bracketed by 1 and 1+2^-63.
// Promotion produces a scoped rational with both NUM and DEN physically BIG.
func algebraicLargeDyadicPromotionRoot(t *testing.T, c *AlgebraicRootCore) Algebraic {
	t.Helper()
	p, z, b := c.config.Polynomials, c.config.Rationals.Integers(), c.config.Binary
	var coefficients [3]unumeral.Int
	for i, text := range []string{"36893488147419103234", "-55340232221128654849", "18446744073709551616"} {
		serviceMust(t, z.SetString(&coefficients[i], text))
	}
	poly, e := p.NewPolynomial()
	serviceMust(t, e)
	serviceMust(t, p.SetInts(poly, coefficients[:]))
	v, e := p.wholeView(poly)
	serviceMust(t, e)
	lower := unumeral.NewBinaryInt(1)
	var upper unumeral.BinaryRational
	n := unumeral.NewInt(0)
	serviceMust(t, z.SetString(&n, "9223372036854775809"))
	serviceMust(t, b.SetIntegerPower(&upper, &n, 63))
	var root Algebraic
	serviceMust(t, c.SetRoot(&root, v, &lower, &upper, false))
	serviceMust(t, p.DelPolynomial(&poly))
	serviceMust(t, b.Del(&upper))
	serviceMust(t, z.Del(&n))
	for i := range coefficients {
		serviceMust(t, z.Del(&coefficients[i]))
	}
	return root
}

// Exercise the actual prefix and source branch through conversion, old-root
// deletion, and a refused basic-cell construction. The deliberately malformed
// denominator requested size separately guards the scoped rational destructor.
func TestAlgebraicPromotionRationalDestructorJoinsBodyAndPreservesCursor(t *testing.T) {
	c, _ := algebraicCellSourceFixtureProfile(t, true)
	root := algebraicLargeDyadicPromotionRoot(t, c)
	l, e := c.location(&root)
	serviceMust(t, e)
	r, e := c.load(l)
	serviceMust(t, e)
	v, e := c.sourcePolynomial(&r)
	serviceMust(t, e)
	sign := 1
	if r.flags&2 != 0 {
		sign = -1
	}
	interval, e := c.refineRecord(l, &r, v, sign, 0, false)
	serviceMust(t, e)
	if interval {
		t.Fatal("actual root not found")
	}
	serviceMust(t, c.beginRefinementQ())
	serviceMust(t, c.refinementConversion.run(c, r.interval.BorrowLower(), &c.localQ))
	serviceMust(t, c.Del(&root))
	z := c.config.Rationals.Integers()
	z.Budget().FailAfter(0)
	body := c.SetTake(&root, &c.localQ)
	z.Budget().FailAfter(-1)
	if body == nil || !numericPhysicalReservationRefusal(body) {
		t.Fatal("basic-cell construction did not refuse", body)
	}
	ref := c.localQ.BorrowDenominator().Cell
	if ref == 0 || z.IsFailureRef(ref) {
		t.Fatal("actual source denominator missing")
	}
	numRef := c.localQ.BorrowNumerator().Cell
	if numRef == 0 || z.IsFailureRef(numRef) {
		t.Fatal("actual BIG numerator missing")
	}
	cell := z.Cell(ref)
	requested := cell.Backing.Requested
	cell.Backing.Requested = uint64(^uint32(0)) + 1
	z.Store(ref, cell)
	cause := c.finishRefinementQ(body)
	s := &c.refinementQ
	if !errors.Is(cause, body) || !errors.Is(cause, arena.ErrBounds) || s.sourceStop == nil || s.bodyCause == nil {
		t.Fatal("source destructor discarded body or source error", cause)
	}
	if !s.active || !c.localActive || s.sourceCursor != 1 || c.localQ.BorrowNumerator().Cell != 0 || c.localQ.BorrowDenominator().Cell != ref {
		t.Fatal("numerator-before-denominator source cursor", s.sourceCursor)
	}
	if unumeral.ValidNumericRef(z.Cells, numRef) {
		t.Fatal("numerator was not released before denominator stop")
	}
	state, used, stats := *s, z.Budget().Used(), z.RuntimeOwner().Stats()
	if e := c.finishRefinementQ(nil); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("stopped source destructor restarted", e)
	}
	if state != *s || used != z.Budget().Used() || stats != z.RuntimeOwner().Stats() {
		t.Fatal("stopped source destruction changed state")
	}
	if e := c.Refine(&root); !errors.Is(e, ErrAlgebraicPending) {
		t.Fatal("pending scoped rational replaced", e)
	}
	// Only the intentionally malformed deallocation guard is restored. The
	// source cursor remains at DEN; owning cleanup has its own cursor.
	cell = z.Cell(ref)
	cell.Backing.Requested = requested
	z.Store(ref, cell)
	serviceMust(t, c.CleanupRetained())
	if s.active || c.localActive || s.ownCursor != 2 || s.sourceCursor != 1 || s.sourceStop == nil {
		t.Fatal("owning acknowledgement replayed source destruction", s.sourceCursor, s.ownCursor)
	}
	serviceMust(t, c.Del(&root))
}

// Qualified signed-host TLS accounting guard, reached through real runtime
// reallocations and actual source NUM destruction; no pending flags are minted.
func TestAlgebraicPromotionRationalDestructorActualSignedHostGuard(t *testing.T) {
	c, _, runtime := algebraicNegTLSFixture(t, nil)
	var root Algebraic
	root = algebraicLargeDyadicPromotionRoot(t, c)
	l, e := c.location(&root)
	serviceMust(t, e)
	r, e := c.load(l)
	serviceMust(t, e)
	v, e := c.sourcePolynomial(&r)
	serviceMust(t, e)
	sign := 1
	if r.flags&2 != 0 {
		sign = -1
	}
	interval, e := c.refineRecord(l, &r, v, sign, 0, false)
	serviceMust(t, e)
	if interval {
		t.Fatal("actual root not found")
	}
	serviceMust(t, c.beginRefinementQ())
	serviceMust(t, c.refinementConversion.run(c, r.interval.BorrowLower(), &c.localQ))
	serviceMust(t, c.Del(&root))
	z := c.config.Rationals.Integers()
	z.Budget().FailAfter(0)
	body := c.SetTake(&root, &c.localQ)
	z.Budget().FailAfter(-1)
	if body == nil || !numericPhysicalReservationRefusal(body) {
		t.Fatal("missing real basic-cell refusal", body)
	}
	contextTLSNearMinimum(t, runtime)
	lane, e := runtime.Allocate(8)
	serviceMust(t, e)
	delta := int64(-1000000)
	if e = runtime.Reallocate(&lane, uint64(16)+uint64(delta)-8); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	beforeBlocks := runtime.Stats().LiveBlocks
	cause := c.finishRefinementQ(body)
	if !errors.Is(cause, body) || !errors.Is(cause, arena.ErrBounds) || c.refinementQ.sourceCursor != 0 || !c.refinementQ.active {
		t.Fatal("actual source NUM guard not retained", cause, c.refinementQ.sourceCursor)
	}
	if runtime.Stats().LiveBlocks >= beforeBlocks {
		t.Fatal("source NUM guard did not consume its actual physical block")
	}
	sourceCursor := c.refinementQ.sourceCursor
	contextTLSFinish(t, runtime, c.CleanupRetained)
	if c.refinementQ.active || c.localActive || c.refinementQ.sourceCursor != sourceCursor || c.refinementQ.ownCursor != 2 {
		t.Fatal("owning continuation replayed source cursor")
	}
	contextTLSFinish(t, runtime, func() error {
		if lane == (umemory.Block{}) {
			return nil
		}
		return runtime.Deallocate(&lane)
	})
	serviceMust(t, c.Del(&root))
}

func TestAlgebraicPromotionConverterEndStopPrecedesScopedRational(t *testing.T) {
	c, p, runtime := algebraicNegTLSFixture(t, nil)
	z := c.config.Rationals.Integers()
	coefficient := [3]unumeral.Int{unumeral.NewInt(2), unumeral.NewInt(0), unumeral.NewInt(0)}
	serviceMust(t, z.SetString(&coefficient[1], "-8589934593"))
	serviceMust(t, z.SetString(&coefficient[2], "4294967296"))
	poly, e := p.NewPolynomial()
	serviceMust(t, e)
	serviceMust(t, p.SetInts(poly, coefficient[:]))
	v, e := p.wholeView(poly)
	serviceMust(t, e)
	lower, upper := unumeral.NewBinaryInt(0), unumeral.NewBinaryIntPower(1, 31)
	var root Algebraic
	serviceMust(t, c.SetRoot(&root, v, &lower, &upper, false))
	serviceMust(t, p.DelPolynomial(&poly))
	for i := range coefficient {
		serviceMust(t, z.Del(&coefficient[i]))
	}
	l, e := c.location(&root)
	serviceMust(t, e)
	r, e := c.load(l)
	serviceMust(t, e)
	v, e = c.sourcePolynomial(&r)
	serviceMust(t, e)
	interval, e := c.refineRecord(l, &r, v, 1, 0, false)
	serviceMust(t, e)
	if interval {
		t.Fatal("actual dyadic root not found")
	}
	// Warm the real integer scratch by an independent retained-manager multiply.
	// Its old receiving capacity becomes scratch, without fabricating any field.
	a, seed := unumeral.NewInt(0), unumeral.NewInt(0)
	serviceMust(t, z.SetString(&a, "18446744073709551616"))
	serviceMust(t, z.SetString(&seed, "340282366920938463463374607431768211456"))
	serviceMust(t, z.Mul(&a, &a, &seed))
	serviceMust(t, z.Del(&a))
	serviceMust(t, z.Del(&seed))
	serviceMust(t, c.beginRefinementQ())
	contextTLSNearMinimum(t, runtime)
	lane, e := runtime.Allocate(8)
	serviceMust(t, e)
	delta := int64(-1000000)
	if e = runtime.Reallocate(&lane, uint64(16)+uint64(delta)-8); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	body := c.refinementConversion.run(c, r.interval.BorrowLower(), &c.localQ)
	if !errors.Is(body, arena.ErrBounds) || c.refinementConversion.stop == nil {
		t.Fatal("actual converter END stop was not reached", body)
	}
	if !c.config.Rationals.MatchesRawPowerInvocation(&c.refinementConversion.invocation.Power) {
		t.Fatal("actual stopped power entry missing")
	}
	numRef, numVal := c.localQ.BorrowNumerator().Cell, c.localQ.BorrowNumerator().Val
	denRef, denVal := c.localQ.BorrowDenominator().Cell, c.localQ.BorrowDenominator().Val
	stats, used := runtime.Stats(), z.Budget().Used()
	cause := c.finishRefinementQ(body)
	if !errors.Is(cause, body) || !c.refinementQ.active || !c.localActive || c.refinementQ.sourceCursor != 0 || c.refinementQ.childStop == nil || c.refinementQ.sourceStop != nil || c.localQ.BorrowNumerator().Cell != numRef || c.localQ.BorrowNumerator().Val != numVal || c.localQ.BorrowDenominator().Cell != denRef || c.localQ.BorrowDenominator().Val != denVal {
		t.Fatal("older scoped rational destroyed after child END stop", cause)
	}
	if stats != runtime.Stats() || used != z.Budget().Used() || !unumeral.ValidNumericRef(z.Cells, numRef) || z.Cell(numRef).Released {
		t.Fatal("child END stop consumed older rational storage")
	}
	contextTLSFinish(t, runtime, c.CleanupRetained)
	if c.refinementQ.active || c.localActive || c.refinementQ.ownCursor != 2 || c.refinementQ.sourceCursor != 0 {
		t.Fatal("owning cleanup lost source END cursor")
	}
	contextTLSFinish(t, runtime, func() error { return c.Del(&root) })
	contextTLSFinish(t, runtime, func() error {
		if lane == (umemory.Block{}) {
			return nil
		}
		return runtime.Deallocate(&lane)
	})
}

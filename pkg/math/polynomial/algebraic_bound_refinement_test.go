package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"strconv"
	"testing"
)

func algebraicWideDyadicRoot(t *testing.T, c *AlgebraicRootCore) Algebraic {
	t.Helper()
	p := c.config.Polynomials
	poly, e := p.NewPolynomial()
	serviceMust(t, e)
	serviceMust(t, p.SetInts(poly, []unumeral.Int{unumeral.NewInt(-2), unumeral.NewInt(0), unumeral.NewInt(1)}))
	v, e := p.wholeView(poly)
	serviceMust(t, e)
	lower, upper := unumeral.NewBinaryIntPower(1, 64), unumeral.NewBinaryInt(2)
	var root Algebraic
	serviceMust(t, c.SetRoot(&root, v, &lower, &upper, true))
	serviceMust(t, p.DelPolynomial(&poly))
	return root
}

// Explicit malformed deallocation metadata guards source destruction. This
// is separate from source-valid allocation-refusal evidence.
func TestAlgebraicPrecisionBoundWidthDestructorStopRetainsOlderValues(t *testing.T) {
	c, p, _ := algebraicCompareFixture(t)
	root := algebraicWideDyadicRoot(t, c)
	p.config.Cooperation = func() error { return nil }
	q, z := c.config.Rationals, c.config.Rationals.Integers()
	count := 0
	var ref unumeral.NumericCellRef
	var requested uint64
	p.config.Checkpoint = func() bool {
		count++
		s := &c.bounds.refinement
		if s.live != 1<<boundRefinementWidth {
			t.Fatal("checkpoint width/midpoint lifetime", count, s.live)
		}
		if count == 2 {
			ref = s.values[boundRefinementWidth].BorrowNumerator().Cell
			if ref == 0 || z.IsFailureRef(ref) {
				t.Fatal("actual BIG width missing")
			}
			cell := z.Cell(ref)
			requested = cell.Backing.Requested
			cell.Backing.Requested = uint64(^uint32(0)) + 1
			z.Store(ref, cell)
			return false
		}
		return true
	}
	out := unumeral.NewRational()
	body := c.GetLowerRationalAtPrecision(&root, &out, 2)
	if !errors.Is(body, ErrUPolynomialCanceled) || !errors.Is(body, arena.ErrBounds) || count != 2 {
		t.Fatal("width source destructor error was lost", body, count)
	}
	s := &c.bounds
	if s.refinement.live != 1<<boundRefinementWidth || s.refinement.stop == nil || s.remaining != 2 {
		t.Fatal("source width failure destroyed older bound values", s.refinement.live, s.remaining)
	}
	if e := c.GetLowerRationalAtPrecision(&root, &out, 2); !errors.Is(e, ErrAlgebraicPending) {
		t.Fatal("stopped refinement replaced", e)
	}
	// Restore only the deliberately malformed requested-size metadata before
	// explicit owning continuation; no missing scratch or source value is repaired.
	cell := z.Cell(ref)
	cell.Backing.Requested = requested
	z.Store(ref, cell)
	serviceMust(t, c.CleanupRetained())
	if s.refinement.live != 0 || s.remaining != 0 || s.refinement.pending(c) {
		t.Fatal("owning cleanup retained refinement values")
	}
	p.config.Checkpoint = func() bool { return true }
	serviceMust(t, c.GetLowerRationalAtPrecision(&root, &out, 2))
	serviceMust(t, q.Del(&out))
	serviceMust(t, c.Del(&root))
}

func TestAlgebraicPrecisionBoundCancellationDestroysConstructedWidth(t *testing.T) {
	c, p, _ := algebraicCompareFixture(t)
	root := algebraicWideDyadicRoot(t, c)
	p.config.Cooperation = func() error { return nil }
	p.config.Checkpoint = func() bool {
		if c.bounds.refinement.live != 1<<boundRefinementWidth {
			t.Fatal("width not constructed before first checkpoint")
		}
		if e := c.CleanupRetained(); !errors.Is(e, ErrAlgebraicPending) {
			t.Fatal("executing callback obtained bounds cleanup", e)
		}
		return false
	}
	out := unumeral.NewRational()
	body := c.GetUpperRationalAtPrecision(&root, &out, 2)
	if !errors.Is(body, ErrUPolynomialCanceled) {
		t.Fatal(body)
	}
	if c.bounds.refinement.live != 0 || c.bounds.remaining != 0 || c.bounds.refinement.stop != nil {
		t.Fatal("ordinary cancellation retained source scoped values")
	}
	serviceMust(t, c.CleanupRetained())
	serviceMust(t, c.config.Rationals.Del(&out))
	serviceMust(t, c.Del(&root))
}

func TestAlgebraicPrecisionBoundRefinementReachedReservations(t *testing.T) {
	success, issued := false, false
	for ordinal := int64(0); ordinal < 16384 && !success; ordinal++ {
		t.Run(strconv.FormatInt(ordinal, 10), func(t *testing.T) {
			c, p, _ := algebraicCompareFixture(t)
			root := algebraicWideDyadicRoot(t, c)
			p.config.Checkpoint = func() bool { return true }
			p.config.Cooperation = func() error { return nil }
			q := c.config.Rationals
			var out [32]unumeral.Rational
			for i := range out {
				out[i] = unumeral.NewRational()
			}
			q.Budget().FailAfter(ordinal)
			var body error
			for i := range out {
				body = c.GetLowerRationalAtPrecision(&root, &out[i], 0)
				if body != nil {
					break
				}
			}
			q.Budget().FailAfter(-1)
			if body != nil && !numericPhysicalReservationRefusal(body) {
				t.Fatal("unexpected refinement refusal", body)
			}
			if unumeral.NumericAllocationPending(&c.bounds.refinement.roots) {
				issued = true
				if e := c.GetLowerRationalAtPrecision(&root, &out[0], 0); !errors.Is(e, ErrAlgebraicPending) {
					t.Fatal("replacement lost actual refinement root", e)
				}
			}
			serviceMust(t, c.CleanupRetained())
			if body == nil {
				want := unumeral.NewRational()
				serviceMust(t, q.SetString(&want, "36893488147419103233/36893488147419103232"))
				for i := range out {
					equal, e := q.Equal(&out[i], &want)
					serviceMust(t, e)
					if !equal {
						t.Fatal("independent one-bisection lower bound", i)
					}
				}
				serviceMust(t, q.Del(&want))
			}
			for i := range out {
				serviceMust(t, q.Del(&out[i]))
			}
			serviceMust(t, c.Del(&root))
			success = body == nil
		})
		if t.Failed() {
			return
		}
	}
	// MPBQ_DEBUG creates and destroys independent rational snapshots before
	// body arithmetic, warming the same source allocator/descriptor storage.
	// This concrete checked campaign reaches success without an unjoined body
	// root; do not invent one. The erased build must witness the real frontier.
	if !success || (!polynomialSourceDebug && !issued) {
		t.Fatal("missing success/actual erased-build refinement allocation root", success, issued)
	}
	t.Log("completed 32-output reservation campaign; retained body root", issued)
}

func TestAlgebraicPrecisionBoundRefinementPreservesOlderAllocation(t *testing.T) {
	c, p := algebraicCellSourceFixtureProfile(t, true)
	root := algebraicWideDyadicRoot(t, c)
	z := c.config.Rationals.Integers()
	runtime := z.RuntimeOwner()
	destination := unumeral.NewInt(0)
	var older unumeral.IntegerCopyAllocationReceipt
	calls := 0
	p.config.Cooperation = func() error { return nil }
	p.config.Checkpoint = func() bool {
		calls++
		if calls != 1 {
			t.Fatal("refinement continued past older pending allocation")
		}
		pad := algebraicCellSourceTLSBoundary(t, c)
		body := z.CopyFromOwnerWithReceipt(&destination, z, &z.Two64, &older)
		if body == nil || !z.MatchesCopyAllocationInvocation(&older) {
			t.Fatal("missing actual independently owned allocation", body)
		}
		serviceMust(t, runtime.SetMaxSize(0))
		serviceMust(t, runtime.Deallocate(&pad))
		return true
	}
	out := unumeral.NewRational()
	body := c.GetLowerRationalAtPrecision(&root, &out, 0)
	if body == nil || calls != 1 || unumeral.NumericAllocationPending(&c.bounds.refinement.roots) {
		t.Fatal("refinement acquired older allocation", body, calls)
	}
	state, used, stats := z.CopyAllocation, z.Budget().Used(), runtime.Stats()
	serviceMust(t, c.CleanupRetained())
	if !z.MatchesCopyAllocationInvocation(&older) || state != z.CopyAllocation || used != z.Budget().Used() || stats != runtime.Stats() {
		t.Fatal("refinement cleanup consumed older independent allocation")
	}
	serviceMust(t, z.CleanupCopyAllocationInvocation(&older))
	serviceMust(t, z.Del(&destination))
	serviceMust(t, c.config.Rationals.Del(&out))
	serviceMust(t, c.Del(&root))
}

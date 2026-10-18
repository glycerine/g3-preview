package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	primegen "github.com/glycerine/g3/pkg/util/prime"
)

// The eight constructor-owned integer cells exhaust the real descriptor
// limit. Small coefficient copies still finish before the interval assertion's
// exponent alignment acquires physical storage and cannot publish a cell.
func algebraicCopyBoundedTarget(t *testing.T) *AlgebraicRootCore {
	t.Helper()
	budget := arena.NewBudget(1 << 24)
	diagnostics, e := unumeral.NewRationalService(serviceConfig(budget))
	serviceMust(t, e)
	profile := umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	q, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: unumeral.IntegerConfig{
		Cells: arena.Config{PageSize: 8, SlotLimit: 8, Budget: budget},
		Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}, Profile: profile,
	}})
	serviceMust(t, e)
	p, e := NewUPolynomialManager(UPolynomialConfig{Integer: q.Integers(), Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }})
	serviceMust(t, e)
	primes, e := primegen.NewGenerator(budget)
	serviceMust(t, e)
	factors, e := NewUPolynomialFactorizationService(p, UPolynomialFactorizationConfig{Primes: primes})
	serviceMust(t, e)
	b, e := unumeral.NewBinaryRationalManager(q.Integers(), unumeral.BinaryRationalConfig{Diagnostics: diagnostics})
	serviceMust(t, e)
	intervals, e := unumeral.NewBinaryIntervalManager(b)
	serviceMust(t, e)
	allocator, e := umemory.NewSmallAllocator(umemory.SmallAllocatorConfig{Runtime: q.Integers().RuntimeOwner(), Profile: profile})
	serviceMust(t, e)
	c, e := NewAlgebraicRootCore(AlgebraicRootConfig{Rationals: q, Polynomials: p, Factors: factors, Binary: b, Intervals: intervals, Allocator: allocator, Checkpoint: p.config.Checkpoint, Cooperation: p.config.Cooperation})
	serviceMust(t, e)
	t.Cleanup(func() {
		serviceMust(t, c.Close())
		serviceMust(t, allocator.Close())
		intervals.Close()
		serviceMust(t, b.CloseChecked())
		serviceMust(t, factors.Close())
		serviceMust(t, primes.Close())
		serviceMust(t, p.Close())
		serviceMust(t, q.CloseChecked())
		serviceMust(t, diagnostics.Close())
		if budget.Used() != 0 {
			t.Fatal("bounded target leaked charged storage", budget.Used())
		}
	})
	return c
}

func TestAlgebraicCopyFromCoreBoundShiftPhysicalRefusal(t *testing.T) {
	t.Run("owned", func(t *testing.T) { algebraicCopyBoundShiftPhysicalRefusal(t, false) })
	if unumeral.SourceExtendedGCDDebug {
		t.Run("foreign-replacement", func(t *testing.T) { algebraicCopyBoundShiftPhysicalRefusal(t, true) })
	}
}

func algebraicCopyBoundShiftPhysicalRefusal(t *testing.T, replacement bool) {
	source, _ := algebraicFixture(t)
	src := algebraicCompareRoot(t, source, []int32{-1, 0, 2}, 1, 128, 1, 0, true)
	t.Cleanup(func() { serviceMust(t, source.Del(&src)) })
	target := algebraicCopyBoundedTarget(t)
	var dst Algebraic
	t.Cleanup(func() {
		serviceMust(t, target.CleanupRetained())
		serviceMust(t, target.CleanupValue(&dst))
	})
	body := target.CopyFromCore(&dst, source, &src)
	if !unumeral.SourceExtendedGCDDebug {
		serviceMust(t, body)
		if target.config.Binary.Tmp != (unumeral.Int{}) {
			t.Fatal("erased interval assertion evaluated exponent alignment")
		}
		return
	}
	z, s := target.config.Rationals.Integers(), &target.foreignCopy
	if !errors.Is(body, arena.ErrSlotLimit) || !unumeral.NumericAllocationPending(&s.roots) || s.roots.Cell.Object == (umemory.ObjectRef{}) {
		t.Fatal("integrated bound shift did not retain its physical prefix", body)
	}
	if z.MatchesCopyAllocationInvocation(&s.boundCopy) || z.MatchesCopyAllocationInvocation(&target.coefficientCopy) {
		t.Fatal("shift acquired completed SET's allocation authority")
	}
	if !s.active || source.foreignCopyBorrowers != 1 {
		t.Fatal("physical prefix lost source lifetime")
	}
	l, e := target.locationCleanup(&dst)
	serviceMust(t, e)
	r, e := target.load(l)
	serviceMust(t, e)
	if r.size != 3 || l.coefficientInitialized != 3 || *r.interval.BorrowLower() != (unumeral.BinaryRational{}) || *r.interval.BorrowUpper() != (unumeral.BinaryRational{}) {
		t.Fatal("interval assertion failure changed copied prefix or assigned bounds")
	}
	physical := s.roots.Cell.Object
	if _, e := z.Allocator.ObjectExtent(physical); e != nil {
		t.Fatal("retained shift backing is not physically live", e)
	}
	var foreign unumeral.Int
	var foreignReceipt unumeral.IntegerCopyAllocationReceipt
	var foreignPhysical umemory.ObjectRef
	if replacement {
		// An independent explicit owner acknowledges the first allocation,
		// then obtains its own failed SET allocation. The algebraic scope has
		// no authority to absorb that later child merely because it is live.
		serviceMust(t, z.CleanupPendingAllocation())
		e := z.CopyFromOwnerWithReceipt(&foreign, source.config.Rationals.Integers(), &source.config.Rationals.Integers().Two64, &foreignReceipt)
		if !errors.Is(e, arena.ErrSlotLimit) || !z.MatchesCopyAllocationInvocation(&foreignReceipt) || z.PendingCell.Object == (umemory.ObjectRef{}) || z.PendingCell.Object == physical {
			t.Fatal("independent replacement did not acquire distinct physical storage", e)
		}
		foreignPhysical = z.PendingCell.Object
	}
	serviceMust(t, target.CleanupRetained())
	if s.active || source.foreignCopyBorrowers != 0 || !replacement && z.PendingCell.Object != (umemory.ObjectRef{}) {
		t.Fatal("owning cleanup did not release exact shift prefix and source borrow")
	}
	if _, e := z.Allocator.ObjectExtent(physical); e == nil {
		t.Fatal("owning cleanup retained shift backing")
	}
	if replacement {
		if !z.MatchesCopyAllocationInvocation(&foreignReceipt) || z.PendingCell.Object != foreignPhysical {
			t.Fatal("algebraic cleanup acquired independent replacement allocation")
		}
		if _, e := z.Allocator.ObjectExtent(foreignPhysical); e != nil {
			t.Fatal("algebraic cleanup released independent physical storage", e)
		}
		serviceMust(t, z.CleanupCopyAllocationInvocation(&foreignReceipt))
		serviceMust(t, z.CleanupRawNumeral(&foreign))
	}
}

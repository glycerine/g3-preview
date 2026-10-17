package polynomial

import (
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestAlgebraicCellAllocationExactPendingCleanupAndRetry(t *testing.T) {
	for _, replace := range []bool{false, true} {
		t.Run(map[bool]string{false: "own", true: "later-foreign"}[replace], func(t *testing.T) {
			c, _ := algebraicCellSourceFixtureProfile(t, true)
			q := unumeral.NewRational()
			serviceMust(t, c.config.Rationals.SetFraction(&q, 5, 3))
			var a Algebraic
			defer func() { serviceMust(t, c.Del(&a)); serviceMust(t, c.config.Rationals.Del(&q)) }()
			pad := algebraicCellSourceTLSBoundary(t, c)
			runtime := c.config.Rationals.Integers().RuntimeOwner()
			before := q
			e := c.SetTake(&a, &q)
			allocator := c.config.Allocator
			if e == nil || a != (Algebraic{}) || q != before || !allocator.MatchesPendingAllocationOutcome(&c.cellAllocation) {
				t.Fatal("actual unjoined cell allocation missing", e)
			}
			serviceMust(t, runtime.SetMaxSize(0))
			serviceMust(t, runtime.Deallocate(&pad))
			if replace {
				serviceMust(t, allocator.CleanupPending())
				pad = algebraicCellSourceTLSBoundary(t, c)
				var later umemory.SmallAllocatorAllocationOutcome
				_, e = allocator.AllocateWithOutcome(24, &later)
				if e == nil || !allocator.MatchesPendingAllocationOutcome(&later) || allocator.MatchesPendingAllocationOutcome(&c.cellAllocation) {
					t.Fatal("actual later allocation not distinct", e)
				}
				used, stats := c.config.Rationals.Budget().Used(), runtime.Stats()
				serviceMust(t, c.CleanupRetained())
				if !allocator.MatchesPendingAllocationOutcome(&later) || used != c.config.Rationals.Budget().Used() || stats != runtime.Stats() {
					t.Fatal("stale core outcome consumed later allocation")
				}
				serviceMust(t, runtime.SetMaxSize(0))
				serviceMust(t, runtime.Deallocate(&pad))
				serviceMust(t, allocator.CleanupPending())
			} else {
				serviceMust(t, c.CleanupRetained())
				if allocator.Stats().PendingBlocks != 0 || c.cellPending.Block() != (umemory.Block{}) {
					t.Fatal("exact allocation retained after cleanup")
				}
			}
			serviceMust(t, c.SetTake(&a, &q))
			if q != unumeral.NewRational() {
				t.Fatal("successful retry did not consume input")
			}
		})
	}
}

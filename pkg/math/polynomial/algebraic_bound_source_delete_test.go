// Source: algebraic_numbers.cpp:2998-3002,3026-3030; mpbq.h:78.
package polynomial

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestAlgebraicBoundTemporaryConsumedReleaseSourceGuard(t *testing.T) {
	for _, upper := range []bool{false, true} {
		t.Run(fmt.Sprint("upper=", upper), func(t *testing.T) {
			reached := false
			for offset := int64(4); offset < 400 && !reached; offset += 4 {
				t.Run(fmt.Sprint(offset), func(t *testing.T) {
					var c *AlgebraicRootCore
					var rt *umemory.Runtime
					var lane, held umemory.Block
					armed := false
					callbacks := 0
					transport := func(f umemory.Failure) error {
						if !armed || !c.bounds.conversion.executing || c.bounds.values[0].BorrowNumerator().Cell == 0 {
							return f
						}
						armed = false
						callbacks++
						serviceMust(t, rt.SetMaxSize(0))
						lane = algebraicCellDeletionTLSLane(t, rt)
						return f
					}
					c, _, rt = algebraicNegTLSFixture(t, transport)
					root := algebraicBigBinaryBoundsRoot(t, c)
					q := c.config.Rationals
					z := q.Integers()
					out := unumeral.NewRational()
					defer func() {
						armed = false
						serviceMust(t, rt.SetMaxSize(0))
						contextTLSFinish(t, rt, c.cleanupBounds)
						contextTLSFinish(t, rt, func() error { return q.Del(&out) })
						contextTLSFinish(t, rt, func() error { return c.CleanupValue(&root) })
						for _, block := range []*umemory.Block{&lane, &held} {
							contextTLSFinish(t, rt, func() error {
								if *block == (umemory.Block{}) {
									return nil
								}
								return rt.Deallocate(block)
							})
						}
					}()
					pad := int64(100000) - rt.State().LocalSize - offset
					if pad <= 0 {
						t.Fatal(rt.State())
					}
					var e error
					held, e = rt.Allocate(uint64(pad))
					serviceMust(t, e)
					serviceMust(t, rt.SetMaxSize(1))
					armed = true
					var body error
					if upper {
						body = c.GetUpperRational(&root, &out)
					} else {
						body = c.GetLowerRational(&root, &out)
					}
					armed = false
					serviceMust(t, rt.SetMaxSize(0))
					if callbacks == 0 || c.bounds.conversion.stop != nil || c.bounds.remaining == 0 {
						return
					}
					n := c.bounds.values[0].BorrowNumerator()
					if n.Cell == 0 || !z.Cell(n.Cell).Released {
						return
					}
					reached = true
					if callbacks != 1 || c.bounds.remaining != 1 || !errors.Is(c.bounds.sourceStop, arena.ErrBounds) {
						t.Fatal("actual raw bound source stop", callbacks, c.bounds.remaining, c.bounds.sourceStop)
					}
					t.Logf("actual conversion refusal callback and bound NUM DT reached at offset %d", offset)
					if !errors.Is(body, arena.ErrBounds) {
						t.Fatal("actual temporary NUM release stop", body)
					}
					before, used, original := rt.Stats(), z.Budget().Used(), *n
					remaining := c.bounds.remaining
					if e := c.finishBounds(false); !errors.Is(e, arena.ErrInvalidSlot) {
						t.Error("literal temporary DT acknowledged consumed release", e)
					}
					if rt.Stats() != before || z.Budget().Used() != used || *n != original || c.bounds.remaining != remaining {
						t.Error("source retry changed stopped temporary")
					}
					contextTLSFinish(t, rt, c.cleanupBounds)
					if c.bounds.remaining != 0 {
						t.Fatal("owning cleanup retained bound temporary")
					}
				})
			}
			if !reached {
				t.Fatal("actual bound temporary release frontier not reached")
			}
		})
	}
}

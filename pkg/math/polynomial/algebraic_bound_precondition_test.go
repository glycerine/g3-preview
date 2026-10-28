package polynomial

// Source: algebraic_numbers.cpp:2987-3009,3014-3037; actual predicate scopes.

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestAlgebraicBoundDeclaresTemporaryBeforePredicate(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source predicate assertion compiled out")
	}
	for _, upper := range []bool{false, true} {
		t.Run(fmt.Sprint(upper), func(t *testing.T) {
			var c *AlgebraicRootCore
			active, callbacks := false, 0
			var remaining uint8
			var cleanup error
			transport := func(f umemory.Failure) error {
				if active && c.predicate.active {
					active = false
					callbacks++
					remaining = c.bounds.remaining
					cleanup = c.cleanupBounds()
				}
				return f
			}
			var rt *umemory.Runtime
			c, _, rt = algebraicNegTLSFixture(t, transport)
			root := algebraicBigBinaryBoundsRootWithMinimal(t, c, false)
			q := c.config.Rationals
			out := unumeral.NewRational()
			serviceMust(t, q.SetInt64(&out, 99))
			pad := algebraicCellSourceTLSBoundary(t, c)
			active = true
			var body error
			if upper {
				body = c.GetUpperRational(&root, &out)
			} else {
				body = c.GetLowerRational(&root, &out)
			}
			active = false
			serviceMust(t, rt.SetMaxSize(0))
			if callbacks != 1 || body == nil {
				t.Fatal("actual predicate allocation transport not reached", callbacks, body)
			}
			if remaining != 1 || !errors.Is(cleanup, ErrAlgebraicPending) {
				t.Error("older scoped BIN absent or cleaned during predicate", remaining, cleanup)
			}
			if c.bounds.remaining != 0 || c.bounds.sourceStop != nil {
				t.Error("body failure without source stop skipped older BIN DT", c.bounds.remaining, c.bounds.sourceStop)
			}
			contextTLSFinish(t, rt, c.CleanupRetained)
			serviceMust(t, q.Del(&out))
			serviceMust(t, c.Del(&root))
			serviceMust(t, rt.Deallocate(&pad))
		})
	}
}

func TestAlgebraicBoundPredicateSourceStopProtectsOlderTemporary(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source predicate assertion compiled out")
	}
	for _, owned := range []bool{false, true} {
		t.Run(fmt.Sprint("owned=", owned), func(t *testing.T) {
			reached := false
			for offset := int64(0); offset < 4096 && !reached; offset += 4 {
				t.Run(fmt.Sprint(offset), func(t *testing.T) {
					var c *AlgebraicRootCore
					var rt *umemory.Runtime
					var lane, pad umemory.Block
					armed := false
					transport := func(f umemory.Failure) error {
						if !armed || !c.refinement.executing || !c.predicate.active {
							return f
						}
						z := c.config.Rationals.Integers()
						if z.AddMulInvocation.Tmp.Cell == 0 || z.IsFailureRef(z.AddMulInvocation.Tmp.Cell) {
							return f
						}
						armed = false
						reached = true
						serviceMust(t, rt.SetMaxSize(0))
						lane = algebraicCellDeletionTLSLane(t, rt)
						return f
					}
					c, _, rt = algebraicNegTLSFixture(t, transport)
					root := algebraicBigBinaryBoundsRootWithMinimal(t, c, false)
					q := c.config.Rationals
					raw := unumeral.NewRational()
					var service *unumeral.RationalService
					var out unumeral.OwnedRational
					if owned {
						service, _ = serviceNew(t)
						var e error
						out, e = service.NewValue()
						serviceMust(t, e)
						serviceMust(t, service.SetInt64(&out, 99))
					}
					defer func() {
						armed = false
						serviceMust(t, rt.SetMaxSize(0))
						contextTLSFinish(t, rt, c.CleanupRetained)
						contextTLSFinish(t, rt, func() error { return q.Del(&raw) })
						contextTLSFinish(t, rt, func() error { return c.CleanupValue(&root) })
						if owned {
							serviceMust(t, service.Del(&out))
						}
						for _, block := range []*umemory.Block{&lane, &pad} {
							contextTLSFinish(t, rt, func() error {
								if *block == (umemory.Block{}) {
									return nil
								}
								return rt.Deallocate(block)
							})
						}
					}()
					n := int64(100000) - rt.State().LocalSize - offset - 8
					if n <= 0 {
						t.Fatal(rt.State())
					}
					var e error
					pad, e = rt.Allocate(uint64(n))
					serviceMust(t, e)
					serviceMust(t, rt.SetMaxSize(1))
					armed = true
					var body error
					if owned {
						body = c.GetUpperOwnedRational(&root, service, &out)
					} else {
						body = c.GetUpperRational(&root, &raw)
					}
					armed = false
					serviceMust(t, rt.SetMaxSize(0))
					if !reached {
						return
					}
					if !errors.Is(body, arena.ErrBounds) || c.predicate.childStop == nil || !c.predicate.active {
						t.Fatal("actual predicate/refinement source stop", body)
					}
					if c.bounds.remaining != 1 || !errors.Is(c.bounds.sourceStop, c.predicate.childStop) {
						t.Error("predicate source stop crossed older BIN DT", c.bounds.remaining, c.bounds.sourceStop)
					}
					if owned && (c.toOwnedRational.tmpDone || c.toOwnedRational.sourceCause != c.bounds.sourceStop || out.Raw.Num != unumeral.NewInt(99)) {
						t.Error("bound source stop crossed owning wrapper TMP", c.toOwnedRational.tmpDone)
					}
					values, remaining, used, stats := c.bounds.values, c.bounds.remaining, q.Budget().Used(), rt.Stats()
					if e := c.finishBounds(false); !errors.Is(e, c.predicate.childStop) {
						t.Error("source cleanup lost actual younger stop", e)
					}
					if c.bounds.values != values || c.bounds.remaining != remaining || used != q.Budget().Used() || stats != rt.Stats() {
						t.Error("source retry changed stopped bound")
					}
					prior := c.predicate
					var reported error
					_, e = c.isRationalWithSourceStop(&root, &reported)
					if e == nil || reported != nil || !reflect.DeepEqual(prior, c.predicate) {
						t.Error("refused predicate adopted older stop", e, reported)
					}
					t.Logf("actual nested refinement release stop at offset %d", offset)
				})
			}
			if !reached {
				t.Fatal("actual predicate/refinement stop not reached")
			}
		})
	}
}

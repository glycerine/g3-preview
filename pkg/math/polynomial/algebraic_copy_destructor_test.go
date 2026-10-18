package polynomial

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

// Qualified LP64 signed-host frontier: public realloc operations reach the
// existing TLS overflow guard. No source flags or private accounting counters
// are fabricated. The original mutable SET refusal then encounters a stopped
// literal scoped-mpq NUM destructor, followed by a separate owning stop.
func TestAlgebraicCopyFromCoreSourceDestructorAndOwningStop(t *testing.T) {
	reached := false
	for offset := int64(0); offset < 1024 && !reached; offset += 4 {
		t.Run(fmt.Sprint(offset), func(t *testing.T) {
			source, _ := algebraicFixture(t)
			src := algebraicForeignValue(t, source, 1)
			var target *AlgebraicRootCore
			var rt *umemory.Runtime
			var lane, pad umemory.Block
			var dst Algebraic
			armed := false
			transport := func(f umemory.Failure) error {
				if !armed || target == nil || !target.foreignCopy.running || !target.constQ.running || target.constQ.executing {
					return f
				}
				z := target.config.Rationals.Integers()
				if !unumeral.ValidNumericRef(z.Cells, target.localQ.BorrowNumerator().Cell) || !unumeral.ValidNumericRef(z.Cells, target.localQ.BorrowDenominator().Cell) {
					return f
				}
				armed, reached = false, true
				for _, e := range []error{source.Del(&src), source.Close(), source.CleanupRetained(), target.Del(&dst), target.Close(), target.CleanupRetained(), target.cleanupForeignCopy()} {
					if !errors.Is(e, ErrAlgebraicPending) {
						t.Fatal("mutable SET callback obtained a borrowed lifetime", e)
					}
				}
				serviceMust(t, rt.SetMaxSize(0))
				contextTLSNearMinimum(t, rt)
				var e error
				lane, e = rt.Allocate(8)
				serviceMust(t, e)
				delta := int64(-1000000)
				if e := rt.Reallocate(&lane, uint64(16)+uint64(delta)-8); !errors.Is(e, arena.ErrBounds) {
					t.Fatal("public TLS guard not reached", e)
				}
				return f
			}
			target, _, rt = algebraicNegTLSFixture(t, transport)
			z := target.config.Rationals.Integers()
			one := unumeral.NewInt(1)
			serviceMust(t, target.SetInteger(&dst, &one))
			sourceBindings, targetBindings := source.config.Rationals.Integers().CallerBindings(), z.CallerBindings()
			defer func() {
				armed = false
				serviceMust(t, rt.SetMaxSize(0))
				contextTLSFinish(t, rt, target.CleanupRetained)
				contextTLSFinish(t, rt, func() error { return target.CleanupValue(&dst) })
				for _, block := range []*umemory.Block{&lane, &pad} {
					contextTLSFinish(t, rt, func() error {
						if *block == (umemory.Block{}) {
							return nil
						}
						return rt.Deallocate(block)
					})
				}
				serviceMust(t, source.Del(&src))
			}()
			n := int64(100000) - rt.State().LocalSize - offset - 8
			if n <= 0 {
				t.Fatal("TLS extent", rt.State())
			}
			var e error
			pad, e = rt.Allocate(uint64(n))
			serviceMust(t, e)
			serviceMust(t, rt.SetMaxSize(1))
			armed = true
			body := target.CopyFromCore(&dst, source, &src)
			armed = false
			serviceMust(t, rt.SetMaxSize(0))
			if source.config.Rationals.Integers().CallerBindings() != sourceBindings || z.CallerBindings() != targetBindings {
				t.Fatal("copy kept synchronous numeral bindings after return")
			}
			if !reached {
				if body != nil && !numericPhysicalReservationRefusal(body) {
					t.Fatal("unexpected allocation prefix", body)
				}
				return
			}
			q, s := &target.constQ, &target.foreignCopy
			if !numericPhysicalReservationRefusal(body) || !errors.Is(body, arena.ErrBounds) || q.bodyCause == nil || q.sourceStop == nil || q.sourceCursor != 0 || !q.active || !s.active || source.foreignCopyBorrowers != 1 {
				t.Fatal("cross-owner body/destructor frontier missing", body, q.sourceCursor)
			}
			stop, bodyCause := q.sourceStop, q.bodyCause
			stats, used := rt.Stats(), z.Budget().Used()
			if again := target.finishConstQ(nil); !errors.Is(again, stop) || q.sourceCursor != 0 || q.sourceStop != stop || rt.Stats() != stats || z.Budget().Used() != used || source.foreignCopyBorrowers != 1 {
				t.Fatal("stopped source destructor replayed or released its borrow", again)
			}
			before := rt.Stats().LiveBlocks
			own := target.CleanupRetained()
			if !errors.Is(own, arena.ErrBounds) || rt.Stats().LiveBlocks >= before || !s.active || s.running || s.cleaning || source.foreignCopyBorrowers != 1 {
				t.Fatal("owning stop lost physical progress or source borrow", own)
			}
			if source.config.Rationals.Integers().CallerBindings() != sourceBindings || z.CallerBindings() != targetBindings {
				t.Fatal("stopped owning cleanup kept synchronous bindings")
			}
			if e := source.Close(); !errors.Is(e, ErrAlgebraicPending) {
				t.Fatal("stopped owning cleanup released source lifetime", e)
			}
			contextTLSFinish(t, rt, target.CleanupRetained)
			if s.active || source.foreignCopyBorrowers != 0 || q.active || target.localActive || q.sourceCursor != 0 || q.ownCursor != 2 || q.sourceStop != stop || q.bodyCause != bodyCause {
				t.Fatal("owning continuation changed source history or retained borrow")
			}
		})
	}
	if !reached {
		t.Fatal("mutable SET callback and literal NUM destructor stop not reached")
	}
}

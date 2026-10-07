// Source: src/util/buffer.h:29-59,146-151,210-242; src/math/polynomial/polynomial.cpp:3485-3488.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialContentBufferSourcePushResizeAndRetainedGrowth(t *testing.T) {
	z, budget := mpzPublicationDirectOwner(t)
	var b polynomialContentUnsignedBuffer
	b.construct(z.Memory)
	serviceMust(t, b.resize(128, 7))
	budget.FailAfter(0)
	e := b.resize(129, 9)
	if !numericPhysicalReservationRefusal(e) || b.count != 128 || b.capacity != 128 {
		t.Fatal("source push expansion prefix", b.count, b.capacity, e)
	}
	a, x := b.get(127)
	serviceMust(t, x)
	if a != 7 {
		t.Fatal("old source prefix")
	}
	raw := []util.Block{}
	if b.pending != (util.Block{}) {
		raw = append(raw, b.pending)
	}
	integerPhysicalAuditWithRaw(t, z, raw)
	budget.FailAfter(-1)
	serviceMust(t, b.close(true))
	var c polynomialContentUnsignedBuffer
	c.construct(z.Memory)
	serviceMust(t, c.resize(130, 4))
	serviceMust(t, c.resize(2, 8))
	if c.count != 2 || c.capacity != 256 {
		t.Fatal("shrink retains backing", c.count, c.capacity)
	}
	a, x = c.get(1)
	serviceMust(t, x)
	if a != 4 {
		t.Fatal("shrink does not refill old fields")
	}
	serviceMust(t, c.resize(4, 8))
	a, x = c.get(3)
	serviceMust(t, x)
	if a != 8 {
		t.Fatal("ascending fill")
	}
	c.reset()
	if c.count != 0 || c.capacity != 256 {
		t.Fatal("logical reset")
	}
	serviceMust(t, c.close(false))
}

func TestPolynomialContentBufferReturnedBlockSourceDestructorAndAbort(t *testing.T) {
	budget := arena.NewBudget(1 << 20)
	r, e := util.NewRuntime(util.RuntimeConfig{Budget: budget, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	var b polynomialContentUnsignedBuffer
	b.construct(r)
	serviceMust(t, b.resize(128, 0))
	prep, e := r.Allocate(99980 - 8)
	serviceMust(t, e)
	before := r.Stats()
	serviceMust(t, r.SetMaxSize(1))
	e = b.push(9)
	var failure util.Failure
	if !errors.As(e, &failure) || failure.Kind != util.FailureMemout || !r.State().OutOfMemory || b.pending == (util.Block{}) {
		t.Fatal("actual returned allocation", e, r.State(), b.pending)
	}
	if b.count != 128 || b.raw != (util.Block{}) || r.Stats().LiveBlocks != before.LiveBlocks+1 {
		t.Fatal("published old range", b.count, b.raw, r.Stats())
	}
	for _, raw := range []util.Block{prep, b.pending} {
		_, e = r.Size(raw)
		serviceMust(t, e)
	}
	serviceMust(t, b.close(false))
	if !b.finalized || b.pending == (util.Block{}) || r.Stats().LiveBlocks != before.LiveBlocks+1 {
		t.Fatal("ordinary source destructor consumed unpublished block")
	}
	serviceMust(t, r.SetMaxSize(0))
	serviceMust(t, b.close(true))
	if b.pending != (util.Block{}) || r.Stats().LiveBlocks != before.LiveBlocks {
		t.Fatal("owning abort did not drain exact block")
	}
	serviceMust(t, r.Deallocate(&prep))
	serviceMust(t, r.Close())
	if budget.Used() != 0 {
		t.Fatal("full refund", budget.Used())
	}
}

// Source: src/util/mpf.cpp:233-300; explicit signed-host Runtime guard profile.
// Licensed under the MIT license. Source author: Christoph Wintersteiger.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestArbitraryFloatExponentRationalSetFirstOwnRawStopHistory(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	rt, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	m, e := NewArbitraryFloatManager(IntegerConfig{Memory: rt, Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	q := NewRational()
	zero := NewRational()
	ex := NewInt(0)
	var out, after ArbitraryFloat
	serviceMust(t, m.rational.SetFraction(&q, 3, 2))
	serviceMust(t, m.MkZero(8, 24, false, &after))
	pad, e := rt.Allocate(uint64(99990 - rt.State().LocalSize - 8))
	serviceMust(t, e)
	serviceMust(t, rt.SetMaxSize(1))
	var r ArbitraryFloatRationalSetReceipt
	body := m.SetExponentRationalWithReceipt(&out, 8, 4000, ArbitraryFloatNearestEven, &ex, &q, &r)
	h, e := m.rationalSetHistory(&r)
	serviceMust(t, e)
	if body == nil || h.transferred.Block() == (memory.Block{}) || h.bodyCause == nil || !errors.Is(body, h.bodyCause) {
		t.Fatal("actual raw transfer failure", body, h)
	}
	integerPhysicalAuditWithRaw(t, m.integer, []memory.Block{pad}, &q.num, &q.den, &out.significand, &after.significand, &m.rationalSet.sig.num, &m.rationalSet.sig.den, &m.rationalSet.exp, &m.rationalSet.p, &m.rationalSet.t.num, &m.rationalSet.t.den, &m.rationalSet.sq.num, &m.rationalSet.sq.den, &m.rational.ltTmp1.num, &m.rational.ltTmp1.den, &m.rational.ltTmp2.num, &m.rational.ltTmp2.den)
	if b.Used() != rt.Stats().TotalBytes+m.OwnedStorageBytes() {
		t.Fatal("before lift cold graph")
	}
	serviceMust(t, rt.SetMaxSize(0))
	contextTLSNearMinimum(t, rt)
	lane, e := rt.Allocate(8)
	serviceMust(t, e)
	delta := int64(-99950) - rt.State().LocalSize
	if e = rt.Reallocate(&lane, uint64(16)+uint64(delta)-8); e != arena.ErrBounds {
		t.Fatal(e)
	}
	serviceMust(t, m.CleanupExponentRationalSetWithReceipt(&r, false))
	serviceMust(t, m.CleanupValue(&out))
	originalBody, originalSource := h.bodyCause, h.sourceCause
	issuedRaw := h.originalPending
	firstOwn := m.CleanupExponentRationalSetWithReceipt(&r, true)
	if firstOwn != arena.ErrBounds || h.ownCause != firstOwn || h.bodyCause != originalBody || h.originalPending != issuedRaw {
		t.Fatal("first-own source raw-release history", firstOwn, h.ownCause)
	}
	serviceMust(t, m.CleanupExponentRationalSetWithReceipt(&r, true))
	recovery, e := rt.Allocate(200000)
	serviceMust(t, e)
	b.FailAfter(0)
	serviceMust(t, m.SetExponentRational(&after, 8, 24, ArbitraryFloatNearestEven, &ex, &zero))
	b.FailAfter(-1)
	current := m.rationalSet
	used := b.Used()
	serviceMust(t, m.CleanupExponentRationalSetWithReceipt(&r, true))
	if h.bodyCause != originalBody || h.sourceCause != originalSource || h.ownCause != firstOwn || h.originalPending != issuedRaw || m.rationalSet != current || b.Used() != used {
		t.Fatal("ack/new inline invocation erased first-own record")
	}
	for _, v := range []*ArbitraryFloat{&out, &after} {
		serviceMust(t, m.CleanupValue(v))
	}
	serviceMust(t, m.rational.Del(&q))
	if e = m.SourceFinalize(); e != nil && !m.integer.sourceGuard {
		t.Fatal(e)
	}
	serviceMust(t, m.Close())
	for _, p := range []*memory.Block{&pad, &lane, &recovery} {
		contextTLSFinish(t, rt, func() error {
			if *p == (memory.Block{}) {
				return nil
			}
			return rt.Deallocate(p)
		})
	}
	contextTLSFinish(t, rt, rt.Close)
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

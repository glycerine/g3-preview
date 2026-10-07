// Source: src/util/mpf.cpp:233-300; explicit signed-host Runtime guard profile.
// Licensed under the MIT license. Source author: Christoph Wintersteiger.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestArbitraryFloatPlainRationalSetFirstOwnRawStopHistory(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	rt, e := NewRuntime(RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	m, e := NewArbitraryFloatManager(IntegerConfig{Memory: rt, Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	q := NewRational()
	zero := NewRational()
	var out, after ArbitraryFloat
	serviceMust(t, m.rational.SetFraction(&q, 3, 2))
	serviceMust(t, m.MkZero(8, 24, false, &after))
	pad, e := rt.Allocate(uint64(99990 - rt.State().LocalSize - 8))
	serviceMust(t, e)
	serviceMust(t, rt.SetMaxSize(1))
	var r ArbitraryFloatPlainRationalSetReceipt
	body := m.SetRationalWithReceipt(&out, 8, 4000, ArbitraryFloatNearestEven, &q, &r)
	h, e := m.rationalSetHistory(&r.child)
	serviceMust(t, e)
	if body == nil || h.Transferred.Block() == (Block{}) || h.bodyCause == nil || !errors.Is(body, h.bodyCause) {
		t.Fatal("actual raw transfer failure", body, h)
	}
	// The actual issued child record cannot authorize a copied parent frame.
	copied := *r.frame
	forged := r
	forged.frame = &copied
	beforeParent, beforeHistory := m.plainRationalSet, *h
	beforeUsed, beforeState := b.Used(), rt.State()
	if e = m.CleanupPlainRationalSetWithReceipt(&forged, true); e != ErrRuntimeOwner {
		t.Fatal("copied parent accepted", e)
	}
	if m.plainRationalSet != beforeParent || *h != beforeHistory || b.Used() != beforeUsed || rt.State() != beforeState {
		t.Fatal("rejected parent touched child or history")
	}
	integerPhysicalAuditWithRaw(t, m.integer, []Block{pad}, &q.Num, &q.Den, &out.significand, &after.significand, &m.rationalSet.sig.Num, &m.rationalSet.sig.Den, &m.rationalSet.exp, &m.rationalSet.p, &m.rationalSet.t.Num, &m.rationalSet.t.Den, &m.rationalSet.sq.Num, &m.rationalSet.sq.Den, &m.rational.LtTmp1.Num, &m.rational.LtTmp1.Den, &m.rational.LtTmp2.Num, &m.rational.LtTmp2.Den)
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
	serviceMust(t, m.CleanupPlainRationalSetWithReceipt(&r, false))
	serviceMust(t, m.CleanupValue(&out))
	originalBody, originalSource := h.plainBodyCause, h.plainSourceCause
	issuedRaw := h.originalPending
	firstOwn := m.CleanupPlainRationalSetWithReceipt(&r, true)
	if firstOwn != arena.ErrBounds || h.plainOwnCause != firstOwn || h.plainBodyCause != originalBody || h.originalPending != issuedRaw {
		t.Fatal("first-own source raw-release history", firstOwn, h.ownCause)
	}
	serviceMust(t, m.CleanupPlainRationalSetWithReceipt(&r, true))
	recovery, e := rt.Allocate(200000)
	serviceMust(t, e)
	b.FailAfter(0)
	serviceMust(t, m.SetRational(&after, 8, 24, ArbitraryFloatNearestEven, &zero))
	for i := 0; i < 3; i++ {
		before := b.Used()
		var erased ArbitraryFloatPlainRationalSetReceipt
		serviceMust(t, m.SetRationalWithReceipt(&after, 8, 24, ArbitraryFloatNearestEven, &zero, &erased))
		if erased != (ArbitraryFloatPlainRationalSetReceipt{}) || b.Used() != before {
			t.Fatal("inline caller created an allocation history")
		}
	}
	b.FailAfter(-1)
	current := m.plainRationalSet
	used := b.Used()
	serviceMust(t, m.CleanupPlainRationalSetWithReceipt(&r, true))
	if h.plainBodyCause != originalBody || h.plainSourceCause != originalSource || h.plainOwnCause != firstOwn || h.originalPending != issuedRaw || m.plainRationalSet != current || b.Used() != used {
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
	for _, p := range []*Block{&pad, &lane, &recovery} {
		contextTLSFinish(t, rt, func() error {
			if *p == (Block{}) {
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

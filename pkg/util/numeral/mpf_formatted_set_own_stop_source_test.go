// Source: src/util/mpf.cpp:335-374; explicit signed-host Runtime guard profile.
package numeral

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestArbitraryFloatFormattedSetFirstOwnRawStopHistory(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	m, e := NewArbitraryFloatManager(IntegerConfig{Memory: rt, Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	one := NewInt(1)
	var x, out, z, after ArbitraryFloat
	serviceMust(t, m.SetComponents(&x, 8, 4000, false, 0, &one))
	serviceMust(t, m.MkZero(8, 24, true, &z))
	_, e = m.powerTwoWrapper(3999, false)
	serviceMust(t, e)
	_, e = m.IsNaN(&x)
	serviceMust(t, e)
	_, e = m.IsInf(&x)
	serviceMust(t, e)
	_, e = m.IsNormal(&x)
	serviceMust(t, e)
	pad, e := rt.Allocate(uint64(99990 - rt.State().LocalSize - 8))
	serviceMust(t, e)
	serviceMust(t, rt.SetMaxSize(1))
	var r ArbitraryFloatFormattedSetReceipt
	body := m.SetFormatWithReceipt(&out, 8, 24, ArbitraryFloatNearestEven, &x, &r)
	h, e := m.formattedSetHistory(&r)
	serviceMust(t, e)
	if body == nil || h.Transferred.Block() == (umemory.Block{}) || h.bodyCause == nil || !errors.Is(body, h.bodyCause) {
		t.Fatal("actual raw transfer failure", body, h)
	}
	integerPhysicalAuditWithRaw(t, m.integer, []umemory.Block{pad}, &x.significand, &out.significand, &z.significand)
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
	serviceMust(t, m.CleanupFormattedSetWithReceipt(&r, false))
	serviceMust(t, m.CleanupValue(&out))
	originalBody, originalSource := h.bodyCause, h.sourceCause
	issuedRaw := h.originalPending
	firstOwn := m.CleanupFormattedSetWithReceipt(&r, true)
	if firstOwn != arena.ErrBounds || h.ownCause != firstOwn || h.bodyCause != originalBody || h.originalPending != issuedRaw {
		t.Fatal("first-own source raw-release history", firstOwn, h.ownCause)
	}
	serviceMust(t, m.CleanupFormattedSetWithReceipt(&r, true))
	recovery, e := rt.Allocate(200000)
	serviceMust(t, e)
	b.FailAfter(0)
	serviceMust(t, m.SetFormat(&after, 8, 24, ArbitraryFloatNearestEven, &z))
	b.FailAfter(-1)
	current := m.formattedSet
	used := b.Used()
	serviceMust(t, m.CleanupFormattedSetWithReceipt(&r, true))
	if h.bodyCause != originalBody || h.sourceCause != originalSource || h.ownCause != firstOwn || h.originalPending != issuedRaw || m.formattedSet != current || b.Used() != used {
		t.Fatal("ack/new inline invocation erased first-own record")
	}
	for _, v := range []*ArbitraryFloat{&x, &out, &z, &after} {
		serviceMust(t, m.CleanupValue(v))
	}
	if e = m.SourceFinalize(); e != nil && !m.integer.sourceGuard {
		t.Fatal(e)
	}
	serviceMust(t, m.Close())
	for _, p := range []*umemory.Block{&pad, &lane, &recovery} {
		contextTLSFinish(t, rt, func() error {
			if *p == (umemory.Block{}) {
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

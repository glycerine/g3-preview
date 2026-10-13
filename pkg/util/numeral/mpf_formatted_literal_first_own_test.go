// Source: src/util/mpf.cpp:194-231,233-300; signed-host first owning release.
package numeral

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestArbitraryFloatFormattedLiteralFirstOwnRawStop(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	m, e := NewArbitraryFloatManager(IntegerConfig{Memory: rt, Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	var out ArbitraryFloat
	pad, e := rt.Allocate(uint64(99990 - rt.State().LocalSize - 8))
	serviceMust(t, e)
	serviceMust(t, rt.SetMaxSize(1))
	body := m.SetFormattedLiteral(&out, 8, 4000, ArbitraryFloatNearestEven, "1.5p0")
	s := &m.formattedLiteral
	if body == nil || s.child.frame == nil || s.child.frame.Transferred.Block() == (umemory.Block{}) || !errors.Is(body, s.child.frame.bodyCause) {
		t.Fatal("actual formatted child raw transfer", body, s.child.frame, s.childInvocation)
	}
	c := &m.rationalSet
	integerPhysicalAuditWithRaw(t, m.integer, []umemory.Block{pad}, &out.significand, &s.ex, &s.q.Num, &s.q.Den, &c.sig.Num, &c.sig.Den, &c.exp, &c.p, &c.p2, &c.prev, &c.t.Num, &c.t.Den, &c.sq.Num, &c.sq.Den, &m.rational.NTmp, &m.rational.AddTmp1, &m.rational.AddTmp2, &m.rational.LtTmp1.Num, &m.rational.LtTmp1.Den, &m.rational.LtTmp2.Num, &m.rational.LtTmp2.Den)
	if b.Used() != rt.Stats().TotalBytes+m.OwnedStorageBytes() {
		t.Fatal("formatted first-own physical graph", b.Used(), rt.Stats(), m.OwnedStorageBytes())
	}
	serviceMust(t, rt.SetMaxSize(0))
	contextTLSNearMinimum(t, rt)
	lane, e := rt.Allocate(8)
	serviceMust(t, e)
	delta := int64(-99950) - rt.State().LocalSize
	if e = rt.Reallocate(&lane, uint64(16)+uint64(delta)-8); e != arena.ErrBounds {
		t.Fatal("real TLS first-own domain", e)
	}
	oldBody, oldSource, oldIssued := s.bodyCause, s.sourceCause, s.child.frame.issued
	firstOwn := m.CleanupFormattedLiteral()
	if firstOwn != arena.ErrBounds || s.ownCause != firstOwn || s.bodyCause != oldBody || s.sourceCause != oldSource || s.child.frame.issued != oldIssued {
		t.Fatal("first owning stop altered formatted history", firstOwn, s.ownCause, s.sourceCause)
	}
	serviceMust(t, m.CleanupFormattedLiteral())
	serviceMust(t, m.CleanupValue(&out))
	recovery, e := rt.Allocate(200000)
	serviceMust(t, e)
	if e := m.SourceFinalize(); e != nil && !errors.Is(e, oldSource) && !m.integer.sourceGuard {
		t.Fatal(e)
	}
	contextTLSFinish(t, rt, m.Close)
	for _, block := range []*umemory.Block{&pad, &lane, &recovery} {
		contextTLSFinish(t, rt, func() error {
			if *block == (umemory.Block{}) {
				return nil
			}
			return rt.Deallocate(block)
		})
	}
	contextTLSFinish(t, rt, rt.Close)
	if b.Used() != 0 {
		t.Fatal("formatted first-own refund", b.Used())
	}
}

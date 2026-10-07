// Source: src/util/mpf.cpp:233-300; src/util/mpz.h:156-164.
package util

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestArbitraryFloatPlainRationalSetCallbackOwnerRefusal(t *testing.T) {
	b := arena.NewBudget(1 << 27)
	var m *ArbitraryFloatManager
	var out ArbitraryFloat
	q := NewRational()
	armed := false
	calls := 0
	rt, e := NewRuntime(RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f Failure) error {
		if armed {
			calls++
			t.Logf("actual callback flags value=%v copy=%v raw=%v", m.valueExecuting, m.integer.CopyAllocation.Executing, m.integer.rawInvocationExecuting())
			beforeQ, beforeOut := q, out
			used := b.Used()
			parent := m.plainRationalSet
			scope := m.rationalSet
			iz := m.integer.SourceFinalizeStage
			qz := m.rational.cleanupStage
			for index, call := range []func() error{m.SourceFinalize, m.Close, m.integer.SourceFinalize, m.integer.CloseChecked, m.rational.SourceFinalize, m.rational.CloseChecked, func() error { return m.SetRational(&out, 8, 4000, ArbitraryFloatNearestEven, &q) }, func() error { return m.CleanupPlainRationalSet() }} {
				if err := call(); err != ErrNumericPending {
					t.Fatalf("actual callback owner refusal index%d: %v", index, err)
				}
			}
			if q != beforeQ || out != beforeOut || b.Used() != used || m.rationalSet != scope || m.plainRationalSet != parent || m.integer.SourceFinalizeStage != iz || m.rational.cleanupStage != qz {
				t.Fatal("callback mutated actual parent/older owner")
			}
		}
		return f
	}})
	serviceMust(t, e)
	m, e = NewArbitraryFloatManager(IntegerConfig{Memory: rt, Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	serviceMust(t, m.rational.SetFraction(&q, 3, 2))
	pad, e := rt.Allocate(uint64(99990 - rt.State().LocalSize - 8))
	serviceMust(t, e)
	serviceMust(t, rt.SetMaxSize(1))
	armed = true
	body := m.SetRational(&out, 8, 4000, ArbitraryFloatNearestEven, &q)
	armed = false
	if body == nil || calls != 1 {
		t.Fatal("reached real SET Power allocation", body, calls)
	}
	integerPhysicalAuditWithRaw(t, m.integer, []Block{pad}, &q.Num, &q.Den, &out.significand, &m.rationalSet.sig.Num, &m.rationalSet.sig.Den, &m.rationalSet.exp, &m.rationalSet.p, &m.rationalSet.t.Num, &m.rationalSet.t.Den, &m.rationalSet.sq.Num, &m.rationalSet.sq.Den, &m.rational.LtTmp1.Num, &m.rational.LtTmp1.Den, &m.rational.LtTmp2.Num, &m.rational.LtTmp2.Den)
	if b.Used() != rt.Stats().TotalBytes+m.OwnedStorageBytes() {
		t.Fatal("before-lift exact cold graph")
	}
	serviceMust(t, rt.SetMaxSize(0))
	serviceMust(t, rt.Deallocate(&pad))
	serviceMust(t, m.CleanupPlainRationalSet())
	serviceMust(t, m.CleanupValue(&out))
	serviceMust(t, m.CleanupRound())
	serviceMust(t, m.rational.Del(&q))
	if e = m.SourceFinalize(); e != nil && !m.integer.sourceGuard {
		t.Fatal(e)
	}
	serviceMust(t, m.Close())
	serviceMust(t, rt.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

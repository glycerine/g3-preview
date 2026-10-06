// Source: src/util/mpf.cpp:233-300; src/util/mpz.h:156-164.
// Licensed under the MIT license. Source author: Christoph Wintersteiger.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestArbitraryFloatPlainRationalSetCallbackOwnerRefusal(t *testing.T) {
	b := arena.NewBudget(1 << 27)
	var m *ArbitraryFloatManager
	var out ArbitraryFloat
	q := NewRational()
	armed := false
	calls := 0
	rt, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f memory.Failure) error {
		if armed {
			calls++
			t.Logf("actual callback flags value=%v copy=%v raw=%v", m.valueExecuting, m.integer.copyAllocation.executing, m.integer.rawInvocationExecuting())
			beforeQ, beforeOut := q, out
			used := b.Used()
			parent := m.plainRationalSet
			scope := m.rationalSet
			iz := m.integer.sourceFinalizeStage
			qz := m.rational.cleanupStage
			for index, call := range []func() error{m.SourceFinalize, m.Close, m.integer.SourceFinalize, m.integer.CloseChecked, m.rational.SourceFinalize, m.rational.CloseChecked, func() error { return m.SetRational(&out, 8, 4000, ArbitraryFloatNearestEven, &q) }, func() error { return m.CleanupPlainRationalSet() }} {
				if err := call(); err != ErrPolynomialPending {
					t.Fatalf("actual callback owner refusal index%d: %v", index, err)
				}
			}
			if q != beforeQ || out != beforeOut || b.Used() != used || m.rationalSet != scope || m.plainRationalSet != parent || m.integer.sourceFinalizeStage != iz || m.rational.cleanupStage != qz {
				t.Fatal("callback mutated actual parent/older owner")
			}
		}
		return f
	}})
	serviceMust(t, e)
	m, e = NewArbitraryFloatManager(IntegerConfig{Memory: rt, Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
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
	integerPhysicalAuditWithRaw(t, m.integer, []memory.Block{pad}, &q.num, &q.den, &out.significand, &m.rationalSet.sig.num, &m.rationalSet.sig.den, &m.rationalSet.exp, &m.rationalSet.p, &m.rationalSet.t.num, &m.rationalSet.t.den, &m.rationalSet.sq.num, &m.rationalSet.sq.den, &m.rational.ltTmp1.num, &m.rational.ltTmp1.den, &m.rational.ltTmp2.num, &m.rational.ltTmp2.den)
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

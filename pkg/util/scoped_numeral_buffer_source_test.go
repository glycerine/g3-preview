// Source: src/util/scoped_numeral_buffer.h; src/util/buffer.h.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func scopedBufferFixture(t *testing.T) (*IntegerModularManager, *scopedIntegerModularBuffer128) {
	t.Helper()
	z, _ := mpzPublicationDirectOwner(t)
	m, e := NewIntegerModularManager(z)
	serviceMust(t, e)
	b := new(scopedIntegerModularBuffer128)
	b.construct(m)
	t.Cleanup(func() { serviceMust(t, b.cleanup()); serviceMust(t, m.Close()) })
	return m, b
}
func scopedBufferAudit(t *testing.T, b *scopedIntegerModularBuffer128, extra ...*Int) {
	t.Helper()
	values := append([]Int(nil), b.values...)
	roots := append([]*Int{}, extra...)
	for i := range values {
		roots = append(roots, &values[i])
	}
	integerPhysicalAudit(t, b.manager.manager, roots...)
}
func TestScopedNumeralBufferNativeGrowthResetAndResize(t *testing.T) {
	m, b := scopedBufferFixture(t)
	for i := int32(0); i < 130; i++ {
		v := NewInt(i - 60)
		serviceMust(t, b.push(&v))
	}
	if len(b.values) != 130 {
		t.Fatal(len(b.values))
	}
	for i := uint32(0); i < 130; i++ {
		v, e := b.get(i)
		serviceMust(t, e)
		s, e := m.manager.String(&v)
		serviceMust(t, e)
		want := NewInt(int32(i) - 60)
		ws, e := m.manager.String(&want)
		serviceMust(t, e)
		if s != ws {
			t.Fatal(i, s, ws)
		}
	}
	serviceMust(t, b.shrink(2))
	serviceMust(t, b.Resize(5))
	for i := uint32(2); i < 5; i++ {
		v, e := b.get(i)
		serviceMust(t, e)
		if v != NewInt(0) {
			t.Fatal("resize zero construction", i, v)
		}
	}
	capacity := cap(b.values)
	serviceMust(t, b.reset())
	if len(b.values) != 0 || cap(b.values) != capacity {
		t.Fatal("reset must retain native backing")
	}
	serviceMust(t, b.Resize(3))
	scopedBufferAudit(t, b)
	serviceMust(t, b.sourceClose())
	if b.values != nil {
		t.Fatal("source close retained native backing")
	}
}
func TestScopedNumeralBufferBigCopiesAndFieldNormalization(t *testing.T) {
	m, b := scopedBufferFixture(t)
	z := m.manager
	var a Int
	serviceMust(t, z.Copy(&a, &z.Two64))
	serviceMust(t, b.push(&a))
	serviceMust(t, z.SetInt64(&a, -3))
	v, e := b.get(0)
	serviceMust(t, e)
	s, e := z.String(&v)
	serviceMust(t, e)
	if s != "18446744073709551616" {
		t.Fatal("source deep set", s)
	}
	serviceMust(t, b.Resize(0))
	serviceMust(t, m.SetZPUint64(7))
	serviceMust(t, z.SetInt64(&a, -11))
	serviceMust(t, b.push(&a))
	v, e = b.get(0)
	serviceMust(t, e)
	s, e = z.String(&v)
	serviceMust(t, e)
	if s != "3" {
		t.Fatal("source manager normalization", s)
	}
	scopedBufferAudit(t, b, &a)
	serviceMust(t, z.Del(&a))
}
func TestScopedNumeralBufferNativeResizeIgnoresArenaFailure(t *testing.T) {
	m, b := scopedBufferFixture(t)
	budget := m.manager.BudgetField
	before, stats := budget.Used(), m.manager.Memory.Stats()
	budget.FailAfter(0)
	serviceMust(t, b.Resize(4096))
	budget.FailAfter(-1)
	if len(b.values) != 4096 || budget.Used() != before || m.manager.Memory.Stats() != stats {
		t.Fatal(len(b.values), before, budget.Used(), stats, m.manager.Memory.Stats())
	}
}
func TestScopedNumeralBufferSourceStopBeforeOlderNumeral(t *testing.T) {
	for _, route := range []string{"reset", "shrink", "destructor"} {
		t.Run(route, func(t *testing.T) {
			m, b := scopedBufferFixture(t)
			z := m.manager
			serviceMust(t, b.push(&z.Two64))
			serviceMust(t, b.push(&z.Two64))
			older, e := b.get(1)
			serviceMust(t, e)
			var large Int
			limbs := make([]uint32, 8192)
			limbs[0], limbs[len(limbs)-1] = 17, 1
			serviceMust(t, z.SetLimbs(&large, limbs))
			t.Cleanup(func() { serviceMust(t, z.Del(&large)) })
			z.BudgetField.FailAfter(0)
			body := b.set(0, &large)
			if !numericPhysicalReservationRefusal(body) {
				t.Fatal("actual source replacement refusal", body)
			}
			first, e := b.get(0)
			serviceMust(t, e)
			if first.Cell != z.FailedRef {
				t.Fatal("actual dead token", first)
			}
			source := b.reset
			if route == "shrink" {
				source = func() error { return b.shrink(0) }
			}
			if route == "destructor" {
				source = b.sourceClose
			}
			e = source()
			now, x := b.get(1)
			serviceMust(t, x)
			if !errors.Is(e, arena.ErrInvalidSlot) || len(b.values) != 2 || now != older {
				t.Fatal("source consumed dead token or older numeral", e, len(b.values), now, older)
			}
			if again := source(); again != e {
				t.Fatal("source first cause changed", e, again)
			}
			scopedBufferAudit(t, b, &large)
			z.BudgetField.FailAfter(-1)
			serviceMust(t, b.cleanup())
			integerPhysicalAudit(t, z, &large)
		})
	}
}
func TestScopedNumeralBufferBorrowedNativeView(t *testing.T) {
	m, b := scopedBufferFixture(t)
	serviceMust(t, b.Resize(130))
	for i, v := range []int32{42, 14, 28} {
		value := NewInt(v)
		serviceMust(t, b.set(uint32(i), &value))
	}
	view, e := b.view()
	serviceMust(t, e)
	if view.Count != 130 || view.Values == nil || view.Raw != (umemory.Block{}) {
		t.Fatal("native borrowed view", view)
	}
	serviceMust(t, view.store(2, NewInt(56)))
	actual, e := b.get(2)
	serviceMust(t, e)
	if actual != NewInt(56) {
		t.Fatal("borrowed view is shadow storage", actual)
	}
	var g Int
	serviceMust(t, m.GCDManyIndexed(view, &g))
	str, e := m.manager.String(&g)
	serviceMust(t, e)
	if str != "14" {
		t.Fatal("real indexed consumer", str)
	}
	scopedBufferAudit(t, b, &g)
	serviceMust(t, m.Del(&g))
}

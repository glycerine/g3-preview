package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/cooperate"
)

func upolyAdoptionOther(t *testing.T, q *util.RationalManager, limit *control.Reslimit, coop *cooperate.Cooperation) *UPolynomialManager {
	t.Helper()
	m, e := NewUPolynomialManager(UPolynomialConfig{Integer: q.Integers(), Checkpoint: limit.Inc, Cooperation: func() error { return coop.Cooperate("upolynomial") }})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if e := m.Close(); e != nil && !errors.Is(e, arena.ErrClosed) {
			t.Error(e)
		}
	})
	return m
}
func TestUPolynomialStoreIndependentAdoptionTransferAndOldPhysicalViews(t *testing.T) {
	m, q, b, limit, coop := upolySourceOwnerServices(t)
	other := upolyAdoptionOther(t, q, limit, coop)
	p, e := m.NewPolynomial()
	if e != nil {
		t.Fatal(e)
	}
	source, e := other.NewPolynomial()
	if e != nil {
		t.Fatal(e)
	}
	var n util.Int
	if e = q.Integers().SetString(&n, "123456789012345678901234567890"); e != nil {
		t.Fatal(e)
	}
	if e = other.SetInts(source, []util.Int{n}); e != nil {
		t.Fatal(e)
	}
	if e = q.Integers().Del(&n); e != nil {
		t.Fatal(e)
	}
	v, e := other.View(source, 0, 1)
	if e != nil {
		t.Fatal(e)
	}
	store, e := other.CoefficientStore()
	if e != nil {
		t.Fatal(e)
	}
	roots := m.Stats().IntegerLiveCells
	if e = m.SwapFrom(p, other, source); e != nil {
		t.Fatal(e)
	}
	if m.Stats().IntegerLiveCells != roots {
		t.Fatal("transfer copied numeral roots")
	}
	if e = other.DelPolynomial(&source); e != nil {
		t.Fatal(e)
	}
	if e = other.Close(); e != nil {
		t.Fatal(e)
	}
	st, e := store.Stats()
	if e != nil || st.Owners != 1 || st.OwnedNumeralCells != 1 {
		t.Fatal("actual imported retention", st, e)
	}
	if e = m.ReadView(v, 0, &n); e != nil {
		t.Fatal("old source pointer survives actual ownership swap", e)
	}
	text, e := q.Integers().String(&n)
	if e != nil || text != "123456789012345678901234567890" {
		t.Fatal(text, e)
	}
	if e = q.Integers().Del(&n); e != nil {
		t.Fatal(e)
	}
	// Growth allocates new primary storage and frees the old imported buffer.
	if e = m.Reserve(p, 3); e != nil {
		t.Fatal(e)
	}
	if e = m.ReadView(v, 0, &n); !errors.Is(e, ErrUPolynomialOwner) {
		t.Fatal("actual imported free invalidates old view", e)
	}
	st, e = store.Stats()
	if e != nil || st.LiveBackings != 0 || st.OwnedNumeralCells != 0 {
		t.Fatal(st, e)
	}
	b.FailAfter(0)
	if e = m.BorrowCoefficientStore(store); e != nil {
		t.Fatal("existing binding0alloc", e)
	}
	b.FailAfter(-1)
	if e = m.DelPolynomial(&p); e != nil {
		t.Fatal(e)
	}
	if e = m.Close(); e != nil {
		t.Fatal(e)
	}
	if _, e = store.Stats(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}
func TestUPolynomialStoreAdoptionFailurePrefixAndCohortGraph(t *testing.T) {
	m, q, b, limit, coop := upolySourceOwnerServices(t)
	other := upolyAdoptionOther(t, q, limit, coop)
	p := upolySourceNew(t, m, []int32{1, 2})
	source := upolySourceNew(t, other, []int32{3, 4})
	av, bv := mustUPolyView(t, m, p), mustUPolyView(t, other, source)
	sa, e := m.CoefficientStore()
	if e != nil {
		t.Fatal(e)
	}
	sb, e := other.CoefficientStore()
	if e != nil {
		t.Fatal(e)
	}
	baseA, e := sa.Stats()
	if e != nil {
		t.Fatal(e)
	}
	baseB, e := sb.Stats()
	if e != nil {
		t.Fatal(e)
	}
	external := b.Used() - m.Stats().ExclusiveBytes - other.Stats().ExclusiveBytes - baseA.TotalBytes - baseB.TotalBytes
	b.FailAfter(1)
	err := m.SwapFrom(p, other, source)
	if !errors.Is(err, arena.ErrInjectedFailure) {
		t.Fatal(err)
	}
	a, e := sa.Stats()
	if e != nil {
		t.Fatal(e)
	}
	bb, e := sb.Stats()
	if e != nil {
		t.Fatal(e)
	}
	// First destination binding is retained; second fails before scalar swap.
	if a.Owners != 1 || bb.Owners != 2 || b.Used() != external+m.Stats().ExclusiveBytes+other.Stats().ExclusiveBytes+a.TotalBytes+bb.TotalBytes {
		t.Fatal("real cold retained prefix exact graph", a, bb, m.Stats(), other.Stats(), b.Used())
	}
	if ok, e := m.EqualViews(av, mustUPolyView(t, m, p)); e != nil || !ok {
		t.Fatal(ok, e)
	}
	if ok, e := other.EqualViews(bv, mustUPolyView(t, other, source)); e != nil || !ok {
		t.Fatal(ok, e)
	}
	b.FailAfter(-1)
	if e = m.SwapFrom(p, other, source); e != nil {
		t.Fatal(e)
	}
	if ok, e := m.EqualViews(bv, mustUPolyView(t, m, p)); e != nil || !ok {
		t.Fatal(ok, e)
	}
	if ok, e := other.EqualViews(av, mustUPolyView(t, other, source)); e != nil || !ok {
		t.Fatal(ok, e)
	}
}
func TestUPolynomialStoreRelatedInheritsImportedBindingsAndUnwinds(t *testing.T) {
	m, q, b, limit, coop := upolySourceOwnerServices(t)
	other := upolyAdoptionOther(t, q, limit, coop)
	p := upolySourceNew(t, other, []int32{5, 6})
	v := mustUPolyView(t, other, p)
	s, e := other.CoefficientStore()
	if e != nil {
		t.Fatal(e)
	}
	if e = m.BorrowCoefficientStore(s); e != nil {
		t.Fatal(e)
	}
	primary, e := m.CoefficientStore()
	if e != nil {
		t.Fatal(e)
	}
	primaryBefore, e := primary.Stats()
	if e != nil {
		t.Fatal(e)
	}
	refused, finished := 0, 0
	used := b.Used()
	before, e := s.Stats()
	if e != nil {
		t.Fatal(e)
	}
	for ordinal := int64(0); ordinal < 32; ordinal++ {
		b.FailAfter(ordinal)
		child, err := NewRelatedUPolynomialManager(m)
		now, e := s.Stats()
		primaryNow, pe := primary.Stats()
		if pe != nil {
			t.Fatal(pe)
		}
		if e != nil {
			t.Fatal(e)
		}
		if err != nil {
			refused++
		}
		if err != nil && (now != before || primaryNow != primaryBefore || b.Used() != used) {
			t.Fatal("constructor retained context unwind", ordinal, err, now, b.Used(), used)
		}
		b.FailAfter(-1)
		if err == nil {
			finished++
			var n util.Int
			if e = child.ReadView(v, 1, &n); e != nil {
				t.Fatal(e)
			}
			val, e := q.Integers().Int64(&n)
			if e != nil || val != 6 {
				t.Fatal(val, e)
			}
			if e = q.Integers().Del(&n); e != nil {
				t.Fatal(e)
			}
			if e = child.Close(); e != nil {
				t.Fatal(e)
			}
			break
		}
		if !errors.Is(err, arena.ErrInjectedFailure) {
			t.Fatal(err)
		}
	}
	if refused == 0 || finished != 1 {
		t.Fatal("actual refusal/success endpoints", refused, finished)
	}
}

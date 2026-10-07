// Licensed under the MIT license.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestUPolynomialStoreRelatedConstructorFailureUnwinds(t *testing.T) {
	m, _, b := upolySourceOwner(t)
	s, e := m.CoefficientStore()
	if e != nil {
		t.Fatal(e)
	}
	before, e := s.Stats()
	if e != nil {
		t.Fatal(e)
	}
	used := b.Used()
	refused, succeeded := 0, 0
	for ordinal := int64(0); ordinal < 32; ordinal++ {
		b.FailAfter(ordinal)
		child, err := NewRelatedUPolynomialManager(m)
		immediate, e := s.Stats()
		if e != nil {
			t.Fatal(e)
		}
		if err != nil {
			refused++
			if !errors.Is(err, arena.ErrInjectedFailure) || immediate != before || b.Used() != used {
				t.Fatalf("ordinal%d: %v stats%+v before%+v bytes%d/%d", ordinal, err, immediate, before, b.Used(), used)
			}
		} else {
			succeeded++
			if immediate.Owners != before.Owners+1 {
				t.Fatal(immediate)
			}
		}
		b.FailAfter(-1)
		if child != nil {
			if e = child.Close(); e != nil {
				t.Fatal(e)
			}
		}
		if b.Used() != used {
			t.Fatal("constructor graph not refunded", b.Used(), used)
		}
		if succeeded != 0 {
			break
		}
	}
	if refused == 0 || succeeded != 1 {
		t.Fatal(refused, succeeded)
	}
}
func TestUPolynomialStoreCopiedClosedGuardsAndDistinctOrdinary(t *testing.T) {
	m, q, _, limit, coop := upolySourceOwnerServices(t)
	s, e := m.CoefficientStore()
	if e != nil {
		t.Fatal(e)
	}
	copied := *m
	if _, e = NewRelatedUPolynomialManager(&copied); !errors.Is(e, ErrUPolynomialOwner) {
		t.Fatal(e)
	}
	copiedStore := *s
	if _, e = copiedStore.Stats(); !errors.Is(e, ErrUPolynomialOwner) {
		t.Fatal(e)
	}
	other, e := NewUPolynomialManager(UPolynomialConfig{Integer: q.Integers(), Checkpoint: limit.Inc, Cooperation: func() error { return coop.Cooperate("upolynomial") }})
	if e != nil {
		t.Fatal(e)
	}
	os, e := other.CoefficientStore()
	if e != nil || os == s {
		t.Fatal("ordinary stores must stay distinct", e)
	}
	p, e := m.NewPolynomial()
	if e != nil {
		t.Fatal(e)
	}
	if e = m.SetInts(p, []util.Int{util.NewInt(5)}); e != nil {
		t.Fatal(e)
	}
	v, e := m.View(p, 0, 1)
	if e != nil {
		t.Fatal(e)
	}
	var out util.Int
	if e = other.ReadView(v, 0, &out); !errors.Is(e, ErrUPolynomialOwner) {
		t.Fatal("independent store origin collision", e)
	}
	if e = m.DelPolynomial(&p); e != nil {
		t.Fatal(e)
	}
	if e = other.Close(); e != nil {
		t.Fatal(e)
	}
	if e = m.Close(); e != nil {
		t.Fatal(e)
	}
	if _, e = NewRelatedUPolynomialManager(m); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	if _, e = s.Stats(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}
func TestUPolynomialStoreGrowthGenerationAndRealCloseGuards(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	child, e := NewRelatedUPolynomialManager(m)
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if e := child.Close(); e != nil {
			t.Error(e)
		}
	}()
	p, e := m.NewPolynomial()
	if e != nil {
		t.Fatal(e)
	}
	if e = m.SetInts(p, []util.Int{util.NewInt(1), util.NewInt(2)}); e != nil {
		t.Fatal(e)
	}
	v, e := m.View(p, 0, 2)
	if e != nil {
		t.Fatal(e)
	}
	var out util.Int
	if e = child.ReadView(v, 1, &out); e != nil {
		t.Fatal(e)
	}
	if e = m.Close(); !errors.Is(e, ErrUPolynomialLive) {
		t.Fatal(e)
	}
	if e = m.Reserve(p, 3); e != nil {
		t.Fatal(e)
	}
	if e = child.ReadView(v, 0, &out); !errors.Is(e, ErrUPolynomialOwner) {
		t.Fatal("growth preserves no old borrow", e)
	}
	v, e = m.View(p, 0, 3)
	if e != nil {
		t.Fatal(e)
	}
	if e = m.DelPolynomial(&p); e != nil {
		t.Fatal(e)
	}
	if e = child.ReadView(v, 0, &out); !errors.Is(e, ErrUPolynomialOwner) {
		t.Fatal("free stale", e)
	}
	seq, e := m.NewSequence()
	if e != nil {
		t.Fatal(e)
	}
	if e = m.Close(); !errors.Is(e, ErrUPolynomialLive) {
		t.Fatal(e)
	}
	if e = seq.Close(); e != nil {
		t.Fatal(e)
	}
	if e = m.Close(); e != nil {
		t.Fatal(e)
	}
}

func TestUPolynomialStoreSingleChargeCohortGraph(t *testing.T) {
	m, _, b := upolySourceOwner(t)
	store, e := m.CoefficientStore()
	if e != nil {
		t.Fatal(e)
	}
	initial, e := store.Stats()
	if e != nil {
		t.Fatal(e)
	}
	parent := m.Stats()
	numericExternal := b.Used() - parent.ExclusiveBytes - initial.TotalBytes
	child, e := NewRelatedUPolynomialManager(m)
	if e != nil {
		t.Fatal(e)
	}
	p, e := child.NewPolynomial()
	if e != nil {
		t.Fatal(e)
	}
	if e = child.SetInts(p, []util.Int{util.NewInt(1), util.NewInt(2), util.NewInt(3)}); e != nil {
		t.Fatal(e)
	}
	graph, e := store.Stats()
	if e != nil {
		t.Fatal(e)
	}
	a, c := m.Stats(), child.Stats()
	if b.Used() != numericExternal+a.ExclusiveBytes+c.ExclusiveBytes+graph.TotalBytes {
		t.Fatal("cohort exact physical charge", b.Used(), numericExternal, a, c, graph)
	}
	if a.SharedStoreBytes != graph.TotalBytes || c.SharedStoreBytes != graph.TotalBytes || a.TotalBytes != a.ExclusiveBytes+graph.TotalBytes || c.TotalBytes != c.ExclusiveBytes+graph.TotalBytes {
		t.Fatal("descriptive and additive stats", a, c, graph)
	}
	if graph.Owners != 2 || graph.LiveBackings != 1 || graph.OwnedNumeralCells != 0 {
		t.Fatal(graph)
	}
	if e = child.DelPolynomial(&p); e != nil {
		t.Fatal(e)
	}
	if e = child.Close(); e != nil {
		t.Fatal(e)
	}
	final, e := store.Stats()
	if e != nil {
		t.Fatal(e)
	}
	if final.Owners != 1 || final.LiveBackings != 0 || final.BackingBytes != 0 {
		t.Fatal(final)
	}
}

func TestUPolynomialStoreRelatedExtendedGCDIntermediateProvenance(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	child, e := NewRelatedUPolynomialManager(m)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if e := child.Close(); e != nil {
			t.Error(e)
		}
	})
	if e = child.SetZPUint64(7); e != nil {
		t.Fatal(e)
	}
	a := upolySourceNew(t, child, []int32{1, 0, 1})
	bv := upolySourceNew(t, child, []int32{1, 1})
	u := upolySourceNew(t, child, nil)
	v := upolySourceNew(t, child, nil)
	d := upolySourceNew(t, child, nil)
	// Actual source helper cleanup destroys all polynomial owners before the child.
	if e = child.ExtendedGCD(a, bv, u, v, d); e != nil {
		t.Fatal("actual sibling intermediate raw view", e)
	}
	for _, pair := range []struct {
		actual   UPolynomial
		expected []int32
	}{{u, []int32{-3}}, {v, []int32{-3, 3}}, {d, []int32{1}}} {
		expected := upolySourceNew(t, child, pair.expected)
		ok, e := child.Equal(pair.actual, expected)
		if e != nil || !ok {
			t.Fatal("frozen mod7 Bezout coefficients", pair.expected, ok, e)
		}
	}
}

func TestUPolynomialStoreBigChildRootsSurviveParentClose(t *testing.T) {
	m, q, _ := upolySourceOwner(t)
	child, e := NewRelatedUPolynomialManager(m)
	if e != nil {
		t.Fatal(e)
	}
	p, e := child.NewPolynomial()
	if e != nil {
		t.Fatal(e)
	}
	var a util.Int
	if e = q.Integers().SetString(&a, "123456789012345678901234567890"); e != nil {
		t.Fatal(e)
	}
	if e = child.SetInts(p, []util.Int{a}); e != nil {
		t.Fatal(e)
	}
	if e = q.Integers().Del(&a); e != nil {
		t.Fatal(e)
	}
	store, e := m.CoefficientStore()
	if e != nil {
		t.Fatal(e)
	}
	before, e := store.Stats()
	if e != nil || before.OwnedNumeralCells != 1 {
		t.Fatal(before, e)
	}
	if e = m.Close(); e != nil {
		t.Fatal(e)
	}
	after, e := store.Stats()
	if e != nil || after.OwnedNumeralCells != 1 || after.BackingBytes != before.BackingBytes {
		t.Fatal(after, e)
	}
	if e = child.ReadCoefficient(p, 0, &a); e != nil {
		t.Fatal(e)
	}
	str, e := q.Integers().String(&a)
	if e != nil || str != "123456789012345678901234567890" {
		t.Fatal(str, e)
	}
	if e = q.Integers().Del(&a); e != nil {
		t.Fatal(e)
	}
	if e = child.DelPolynomial(&p); e != nil {
		t.Fatal(e)
	}
	after, e = store.Stats()
	if e != nil || after.OwnedNumeralCells != 0 || after.LiveBackings != 0 {
		t.Fatal(after, e)
	}
	if e = child.Close(); e != nil {
		t.Fatal(e)
	}
}

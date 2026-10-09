package polynomial

import (
	"github.com/glycerine/g3/pkg/util"
	"testing"
)

func TestUPolynomialNativeDegreeSubsetOracle(t *testing.T) {
	m, q, b := upolySourceOwner(t)
	fs, e := m.NewFactors()
	serviceMust(t, e)
	defer func() { serviceMust(t, fs.Close()) }()
	b.FailAfter(1) // Only the actual public degree owner, no degree-flag backing.
	ds, e := NewUPolynomialDegreeSet(fs)
	b.FailAfter(-1)
	serviceMust(t, e)
	defer func() { b.FailAfter(-1); serviceMust(t, ds.Close()) }()
	used, before := b.Used(), q.Integers().Memory.Stats()
	b.FailAfter(0)
	expected := map[uint32]bool{0: true}
	total := uint32(0)
	for _, degree := range []uint32{0, 1, 31, 32, 33, 65, 129, 4097, 65} {
		next := make(map[uint32]bool)
		for k := range expected {
			next[k] = true
			next[k+degree] = true
		}
		expected = next
		total += degree
		serviceMust(t, ds.state.includeDegree(degree))
		max, e := ds.MaxDegree()
		if e != nil || max != total {
			t.Fatal(max, total, e)
		}
		for k := uint32(0); k <= total; k++ {
			got, e := ds.InSet(k)
			if e != nil || got != expected[k] {
				t.Fatal(degree, k, got, expected[k], e)
			}
		}
	}
	if b.Used() != used || q.Integers().Memory.Stats() != before || ds.Stats().WordBytes != 0 {
		t.Fatal("native degree storage charged Runtime")
	}
}

func TestUPolynomialNativeFactorGrowthMovesBIGAndPreservesViews(t *testing.T) {
	m, q, b := upolySourceOwner(t)
	fs, e := m.NewFactors()
	serviceMust(t, e)
	defer func() { b.FailAfter(-1); serviceMust(t, fs.Close()) }()
	big := upolySourceNew(t, m, nil)
	var coefficient util.Int
	serviceMust(t, q.Integers().SetUint64(&coefficient, 1<<40))
	serviceMust(t, m.SetInts(big, []util.Int{coefficient, util.NewInt(1)}))
	serviceMust(t, q.Integers().Del(&coefficient))
	serviceMust(t, fs.PushCopy(mustUPolyView(t, m, big), 1))
	view, e := fs.FactorView(0)
	serviceMust(t, e)
	first, e := m.read(view, 0)
	serviceMust(t, e)
	token := first.NativeCellToken()
	old := fs.factors
	small := upolySourceNew(t, m, []int32{2, 1})
	used, before := b.Used(), q.Integers().Memory.Stats()
	b.FailAfter(0)
	for i := uint32(1); i <= 4097; i++ {
		serviceMust(t, fs.PushCopy(mustUPolyView(t, m, small), i%7+1))
	}
	for i := range old {
		if old[i] != (upolyVectorState{}) {
			t.Fatal("relocated old record still owns backing", old[i])
		}
	}
	for i := uint32(1); i <= 4097; i++ {
		mult, e := fs.Multiplicity(i)
		if e != nil || mult != i%7+1 {
			t.Fatal(i, mult, e)
		}
		v, e := fs.FactorView(i)
		serviceMust(t, e)
		row, e := m.read(v, 0)
		serviceMust(t, e)
		if *row != util.NewInt(2) {
			t.Fatal(i, *row)
		}
	}
	first, e = m.read(view, 0)
	serviceMust(t, e)
	if first.NativeCellToken() != token || !q.Integers().IsLiveCellToken(token) {
		t.Fatal("factor relocation lost actual BIG owner")
	}
	stats := fs.Stats()
	if stats.OuterBytes != 0 || stats.MultiplicityBytes != 0 || stats.TotalBytes != stats.OwnerBytes || b.Used() != used || q.Integers().Memory.Stats() != before {
		t.Fatal("native factor growth charged Runtime", stats)
	}
	serviceMust(t, fs.Clear())
	if q.Integers().IsLiveCellToken(token) {
		t.Fatal("clear retained actual factor BIG")
	}
	b.FailAfter(-1)
}

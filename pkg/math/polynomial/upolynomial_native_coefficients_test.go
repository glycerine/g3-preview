package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestUPolynomialNativeCoefficientScratchGrowthAndReset(t *testing.T) {
	m, q, b, _, _ := upolySourceOwnerServices(t)
	states := []*upolyVectorState{&m.basic, &m.div1, &m.div2, &m.exactDiv, &m.gcd1, &m.gcd2, &m.cra, &m.sqf1, &m.sqf2, &m.pw, &m.db, &m.dbab1, &m.dbab2, &m.tr, &m.push}
	for i := range m.mgcd {
		states = append(states, &m.mgcd[i])
	}
	before, used := q.Integers().Memory.Stats(), b.Used()
	b.FailAfter(0)
	t.Cleanup(func() { b.FailAfter(-1) })
	for _, s := range states {
		for _, n := range []uint32{4097, 0, 129, 1, 33} {
			if e := m.reserve(s, n); e != nil {
				t.Fatal("native growth", n, e)
			}
			a := m.values(s)
			for i := uint32(0); i < n; i++ {
				if a[i] != (util.Int{}) {
					t.Fatal("new/default field", n, i, a[i])
				}
				a[i] = util.NewInt(int32(37*i) - 128)
			}
			for i := uint32(0); i < n; i++ {
				row, e := m.read(m.stateView(s), i)
				if e != nil || *row != util.NewInt(int32(37*i)-128) {
					t.Fatal("independent row", n, i, row, e)
				}
			}
			if n != 0 {
				if _, e := m.read(m.stateView(s), n); !errors.Is(e, arena.ErrBounds) {
					t.Fatal("borrowed bounds", e)
				}
			}
			if e := m.shrink(s, n/2); e != nil {
				t.Fatal(e)
			}
			for i := n / 2; i < n; i++ {
				if a[i] != util.NewInt(int32(37*i)-128) {
					t.Fatal("inline Del must preserve its scalar field", n, i, a[i])
				}
			}
			if e := m.reset(s); e != nil {
				t.Fatal(e)
			}
			if s.size != 0 {
				t.Fatal("logical reset", s.size)
			}
		}
		m.freeBacking(s)
	}
	if q.Integers().Memory.Stats() != before || b.Used() != used {
		t.Fatal("native coefficients changed Runtime accounting")
	}
	ss, e := m.store.Stats()
	if e != nil || ss.LiveBackings != 0 || ss.BackingBytes != 0 {
		t.Fatal("native backing cleanup", ss, e)
	}
}

func TestUPolynomialNativeCoefficientGrowthMovesActualBIGAndView(t *testing.T) {
	m, q, b, _, _ := upolySourceOwnerServices(t)
	p, e := m.NewPolynomial()
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		b.FailAfter(-1)
		if p != (UPolynomial{}) {
			if e := m.DelPolynomial(&p); e != nil {
				t.Error(e)
			}
		}
	})
	var original util.Int
	if e := q.Integers().SetInt64(&original, 1<<40); e != nil {
		t.Fatal(e)
	}
	if e := m.SetInts(p, []util.Int{original}); e != nil {
		t.Fatal(e)
	}
	if e := q.Integers().Del(&original); e != nil {
		t.Fatal(e)
	}
	d, e := m.descriptor(p)
	if e != nil {
		t.Fatal(e)
	}
	token := m.values(&d.state)[0]
	old, e := m.View(p, 0, 1)
	if e != nil {
		t.Fatal(e)
	}
	before, used := q.Integers().Memory.Stats(), b.Used()
	cells := q.Integers().StorageStats()
	b.FailAfter(0)
	if e := m.Reserve(p, 4097); e != nil {
		t.Fatal("native relocation", e)
	}
	if m.values(&d.state)[0] != token || q.Integers().StorageStats() != cells {
		t.Fatal("relocation copied/lost actual BIG ownership")
	}
	if _, e := m.read(old, 0); !errors.Is(e, ErrUPolynomialOwner) {
		t.Fatal("relocated view remains live", e)
	}
	v, e := m.View(p, 0, 4097)
	if e != nil {
		t.Fatal(e)
	}
	row, e := m.read(v, 0)
	if e != nil {
		t.Fatal(e)
	}
	value, e := q.Integers().Int64(row)
	if e != nil || value != 1<<40 {
		t.Fatal("actual moved BIG value", value, e)
	}
	for i := uint32(1); i < v.count; i++ {
		row, e := m.read(v, i)
		if e != nil || *row != (util.Int{}) {
			t.Fatal("zero extension", i, row, e)
		}
	}
	if q.Integers().Memory.Stats() != before || b.Used() != used {
		t.Fatal("native BIG relocation changed Runtime accounting")
	}
	b.FailAfter(-1)
	if e := m.Reset(p); e != nil {
		t.Fatal(e)
	}
	if q.Integers().IsLiveCellToken(token.NativeCellToken()) {
		t.Fatal("reset retained actual BIG")
	}
}

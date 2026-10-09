package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestUPolynomialNativeDescriptorAndBackingDirectories(t *testing.T) {
	m, _, b, _, _ := upolySourceOwnerServices(t)
	const count = 4097
	owned := make([]UPolynomial, count)
	t.Cleanup(func() {
		b.FailAfter(-1)
		for i := range owned {
			if owned[i] != (UPolynomial{}) {
				if e := m.DelPolynomial(&owned[i]); e != nil {
					t.Error(e)
				}
			}
		}
	})
	before, used := m.IntegerManager().Memory.Stats(), b.Used()
	b.FailAfter(0)
	for i := range owned {
		p, e := m.NewPolynomial()
		if e != nil {
			t.Fatal("native descriptor growth", i, e)
		}
		owned[i] = p
	}
	if m.Stats().LivePolynomials != count || m.Stats().PageBytes != 0 || m.Stats().DirectoryBytes != 0 || b.Used() != used || m.IntegerManager().Memory.Stats() != before {
		t.Fatal("native descriptor directory changed accounting", m.Stats())
	}
	stale := owned[0]
	if e := m.DelPolynomial(&owned[0]); e != nil {
		t.Fatal(e)
	}
	var e error
	owned[0], e = m.NewPolynomial()
	if e != nil || owned[0].slot != stale.slot || owned[0].generation == stale.generation {
		t.Fatal("native descriptor recycling", owned[0], stale, e)
	}
	if _, e := m.Size(stale); !errors.Is(e, ErrUPolynomialOwner) {
		t.Fatal("stale scalar owner", e)
	}
	// Both backing directories and coefficient arrays grow natively without
	// consuming numeric allocation reservations.
	for i := 0; i < 257; i++ {
		b.FailAfter(0)
		if e := m.Reserve(owned[i], 1); e != nil {
			t.Fatal("native backing growth reserved bytes", i, e)
		}
		b.FailAfter(-1)
		value := util.NewInt(int32(17*i + 3))
		if e := m.SetCoefficient(owned[i], 0, &value); e != nil {
			t.Fatal(e)
		}
	}
	for i := 0; i < 257; i++ {
		var value util.Int
		if e := m.ReadCoefficient(owned[i], 0, &value); e != nil || value != util.NewInt(int32(17*i+3)) {
			t.Fatal("independent backing row", i, value, e)
		}
	}
	stats, e := m.store.Stats()
	if e != nil || stats.LiveBackings != 257 || stats.DirectoryBytes != 0 {
		t.Fatal("native backing directory", stats, e)
	}
}

func TestUPolynomialNativeBorrowedStoreDirectory(t *testing.T) {
	m, q, b, limit, cooperation := upolySourceOwnerServices(t)
	foreign := make([]*UPolynomialManager, 33)
	for i := range foreign {
		foreign[i] = upolyAdoptionOther(t, q, limit, cooperation)
	}
	before, used := q.Integers().Memory.Stats(), b.Used()
	b.FailAfter(0)
	t.Cleanup(func() { b.FailAfter(-1) })
	for repeat := 0; repeat < 2; repeat++ {
		for i, other := range foreign {
			if e := m.BorrowCoefficientStore(other.store); e != nil {
				t.Fatal("native store binding", i, e)
			}
			ss, e := other.store.Stats()
			if e != nil || ss.Owners != 2 || m.boundStore(other.store.origin) != other.store {
				t.Fatal("exact retained store", i, ss, e)
			}
		}
	}
	if len(m.bindings) != len(foreign) || b.Used() != used || q.Integers().Memory.Stats() != before || m.Stats().DirectoryBytes != 0 {
		t.Fatal("native binding directory changed Runtime accounting")
	}
}

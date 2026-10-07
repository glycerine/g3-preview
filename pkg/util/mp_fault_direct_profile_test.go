package util

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

// Explicit supported source allocator profile for action-timing refusal tests.
// Default pooling and its allocation-free warmed paths are exercised separately.
func rationalPublicationDirectOwner(t *testing.T, synch bool) (*RationalManager, *arena.Budget) {
	t.Helper()
	b := arena.NewBudget(1 << 24)
	m, e := NewRationalManager(RationalConfig{Integers: IntegerConfig{Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true, Windows: false}}, Synchronized: synch})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		m.Close()
		if b.Used() != 0 {
			t.Errorf("direct rational owner retained %d", b.Used())
		}
	})
	return m, b
}
func servicePublicationDirectOwner(t *testing.T) (*RationalService, *arena.Budget) {
	t.Helper()
	b := arena.NewBudget(1 << 24)
	c := serviceConfig(b)
	c.Integers.Profile = SmallAllocatorProfile{PointerBits: 64, Debug: true, Windows: false}
	s, e := NewRationalService(c)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if e := s.Close(); e != nil {
			t.Error(e)
		}
		if b.Used() != 0 {
			t.Errorf("direct service owner retained %d", b.Used())
		}
	})
	return s, b
}

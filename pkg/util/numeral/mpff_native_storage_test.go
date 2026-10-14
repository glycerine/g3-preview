// Source: src/util/mpff.cpp:36-109; src/util/mpff.h.
package numeral

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestFastFloatNativeStorageGrowthAndRecycledGeneration(t *testing.T) {
	c := DefaultFastFloatConfig(serviceConfig(arena.NewBudget(1 << 26)).Integers.Words)
	c.InitialCapacity = 2
	m, err := NewFastFloatManager(c)
	serviceMust(t, err)
	defer m.Close()
	if m.one.index() != 1 || m.next != 2 {
		t.Fatalf("constructor did not reserve zero ID before one: one=%d next=%d", m.one.index(), m.next)
	}
	for word := uint32(0); word < m.precision; word++ {
		if got := m.significands[word]; got != 0 {
			t.Fatalf("reserved zero slot word %d contains %#x", word, got)
		}
	}
	budget := m.config.Storage.Descriptors.Budget
	before := budget.Used()
	budget.FailAfter(0)
	defer budget.FailAfter(-1)
	var values [129]FastFloat
	for i := range values {
		serviceMust(t, m.SetInt32(&values[i], int32(i+1)))
	}
	for i := range values {
		if fastExact(m, &values[i]).Num().Int64() != int64(i+1) {
			t.Fatal("growth changed stored value", i)
		}
	}
	stale := values[64]
	index := stale.index()
	generation := m.read(&m.metadata, 2*index+1)
	serviceMust(t, m.Reset(&values[64]))
	for word := uint32(0); word < m.precision; word++ {
		if got := m.significands[index*m.precision+word]; got != 0 {
			t.Fatalf("deleted significand slot %d word %d retained %#x", index, word, got)
		}
	}
	serviceMust(t, m.SetInt32(&values[64], 777))
	if values[64].index() != index || m.read(&m.metadata, 2*index+1) != generation+1 {
		t.Fatal("recycled generation did not advance")
	}
	if ffGeneration(stale.bits) != 0 && !errors.Is(m.check(&stale), ErrFastFloatOwner) {
		t.Fatal("checked profile accepted stale handle")
	}
	if budget.Used() != before {
		t.Fatal("Go storage charged as arena allocation", before, budget.Used())
	}
	for i := range values {
		serviceMust(t, m.Del(&values[i]))
	}
}

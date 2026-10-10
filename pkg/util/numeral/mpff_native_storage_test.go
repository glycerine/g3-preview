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

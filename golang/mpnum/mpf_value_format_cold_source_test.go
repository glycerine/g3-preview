// Source: src/util/mpf.cpp:1519-1569; src/util/mpq.cpp:156-185.
// Licensed under the MIT license. Source author: Christoph Wintersteiger.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestArbitraryFloatToStringColdPrefixEightImmediateGraph(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	m, e := NewArbitraryFloatManager(IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	var a ArbitraryFloat
	n := NewInt(1)
	serviceMust(t, m.SetComponents(&a, 8, 260, false, 0, &n))
	text := &formatTestText{}
	b.FailAfter(8)
	_, body := m.ToStringInto(&a, text)
	t.Log("actual cold body", body, "constructed", text.scope.constructed, "source", text.scope.sourceCursor, "decimal count", text.scope.decimal.count)
	integerPhysicalAuditWithRaw(t, m.integer, nil, &a.significand, &m.rational.nTmp)
	b.FailAfter(-1)
	serviceMust(t, text.scope.Cleanup(false))
	serviceMust(t, text.scope.Cleanup(true))
	serviceMust(t, text.scope.EndContext())
	serviceMust(t, m.Del(&a))
	serviceMust(t, m.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

package ast

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestASTFloatingPointNativeValueBucketsGrowthCollisionAndReuse(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	_, e = m.RegisterFloatingPointPlugin()
	astTestMust(t, e)
	p := m.floatingPointPlugin
	const count = 257
	var value util.ArbitraryFloat
	before, storage := m.BudgetField.Used(), p.numeric.RuntimeOwner().Stats()
	m.BudgetField.FailAfter(0)
	for i := 0; i < count; i++ {
		// Hash ignores sign and format; all four values share a bucket but
		// denote different MPFs. Repeating each must return its canonical ID.
		sig := util.NewInt(int32(i + 1))
		for variant := 0; variant < 4; variant++ {
			astTestMust(t, p.numeric.SetComponents(&value, uint32(8+variant/2), uint32(24+variant/2), variant%2 != 0, 3, &sig))
			id, e := p.mkValueID(&value)
			astTestMust(t, e)
			if id != uint32(4*i+variant) {
				t.Fatal("independent sign/format canonical ID", i, variant, id)
			}
			again, e := p.mkValueID(&value)
			astTestMust(t, e)
			if again != id || len(p.freeIDs) != 1 {
				t.Fatal("duplicate did not reuse native ID slot", again, id, p.freeIDs)
			}
		}
	}
	for i := 0; i < count; i++ {
		id := uint32(4*i + 1)
		astTestMust(t, p.recycledValueID(id))
		_, index, e := p.findValueID(id)
		astTestMust(t, e)
		if index >= 0 {
			t.Fatal("erased collision still canonical", id)
		}
		for _, variant := range []uint32{0, 2, 3} {
			_, index, e := p.findValueID(uint32(4*i) + variant)
			astTestMust(t, e)
			if index < 0 {
				t.Fatal("erase lost a distinct colliding value", i, variant)
			}
		}
	}
	sig := util.NewInt(99999)
	astTestMust(t, p.numeric.SetComponents(&value, 8, 24, false, 3, &sig))
	id, e := p.mkValueID(&value)
	astTestMust(t, e)
	if id != 4*(count-1)+1 || len(p.freeIDs) != count || p.nextID != 4*count+1 {
		t.Fatal("native LIFO reuse", id, p.freeIDs, p.nextID)
	}
	if m.BudgetField.Used() != before || p.numeric.RuntimeOwner().Stats() != storage {
		t.Fatal("native buckets or free IDs changed numerical/object storage")
	}
	m.BudgetField.FailAfter(-1)
	astTestMust(t, p.numeric.Del(&value))
	astFloatingPointConstructorRawGraph(t, m.numerals, p)
	astTestMust(t, m.Close())
}

// Source: src/sat/sat_simplifier_params.pyg; sat_simplifier.cpp:1482–1522. Licensed under the MIT license.
package g3

import "testing"

func TestSATSimplifierAllDescriptorsAndStatistics(t *testing.T) {
	s, _ := satSimplifierFixture(t, 0)
	d := pdnew(t, symbolManagerTest(t, nil, false), nil)
	satOK(t, collectSimplifierParamDescrs(d))
	n, e := d.Size()
	satOK(t, e)
	if n != 17 {
		t.Fatal("all source descriptions", n)
	}
	st, r := statFixture(t, nil)
	names := [5]string{"subsumed", "subsumption resolution", "elim literals", "elim bool vars", "elim blocked clauses"}
	var keys [5]StatisticsKey
	for i, n := range names {
		keys[i] = statKey(t, r, n)
	}
	s.numSubsumed = 11
	s.numSubRes = 12
	s.numElimLits = 13
	s.numElimVars = 14
	s.numBlocked = 15
	satOK(t, s.collectStatistics(st, keys))
	for i := uint32(0); i < 5; i++ {
		v, e := st.UintValue(i)
		satOK(t, e)
		if v != 11+i {
			t.Fatal("actual source stats order", i, v)
		}
	}
	s.resetStatistics()
	if s.numSubsumed|s.numSubRes|s.numElimLits|s.numElimVars|s.numBlocked != 0 {
		t.Fatal("source reset all counters")
	}
}

// Source: src/util/mpf.cpp:239-248.
// Licensed under the MIT license. Source author: Christoph Wintersteiger.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"testing"
)

func TestArbitraryFloatExponentRationalSetZeroUnreadForeignFields(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	cfg := IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}
	m, e := NewArbitraryFloatManager(cfg)
	serviceMust(t, e)
	foreign, e := NewIntegerManager(cfg)
	serviceMust(t, e)
	ex := Int{}
	serviceMust(t, foreign.SetInt64(&ex, 1<<40))
	den := Int{}
	serviceMust(t, foreign.SetInt64(&den, 1<<41))
	var out ArbitraryFloat
	serviceMust(t, m.MkZero(8, 24, false, &out))
	q := NewRational()
	q.den = den
	beforeEx, beforeDen := ex, den
	used := b.Used()
	b.FailAfter(0)
	for repeat := 0; repeat < 4; repeat++ {
		serviceMust(t, m.SetExponentRational(&out, 8, 24, ArbitraryFloatNearestEven, &ex, &q))
		if ex != beforeEx || den != beforeDen || q.den != beforeDen || b.Used() != used {
			t.Fatal("unread zero fields mutated/allocated")
		}
	}
	b.FailAfter(-1)
	integerPhysicalAudit(t, foreign, &ex, &den)
	serviceMust(t, m.Del(&out))
	serviceMust(t, m.SourceFinalize())
	serviceMust(t, m.Close())
	serviceMust(t, foreign.Del(&ex))
	serviceMust(t, foreign.Del(&den))
	serviceMust(t, foreign.CloseChecked())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

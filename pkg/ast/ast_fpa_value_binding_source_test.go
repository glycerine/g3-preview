// Source: src/ast/fpa_decl_plugin.h:96-108; src/ast/fpa_decl_plugin.cpp:51-62.
package ast

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestASTFloatingPointValueTableUsesActualMPFHashAndEquality(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	_, e = m.RegisterFloatingPointPlugin()
	astTestMust(t, e)
	p := m.floatingPointPlugin
	var sig util.Int
	var a, b, c util.ArbitraryFloat
	astTestMust(t, p.numeric.SetComponents(&a, 8, 24, false, 3, &sig))
	astTestMust(t, p.numeric.SetComponents(&b, 8, 24, true, 3, &sig))
	astTestMust(t, p.numeric.SetComponents(&c, 8, 24, false, 3, &sig))
	for _, n := range []*util.ArbitraryFloat{&a, &b, &c} {
		astTestMust(t, p.values.push(n))
	}
	procs := p.valueTableProcs()
	h0, e := procs.hash(0)
	astTestMust(t, e)
	h1, e := procs.hash(1)
	astTestMust(t, e)
	if h0 != h1 {
		t.Fatal("source hash ignores sign", h0, h1)
	}
	for i, want := range []uint32{0, 1, 0} {
		got, _, e := p.table.insertIfAbsent(m.BudgetField, procs, uint32(i))
		astTestMust(t, e)
		if got != want {
			t.Fatal("actual equality instead of ID equality", i, got, want)
		}
	}
	astFloatingPointConstructorRawGraph(t, m.numerals, p)
	astTestMust(t, m.Close())
}

// Source: src/ast/fpa_decl_plugin.cpp:52-62. These are real populated rows,
// with canonical ID equality determined by MPF hash/eq rather than row number.
func TestASTFloatingPointValueIDLiteralCanonicalization(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	_, e = m.RegisterFloatingPointPlugin()
	astTestMust(t, e)
	p := m.floatingPointPlugin
	sig := util.NewInt(19)
	var a util.ArbitraryFloat
	astTestMust(t, p.numeric.SetComponents(&a, 8, 24, false, 7, &sig))
	first, e := p.mkValueID(&a)
	astTestMust(t, e)
	again, e := p.mkValueID(&a)
	astTestMust(t, e)
	if first != again {
		t.Fatal("actual MPF canonical id", first, again)
	}
	if len(p.ids.Free) != 1 || p.ids.Free[0] == first {
		t.Fatal("duplicate source id not recycled", p.ids.Free)
	}
	duplicate := p.ids.Free[0]
	row, e := p.values.at(duplicate)
	astTestMust(t, e)
	words, e := p.numeric.ValueWords(&row)
	astTestMust(t, e)
	if words[3] != 19 || words[4] != 0 || words[5] != 0 {
		t.Fatal("source inline DEL changed nonowning numeral bits", row)
	}
	astFloatingPointConstructorRawGraph(t, m.numerals, p)
	sig = util.NewInt(20)
	astTestMust(t, p.numeric.SetComponents(&a, 8, 24, false, 7, &sig))
	distinct, e := p.mkValueID(&a)
	astTestMust(t, e)
	if distinct != duplicate || distinct == first {
		t.Fatal("literal recycled-id reuse", distinct, duplicate, first)
	}
	astFloatingPointConstructorRawGraph(t, m.numerals, p)
	astTestMust(t, p.numeric.Del(&a))
	astTestMust(t, m.Close())
}

func TestASTFloatingPointValueIDEveryActualReservationPrefix(t *testing.T) {
	completed := false
	for ordinal := int64(0); ordinal < 64; ordinal++ {
		m := astTestManager(t)
		_, e := m.RegisterArithmeticPlugin()
		astTestMust(t, e)
		_, e = m.RegisterBitVectorPlugin()
		astTestMust(t, e)
		_, e = m.RegisterFloatingPointPlugin()
		astTestMust(t, e)
		p := m.floatingPointPlugin
		sig := util.NewInt(19)
		var a util.ArbitraryFloat
		astTestMust(t, p.numeric.SetComponents(&a, 8, 24, false, 7, &sig))
		base := m.BudgetField.Used() - m.Stats().AllocatedBytes - m.numerals.RuntimeOwner().Stats().TotalBytes
		m.BudgetField.FailAfter(ordinal)
		id, call := p.mkValueID(&a)
		astFloatingPointConstructorRawGraph(t, m.numerals, p)
		if m.BudgetField.Used() != base+m.Stats().AllocatedBytes+m.numerals.RuntimeOwner().Stats().TotalBytes {
			t.Fatal("before-lift actual named cold graph", ordinal, call)
		}
		if p.ids.Next != 1 {
			t.Fatal("source ID acquisition rolled back", ordinal, p.ids.Next)
		}
		if call == nil {
			completed = true
			if id != 0 {
				t.Fatal("initial literal ID", id)
			}
			value, read := p.values.at(id)
			astTestMust(t, read)
			equal, read := p.numeric.EqCore(&a, &value)
			astTestMust(t, read)
			if !equal {
				t.Fatal("actual row publication")
			}
		}
		m.BudgetField.FailAfter(-1)
		astTestMust(t, p.numeric.Del(&a))
		astTestMust(t, m.Close())
		if completed {
			t.Log("first successful source ID prefix", ordinal)
			break
		}
	}
	if !completed {
		t.Fatal("no actual populated table success")
	}
}

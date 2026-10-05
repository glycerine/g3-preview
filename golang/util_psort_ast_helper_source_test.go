// Source: src/util/sorting_network.h:269-494,645-653,918-1013.
// Source author: Nikolaj Bjorner (2013-11-07).
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestPsortASTNetworkDefinedPairwiseAndErasedBimanderHelpers(t *testing.T) {
	for n := uint32(1); n <= 5; n++ {
		for helper := 0; helper < 3; helper++ {
			m := astTestManager(t)
			c, e := NewPsortASTContext(m)
			astTestMust(t, e)
			r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: m.budget})
			astTestMust(t, e)
			s, e := NewPsortASTNetwork(c, r, m.budget, nil)
			astTestMust(t, e)
			in, e := NewRuntimeASTLiteralVector(r, m.budget, nil)
			astTestMust(t, e)
			ors, e := NewRuntimeASTLiteralVector(r, m.budget, nil)
			astTestMust(t, e)
			sort, e := m.MkBoolSort()
			astTestMust(t, e)
			var inputs []ASTHandle
			for i := uint32(0); i < n; i++ {
				h, e := m.MkFreshConst("a", sort)
				astTestMust(t, e)
				astTestPin(t, m, h)
				inputs = append(inputs, h)
				astTestMust(t, in.Push(h))
			}
			p, e := in.CPtr()
			astTestMust(t, e)
			// Direct kernel fixtures cover actual source definitions. Bimander remains
			// unreachable through the literal n>=4 && false public policy.
			s.running = true
			var result ASTHandle
			if helper == 0 {
				result, e = s.atMostOnePairwise(n, psortASTPointer{span: p})
			} else {
				result, e = s.atMostOneBimander(helper == 2, n, psortASTPointer{span: p}, ors)
			}
			s.running = false
			astTestMust(t, e)
			q := psortOracleCNF{ids: map[ASTHandle]int{}}
			rid := q.gate(t, m, result)
			sz, e := m.RefVectorSize(c.clauses)
			astTestMust(t, e)
			for i := uint32(0); i < sz; i++ {
				h, e := m.RefVectorGet(c.clauses, i)
				astTestMust(t, e)
				id := q.gate(t, m, h)
				q.clauses = append(q.clauses, []int{id})
			}
			for _, h := range inputs {
				q.gate(t, m, h)
			}
			for bits := uint32(0); bits < 1<<n; bits++ {
				values := make([]int8, q.count+1)
				pop := 0
				for i, h := range inputs {
					values[q.ids[h]] = -1
					if bits&(1<<i) != 0 {
						values[q.ids[h]] = 1
						pop++
					}
				}
				positive := append([]int8(nil), values...)
				positive[rid] = 1
				if got := psortOracleSAT(q.clauses, positive); got != (pop <= 1) {
					t.Fatal("source helper positive clause semantics", n, helper, bits, got)
				}
				if helper == 0 {
					negative := append([]int8(nil), values...)
					negative[rid] = -1
					if got := psortOracleSAT(q.clauses, negative); got == (pop <= 1) {
						t.Fatal("pairwise bidirectional AND semantics", n, bits, got)
					}
				}
			}
			astTestMust(t, s.Close())
			astTestMust(t, in.Close())
			astTestMust(t, ors.Close())
			astTestMust(t, r.Close())
			astTestMust(t, c.Close())
		}
	}
}
func TestPsortASTNetworkDirectMergeAppendPrefixAndLexicalScopeRetirement(t *testing.T) {
	m := astTestManager(t)
	c, e := NewPsortASTContext(m)
	astTestMust(t, e)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: m.budget})
	astTestMust(t, e)
	s, e := NewPsortASTNetwork(c, r, m.budget, nil)
	astTestMust(t, e)
	in, e := NewRuntimeASTLiteralVector(r, m.budget, nil)
	astTestMust(t, e)
	out, e := NewRuntimeASTLiteralVector(r, m.budget, nil)
	astTestMust(t, e)
	sort, e := m.MkBoolSort()
	astTestMust(t, e)
	for i := 0; i < 2; i++ {
		h, e := m.MkFreshConst("a", sort)
		astTestMust(t, e)
		astTestPin(t, m, h)
		astTestMust(t, in.Push(h))
	}
	p, e := in.CPtr()
	astTestMust(t, e)
	// Empty one-sided merge copies actual raw handles, even with m_t unread.
	s.running = true
	e = s.merge(0, psortASTPointer{}, 2, psortASTPointer{span: p}, out)
	s.running = false
	astTestMust(t, e)
	if s.modeInitialized || s.frames != nil {
		t.Fatal("empty side must erase mode/cost/frame reads")
	}
	_, e = s.Eq(true, 0, 1, p)
	astTestMust(t, e)
	// The next merge appends; source does not clear the previous caller output.
	old, e := out.At(0)
	astTestMust(t, e)
	s.running = true
	e = s.merge(1, psortASTPointer{span: p}, 1, psortASTPointer{span: p, offset: 1}, out)
	s.running = false
	astTestMust(t, e)
	n, e := out.Size()
	astTestMust(t, e)
	got, e := out.At(0)
	astTestMust(t, e)
	if n != 4 || got != old || s.frames != nil {
		t.Fatal("caller prefix/finished lexical owner retirement", n, got, old)
	}
	astTestMust(t, s.SourceFinalize())
	before := s.budget.Used()
	if _, e = s.Ge(true, 0, 0, ASTLiteralVectorSpan{}); e == nil || s.budget.Used() != before {
		t.Fatal("ended checked facade allowed body mutation")
	}
	astTestMust(t, s.Close())
	astTestMust(t, in.Close())
	astTestMust(t, out.Close())
	astTestMust(t, r.Close())
	astTestMust(t, c.Close())
}

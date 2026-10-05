// Source: src/util/sorting_network.h:190-260,597-1046.
// Source author: Nikolaj Bjorner (2013-11-07).
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

// The test oracle independently lowers actual Boolean AST gates into their
// truth-table CNF and exhausts assignments with unit propagation. It has no
// cardinality/sorting knowledge and is not a production solver substitution.
type psortOracleCNF struct {
	ids     map[ASTHandle]int
	clauses [][]int
	count   int
}

func (q *psortOracleCNF) gate(t *testing.T, m *ASTManager, h ASTHandle) int {
	if id, ok := q.ids[h]; ok {
		return id
	}
	q.count++
	id := q.count
	q.ids[h] = id
	r, e := m.internal(h)
	astTestMust(t, e)
	app := m.appAt(r.slot)
	info, e := m.funcInfo(m.declAt(app.decl.slot).info)
	astTestMust(t, e)
	var args []int
	for i := uint32(0); i < app.count; i++ {
		r, e := m.readInternal(app.args, i)
		astTestMust(t, e)
		args = append(args, q.gate(t, m, m.external(r)))
	}
	if info.family != 0 {
		if len(args) != 0 {
			t.Fatal("oracle unsupported nonbasic compound")
		}
		return id
	}
	switch info.kind {
	case ASTOpTrue:
		q.clauses = append(q.clauses, []int{id})
	case ASTOpFalse:
		q.clauses = append(q.clauses, []int{-id})
	case ASTOpNot:
		if len(args) != 1 {
			t.Fatal("oracle NOT arity")
		}
		q.clauses = append(q.clauses, []int{-id, -args[0]}, []int{id, args[0]})
	case ASTOpAnd:
		reverse := []int{id}
		for _, a := range args {
			q.clauses = append(q.clauses, []int{-id, a})
			reverse = append(reverse, -a)
		}
		q.clauses = append(q.clauses, reverse)
	case ASTOpOr:
		reverse := []int{-id}
		for _, a := range args {
			q.clauses = append(q.clauses, []int{id, -a})
			reverse = append(reverse, a)
		}
		q.clauses = append(q.clauses, reverse)
	default:
		t.Fatal("oracle unsupported basic operation", info.kind)
	}
	return id
}
func psortOracleSAT(clauses [][]int, values []int8) bool {
	for {
		changed := false
		for _, clause := range clauses {
			satisfied := false
			unknown, last := 0, 0
			for _, lit := range clause {
				id := lit
				if id < 0 {
					id = -id
				}
				v := values[id]
				if v == 0 {
					unknown++
					last = lit
				} else if v > 0 && lit > 0 || v < 0 && lit < 0 {
					satisfied = true
					break
				}
			}
			if satisfied {
				continue
			}
			if unknown == 0 {
				return false
			}
			if unknown == 1 {
				id := last
				v := int8(1)
				if id < 0 {
					id = -id
					v = -1
				}
				if values[id] == 0 {
					values[id] = v
					changed = true
				} else if values[id] != v {
					return false
				}
			}
		}
		if !changed {
			break
		}
	}
	pick := 0
	for _, clause := range clauses {
		satisfied := false
		candidate := 0
		for _, lit := range clause {
			id := lit
			if id < 0 {
				id = -id
			}
			v := values[id]
			if v > 0 && lit > 0 || v < 0 && lit < 0 {
				satisfied = true
				break
			}
			if v == 0 && candidate == 0 {
				candidate = id
			}
		}
		if !satisfied && candidate != 0 {
			pick = candidate
			break
		}
	}
	if pick == 0 {
		return true
	}
	next := append([]int8(nil), values...)
	next[pick] = 1
	if psortOracleSAT(clauses, next) {
		return true
	}
	next = append([]int8(nil), values...)
	next[pick] = -1
	return psortOracleSAT(clauses, next)
}
func TestPsortASTNetworkRecursiveFamiliesIndependentClauseOracle(t *testing.T) {
	cases := [][2]uint32{{5, 1}, {5, 2}, {10, 3}, {10, 5}, {12, 4}, {12, 10}}
	for _, tc := range cases {
		for mode := 0; mode < 3; mode++ {
			for _, full := range []bool{false, true} {
				n, k := tc[0], tc[1]
				m := astTestManager(t)
				c, e := NewPsortASTContext(m)
				astTestMust(t, e)
				r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: m.budget})
				astTestMust(t, e)
				s, e := NewPsortASTNetwork(c, r, m.budget, nil)
				astTestMust(t, e)
				in, e := NewRuntimeASTLiteralVector(r, m.budget, nil)
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
				var result ASTHandle
				switch mode {
				case 0:
					result, e = s.Ge(full, k, n, p)
				case 1:
					result, e = s.Le(full, k, n, p)
				case 2:
					result, e = s.Eq(full, k, n, p)
				}
				if e != nil {
					t.Fatal(n, k, mode, full, e)
				}
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
				for pop := uint32(0); pop <= n; pop++ {
					values := make([]int8, q.count+1)
					for i, h := range inputs {
						values[q.ids[h]] = -1
						if uint32(i) < pop {
							values[q.ids[h]] = 1
						}
					}
					want := pop >= k
					if mode == 1 {
						want = pop <= k
					} else if mode == 2 {
						want = pop == k
					}
					yes := append([]int8(nil), values...)
					yes[rid] = 1
					got := psortOracleSAT(q.clauses, yes)
					if got != want {
						t.Fatal("recursive existential oracle", n, k, mode, full, pop, got, want)
					}
					if full {
						no := append([]int8(nil), values...)
						no[rid] = -1
						if got := psortOracleSAT(q.clauses, no); got == want {
							t.Fatal("recursive negative oracle", n, k, mode, pop, got, want)
						}
					}
				}
				astTestMust(t, s.Close())
				astTestMust(t, in.Close())
				astTestMust(t, r.Close())
				astTestMust(t, c.Close())
			}
		}
	}
}

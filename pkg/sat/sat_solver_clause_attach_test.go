package sat

import (
	"fmt"
	"github.com/glycerine/g3/pkg/util/lbool"
	"testing"
)

func TestSATSolverClauseAttachDetachActualKinds(t *testing.T) {
	s, p, v, _ := satLifecycleFixture(t, 5)
	for _, lits := range [][]Literal{{0, 2, 4}, {0, 2, 4, 6}} {
		c, e := p.clauses.MkClause(lits, false)
		satOK(t, e)
		_, e = s.attach(c)
		satOK(t, e)
		for _, l := range lits[:2] {
			w, e := v.watches.Get(l.Negated().index())
			satOK(t, e)
			n, e := w.Size()
			satOK(t, e)
			if n != 1 {
				t.Fatal(n)
			}
		}
		satOK(t, s.detachClause(c))
		satOK(t, p.clauses.Delete(c))
	}
	satOK(t, s.mkBinClause(0, 2, false))
	satOK(t, s.detachBinClause(0, 2, false))
	for _, idx := range []uint32{1, 3, 5} {
		w, e := v.watches.Get(idx)
		satOK(t, e)
		n, e := w.Size()
		satOK(t, e)
		if n != 0 {
			t.Fatal(idx, n)
		}
	}
}
func TestSATSolverClauseScopedReinitOrdering(t *testing.T) {
	s, p, v, _ := satLifecycleFixture(t, 4)
	v.scopeLevel = 1
	satOK(t, p.assign(1, satJustification{}))
	satOK(t, p.assign(3, satJustification{}))
	c, e := s.MkClauseCore(3, []Literal{0, 2, 4}, false)
	satOK(t, e)
	if len(s.reinit) != 1 || s.reinit[0].ref != c || len(s.Original) != 1 || p.ternaryPropagations != 1 {
		t.Fatal("attachment/reinit/publication")
	}
	flag, e := p.clauses.OnReinitStack(c)
	satOK(t, e)
	if !flag {
		t.Fatal("flag")
	}
}

func TestSATSolverClauseLearnedWatchActualSelection(t *testing.T) {
	s, p, v, _ := satLifecycleFixture(t, 4)
	v.scopeLevel = 2
	satOK(t, p.assign(3, satJustification{}))
	satOK(t, v.level.Write(1, 1))
	satOK(t, p.assign(5, satJustification{}))
	satOK(t, v.level.Write(2, 2))
	satOK(t, p.assign(7, satJustification{}))
	satOK(t, v.level.Write(3, 1))
	c, e := s.MkClauseCore(4, []Literal{0, 2, 4, 6}, true)
	satOK(t, e)
	x, e := p.clauses.Literal(c, 1)
	satOK(t, e)
	if x != 4 || len(s.learned) != 1 || len(s.reinit) != 0 || p.clausePropagations != 1 {
		t.Fatal("actual highest-level learned watch", x)
	}
	if satJustificationTestValue(t, &v.justification, 0) != newSATClauseJustification(satClauseOffset(c)) {
		t.Fatal("actual clause assignment")
	}
	satLifecycleAssertOwned(t, s)
}

func TestSATSolverAttachBinaryAndTernaryFiniteMatrices(t *testing.T) {
	vals := []lbool.LBool{lbool.False, lbool.Undef, lbool.True}
	for _, scope := range []uint32{0, 2} {
		for a := 0; a < 3; a++ {
			for b := 0; b < 3; b++ {
				t.Run(fmt.Sprintf("bin/%d/%d%d", scope, a, b), func(t *testing.T) {
					s, p, v, _ := satLifecycleFixture(t, 2)
					v.scopeLevel = scope
					for i, x := range []int{a, b} {
						if vals[x] != lbool.Undef {
							satOK(t, p.assign(MakeLiteral(BoolVar(i), vals[x] == lbool.False), satJustification{}))
						}
					}
					satOK(t, s.mkBinClause(0, 2, false))
					prop := a == 0 || b == 0
					if p.binaryPropagations != uint32(map[bool]int{true: 1}[prop]) || p.inconsistent != (a == 0 && b == 0) {
						t.Fatal("independent binary truth", a, b)
					}
					if prop {
						target := 0
						if b != 0 {
							target = 1
						}
						actual, e := v.valueVar(BoolVar(target))
						satOK(t, e)
						want := vals[[]int{a, b}[target]]
						if want == lbool.Undef {
							want = lbool.True
						}
						if actual != want {
							t.Fatal("source first-false precedence", actual, want)
						}
					}
					watches := uint32(1)
					if prop && scope == 0 {
						watches = 0
					}
					for _, idx := range []uint32{1, 3} {
						w, e := v.watches.Get(idx)
						satOK(t, e)
						n, e := w.Size()
						satOK(t, e)
						if n != watches {
							t.Fatal("binary watcher publication", n, watches)
						}
					}
					satLifecycleAssertOwned(t, s)
				})
			}
		}
		for a := 0; a < 3; a++ {
			for b := 0; b < 3; b++ {
				for c := 0; c < 3; c++ {
					t.Run(fmt.Sprintf("ter/%d/%d%d%d", scope, a, b, c), func(t *testing.T) {
						s, p, v, _ := satLifecycleFixture(t, 3)
						v.scopeLevel = scope
						input := []int{a, b, c}
						oldTrail := uint32(0)
						for i, x := range input {
							if vals[x] != lbool.Undef {
								satOK(t, p.assign(MakeLiteral(BoolVar(i), vals[x] == lbool.False), satJustification{}))
								oldTrail++
							}
						}
						clause, e := s.mkTerClause([]Literal{0, 2, 4}, false)
						satOK(t, e)
						target := -1
						if scope > 0 {
							for i := 0; i < 3; i++ {
								j, k := 1, 2
								if i == 1 {
									j, k = 0, 2
								} else if i == 2 {
									j, k = 0, 1
								}
								if input[j] == 0 && input[k] == 0 {
									target = i
									break
								}
							}
						}
						wantCount := uint32(0)
						if target >= 0 {
							wantCount = 1
						}
						if p.ternaryPropagations != wantCount || p.inconsistent != (target >= 0 && input[target] == 0) {
							t.Fatal("independent ternary precedence", input, target)
						}
						for i, idx := range []uint32{1, 3, 5} {
							w, e := v.watches.Get(idx)
							satOK(t, e)
							n, e := w.Size()
							satOK(t, e)
							if n != 1 {
								t.Fatal("all watches precede propagation", idx, n)
							}
							got, e := w.Read(0)
							satOK(t, e)
							j, k := 1, 2
							if i == 1 {
								j, k = 0, 2
							} else if i == 2 {
								j, k = 0, 1
							}
							expected, e := newSATTernaryWatch(Literal(2*j), Literal(2*k))
							satOK(t, e)
							if !got.Equal(expected) {
								t.Fatal("exact ternary fields")
							}
						}
						n, e := v.trail.Size(v.trailRef)
						satOK(t, e)
						wantTrail := oldTrail
						if target >= 0 && input[target] == 1 {
							wantTrail++
						}
						if n != wantTrail {
							t.Fatal("actual source assignment prefix", n, wantTrail)
						}
						flag, e := p.clauses.OnReinitStack(clause)
						satOK(t, e)
						if flag != (target >= 0) || len(s.reinit) != int(wantCount) {
							t.Fatal("actual reinit flag/publication")
						}
						satLifecycleAssertOwned(t, s)
					})
				}
			}
		}
	}
}
func TestSATSolverAttachNaryReorderAndWatchFaultPrefixes(t *testing.T) {
	success, failures := 0, 0
	firstOnly := false
	for fault := int64(0); fault < 96; fault++ {
		done := false
		t.Run(fmt.Sprint(fault), func(t *testing.T) {
			s, p, v, b := satLifecycleFixture(t, 4)
			v.scopeLevel = 3
			for i := BoolVar(0); i < 3; i++ {
				satOK(t, p.assign(MakeLiteral(i, true), satJustification{}))
				satOK(t, v.level.Write(uint32(i), uint32(i+1)))
			}
			filler, e := newSATBinaryWatch(7, false)
			satOK(t, e)
			old := [2]uint32{}
			for k, idx := range []uint32{7, 5} {
				w, e := v.watches.Get(idx)
				satOK(t, e)
				fill := 40
				if k == 1 {
					fill = 200
				}
				for i := 0; i < fill; i++ {
					satOK(t, w.Push(filler))
				}
				r, e := w.ref()
				satOK(t, e)
				record, e := w.owner.store.record(r)
				satOK(t, e)
				for record.length < record.capacity {
					satOK(t, w.Push(filler))
				}
				old[k] = record.length
			}
			b.FailAfter(fault)
			c, e := s.mkNaryClause(4, []Literal{0, 2, 4, 6}, false)
			satLifecycleAssertOwned(t, s)
			if e != nil {
				if !satRuntimeActualRefusal(v.memory, e) {
					t.Fatal(e)
				}
				failures++
			} else {
				success++
				done = true
			}
			if c != (StoredClauseRef{}) {
				a, x := p.clauses.Literal(c, 0)
				satOK(t, x)
				z, x := p.clauses.Literal(c, 1)
				satOK(t, x)
				if a != 6 || z != 4 || p.clausePropagations != 1 || v.assignment[6] != lbool.True {
					t.Fatal("source swaps+assignment precede watches", a, z)
				}
				w, x := v.watches.Get(7)
				satOK(t, x)
				n, x := w.Size()
				satOK(t, x)
				w2, x := v.watches.Get(5)
				satOK(t, x)
				n2, x := w2.Size()
				satOK(t, x)
				if n == old[0]+1 && n2 == old[1] {
					firstOnly = true
				}
				if n > old[0]+1 || n2 > old[1]+1 || n2 > old[1] && n == old[0] {
					t.Fatal("ordered watch prefixes", n, n2)
				}
			}
			b.FailAfter(-1)
		})
		if done {
			break
		}
	}
	if failures == 0 || success != 1 || !firstOnly {
		t.Fatal("genuine both watcher boundaries", failures, success, firstOnly)
	}
}

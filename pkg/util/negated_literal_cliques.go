// Source: src/util/max_cliques.h:26-137; src/sat/sat_solver.cpp:3113-3117.
package util

import (
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
)

// NegatedLiteralCliques greedily partitions complement-paired unsigned literals.
// Negation is index XOR 1. Reachability caches persist across queries, as in
// max_cliques.h; callers needing an independent query construct a fresh owner.
type NegatedLiteralCliques struct {
	self                        *NegatedLiteralCliques
	runtime                     *Runtime
	budget                      *arena.Budget
	ownerBytes                  uint64
	next, tc                    *RuntimeUnsignedRows
	reachable                   [2]cliqueUintSet
	seen1, seen2                cliqueUintSet
	todo, clique                RuntimeUintColumn
	vars                        cliqueUintSet
	callActive, closing, closed bool
	closeCursor                 uint32
}

func NewNegatedLiteralCliques(r *Runtime, b *arena.Budget) (*NegatedLiteralCliques, error) {
	if r == nil || r.Budget() != b {
		return nil, ErrRuntimeVectorOwner
	}
	if e := r.CheckOpen(); e != nil {
		return nil, e
	}
	bytes := uint64(reflect.TypeOf(NegatedLiteralCliques{}).Size())
	if e := b.Reserve(bytes); e != nil {
		return nil, e
	}
	g := &NegatedLiteralCliques{runtime: r, budget: b, ownerBytes: bytes}
	g.self = g
	for _, s := range []*cliqueUintSet{&g.reachable[0], &g.reachable[1], &g.seen1, &g.seen2, &g.vars} {
		s.words.Bind(r)
	}
	g.todo.Bind(r)
	g.clique.Bind(r)
	var e error
	g.next, e = NewRuntimeUnsignedRows(r, b)
	if e != nil {
		return g, e
	}
	g.tc, e = NewRuntimeUnsignedRows(r, b)
	return g, e
}
func (g *NegatedLiteralCliques) open() error {
	if g == nil || g.self != g {
		return ErrRuntimeVectorOwner
	}
	if g.closing || g.closed || g.callActive || g.next == nil || g.tc == nil {
		return arena.ErrClosed
	}
	return g.runtime.CheckOpen()
}
func cliqueMaxPlusOne(a, b uint32) uint32 {
	if b > a {
		a = b
	}
	return a + 1
}
func (g *NegatedLiteralCliques) AddEdge(src, dst uint32) error {
	if e := g.open(); e != nil {
		return e
	}
	if e := g.next.ReserveEmptyRows(cliqueMaxPlusOne(src, dst)); e != nil {
		return e
	}
	if e := g.next.ReserveEmptyRows(cliqueMaxPlusOne(src^1, dst^1)); e != nil {
		return e
	}
	if e := cliqueRowPush(g.next, src, dst); e != nil {
		return e
	}
	return cliqueRowPush(g.next, dst, src)
}
func (g *NegatedLiteralCliques) getReachable(p uint32, goal, out *cliqueUintSet) error {
	if e := g.seen1.reset(); e != nil {
		return e
	}
	if e := g.todo.Shrink(0); e != nil {
		return e
	}
	if e := g.todo.Push(p); e != nil {
		return e
	}
	for i := uint32(0); ; i++ {
		n, e := g.todo.Size()
		if e != nil {
			return e
		}
		if i >= n {
			break
		}
		p, e = g.todo.Read(i)
		if e != nil {
			return e
		}
		seen, e := g.seen1.contains(p)
		if e != nil {
			return e
		}
		if seen {
			continue
		}
		if e = g.seen1.insert(p); e != nil {
			return e
		}
		seen, e = g.seen2.contains(p)
		if e != nil {
			return e
		}
		if seen {
			tc, e := g.tc.Row(p)
			if e != nil {
				return e
			}
			n, e := tc.Size()
			if e != nil {
				return e
			}
			for j := uint32(0); j < n; j++ {
				np, e := tc.Read(j)
				if e != nil {
					return e
				}
				yes, e := goal.contains(np)
				if e != nil {
					return e
				}
				if yes {
					if e = out.insert(np); e != nil {
						return e
					}
				}
			}
		} else {
			np := p ^ 1
			yes, e := goal.contains(np)
			if e != nil {
				return e
			}
			if yes {
				if e = out.insert(np); e != nil {
					return e
				}
			}
			succ, e := g.next.Row(np)
			if e != nil {
				return e
			}
			if e = cliqueAppend(&g.todo, succ); e != nil {
				return e
			}
		}
	}
	n, e := g.todo.Size()
	if e != nil {
		return e
	}
	for i := n; i > 0; {
		i--
		p, e = g.todo.Read(i)
		if e != nil {
			return e
		}
		seen, e := g.seen2.contains(p)
		if e != nil {
			return e
		}
		if seen {
			continue
		}
		if e = g.seen2.insert(p); e != nil {
			return e
		}
		np := p ^ 1
		_, e = g.tc.Row(p)
		if e != nil {
			return e
		}
		yes, e := goal.contains(np)
		if e != nil {
			return e
		}
		if yes {
			if e = cliqueRowPush(g.tc, p, np); e != nil {
				return e
			}
		} else {
			succ, e := g.next.Row(np)
			if e != nil {
				return e
			}
			sz, e := succ.Size()
			if e != nil {
				return e
			}
			for j := uint32(0); j < sz; j++ {
				v, e := succ.Read(j)
				if e != nil {
					return e
				}
				row, e := g.tc.Row(v)
				if e != nil {
					return e
				}
				if e = cliqueRowAppend(g.tc, p, row); e != nil {
					return e
				}
			}
		}
	}
	return nil
}
func (g *NegatedLiteralCliques) Cliques(ps *RuntimeUintColumn, out *RuntimeUnsignedRows) (err error) {
	if err = g.open(); err != nil {
		return err
	}
	num, e := ps.Size()
	if e != nil {
		return e
	}
	max := uint32(0)
	for i := uint32(0); i < num; i++ {
		p, e := ps.Read(i)
		if e != nil {
			return e
		}
		n := cliqueMaxPlusOne(p, p^1)
		if n > max {
			max = n
		}
	}
	if err = g.next.ReserveEmptyRows(max); err != nil {
		return err
	}
	n, e := g.next.RowCount()
	if e != nil {
		return e
	}
	if err = g.tc.ReserveEmptyRows(n); err != nil {
		return err
	}
	g.callActive = true
	g.clique = RuntimeUintColumn{}
	g.clique.Bind(g.runtime)
	g.vars = cliqueUintSet{}
	g.vars.words.Bind(g.runtime)
	defer func() {
		if e := g.vars.words.Finalize(); e != nil {
			if err == nil {
				err = e
			}
			return
		}
		if e := g.clique.Finalize(); e != nil {
			if err == nil {
				err = e
			}
			return
		}
		g.callActive = false
	}()
	for i := uint32(0); i < num; i++ {
		v, e := ps.Read(i)
		if e != nil {
			return e
		}
		if err = g.vars.insert(v); err != nil {
			return err
		}
	}
	for {
		empty, e := g.vars.empty()
		if e != nil {
			return e
		}
		if empty {
			return nil
		}
		if err = g.clique.Shrink(0); err != nil {
			return err
		}
		turn := 0
		if err = g.reachable[turn].assign(&g.vars); err != nil {
			return err
		}
		for {
			empty, e := g.reachable[turn].empty()
			if e != nil {
				return e
			}
			if empty {
				break
			}
			p, e := g.reachable[turn].first()
			if e != nil {
				return e
			}
			if err = g.reachable[turn].remove(p); err != nil {
				return err
			}
			if err = g.vars.remove(p); err != nil {
				return err
			}
			if err = g.clique.Push(p); err != nil {
				return err
			}
			empty, e = g.reachable[turn].empty()
			if e != nil {
				return e
			}
			if empty {
				break
			}
			if err = g.reachable[1-turn].reset(); err != nil {
				return err
			}
			if err = g.getReachable(p, &g.reachable[turn], &g.reachable[1-turn]); err != nil {
				return err
			}
			turn = 1 - turn
		}
		n, e := g.clique.Size()
		if e != nil {
			return e
		}
		if n > 1 {
			if err = out.AppendColumn(&g.clique); err != nil {
				return err
			}
		}
	}
}
func (g *NegatedLiteralCliques) Close() error {
	if g == nil || g.self != g {
		return ErrRuntimeVectorOwner
	}
	if g.closed {
		return nil
	}
	g.closing = true
	for g.closeCursor < 9 {
		var e error
		switch g.closeCursor {
		case 0:
			e = g.vars.words.Finalize()
		case 1:
			e = g.clique.Finalize()
		case 2:
			e = g.todo.Finalize()
		case 3:
			e = g.seen2.words.Finalize()
		case 4:
			e = g.seen1.words.Finalize()
		case 5:
			e = g.reachable[1].words.Finalize()
		case 6:
			e = g.reachable[0].words.Finalize()
		case 7:
			if g.tc != nil {
				e = g.tc.CloseChecked()
			}
		case 8:
			if g.next != nil {
				e = g.next.CloseChecked()
			}
		}
		if e != nil {
			return e
		}
		g.closeCursor++
	}
	g.budget.Release(g.ownerBytes)
	g.ownerBytes = 0
	g.closed = true
	return nil
}

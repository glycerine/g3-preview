// Source: src/util/max_cliques.h:26-137; src/sat/sat_solver.cpp:3113-3117.
package sat

import (
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

type satMaxCliques struct {
	self                        *satMaxCliques
	runtime                     *util.Runtime
	budget                      *arena.Budget
	ownerBytes                  uint64
	next, tc                    *satSolverLiteralRows
	reachable                   [2]satCliqueUintSet
	seen1, seen2                satCliqueUintSet
	todo, clique                RuntimeUintColumn
	vars                        satCliqueUintSet
	callActive, closing, closed bool
	closeCursor                 uint32
}

func newSATMaxCliques(r *util.Runtime, b *arena.Budget) (*satMaxCliques, error) {
	if r == nil || r.Budget() != b {
		return nil, ErrSATOwner
	}
	if e := r.CheckOpen(); e != nil {
		return nil, e
	}
	bytes := uint64(reflect.TypeOf(satMaxCliques{}).Size())
	if e := b.Reserve(bytes); e != nil {
		return nil, e
	}
	g := &satMaxCliques{runtime: r, budget: b, ownerBytes: bytes}
	g.self = g
	for _, s := range []*satCliqueUintSet{&g.reachable[0], &g.reachable[1], &g.seen1, &g.seen2, &g.vars} {
		s.words.Bind(r)
	}
	g.todo.Bind(r)
	g.clique.Bind(r)
	var e error
	g.next, e = newSATSolverLiteralRows(r, b)
	if e != nil {
		return g, e
	}
	g.tc, e = newSATSolverLiteralRows(r, b)
	return g, e
}
func (g *satMaxCliques) open() error {
	if g == nil || g.self != g {
		return ErrSATOwner
	}
	if g.closing || g.closed || g.callActive || g.next == nil || g.tc == nil {
		return arena.ErrClosed
	}
	return g.runtime.CheckOpen()
}
func satCliqueMaxPlusOne(a, b uint32) uint32 {
	if b > a {
		a = b
	}
	return a + 1
}
func (g *satMaxCliques) addEdge(src, dst uint32) error {
	if e := g.open(); e != nil {
		return e
	}
	if e := g.next.reserveEmptyRows(satCliqueMaxPlusOne(src, dst)); e != nil {
		return e
	}
	if e := g.next.reserveEmptyRows(satCliqueMaxPlusOne(src^1, dst^1)); e != nil {
		return e
	}
	if e := satCliqueRowPush(g.next, src, dst); e != nil {
		return e
	}
	return satCliqueRowPush(g.next, dst, src)
}
func (g *satMaxCliques) getReachable(p uint32, goal, out *satCliqueUintSet) error {
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
			tc, e := g.tc.unsignedRow(p)
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
			succ, e := g.next.unsignedRow(np)
			if e != nil {
				return e
			}
			if e = satCliqueAppend(&g.todo, succ); e != nil {
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
		_, e = g.tc.unsignedRow(p)
		if e != nil {
			return e
		}
		yes, e := goal.contains(np)
		if e != nil {
			return e
		}
		if yes {
			if e = satCliqueRowPush(g.tc, p, np); e != nil {
				return e
			}
		} else {
			succ, e := g.next.unsignedRow(np)
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
				row, e := g.tc.unsignedRow(v)
				if e != nil {
					return e
				}
				if e = satCliqueRowAppend(g.tc, p, row); e != nil {
					return e
				}
			}
		}
	}
	return nil
}
func (g *satMaxCliques) cliques(ps *RuntimeUintColumn, out *satSolverLiteralRows) (err error) {
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
		n := satCliqueMaxPlusOne(p, p^1)
		if n > max {
			max = n
		}
	}
	if err = g.next.reserveEmptyRows(max); err != nil {
		return err
	}
	n, e := g.next.rowCount()
	if e != nil {
		return e
	}
	if err = g.tc.reserveEmptyRows(n); err != nil {
		return err
	}
	g.callActive = true
	g.clique = RuntimeUintColumn{}
	g.clique.Bind(g.runtime)
	g.vars = satCliqueUintSet{}
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
			if err = out.appendColumn(&g.clique); err != nil {
				return err
			}
		}
	}
}
func (g *satMaxCliques) Close() error {
	if g == nil || g.self != g {
		return ErrSATOwner
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

// Source: src/sat/sat_scc.cpp; src/sat/sat_scc.h; src/sat/sat_scc_params.pyg.
package sat

import (
	"errors"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
	"github.com/glycerine/g3/pkg/util/statistics"
	"github.com/glycerine/g3/pkg/util/stopwatch"
)

type satSCC struct {
	self                                                  *satSCC
	lifecycle                                             *satSolverClauseLifecycle
	index, lowlink, stack, inStack, roots, frames, toElim satSCCVector
	eliminator                                            satElimEqs
	numElim, nextIndex                                    uint32
	header                                                uint64
	bodyCause, cleanupCause                               error
	report                                                stopwatch.Stopwatch
	reportActive                                          bool
	enabled, active, closing, closed, debugInvariant      bool
}

func newSATSCC(l *satSolverClauseLifecycle, p *util.ParamsRef) (*satSCC, error) {
	if e := l.constructionBorrow(); e != nil {
		return nil, e
	}
	n := uint64(reflect.TypeOf(satSCC{}).Size())
	if e := l.budget.Reserve(n); e != nil {
		return nil, e
	}
	c := &satSCC{lifecycle: l, header: n}
	c.self = c
	c.index.width = 4
	c.lowlink.width = 4
	c.stack.width = 4
	c.inStack.width = 1
	c.roots.width = 4
	// Native LP64 source frame is24 bytes (uint32, bool/padding, two pointers).
	// The checked scalar backing encodes captured physical identities in36 bytes.
	// Runtime charges this actual adapted stride, not source-native geometry.
	c.frames.width = 36
	c.toElim.width = 4
	c.eliminator.lifecycle = l
	c.eliminator.self = &c.eliminator
	if e := c.Update(p); e != nil {
		l.budget.Release(n)
		return nil, e
	}
	return c, nil
}
func (c *satSCC) constructionBorrow() error {
	if c == nil || c.self != c {
		return ErrSATOwner
	}
	if c.closed || c.closing {
		return arena.ErrClosed
	}
	return c.lifecycle.constructionBorrow()
}
func (c *satSCC) open() error {
	if c == nil || c.self != c {
		return ErrSATOwner
	}
	if c.closed || c.closing {
		return arena.ErrClosed
	}
	return c.lifecycle.open()
}
func (c *satSCC) Update(local *util.ParamsRef) (err error) {
	if e := c.constructionBorrow(); e != nil {
		return e
	}
	cfg := c.lifecycle.propagation.config
	g, e := cfg.globals.GetModuleName("sat")
	if e != nil {
		return e
	}
	defer func() {
		if e := cfg.context.Del(&g); err == nil {
			err = e
		}
	}()
	key := "scc"
	enabled, e := cfg.context.GetBoolFallbackName(local, &key, &g, true)
	if e == nil {
		c.enabled = enabled
	}
	return e
}
func (c *satSCC) resetStatistics() { c.numElim = 0 }
func (c *satSCC) collectStatistics(s *statistics.Statistics, key statistics.Key) error {
	if e := c.open(); e != nil {
		return e
	}
	s.UpdateUint(key, c.numElim)
	return nil
}
func (c *satSCC) closeVectors() error {
	r := c.lifecycle.propagation.memory
	for _, v := range [7]*satSCCVector{&c.toElim, &c.frames, &c.roots, &c.inStack, &c.stack, &c.lowlink, &c.index} {
		if e := v.close(r); e != nil {
			return e
		}
	}
	c.active = false
	if c.reportActive {
		if e := c.report.Stop(); e != nil {
			return e
		}
		c.reportActive = false
	}
	return nil
}
func (c *satSCC) Close() error {
	if c == nil || c.self != c {
		return ErrSATOwner
	}
	if c.closed {
		return nil
	}
	c.closing = true
	if e := c.eliminator.cleanupInterrupted(); e != nil {
		return e
	}
	if e := c.closeVectors(); e != nil {
		return e
	}
	c.lifecycle.budget.Release(c.header)
	c.header = 0
	c.closed = true
	return nil
}
func (c *satSCC) newNode(lidx uint32) error {
	p := c.lifecycle.propagation
	r := p.memory
	if e := c.index.put(r, lidx, 0, c.nextIndex); e != nil {
		return e
	}
	if e := c.lowlink.put(r, lidx, 0, c.nextIndex); e != nil {
		return e
	}
	c.nextIndex++
	if e := c.stack.push(r, lidx); e != nil {
		return e
	}
	if e := c.inStack.put(r, lidx, 0, 1); e != nil {
		return e
	}
	list, e := p.variables.watches.Get(lidx)
	if e != nil {
		return e
	}
	ref, e := list.ref()
	if e != nil {
		return e
	}
	it, e := list.owner.store.Iter(ref)
	if e != nil {
		return e
	}
	n, e := c.frames.size(r)
	if e != nil {
		return e
	}
	capacity, e := c.frames.cap(r)
	if e != nil {
		return e
	}
	if n >= capacity {
		if e = c.frames.grow(r); e != nil {
			return e
		}
	}
	if e = c.frames.put(r, n, 0, lidx); e != nil {
		return e
	}
	vals := [8]uint32{1, 0, it.end, it.capacity, uint32(it.payload.Slot), uint32(it.payload.Generation), uint32(it.lifetime.Slot), uint32(it.lifetime.Generation)}
	for k, x := range vals {
		if e = c.frames.put(r, n, uint32(k+1), x); e != nil {
			return e
		}
	}
	return r.WriteUint32(c.frames.raw, 4, n+1)
}
func (c *satSCC) frame(f uint32) (uint32, uint32, satSolverWatchIterator, error) {
	p := c.lifecycle.propagation
	r := p.memory
	var a [9]uint32
	for k := range a {
		x, e := c.frames.get(r, f, uint32(k))
		if e != nil {
			return 0, 0, satSolverWatchIterator{}, e
		}
		a[k] = x
	}
	return a[0], a[1], satSolverWatchIterator{store: p.variables.watches.store, index: a[2], end: a[3], capacity: a[4], payload: arena.VectorRef{Slot: arena.Slot(a[5]), Generation: arena.Generation(a[6])}, lifetime: arena.VectorRef{Slot: arena.Slot(a[7]), Generation: arena.Generation(a[8])}}, nil
}
func (c *satSCC) run() (count uint32, err error) {
	if e := c.open(); e != nil {
		return 0, e
	}
	p := c.lifecycle.propagation
	if p.inconsistent || !c.enabled {
		return 0, nil
	}
	if c.active || c.reportActive {
		return 0, arena.ErrClosed
	}
	if satStorageSourceDebug && c.debugInvariant {
		if e := c.lifecycle.checkInvariant(); e != nil {
			return 0, e
		}
	}
	c.report = stopwatch.Stopwatch{}
	if e := c.report.Start(); e != nil {
		return 0, e
	}
	c.reportActive = true
	c.bodyCause = nil
	c.cleanupCause = nil
	c.active = true
	defer func() {
		c.bodyCause = err
		if e := c.closeVectors(); e != nil {
			c.cleanupCause = e
			if err == nil {
				err = e
			} else if !errors.Is(err, e) {
				err = errors.Join(err, e)
			}
		}
	}()
	r := p.memory
	nv, e := p.variables.level.Size()
	if e != nil {
		return 0, e
	}
	nl := nv * 2
	c.nextIndex = 0
	for _, v := range [2]*satSCCVector{&c.index, &c.lowlink} {
		if e := v.resize(r, nl, ^uint32(0)); e != nil {
			return 0, e
		}
	}
	if e := c.inStack.resize(r, nl, 0); e != nil {
		return 0, e
	}
	if e := c.roots.resize(r, nv, uint32(DefaultLiteral())); e != nil {
		return 0, e
	}
	for lidx := uint32(0); lidx < nl; lidx++ {
		ix, e := c.index.get(r, lidx, 0)
		if e != nil {
			return 0, e
		}
		if ix != ^uint32(0) {
			continue
		}
		el, e := p.variables.wasEliminated(Literal(lidx).variable())
		if e != nil {
			return 0, e
		}
		if el {
			continue
		}
		if e = p.checkpoint(); e != nil {
			return 0, e
		}
		if e = c.newNode(lidx); e != nil {
			return 0, e
		}
		for {
			fn, e := c.frames.size(r)
			if e != nil {
				return 0, e
			}
			if fn == 0 {
				break
			}
			f := fn - 1
			li, first, it, e := c.frame(f)
			if e != nil {
				return 0, e
			}
			if first == 0 {
				w, e := satPropagationCursorRead(&it, it.index)
				if e != nil {
					return 0, e
				}
				lit, e := w.Literal()
				if e != nil {
					return 0, e
				}
				child, e := c.index.get(r, lit.index(), 0)
				if e != nil {
					return 0, e
				}
				if satStorageSourceDebug && child == ^uint32(0) {
					return 0, ErrSATAssertion
				}
				lo, e := c.lowlink.get(r, lit.index(), 0)
				if e != nil {
					return 0, e
				}
				here, e := c.lowlink.get(r, li, 0)
				if e != nil {
					return 0, e
				}
				if lo < here {
					if e = c.lowlink.put(r, li, 0, lo); e != nil {
						return 0, e
					}
				}
				it.index++
			}
			if e = c.frames.put(r, f, 1, 0); e != nil {
				return 0, e
			}
			descend := false
			for it.index != it.end {
				w, e := satPropagationCursorRead(&it, it.index)
				if e != nil {
					return 0, e
				}
				if w.Kind() != 0 {
					it.index++
					continue
				}
				lit, e := w.Literal()
				if e != nil {
					return 0, e
				}
				ix, e := c.index.get(r, lit.index(), 0)
				if e != nil {
					return 0, e
				}
				if ix == ^uint32(0) {
					if e = c.frames.put(r, f, 2, it.index); e != nil {
						return 0, e
					}
					if e = c.newNode(lit.index()); e != nil {
						return 0, e
					}
					descend = true
					break
				}
				inside, e := c.inStack.get(r, lit.index(), 0)
				if e != nil {
					return 0, e
				}
				if inside != 0 {
					lo, e := c.lowlink.get(r, li, 0)
					if e != nil {
						return 0, e
					}
					if ix < lo {
						if e = c.lowlink.put(r, li, 0, ix); e != nil {
							return 0, e
						}
					}
				}
				it.index++
			}
			if descend {
				continue
			}
			if e = c.frames.put(r, f, 2, it.index); e != nil {
				return 0, e
			}
			lo, e := c.lowlink.get(r, li, 0)
			if e != nil {
				return 0, e
			}
			ix, e = c.index.get(r, li, 0)
			if e != nil {
				return 0, e
			}
			if lo == ix {
				if e = c.component(li); e != nil {
					return 0, e
				}
				if p.inconsistent {
					return 0, nil
				}
			}
			if e = c.frames.pop(r); e != nil {
				return 0, e
			}
		}
	}
	n, e := c.toElim.size(r)
	if e != nil {
		return 0, e
	}
	c.numElim += n
	if e = c.eliminator.runViews(&c.roots, &c.toElim); e != nil {
		return 0, e
	}
	if satStorageSourceDebug && c.debugInvariant {
		if e := c.lifecycle.checkInvariant(); e != nil {
			return 0, e
		}
	}
	return n, nil
}
func (c *satSCC) component(li uint32) error {
	p := c.lifecycle.propagation
	r := p.memory
	l := Literal(li)
	v := l.variable()
	rv, e := c.roots.get(r, uint32(v), 0)
	if e != nil {
		return e
	}
	sn, e := c.stack.size(r)
	if e != nil {
		return e
	}
	if sn == 0 {
		return ErrSATAssertion
	}
	if Literal(rv) != DefaultLiteral() {
		for {
			n, e := c.stack.size(r)
			if e != nil {
				return e
			}
			x, e := c.stack.get(r, n-1, 0)
			if e != nil {
				return e
			}
			if e = c.stack.pop(r); e != nil {
				return e
			}
			if e = c.inStack.put(r, x, 0, 0); e != nil {
				return e
			}
			if satStorageSourceDebug {
				rr, e := c.roots.get(r, uint32(Literal(x).variable()), 0)
				if e != nil {
					return e
				}
				if Literal(rr).variable() != Literal(rv).variable() {
					return ErrSATAssertion
				}
			}
			if x == li {
				break
			}
		}
		return nil
	}
	root := DefaultLiteral()
	j := sn - 1
	for {
		x, e := c.stack.get(r, j, 0)
		if e != nil {
			return e
		}
		j--
		if Literal(x) == l.Negated() {
			p.setConflict(satJustification{}, DefaultLiteral())
			return nil
		}
		ext, e := p.variables.IsExternal(Literal(x).variable())
		if e != nil {
			return e
		}
		if ext {
			root = Literal(x)
			break
		}
		if x == li {
			break
		}
	}
	if root == DefaultLiteral() {
		root = l
	}
	for {
		n, e := c.stack.size(r)
		if e != nil {
			return e
		}
		x, e := c.stack.get(r, n-1, 0)
		if e != nil {
			return e
		}
		if e = c.stack.pop(r); e != nil {
			return e
		}
		if e = c.inStack.put(r, x, 0, 0); e != nil {
			return e
		}
		lit := Literal(x)
		vv := lit.variable()
		rr, e := c.roots.get(r, uint32(vv), 0)
		if e != nil {
			return e
		}
		if Literal(rr) == DefaultLiteral() {
			rr = uint32(root)
			if lit.sign() {
				rr = uint32(root.Negated())
			}
			if e = c.roots.put(r, uint32(vv), 0, rr); e != nil {
				return e
			}
			if vv != root.variable() {
				if e = c.toElim.push(r, uint32(vv)); e != nil {
					return e
				}
			}
		}
		if x == li {
			break
		}
	}
	return nil
}

func collectSCCParamDescrs(d *util.ParamDescrs) error {
	return satConfigInsertDescriptor(d, "scc", util.ParamBool, "eliminate Boolean variables by computing strongly connected components", "true")
}

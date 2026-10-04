// Source: src/sat/sat_solver.cpp. Licensed under the MIT license.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"reflect"
)

// Selected source stats/member facade; no additional solver counters are copied.
type satSolverDynamicSubsumption struct {
	self       *satSolverDynamicSubsumption
	analysis   *satSolverConflictAnalysis
	probing    *satSolverProbing
	budget     *arena.Budget
	ownerBytes uint64
	eliminated uint32
	closed     bool
}

func newSATSolverDynamicSubsumption(a *satSolverConflictAnalysis, p *satSolverProbing, b *arena.Budget) (*satSolverDynamicSubsumption, error) {
	if e := a.open(); e != nil {
		return nil, e
	}
	if e := p.open(); e != nil {
		return nil, e
	}
	if a.propagation != p.propagation() || b != a.budget {
		return nil, ErrSATOwner
	}
	n := uint64(reflect.TypeOf(satSolverDynamicSubsumption{}).Size())
	if e := b.Reserve(n); e != nil {
		return nil, e
	}
	d := &satSolverDynamicSubsumption{analysis: a, probing: p, budget: b, ownerBytes: n}
	d.self = d
	return d, nil
}
func (d *satSolverDynamicSubsumption) open() error {
	if d == nil || d.self != d {
		return ErrSATOwner
	}
	if d.closed {
		return arena.ErrClosed
	}
	if e := d.analysis.open(); e != nil {
		return e
	}
	return d.probing.open()
}
func (d *satSolverDynamicSubsumption) Close() error {
	if d == nil || d.self != d {
		return ErrSATOwner
	}
	if d.closed {
		return nil
	}
	d.budget.Release(d.ownerBytes)
	d.ownerBytes = 0
	d.closed = true
	return nil
}
func (d *satSolverDynamicSubsumption) marked(l satLiteral) (bool, error) {
	n, e := d.analysis.propagation.variables.literalMark.read(l.index())
	return n != 0, e
}
func (d *satSolverDynamicSubsumption) mark(l satLiteral) error {
	if satStorageSourceDebug {
		yes, e := d.marked(l)
		if e != nil {
			return e
		}
		if yes {
			return ErrSATAssertion
		}
	}
	return d.analysis.propagation.variables.literalMark.write(l.index(), 1)
}
func (d *satSolverDynamicSubsumption) unmark(l satLiteral) error {
	if satStorageSourceDebug {
		yes, e := d.marked(l)
		if e != nil {
			return e
		}
		if !yes {
			return ErrSATAssertion
		}
	}
	return d.analysis.propagation.variables.literalMark.write(l.index(), 0)
}
func (d *satSolverDynamicSubsumption) eliminateIfMarked(l, fuip satLiteral) error {
	yes, e := d.marked(l)
	if e != nil {
		return e
	}
	if yes && fuip != l {
		return d.unmark(l)
	}
	return nil
}
func (d *satSolverDynamicSubsumption) run() error {
	if e := d.open(); e != nil {
		return e
	}
	lemma := &d.analysis.lemma
	n, e := lemma.size()
	if e != nil {
		return e
	}
	for i := uint32(0); i < n; i++ {
		l, e := lemma.read(i)
		if e != nil {
			return e
		}
		if e = d.mark(satLiteral(l)); e != nil {
			return e
		}
	}
	first, e := lemma.read(0)
	if e != nil {
		return e
	}
	fuip := satLiteral(first)
	for i := uint32(0); i < n; i++ {
		value, e := lemma.read(i)
		if e != nil {
			return e
		}
		l := satLiteral(value)
		yes, e := d.marked(l)
		if e != nil {
			return e
		}
		if !yes {
			continue
		}
		list, e := d.analysis.propagation.variables.watches.Get(l.negated().index())
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
		for {
			w, ok, e := it.Next()
			if e != nil {
				return e
			}
			if !ok {
				break
			}
			if w.Binary() {
				other, e := w.Literal()
				if e != nil {
					return e
				}
				if e = d.eliminateIfMarked(other.negated(), fuip); e != nil {
					return e
				}
			} else if w.Ternary() {
				a, e := w.Literal1()
				if e != nil {
					return e
				}
				b, e := w.Literal2()
				if e != nil {
					return e
				}
				ma, e := d.marked(a)
				if e != nil {
					return e
				}
				firstBranch := false
				if ma {
					mb, e := d.marked(b.negated())
					if e != nil {
						return e
					}
					if mb && fuip != b.negated() {
						if e = d.unmark(b.negated()); e != nil {
							return e
						}
						firstBranch = true
					}
				}
				if !firstBranch {
					ma, e := d.marked(a.negated())
					if e != nil {
						return e
					}
					if ma {
						mb, e := d.marked(b)
						if e != nil {
							return e
						}
						if mb && fuip != a.negated() {
							if e = d.unmark(a.negated()); e != nil {
								return e
							}
						}
					}
				}
			} else {
				break
			}
		}
		cached, found, e := d.probing.cachedImpliedLits(l.negated())
		if e != nil {
			return e
		}
		if found {
			block, end, e := cached.capture()
			if e != nil {
				return e
			}
			for j := uint32(0); j < end; j++ {
				other, e := cached.readCaptured(block, j)
				if e != nil {
					return e
				}
				if e = d.eliminateIfMarked(other.negated(), fuip); e != nil {
					return e
				}
			}
		}
	}
	if satStorageSourceDebug {
		first, e := lemma.read(0)
		if e != nil {
			return e
		}
		yes, e := d.marked(satLiteral(first))
		if e != nil {
			return e
		}
		if !yes {
			return ErrSATAssertion
		}
	}
	j := uint32(0)
	for i := uint32(0); i < n; i++ {
		x, e := lemma.read(i)
		if e != nil {
			return e
		}
		l := satLiteral(x)
		yes, e := d.marked(l)
		if e != nil {
			return e
		}
		if yes {
			if e = d.unmark(l); e != nil {
				return e
			}
			if e = lemma.write(j, x); e != nil {
				return e
			}
			j++
		}
	}
	d.eliminated += n - j
	if satStorageSourceDebug && j < 1 {
		return ErrSATAssertion
	}
	return lemma.shrink(j)
}

// Source: src/sat/sat_solver.cpp:2818-2838,2852-2857,2901-2924; src/sat/sat_solver.h:286,543,566.
// Licensed under the MIT license. Source author: Leonardo de Moura, 2011-05-21.
package g3

func (s *satSolver) numClauses() (uint32, error) {
	if e := s.open(); e != nil {
		return 0, e
	}
	n, e := s.variables.trail.Size(s.variables.trailRef)
	if e != nil {
		return 0, e
	}
	for idx := 0; idx < len(s.variables.watches.lists); idx++ {
		l := satLiteral(uint32(idx)).negated()
		list, e := s.variables.watches.Get(uint32(idx))
		if e != nil {
			return 0, e
		}
		r, e := list.ref()
		if e != nil {
			return 0, e
		}
		it, e := s.variables.watches.store.Iter(r)
		if e != nil {
			return 0, e
		}
		for {
			w, yes, e := it.Next()
			if e != nil {
				return 0, e
			}
			if !yes {
				break
			}
			if w.Binary() {
				l2, e := w.Literal()
				if e != nil {
					return 0, e
				}
				if l.index() < l2.index() {
					n++
				}
			}
		}
	}
	n += uint32(len(s.lifecycle.original))
	n += uint32(len(s.lifecycle.learned))
	return n, nil
}
func (s *satSolver) checkMarks() error {
	if e := s.open(); e != nil {
		return e
	}
	for v := uint32(0); ; v++ {
		n, e := s.variables.level.size()
		if e != nil {
			return e
		}
		if v >= n {
			break
		}
		if satStorageSourceDebug {
			marked, e := s.conflict.analysis.marked(v)
			if e != nil {
				return e
			}
			if marked {
				return ErrSATAssertion
			}
		}
	}
	return nil
}

func (s *satSolver) collectBinClauses(out *satSolverBinPairVector, learned bool) error {
	if e := s.open(); e != nil {
		return e
	}
	sz := len(s.variables.watches.lists)
	for idx := 0; idx < sz; idx++ {
		l := satLiteral(uint32(idx)).negated()
		list, e := s.variables.watches.Get(uint32(idx))
		if e != nil {
			return e
		}
		r, e := list.ref()
		if e != nil {
			return e
		}
		it, e := s.variables.watches.store.Iter(r)
		if e != nil {
			return e
		}
		for {
			w, yes, e := it.Next()
			if e != nil {
				return e
			}
			if !yes {
				break
			}
			if !w.Binary() {
				continue
			}
			if !learned {
				yes, e := w.Learned()
				if e != nil {
					return e
				}
				if yes {
					continue
				}
			}
			l2, e := w.Literal()
			if e != nil {
				return e
			}
			if l.index() > l2.index() {
				continue
			}
			if e = out.Push(newSATSolverBinPair(l, l2)); e != nil {
				return e
			}
		}
	}
	return nil
}

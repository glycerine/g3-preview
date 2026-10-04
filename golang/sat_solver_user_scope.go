// Source: src/sat/sat_solver.cpp:2620–2756,2818–2841; src/sat/sat_simplifier.h:190–194.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package g3

import "github.com/glycerine/g3/golang/arena"

// Actual disjoint clause-vector owner ranges during unsafe-exception compaction.
// The current source cohort is not copied. Kept [0,out), unprocessed [in,end),
// and a current deletion stage describe the actual remaining scalar ownership.
type satSolverUserGC struct {
	in, out, end, drainPrefix, drainSuffix uint32
	learned, running, active               bool
}

func (s *satSolver) userGCVector() *[]satStoredClauseRef {
	if s.userGC.learned {
		return &s.lifecycle.learned
	}
	return &s.lifecycle.original
}
func (s *satSolver) cleanupUserGC() error {
	g := &s.userGC
	if !g.active {
		return nil
	}
	vec := s.userGCVector()
	if uint64(g.end) > uint64(len(*vec)) {
		return arena.ErrBounds
	}
	for g.drainPrefix < g.out {
		c := (*vec)[g.drainPrefix]
		if e := s.clauses.Delete(c); e != nil {
			return e
		}
		g.drainPrefix++
	}
	if g.drainSuffix < g.in {
		g.drainSuffix = g.in
	}
	for g.drainSuffix < g.end {
		c := (*vec)[g.drainSuffix]
		if e := s.clauses.Delete(c); e != nil {
			return e
		}
		g.drainSuffix++
	}
	*vec = (*vec)[:0]
	g.active = false
	s.lifecycle.userContinuation = nil
	return nil
}
func (s *satSolver) gcLit(learned bool, l satLiteral) (err error) {
	g := &s.userGC
	*g = satSolverUserGC{learned: learned, running: true, active: true}
	vec := s.userGCVector()
	g.end = uint32(len(*vec))
	s.lifecycle.userContinuation = s
	defer func() {
		g.running = false
		if err == nil {
			g.active = false
			s.lifecycle.userContinuation = nil
		}
	}()
	for g.in < uint32(len(*vec)) {
		c := (*vec)[g.in]
		yes, e := s.clauses.ContainsLiteral(c, l)
		if e != nil {
			return e
		}
		if !yes {
			yes, e = s.clauses.ContainsLiteral(c, l.negated())
			if e != nil {
				return e
			}
		}
		if yes {
			if e = s.lifecycle.detachClause(c); e != nil {
				return e
			}
			if e = s.lifecycle.delClause(c); e != nil {
				return e
			}
		} else {
			(*vec)[g.out] = c
			g.out++
		}
		g.in++
	}
	*vec = (*vec)[:g.out]
	return nil
}
func (s *satSolver) gcBin(l satLiteral) error {
	for _, r := range s.variables.watches.lists {
		store := s.variables.watches.store
		n, e := store.Size(r)
		if e != nil {
			return e
		}
		out := uint32(0)
		for i := uint32(0); i < n; i++ {
			w, e := store.Read(r, i)
			if e != nil {
				return e
			}
			skip := false
			if w.Binary() {
				a, e := w.Literal()
				if e != nil {
					return e
				}
				skip = a.variable() == l.variable()
			}
			if !skip {
				if e = store.Write(r, out, w); e != nil {
					return e
				}
				out++
			}
		}
		if e = store.Shrink(r, out); e != nil {
			return e
		}
	}
	return nil
}
func (s *satSolver) maxVarClauses(cs []satStoredClauseRef, v uint32) (uint32, error) {
	for _, c := range cs {
		n, e := s.clauses.Size(c)
		if e != nil {
			return v, e
		}
		for i := uint32(0); i < n; i++ {
			l, e := s.clauses.Literal(c, i)
			if e != nil {
				return v, e
			}
			if uint32(l.variable()) > v {
				v = uint32(l.variable())
			}
		}
	}
	return v, nil
}
func (s *satSolver) maxVarBins(learned bool, v uint32) (uint32, error) {
	if e := s.userBinClauses.Reset(); e != nil {
		return v, e
	}
	for idx, r := range s.variables.watches.lists {
		l := satLiteral(idx).negated()
		n, e := s.variables.watches.store.Size(r)
		if e != nil {
			return v, e
		}
		for i := uint32(0); i < n; i++ {
			w, e := s.variables.watches.store.Read(r, i)
			if e != nil {
				return v, e
			}
			if !w.Binary() {
				continue
			}
			if !learned {
				yes, e := w.Learned()
				if e != nil {
					return v, e
				}
				if yes {
					continue
				}
			}
			l2, e := w.Literal()
			if e != nil {
				return v, e
			}
			if l.index() > l2.index() {
				continue
			}
			yes, e := w.Learned()
			if e != nil {
				return v, e
			}
			if e = s.userBinClauses.Push(newSATBinClause(l, l2, yes)); e != nil {
				return v, e
			}
		}
	}
	for i := uint32(0); i < s.userBinClauses.Size(); i++ {
		c, e := s.userBinClauses.Get(i)
		if e != nil {
			return v, e
		}
		if uint32(c.Literal1().variable()) > v {
			v = uint32(c.Literal1().variable())
		}
		if uint32(c.Literal2().variable()) > v {
			v = uint32(c.Literal2().variable())
		}
	}
	return v, nil
}
func (s *satSolver) gcVar(v uint32) error {
	if v > 0 {
		w, e := s.maxVarClauses(s.lifecycle.learned, v-1)
		if e != nil {
			return e
		}
		w, e = s.maxVarClauses(s.lifecycle.original, w)
		if e != nil {
			return e
		}
		w, e = s.maxVarBins(true, w)
		if e != nil {
			return e
		}
		w, e = s.maxVarBins(false, w)
		if e != nil {
			return e
		}
		v, e = s.converter.MaxVar(w)
		if e != nil {
			return e
		}
		for i := uint32(0); ; i++ {
			n, e := s.variables.trail.Size(s.variables.trailRef)
			if e != nil {
				return e
			}
			if i >= n {
				break
			}
			l, e := s.variables.trailRead(i)
			if e != nil {
				return e
			}
			if uint32(l.variable()) > w {
				w = uint32(l.variable())
			}
		}
		if w+1 > v {
			v = w + 1
		}
	}
	n, e := s.variables.level.size()
	if e != nil {
		return e
	}
	if v >= n {
		return nil
	}
	for i := v; ; i++ {
		n, e := s.variables.level.size()
		if e != nil {
			return e
		}
		if i >= n {
			break
		}
		if e = s.variables.queue.DelVar(satBoolVar(i)); e != nil {
			return e
		}
		if e = s.probing.resetCache(satMakeLiteral(satBoolVar(i), true)); e != nil {
			return e
		}
		if e = s.probing.resetCache(satMakeLiteral(satBoolVar(i), false)); e != nil {
			return e
		}
	}
	x := s.variables
	if e = x.watches.Shrink(2 * v); e != nil {
		return e
	}
	if e = x.assignment.shrink(2 * v); e != nil {
		return e
	}
	if e = x.justification.raw.shrink(v); e != nil {
		return e
	}
	if e = x.decision.shrink(v); e != nil {
		return e
	}
	if e = x.eliminated.shrink(v); e != nil {
		return e
	}
	if e = x.external.shrink(v); e != nil {
		return e
	}
	if e = x.activity.values.shrink(v); e != nil {
		return e
	}
	if e = x.level.shrink(v); e != nil {
		return e
	}
	if e = x.mark.shrink(v); e != nil {
		return e
	}
	if e = x.literalMark.shrink(2 * v); e != nil {
		return e
	}
	if e = x.phase.shrink(v); e != nil {
		return e
	}
	if e = x.previousPhase.shrink(v); e != nil {
		return e
	}
	if e = x.assignedSinceGC.shrink(v); e != nil {
		return e
	}
	if e = x.elimTodo.Reset(); e != nil {
		return e
	}
	if e = s.simplifier.subTodo.Reset(); e != nil {
		return e
	}
	return s.simplifier.subBinTodo.Reset()
}
func (s *satSolver) UserPush() error {
	if e := s.open(); e != nil {
		return e
	}
	v, e := s.variables.mkVar(true, false)
	if e != nil {
		return e
	}
	return s.conflict.userScopeLiterals.push(uint32(satMakeLiteral(v, false)))
}
func (s *satSolver) UserPop(scopes uint32) error {
	if e := s.open(); e != nil {
		return e
	}
	if e := s.popToBase(); e != nil {
		return e
	}
	for scopes > 0 {
		n, e := s.conflict.userScopeLiterals.size()
		if e != nil {
			return e
		}
		raw, e := s.conflict.userScopeLiterals.read(n - 1)
		if e != nil {
			return e
		}
		l := satLiteral(raw)
		if e = s.conflict.userScopeLiterals.shrink(n - 1); e != nil {
			return e
		}
		for _, lit := range [2]satLiteral{l, l.negated()} {
			list, e := s.variables.watches.Get(lit.index())
			if e != nil {
				return e
			}
			if e = list.SetEnd(0); e != nil {
				return e
			}
		}
		if e = s.gcLit(true, l); e != nil {
			return e
		}
		if e = s.gcLit(false, l); e != nil {
			return e
		}
		if e = s.gcBin(l); e != nil {
			return e
		}
		scopes--
		for i := uint32(0); ; i++ {
			n, e := s.variables.trail.Size(s.variables.trailRef)
			if e != nil {
				return e
			}
			if i >= n {
				break
			}
			a, e := s.variables.trailRead(i)
			if e != nil {
				return e
			}
			if a == l {
				if e = s.lifecycle.unassignVars(i); e != nil {
					return e
				}
				break
			}
		}
		if e = s.gcVar(uint32(l.variable())); e != nil {
			return e
		}
	}
	s.variables.qhead = 0
	_, e := s.lifecycle.propagate(false, false, false)
	return e
}

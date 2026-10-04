// Source: src/sat/sat_solver.cpp:75-127; src/sat/sat_solver.h:257.
// Licensed under the MIT license. Source authors: Leonardo de Moura (leonardo), 2011-05-21; Nikolaj Bjorner.
package g3

import "github.com/glycerine/g3/golang/arena"

func (s *satSolver) copyFrom(src *satSolver) (err error) {
	if err = s.open(); err != nil {
		return err
	}
	if src == nil || src.self != src {
		return ErrSATOwner
	}
	if err = s.popToBase(); err != nil {
		return err
	}
	if satStorageSourceDebug {
		if !s.converter.Empty() || !src.converter.Empty() {
			return ErrSATAssertion
		}
		if s.variables.scopeLevel != 0 {
			return ErrSATAssertion
		}
	}
	n, e := s.variables.level.size()
	if e != nil {
		return e
	}
	sn, e := src.variables.level.size()
	if e != nil {
		return e
	}
	if n < sn {
		v, e := s.variables.level.size()
		if e != nil {
			return e
		}
		for ; ; v++ {
			sn, e = src.variables.level.size()
			if e != nil {
				return e
			}
			if v >= sn {
				break
			}
			if satStorageSourceDebug {
				elim, e := src.variables.wasEliminated(satBoolVar(v))
				if e != nil {
					return e
				}
				if elim {
					return ErrSATAssertion
				}
			}
			ext, e := src.variables.external.read(v)
			if e != nil {
				return e
			}
			dvar, e := src.variables.decision.read(v)
			if e != nil {
				return e
			}
			created, e := s.MkVar(ext != 0, dvar != 0)
			if e != nil {
				return e
			}
			if satStorageSourceDebug && uint32(created) != v {
				return ErrSATAssertion
			}
		}
	}
	var sz uint32
	if src.variables.scopeLevel == 0 {
		sz, err = src.variables.trail.Size(src.variables.trailRef)
	} else {
		if len(src.lifecycle.scopes) == 0 {
			return arena.ErrBounds
		}
		sz = src.lifecycle.scopes[0].trailLimit
	}
	if err != nil {
		return err
	}
	for i := uint32(0); i < sz; i++ {
		l, e := src.variables.trailRead(i)
		if e != nil {
			return e
		}
		if err = s.propagation.assign(l, satJustification{}); err != nil {
			return err
		}
	}
	// The source captures outer size and each actual watch begin/end. Actual
	// arena iterator generations reject a freed captured payload at the read.
	size := len(src.variables.watches.lists)
	for idx := 0; idx < size; idx++ {
		l := satLiteral(uint32(idx)).negated()
		list, e := src.variables.watches.Get(uint32(idx))
		if e != nil {
			return e
		}
		ref, e := list.ref()
		if e != nil {
			return e
		}
		it, e := src.variables.watches.store.Iter(ref)
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
			// sat_watched.h:88 uses the full packed word, not only kind
			// bits plus the narrower is_learned()==1 predicate.
			if w.val2 != 0 {
				continue
			}
			l2, e := w.Literal()
			if e != nil {
				return e
			}
			if l.index() > l2.index() {
				continue
			}
			pair := [2]satLiteral{l, l2}
			if _, err = s.lifecycle.mkClauseCore(2, pair[:], false); err != nil {
				return err
			}
		}
	}
	if err = s.copyOriginalClauses(src); err != nil {
		return err
	}
	if err = s.conflict.userScopeLiterals.shrink(0); err != nil {
		return err
	}
	return satMUSAppend(&s.conflict.userScopeLiterals, &src.conflict.userScopeLiterals)
}
func (s *satSolver) copyOriginalClauses(src *satSolver) (err error) {
	previous := s.consequenceRunning
	s.consequenceRunning = true
	defer func() { s.consequenceRunning = previous }()
	f, e := s.consequenceFrame()
	if e != nil {
		return e
	}
	defer func() { err = s.consequenceDrop(f, err) }()
	if err = f.vector(s, 0, nil); err != nil {
		return err
	}
	buffer := &f.vectors[0]
	original := src.lifecycle.original
	// This concrete charged clause-vector append helper replaces backing only
	// when its geometric capacity strictly increases. The captured source end
	// remains fixed; guard relocation at each reached old-pointer dereference.
	originalCapacity := cap(original)
	for i := range original {
		if e := src.identity(); e != nil {
			return e
		}
		if cap(src.lifecycle.original) != originalCapacity {
			return ErrSATOwner
		}
		c := original[i]
		if err = buffer.shrink(0); err != nil {
			return err
		}
		for i := uint32(0); ; i++ {
			n, e := src.clauses.Size(c)
			if e != nil {
				return e
			}
			if i >= n {
				break
			}
			l, e := src.clauses.Literal(c, i)
			if e != nil {
				return e
			}
			if err = buffer.push(l.index()); err != nil {
				return err
			}
		}
		n, e := buffer.size()
		if e != nil {
			return e
		}
		if _, err = s.lifecycle.mkClauseCoreSpan(n, satConflictRuntimeSpan(buffer), false); err != nil {
			return err
		}
	}
	return nil
}

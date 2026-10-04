// Source: src/sat/sat_solver.cpp:2859-3017; src/sat/sat_types.h:295-324.
// Licensed under the MIT license. Source author: Leonardo de Moura, 2011-05-21.
package g3

import "github.com/glycerine/g3/golang/arena"

func satDIMACSLiteralInto(out *NumericTextStream, l satLiteral) error {
	sign := ""
	if l.sign() {
		sign = "-"
	}
	if e := out.AppendCString(sign); e != nil {
		return e
	}
	return out.AppendUint32(uint32(l.variable()) + 1)
}
func (s *satSolver) displayBinary(out *NumericTextStream) error {
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
			l2, e := w.Literal()
			if e != nil {
				return e
			}
			if l.index() > l2.index() {
				continue
			}
			if e = out.AppendCString("("); e != nil {
				return e
			}
			if e = satLiteralInto(out, l); e != nil {
				return e
			}
			if e = out.AppendCString(" "); e != nil {
				return e
			}
			if e = satLiteralInto(out, l2); e != nil {
				return e
			}
			if e = out.AppendCString(")\n"); e != nil {
				return e
			}
		}
	}
	return nil
}
func (s *satSolver) displayUnits(out *NumericTextStream) error {
	if e := s.open(); e != nil {
		return e
	}
	var n uint32
	var e error
	if s.variables.scopeLevel == 0 {
		n, e = s.variables.trail.Size(s.variables.trailRef)
	} else {
		if len(s.lifecycle.scopes) == 0 {
			return arena.ErrBounds
		}
		n = s.lifecycle.scopes[0].trailLimit
	}
	if e != nil {
		return e
	}
	for i := uint32(0); i < n; i++ {
		l, e := s.variables.trailRead(i)
		if e != nil {
			return e
		}
		if e = satLiteralInto(out, l); e != nil {
			return e
		}
		if e = out.AppendCString(" "); e != nil {
			return e
		}
	}
	if n != 0 {
		return out.AppendCString("\n")
	}
	return nil
}
func (s *satSolver) displayInto(out *NumericTextStream) error {
	if e := s.open(); e != nil {
		return e
	}
	if e := out.AppendCString("(sat\n"); e != nil {
		return e
	}
	if e := s.displayUnits(out); e != nil {
		return e
	}
	if e := s.displayBinary(out); e != nil {
		return e
	}
	for _, cs := range [2][]satStoredClauseRef{s.lifecycle.original, s.lifecycle.learned} {
		for _, c := range cs {
			if e := s.displaySourceClause(out, c); e != nil {
				return e
			}
			if e := out.AppendCString("\n"); e != nil {
				return e
			}
		}
	}
	return out.AppendCString(")\n")
}
func (s *satSolver) displayJustification(out *NumericTextStream, j satJustification) error {
	if e := s.open(); e != nil {
		return e
	}
	if e := j.DisplayInto(out); e != nil {
		return e
	}
	if j.Clause() {
		c, e := s.clauses.GetClause(j.ClauseOffset())
		if e != nil {
			return e
		}
		return s.displaySourceClause(out, c)
	}
	return nil
}

// These explicit mode branches correspond to the two literal source loop
// bodies. Source request/read order is preserved; no output snapshot is built.
func (s *satSolver) displayHardClauses(out *NumericTextStream, weighted bool, weight uint32) error {
	for i := uint32(0); ; i++ {
		n, e := s.variables.trail.Size(s.variables.trailRef)
		if e != nil {
			return e
		}
		if i >= n {
			break
		}
		if weighted {
			if e = out.AppendUint32(weight); e != nil {
				return e
			}
			if e = out.AppendCString(" "); e != nil {
				return e
			}
		}
		l, e := s.variables.trailRead(i)
		if e != nil {
			return e
		}
		if e = satDIMACSLiteralInto(out, l); e != nil {
			return e
		}
		if e = out.AppendCString(" 0\n"); e != nil {
			return e
		}
	}
	for idx := 0; idx < len(s.variables.watches.lists); idx++ {
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
			l2, e := w.Literal()
			if e != nil {
				return e
			}
			if l.index() >= l2.index() {
				continue
			}
			if weighted {
				if e = out.AppendUint32(weight); e != nil {
					return e
				}
				if e = out.AppendCString(" "); e != nil {
					return e
				}
			}
			if e = satDIMACSLiteralInto(out, l); e != nil {
				return e
			}
			if e = out.AppendCString(" "); e != nil {
				return e
			}
			l2, e = w.Literal()
			if e != nil {
				return e
			}
			if e = satDIMACSLiteralInto(out, l2); e != nil {
				return e
			}
			if e = out.AppendCString(" 0\n"); e != nil {
				return e
			}
		}
	}
	for _, cs := range [2][]satStoredClauseRef{s.lifecycle.original, s.lifecycle.learned} {
		for _, c := range cs {
			n, e := s.clauses.Size(c)
			if e != nil {
				return e
			}
			if weighted {
				if e = out.AppendUint32(weight); e != nil {
					return e
				}
				if e = out.AppendCString(" "); e != nil {
					return e
				}
			}
			for j := uint32(0); j < n; j++ {
				l, e := s.clauses.Literal(c, j)
				if e != nil {
					return e
				}
				if e = satDIMACSLiteralInto(out, l); e != nil {
					return e
				}
				if e = out.AppendCString(" "); e != nil {
					return e
				}
			}
			if e = out.AppendCString("0\n"); e != nil {
				return e
			}
		}
	}
	return nil
}
func (s *satSolver) displayDIMACS(out *NumericTextStream) error {
	if e := s.open(); e != nil {
		return e
	}
	if e := out.AppendCString("p cnf "); e != nil {
		return e
	}
	n, e := s.variables.level.size()
	if e != nil {
		return e
	}
	if e = out.AppendUint32(n); e != nil {
		return e
	}
	if e = out.AppendCString(" "); e != nil {
		return e
	}
	n, e = s.numClauses()
	if e != nil {
		return e
	}
	if e = out.AppendUint32(n); e != nil {
		return e
	}
	if e = out.AppendCString("\n"); e != nil {
		return e
	}
	return s.displayHardClauses(out, false, 0)
}
func (s *satSolver) displayWCNF(out *NumericTextStream, count uint32, lits []satLiteral, weights []uint32) error {
	if e := s.open(); e != nil {
		return e
	}
	max := uint32(0)
	for i := uint32(0); i < count; i++ {
		if uint64(i) >= uint64(cap(weights)) {
			return arena.ErrBounds
		}
		w := weights[:cap(weights)][i]
		if w > max {
			max = w
		}
	}
	max++
	if e := out.AppendCString("p wcnf "); e != nil {
		return e
	}
	n, e := s.variables.level.size()
	if e != nil {
		return e
	}
	if e = out.AppendUint32(n); e != nil {
		return e
	}
	if e = out.AppendCString(" "); e != nil {
		return e
	}
	n, e = s.numClauses()
	if e != nil {
		return e
	}
	if e = out.AppendUint32(n + count); e != nil {
		return e
	}
	if e = out.AppendCString(" "); e != nil {
		return e
	}
	if e = out.AppendUint32(max); e != nil {
		return e
	}
	if e = out.AppendCString("\nc soft "); e != nil {
		return e
	}
	if e = out.AppendUint32(count); e != nil {
		return e
	}
	if e = out.AppendCString("\n"); e != nil {
		return e
	}
	if e = s.displayHardClauses(out, true, max); e != nil {
		return e
	}
	for i := uint32(0); i < count; i++ {
		if uint64(i) >= uint64(cap(weights)) {
			return arena.ErrBounds
		}
		if e = out.AppendUint32(weights[:cap(weights)][i]); e != nil {
			return e
		}
		if e = out.AppendCString(" "); e != nil {
			return e
		}
		if uint64(i) >= uint64(cap(lits)) {
			return arena.ErrBounds
		}
		if e = satLiteralInto(out, lits[:cap(lits)][i]); e != nil {
			return e
		}
		if e = out.AppendCString(" 0\n"); e != nil {
			return e
		}
	}
	return nil // memory-only borrowed stream: source flush has no external sink.
}
func (s *satSolver) displayWatches(out *NumericTextStream) error {
	if e := s.open(); e != nil {
		return e
	}
	for idx := 0; idx < len(s.variables.watches.lists); idx++ {
		list, e := s.variables.watches.Get(uint32(idx))
		if e != nil {
			return e
		}
		if e = satLiteralInto(out, satLiteral(uint32(idx))); e != nil {
			return e
		}
		if e = out.AppendCString(": "); e != nil {
			return e
		}
		r, e := list.ref()
		if e != nil {
			return e
		}
		if e = s.variables.watches.store.DisplayInto(r, s.clauses, out); e != nil {
			return e
		}
		if e = out.AppendCString("\n"); e != nil {
			return e
		}
	}
	return nil
}
func (s *satSolver) displayAssignment(out *NumericTextStream) error {
	if e := s.open(); e != nil {
		return e
	}
	n, e := s.variables.trail.Size(s.variables.trailRef)
	if e != nil {
		return e
	}
	for i := uint32(0); i < n; i++ {
		if i > 0 {
			if e = out.AppendCString(" "); e != nil {
				return e
			}
		}
		l, e := s.variables.trailRead(i)
		if e != nil {
			return e
		}
		if e = satLiteralInto(out, l); e != nil {
			return e
		}
	}
	return out.AppendCString("\n")
}

// An explicit borrowed memory stream replaces source std::cout transport.
// Unbound dispatch remains an error; no synthetic display or search result.
func (s *satSolver) bindDIMACS(out *NumericTextStream) error {
	if e := s.open(); e != nil {
		return e
	}
	if e := out.lock(); e != nil {
		return e
	}
	out.mu.Unlock()
	s.dimacsOutput = out
	return nil
}

// Direct source clause insertion preserves punctuation before reached size
// reads, unlike the independently existing allocator rendering adapter.
func (s *satSolver) displaySourceClause(out *NumericTextStream, c satStoredClauseRef) error {
	if e := out.AppendCString("("); e != nil {
		return e
	}
	for i := uint32(0); ; i++ {
		n, e := s.clauses.Size(c)
		if e != nil {
			return e
		}
		if i >= n {
			break
		}
		if i > 0 {
			if e = out.AppendCString(" "); e != nil {
				return e
			}
		}
		l, e := s.clauses.Literal(c, i)
		if e != nil {
			return e
		}
		if e = satLiteralInto(out, l); e != nil {
			return e
		}
	}
	if e := out.AppendCString(")"); e != nil {
		return e
	}
	for _, f := range [3]struct {
		bit  uint32
		text string
	}{{2, "x"}, {1, "+"}, {4, "*"}} {
		yes, e := s.clauses.flag(c, f.bit)
		if e != nil {
			return e
		}
		if yes {
			if e = out.AppendCString(f.text); e != nil {
				return e
			}
		}
	}
	return nil
}

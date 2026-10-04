// Licensed under the MIT license.
package g3

func (s *satSolverPropagationState) setConflict(c satJustification, l satLiteral) {
	if s.inconsistent {
		return
	}
	s.inconsistent = true
	s.conflict = c
	s.notLiteral = l
}
func (s *satSolverPropagationState) assign(l satLiteral, j satJustification) error {
	if e := s.open(); e != nil {
		return e
	}
	v, e := s.variables.valueLiteral(l)
	if e != nil {
		return e
	}
	switch v {
	case lFalse:
		s.setConflict(j, l.negated())
	case lUndef:
		return s.assignCore(l, j)
	}
	return nil
}
func (s *satSolverPropagationState) assignCore(l satLiteral, j satJustification) error {
	if e := s.open(); e != nil {
		return e
	}
	v := s.variables
	if satStorageSourceDebug {
		value, e := v.valueLiteral(l)
		if e != nil {
			return e
		}
		if value != lUndef {
			return ErrSATAssertion
		}
	}
	if v.scopeLevel == 0 {
		j = satJustification{}
	}
	if e := v.assignment.write(l.index(), int8(lTrue)); e != nil {
		return e
	}
	if e := v.assignment.write(l.negated().index(), int8(lFalse)); e != nil {
		return e
	}
	id := uint32(l.variable())
	if e := v.level.write(id, v.scopeLevel); e != nil {
		return e
	}
	if e := v.justification.write(id, j); e != nil {
		return e
	}
	phase := int8(0)
	if l.sign() {
		phase = 1
	}
	if e := v.phase.write(id, phase); e != nil {
		return e
	}
	if e := v.assignedSinceGC.write(id, 1); e != nil {
		return e
	}
	if e := v.trailPush(l); e != nil {
		return e
	}
	if s.extensionPresent {
		ext, e := v.external.read(id)
		if e != nil {
			return e
		}
		if ext != 0 {
			return ErrSATMissingSourceExtBinding
		}
	}
	if satStorageSourceDebug {
		p, e := v.phase.read(id)
		if e != nil {
			return e
		}
		if (l.sign() && p != 1) || (!l.sign() && p != 0) {
			return ErrSATAssertion
		}
		a, e := v.valueVar(l.variable())
		if e != nil {
			return e
		}
		if (l.sign() && a != lFalse) || (!l.sign() && a != lTrue) {
			return ErrSATAssertion
		}
		a, e = v.valueLiteral(l)
		if e != nil {
			return e
		}
		if a != lTrue {
			return ErrSATAssertion
		}
		a, e = v.valueLiteral(l.negated())
		if e != nil {
			return e
		}
		if a != lFalse {
			return ErrSATAssertion
		}
	}
	return nil
}
func (s *satSolverPropagationState) status(c satStoredClauseRef) (lbool, error) {
	if e := s.open(); e != nil {
		return lUndef, e
	}
	n, e := s.clauses.Size(c)
	if e != nil {
		return lUndef, e
	}
	found := false
	for i := uint32(0); i < n; i++ {
		l, e := s.clauses.Literal(c, i)
		if e != nil {
			return lUndef, e
		}
		v, e := s.variables.valueLiteral(l)
		if e != nil {
			return lUndef, e
		}
		switch v {
		case lTrue:
			return lTrue, nil
		case lUndef:
			found = true
		}
	}
	if found {
		return lUndef, nil
	}
	return lFalse, nil
}

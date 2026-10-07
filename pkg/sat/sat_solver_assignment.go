// Licensed under the MIT license.
package sat

import (
	"github.com/glycerine/g3/pkg/util"
)

func (s *satSolverPropagationState) setConflict(c satJustification, l Literal) {
	if s.inconsistent {
		return
	}
	s.inconsistent = true
	s.conflict = c
	s.notLiteral = l
}
func (s *satSolverPropagationState) assign(l Literal, j satJustification) error {
	if e := s.open(); e != nil {
		return e
	}
	v, e := s.variables.valueLiteral(l)
	if e != nil {
		return e
	}
	switch v {
	case util.LFalse:
		s.setConflict(j, l.Negated())
	case util.LUndef:
		return s.assignCore(l, j)
	}
	return nil
}
func (s *satSolverPropagationState) assignCore(l Literal, j satJustification) error {
	if e := s.open(); e != nil {
		return e
	}
	v := s.variables
	if satStorageSourceDebug {
		value, e := v.valueLiteral(l)
		if e != nil {
			return e
		}
		if value != util.LUndef {
			return ErrSATAssertion
		}
	}
	if v.scopeLevel == 0 {
		j = satJustification{}
	}
	if e := v.assignment.Write(l.index(), int8(util.LTrue)); e != nil {
		return e
	}
	if e := v.assignment.Write(l.Negated().index(), int8(util.LFalse)); e != nil {
		return e
	}
	id := uint32(l.variable())
	if e := v.level.Write(id, v.scopeLevel); e != nil {
		return e
	}
	if e := v.justification.write(id, j); e != nil {
		return e
	}
	phase := int8(0)
	if l.sign() {
		phase = 1
	}
	if e := v.phase.Write(id, phase); e != nil {
		return e
	}
	if e := v.assignedSinceGC.Write(id, 1); e != nil {
		return e
	}
	if e := v.trailPush(l); e != nil {
		return e
	}
	if s.extensionPresent {
		ext, e := v.external.Read(id)
		if e != nil {
			return e
		}
		if ext != 0 {
			return ErrSATMissingSourceExtBinding
		}
	}
	if satStorageSourceDebug {
		p, e := v.phase.Read(id)
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
		if (l.sign() && a != util.LFalse) || (!l.sign() && a != util.LTrue) {
			return ErrSATAssertion
		}
		a, e = v.valueLiteral(l)
		if e != nil {
			return e
		}
		if a != util.LTrue {
			return ErrSATAssertion
		}
		a, e = v.valueLiteral(l.Negated())
		if e != nil {
			return e
		}
		if a != util.LFalse {
			return ErrSATAssertion
		}
	}
	return nil
}
func (s *satSolverPropagationState) status(c StoredClauseRef) (util.LBool, error) {
	if e := s.open(); e != nil {
		return util.LUndef, e
	}
	n, e := s.clauses.Size(c)
	if e != nil {
		return util.LUndef, e
	}
	found := false
	for i := uint32(0); i < n; i++ {
		l, e := s.clauses.Literal(c, i)
		if e != nil {
			return util.LUndef, e
		}
		v, e := s.variables.valueLiteral(l)
		if e != nil {
			return util.LUndef, e
		}
		switch v {
		case util.LTrue:
			return util.LTrue, nil
		case util.LUndef:
			found = true
		}
	}
	if found {
		return util.LUndef, nil
	}
	return util.LFalse, nil
}

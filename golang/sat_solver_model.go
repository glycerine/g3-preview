// Source: src/sat/sat_solver.cpp:1257–1346; src/sat/sat_clause.cpp.
// Licensed under the MIT license. Source author: Leonardo de Moura, 2011.
package g3

func (s *satSolver) mkModel() error {
	if e := s.open(); e != nil {
		return e
	}
	if e := s.model.shrink(0); e != nil {
		return e
	}
	s.variables.modelIsCurrent = true
	s.variables.modelKnown = true
	n, e := s.variables.level.size()
	if e != nil {
		return e
	}
	if e = satSolverResizeModel(&s.model, n); e != nil {
		return e
	}
	model := satSolverModelSpan{column: &s.model}
	for v := uint32(0); v < n; v++ {
		eliminated, e := s.variables.wasEliminated(satBoolVar(v))
		if e != nil {
			return e
		}
		if !eliminated {
			x, e := s.variables.valueVar(satBoolVar(v))
			if e != nil {
				return e
			}
			if e = model.write(satBoolVar(v), x); e != nil {
				return e
			}
		}
	}
	if e = s.converter.applyModelSpan(model); e != nil {
		return e
	}
	// Selected ordinary source profile does not define _EXTERNAL_RELEASE.
	good, e := s.checkModel()
	if e != nil {
		return e
	}
	if !good {
		return &SATSolverException{DefaultException: NewDefaultException("check model failed")}
	}
	return nil
}
func (s *satSolver) checkModel() (bool, error) {
	return s.checkModelSpan(satSolverModelSpan{column: &s.model})
}

// Source check_model accepts a borrowed model, independently of m_model.
func (s *satSolver) checkModelSpan(model satSolverModelSpan) (bool, error) {
	if e := s.open(); e != nil {
		return false, e
	}
	ok := true
	for _, cs := range [2][]satStoredClauseRef{s.lifecycle.original, s.lifecycle.learned} {
		for _, c := range cs {
			yes, e := s.clauses.satisfiedByModelSpan(c, model)
			if e != nil {
				return false, e
			}
			if !yes {
				ok = false
			}
		}
	}
	for idx := uint32(0); idx < uint32(len(s.variables.watches.lists)); idx++ {
		value, e := model.value(satLiteral(idx).negated())
		if e != nil {
			return false, e
		}
		if value != lTrue {
			list, e := s.variables.watches.Get(idx)
			if e != nil {
				return false, e
			}
			n, e := list.Size()
			if e != nil {
				return false, e
			}
			for i := uint32(0); i < n; i++ {
				w, e := list.Read(i)
				if e != nil {
					return false, e
				}
				if !w.Binary() {
					continue
				}
				l, e := w.Literal()
				if e != nil {
					return false, e
				}
				value, e = model.value(l)
				if e != nil {
					return false, e
				}
				if value != lTrue {
					ok = false
				}
			}
		}
	}
	n, e := s.conflict.assumptions.size()
	if e != nil {
		return false, e
	}
	for i := uint32(0); i < n; i++ {
		l, e := s.conflict.assumptions.read(i)
		if e != nil {
			return false, e
		}
		v, e := model.value(satLiteral(l))
		if e != nil {
			return false, e
		}
		if v != lTrue {
			ok = false
		}
	}
	if ok {
		return s.converter.checkModelSpan(model)
	}
	return false, nil
}

// The source model getter is borrowed. This explicit append adapter returns
// caller-owned output while all working model storage stays in the Runtime.
func (s *satSolver) AppendModel(dst satModel) (satModel, error) {
	if e := s.open(); e != nil {
		return dst, e
	}
	n, e := s.model.size()
	if e != nil {
		return dst, e
	}
	model := satSolverModelSpan{column: &s.model}
	for i := uint32(0); i < n; i++ {
		v, e := model.read(satBoolVar(i))
		if e != nil {
			return dst, e
		}
		dst = append(dst, v)
	}
	return dst, nil
}

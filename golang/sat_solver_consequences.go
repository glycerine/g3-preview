// Source: src/sat/sat_solver.cpp:3252–3471; src/sat/sat_solver.h:477–503.
// Licensed under the MIT license. Source author: Nikolaj Bjorner.
package g3

func (s *satSolver) consequenceEntry(out *satSolverLiteralRows) error {
	if e := s.open(); e != nil {
		return e
	}
	if e := out.open(); e != nil {
		return e
	}
	if out.runtime != s.runtime.memory || out.budget != s.budget {
		return ErrSATOwner
	}
	return nil
}
func (s *satSolver) getConsequences(asms []satLiteral, vars []satBoolVar, out *satSolverLiteralRows) (lbool, error) {
	return s.getConsequencesInputs(satConsequenceLiterals(asms), satConsequenceInput{variables: vars, count: uint32(len(vars))}, out)
}
func (s *satSolver) getConsequencesInputs(asms, vars satConsequenceInput, out *satSolverLiteralRows) (result lbool, err error) {
	if err = s.consequenceEntry(out); err != nil {
		return lUndef, err
	}
	s.consequenceRunning = true
	defer func() { s.consequenceRunning = false }()
	f, e := s.consequenceFrame()
	if e != nil {
		return lUndef, e
	}
	defer func() { err = s.consequenceDrop(f, err) }()
	if err = f.vector(s, 0, nil); err != nil {
		return lUndef, err
	}
	result = lTrue
	cfg, e := s.config.State()
	if e != nil {
		return lUndef, e
	}
	if cfg.RestartMax != ^uint32(0) && !s.variables.modelIsCurrent {
		return s.getBoundedConsequencesBody(asms, vars, out)
	}
	if !s.variables.modelIsCurrent {
		n, e := asms.size()
		if e != nil {
			return lUndef, e
		}
		result, err = s.checkSpan(n, asms.cPtr())
		if err != nil {
			return lUndef, err
		}
	}
	if result != lTrue {
		return result, nil
	}
	if err = f.vector(s, 1, &s.model); err != nil {
		return lUndef, err
	}
	for i := uint32(0); ; i++ {
		n, e := vars.size()
		if e != nil {
			return lUndef, e
		}
		if i >= n {
			break
		}
		v, e := vars.read(i)
		if e != nil {
			return lUndef, e
		}
		// Source get_model()[v] reads the live member, not saved mdl.
		value, e := s.model.read(v)
		if e != nil {
			return lUndef, e
		}
		if lbool(int32(value)) == lTrue {
			err = f.vectors[0].push(satMakeLiteral(satBoolVar(v), false).index())
		} else if lbool(int32(value)) == lFalse {
			err = f.vectors[0].push(satMakeLiteral(satBoolVar(v), true).index())
		}
		if err != nil {
			return lUndef, err
		}
	}
	// The literal source false&&asms.empty() core_chunking dispatch remains
	// unreachable; that separate static algorithm family is explicitly pending.
	result, err = s.getLiteralConsequencesBody(asms, satConsequenceColumn(&f.vectors[0]), out)
	if err != nil {
		return result, err
	}
	if err = s.setModel(&f.vectors[1]); err != nil {
		return result, err
	}
	return result, nil
}
func (s *satSolver) getLiteralConsequences(asms, lits satConsequenceInput, out *satSolverLiteralRows) (result lbool, err error) {
	if err = s.consequenceEntry(out); err != nil {
		return lUndef, err
	}
	s.consequenceRunning = true
	defer func() { s.consequenceRunning = false }()
	return s.getLiteralConsequencesBody(asms, lits, out)
}
func (s *satSolver) consequenceInitialize(asms satConsequenceInput) (result lbool, err error) {
	if err = s.popToBase(); err != nil {
		return lUndef, err
	}
	if s.propagation.inconsistent {
		return lFalse, nil
	}
	if err = s.gc.initSearch(); err != nil {
		return lUndef, err
	}
	if _, err = s.lifecycle.propagate(false, false, false); err != nil {
		return lUndef, err
	}
	if s.propagation.inconsistent {
		return lFalse, nil
	}
	n, e := asms.size()
	if e != nil {
		return lUndef, e
	}
	if n == 0 {
		v, e := s.MkVar(true, false)
		if e != nil {
			return lUndef, e
		}
		lit := [1]satLiteral{satMakeLiteral(v, false)}
		err = s.initAssumptionsSpan(1, satSolverLiteralSpan{literals: lit[:]})
	} else {
		err = s.initAssumptionsSpan(n, asms.cPtr())
	}
	if err != nil {
		return lUndef, err
	}
	if _, err = s.lifecycle.propagate(false, false, false); err != nil {
		return lUndef, err
	}
	bad, e := s.checkInconsistent()
	if e != nil {
		return lUndef, e
	}
	if bad {
		return lFalse, nil
	}
	return lTrue, nil
}
func (s *satSolver) getLiteralConsequencesBody(asms, lits satConsequenceInput, out *satSolverLiteralRows) (result lbool, err error) {
	// Source reset precedes actual unfixed_lits, assumptions, bool_var_set ctors.
	if err = s.antecedents.reset(); err != nil {
		return lUndef, err
	}
	f, e := s.consequenceFrame()
	if e != nil {
		return lUndef, e
	}
	defer func() { err = s.consequenceDrop(f, err) }()
	if err = f.set(s, 0, &lits); err != nil {
		return lUndef, err
	}
	if err = f.set(s, 1, &asms); err != nil {
		return lUndef, err
	}
	if err = f.set(s, 2, nil); err != nil {
		return lUndef, err
	}
	unfixed, assumptions, vars := &f.sets[0], &f.sets[1], &f.sets[2]
	for i := uint32(0); ; i++ {
		n, e := lits.size()
		if e != nil {
			return lUndef, e
		}
		if i >= n {
			break
		}
		v, e := lits.read(i)
		if e != nil {
			return lUndef, e
		}
		if err = vars.insert(satLiteral(uint32(satLiteral(v).variable()))); err != nil {
			return lUndef, err
		}
	}
	result, err = s.consequenceInitialize(asms)
	if err != nil || result != lTrue {
		return result, err
	}
	iterations := uint32(0)
	if err = s.extractFixedSet(unfixed, assumptions, vars, out); err != nil {
		return lUndef, err
	}
	if err = s.updateUnfixed(unfixed, vars); err != nil {
		return lUndef, err
	}
	for {
		n, e := unfixed.size()
		if e != nil {
			return lUndef, e
		}
		if n == 0 {
			return lTrue, nil
		}
		if s.variables.scopeLevel > 1 {
			if err = s.lifecycle.pop(s.variables.scopeLevel - 1); err != nil {
				return lUndef, err
			}
		}
		if _, err = s.lifecycle.propagate(false, false, false); err != nil {
			return lUndef, err
		}
		iterations++
		if err = s.propagation.checkpoint(); err != nil {
			return lUndef, err
		}
		span := satConflictRuntimeSpan(&unfixed.values)
		n, e = unfixed.size()
		if e != nil {
			return lUndef, e
		}
		resolves := uint32(0)
		fixed := uint32(0)
		assigned := uint32(0)
		isSat := lTrue
		for i := uint32(0); i < n; i++ {
			lit, e := span.literal(i)
			if e != nil {
				return lUndef, e
			}
			value, e := s.variables.valueLiteral(lit)
			if e != nil {
				return lUndef, e
			}
			if value != lUndef {
				fixed++
				level, e := s.variables.levelVar(lit.variable())
				if e != nil {
					return lUndef, e
				}
				if level <= 1 {
					value, e = s.variables.valueLiteral(lit)
					if e != nil {
						return lUndef, e
					}
					if value == lTrue {
						if err = s.extractFixedConsequence(lit, assumptions, vars, out); err != nil {
							return lUndef, err
						}
					}
				}
				continue
			}
			if err = s.lifecycle.push(); err != nil {
				return lUndef, err
			}
			assigned++
			if err = s.propagation.assign(lit.negated(), satJustification{}); err != nil {
				return lUndef, err
			}
			if _, err = s.lifecycle.propagate(false, false, false); err != nil {
				return lUndef, err
			}
			for s.propagation.inconsistent {
				yes, e := s.conflict.resolveConflict()
				if e != nil {
					return lUndef, e
				}
				if !yes {
					s.propagation.inconsistent = false
					isSat = lUndef
					break
				}
				if _, err = s.lifecycle.propagate(false, false, false); err != nil {
					return lUndef, err
				}
				resolves++
			}
			// Literal source false&&scope_lvl()==1 branch remains unreachable.
		}
		_ = fixed
		_ = assigned // Source verbose-only counters; no fake report output.
		if err = s.extractFixedSet(unfixed, assumptions, vars, out); err != nil {
			return lUndef, err
		}
		if isSat == lTrue {
			if s.variables.scopeLevel == 1 && resolves > 0 {
				isSat = lUndef
			} else {
				isSat, err = s.boundedSearch()
				if err != nil {
					return lUndef, err
				}
				if isSat == lUndef {
					if err = s.gc.restart(); err != nil {
						return lUndef, err
					}
				}
				if err = s.extractFixedSet(unfixed, assumptions, vars, out); err != nil {
					return lUndef, err
				}
			}
		}
		if isSat == lFalse {
			s.propagation.inconsistent = false
		}
		if isSat == lTrue {
			if err = s.deleteUnfixed(unfixed, vars); err != nil {
				return lUndef, err
			}
		}
		if err = s.updateUnfixed(unfixed, vars); err != nil {
			return lUndef, err
		}
		n, e = unfixed.size()
		if e != nil {
			return lUndef, e
		}
		if n != 0 {
			cfg, e := s.config.State()
			if e != nil {
				return lUndef, e
			}
			if cfg.RestartMax <= iterations {
				return lUndef, nil
			}
		}
	}
}
func (s *satSolver) getBoundedConsequences(asms, vars satConsequenceInput, out *satSolverLiteralRows) (result lbool, err error) {
	if err = s.consequenceEntry(out); err != nil {
		return lUndef, err
	}
	s.consequenceRunning = true
	defer func() { s.consequenceRunning = false }()
	return s.getBoundedConsequencesBody(asms, vars, out)
}
func (s *satSolver) boundedExtract(start *uint32, asms satConsequenceInput, vars *satMUSLiteralSet, out *satSolverLiteralRows) (err error) {
	// The literal_set(asms) converting temporary is a real full-expression owner
	// for EACH source call, destroyed before the next simplify/search action.
	f, e := s.consequenceFrame()
	if e != nil {
		return e
	}
	defer func() { err = s.consequenceDrop(f, err) }()
	if err = f.set(s, 0, &asms); err != nil {
		return err
	}
	return s.extractFixedTrail(start, &f.sets[0], vars, out)
}
func (s *satSolver) getBoundedConsequencesBody(asms, vars satConsequenceInput, out *satSolverLiteralRows) (result lbool, err error) {
	f, e := s.consequenceFrame()
	if e != nil {
		return lUndef, e
	}
	defer func() { err = s.consequenceDrop(f, err) }()
	if err = f.set(s, 0, nil); err != nil {
		return lUndef, err
	}
	unfixed := &f.sets[0]
	units, iterations := uint32(0), uint32(0)
	for i := uint32(0); ; i++ {
		n, e := vars.size()
		if e != nil {
			return lUndef, e
		}
		if i >= n {
			break
		}
		v, e := vars.read(i)
		if e != nil {
			return lUndef, e
		}
		if err = unfixed.insert(satLiteral(v)); err != nil {
			return lUndef, err
		}
	}
	if err = s.antecedents.reset(); err != nil {
		return lUndef, err
	}
	result, err = s.consequenceInitialize(asms)
	if err != nil || result != lTrue {
		return result, err
	}
	if err = s.boundedExtract(&units, asms, unfixed, out); err != nil {
		return lUndef, err
	}
	if err = s.simplifyProblem(); err != nil {
		return lUndef, err
	}
	bad, e := s.checkInconsistent()
	if e != nil {
		return lUndef, e
	}
	if bad {
		err = s.fixupConsequenceCore()
		return lFalse, err
	}
	for {
		iterations++
		if satStorageSourceDebug && s.propagation.inconsistent {
			return lUndef, ErrSATAssertion
		}
		result, err = s.boundedSearch()
		if err != nil {
			return lUndef, err
		}
		if result != lUndef {
			err = s.fixupConsequenceCore()
			return result, err
		}
		if err = s.boundedExtract(&units, asms, unfixed, out); err != nil {
			return lUndef, err
		}
		cfg, e := s.config.State()
		if e != nil {
			return lUndef, e
		}
		if s.conflict.conflicts > cfg.MaxConflicts {
			return lUndef, nil
		}
		if err = s.gc.restart(); err != nil {
			return lUndef, err
		}
		if err = s.simplifyProblem(); err != nil {
			return lUndef, err
		}
		bad, e = s.checkInconsistent()
		if e != nil {
			return lUndef, e
		}
		if bad {
			err = s.fixupConsequenceCore()
			return lFalse, err
		}
		if err = s.gc.gc(); err != nil {
			return lUndef, err
		}
		cfg, e = s.config.State()
		if e != nil {
			return lUndef, e
		}
		if cfg.RestartMax <= iterations {
			return lUndef, nil
		}
	}
}

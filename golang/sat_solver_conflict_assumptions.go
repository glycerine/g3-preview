// Source: src/sat/sat_solver.cpp:898–926,1141–1167,1822–1883,1999–2087,2545–2549. Licensed under the MIT license.
package g3

func (s *satSolverConflictResolution) trackingAssumptions() (bool, error) {
	n, e := s.assumptions.size()
	if e != nil {
		return false, e
	}
	if n != 0 {
		return true, nil
	}
	n, e = s.userScopeLiterals.size()
	return n != 0, e
}
func (s *satSolverConflictResolution) isAssumption(l satLiteral) (bool, error) {
	yes, e := s.trackingAssumptions()
	if e != nil || !yes {
		return false, e
	}
	return s.assumptionSet.contains(l)
}
func (s *satSolverConflictResolution) exchangeParallel() error {
	// Source constructor m_par=nullptr, independently of configured num_threads.
	// No actual par instance is supplied yet; nonnull dispatch cannot fake exchange.
	if s.parallelPresent && s.analysis.propagation.variables.scopeLevel == 0 {
		return ErrSATMissingParallelBinding
	}
	return nil
}
func (s *satSolverConflictResolution) popReinit(n uint32) error {
	if e := s.open(); e != nil {
		return e
	}
	if e := s.dynamic.probing.lifecycle.pop(n); e != nil {
		return e
	}
	if e := s.exchangeParallel(); e != nil {
		return e
	}
	return s.reinitAssumptions()
}
func (s *satSolverConflictResolution) reinitAssumptions() error {
	if e := s.open(); e != nil {
		return e
	}
	yes, e := s.trackingAssumptions()
	if e != nil {
		return e
	}
	p := s.analysis.propagation
	if !yes || p.variables.scopeLevel != 0 || p.inconsistent {
		return nil
	}
	if e = s.dynamic.probing.lifecycle.push(); e != nil {
		return e
	}
	for i := uint32(0); !p.inconsistent; i++ {
		n, e := s.userScopeLiterals.size()
		if e != nil {
			return e
		}
		if i >= n {
			break
		}
		l, e := s.userScopeLiterals.read(i)
		if e != nil {
			return e
		}
		if e = p.assign(satLiteral(l).negated(), satJustification{}); e != nil {
			return e
		}
	}
	for i := uint32(0); !p.inconsistent; i++ {
		n, e := s.assumptions.size()
		if e != nil {
			return e
		}
		if i >= n {
			break
		}
		l, e := s.assumptions.read(i)
		if e != nil {
			return e
		}
		if e = p.assign(satLiteral(l), satJustification{}); e != nil {
			return e
		}
	}
	return nil
}
func (s *satSolverConflictResolution) processAntecedentForUnsatCore(l satLiteral) error {
	a := s.analysis
	v := a.propagation.variables
	id := uint32(l.variable())
	if satStorageSourceDebug {
		n, e := v.level.size()
		if e != nil {
			return e
		}
		if id >= n {
			return ErrSATAssertion
		}
	}
	marked, e := a.marked(id)
	if e != nil {
		return e
	}
	if marked {
		return nil
	}
	if e = a.mark(id); e != nil {
		return e
	}
	if e = a.unmark.push(id); e != nil {
		return e
	}
	yes, e := s.isAssumption(l)
	if e != nil {
		return e
	}
	if yes {
		return s.core.push(uint32(l))
	}
	return nil
}
func (s *satSolverConflictResolution) resolveConflictForUnsatCore() error {
	a := s.analysis
	p := a.propagation
	v := p.variables
	if e := s.core.shrink(0); e != nil {
		return e
	}
	if s.conflictLevel == 0 {
		return nil
	}
	if satStorageSourceDebug {
		n, e := a.unmark.size()
		if e != nil {
			return e
		}
		if n != 0 {
			return ErrSATAssertion
		}
		n, e = v.trail.Size(v.trailRef)
		if e != nil {
			return e
		}
		for i := uint32(0); i < n; i++ {
			lit, e := v.trailRead(i)
			if e != nil {
				return e
			}
			marked, e := a.marked(uint32(lit.variable()))
			if e != nil {
				return e
			}
			if marked {
				return ErrSATAssertion
			}
		}
	}
	old, e := a.unmark.size()
	if e != nil {
		return e
	}
	u, e := s.skipLiteralsAboveConflictLevel()
	if e != nil {
		return e
	}
	idx := int32(u)
	if p.notLiteral != satDefaultLiteral() {
		j, e := v.justification.read(uint32(p.notLiteral.variable()))
		if e != nil {
			return e
		}
		if e = s.processAntecedentForUnsatCore(p.notLiteral); e != nil {
			return e
		}
		yes, e := s.isAssumption(p.notLiteral.negated())
		if e != nil {
			return e
		}
		if yes {
			if e = s.core.push(uint32(p.notLiteral.negated())); e != nil {
				return e
			}
		} else if e = s.processConsequent(p.notLiteral, j, 2, nil); e != nil {
			return e
		}
	}
	consequent, j := p.notLiteral, p.conflict
	for {
		if e = s.processConsequent(consequent, j, 2, nil); e != nil {
			return e
		}
		for idx >= 0 {
			lit, e := v.trailRead(uint32(idx))
			if e != nil {
				return e
			}
			marked, e := a.marked(uint32(lit.variable()))
			if e != nil {
				return e
			}
			if marked {
				break
			}
			idx--
		}
		if idx < 0 {
			break
		}
		consequent, e = v.trailRead(uint32(idx))
		if e != nil {
			return e
		}
		level, e := v.levelVar(consequent.variable())
		if e != nil {
			return e
		}
		if level < s.conflictLevel {
			break
		}
		if satStorageSourceDebug && level != s.conflictLevel {
			return ErrSATAssertion
		}
		j, e = v.justification.read(uint32(consequent.variable()))
		if e != nil {
			return e
		}
		idx--
	}
	if e = a.resetUnmark(old); e != nil {
		return e
	}
	cfg, e := p.config.State()
	if e != nil {
		return e
	}
	if cfg.CoreMinimize {
		if s.minCoreValid {
			mn, e := s.minCore.size()
			if e != nil {
				return e
			}
			cn, e := s.core.size()
			if e != nil {
				return e
			}
			if mn < cn {
				if e = s.core.shrink(0); e != nil {
					return e
				}
				span := satConflictRuntimeSpan(&s.minCore)
				for i := uint32(0); i < mn; i++ {
					l, e := span.literal(i)
					if e != nil {
						return e
					}
					if e = s.core.push(uint32(l)); e != nil {
						return e
					}
				}
			}
		}
		return ErrSATMissingMUSBinding
	}
	return nil
}
func (s *satSolverConflictResolution) processAntecedentForInit(l satLiteral) error {
	a := s.analysis
	v := a.propagation.variables
	id := uint32(l.variable())
	if satStorageSourceDebug {
		n, e := v.level.size()
		if e != nil {
			return e
		}
		if id >= n {
			return ErrSATAssertion
		}
	}
	marked, e := a.marked(id)
	if e != nil {
		return e
	}
	if marked {
		return nil
	}
	level, e := v.levelVar(l.variable())
	if e != nil {
		return e
	}
	if level == 0 {
		return nil
	}
	if e = a.mark(id); e != nil {
		return e
	}
	return a.lemma.push(uint32(l.negated()))
}
func (s *satSolverConflictResolution) resolveConflictForInit() (bool, error) {
	if e := s.open(); e != nil {
		return false, e
	}
	if s.conflictLevel == 0 {
		return false, nil
	}
	a := s.analysis
	p := a.propagation
	v := p.variables
	if e := a.lemma.shrink(0); e != nil {
		return false, e
	}
	if e := a.lemma.push(uint32(satDefaultLiteral())); e != nil {
		return false, e
	}
	if p.notLiteral != satDefaultLiteral() {
		if e := s.processAntecedentForInit(p.notLiteral); e != nil {
			return false, e
		}
	}
	consequent, j := p.notLiteral, p.conflict
	idx, e := v.trail.Size(v.trailRef)
	if e != nil {
		return false, e
	}
	if satStorageSourceDebug && idx == 0 {
		return false, ErrSATAssertion
	}
	for !j.None() {
		if e = s.processConsequent(consequent, j, 1, nil); e != nil {
			return false, e
		}
		for {
			idx--
			l, e := v.trailRead(idx)
			if e != nil {
				return false, e
			}
			marked, e := a.marked(uint32(l.variable()))
			if e != nil {
				return false, e
			}
			if marked {
				break
			}
			if satStorageSourceDebug && idx == 0 {
				return false, ErrSATAssertion
			}
		}
		consequent, e = v.trailRead(idx)
		if e != nil {
			return false, e
		}
		id := uint32(consequent.variable())
		level, e := v.levelVar(consequent.variable())
		if e != nil {
			return false, e
		}
		if level == 0 {
			return false, nil
		}
		if satStorageSourceDebug && s.conflictLevel != 1 {
			return false, ErrSATAssertion
		}
		j, e = v.justification.read(id)
		if e != nil {
			return false, e
		}
		if e = a.resetMark(id); e != nil {
			return false, e
		}
	}
	newLevel := uint32(0)
	if e = a.lemma.write(0, uint32(consequent.negated())); e != nil {
		return false, e
	}
	for i := uint32(1); ; i++ {
		n, e := a.lemma.size()
		if e != nil {
			return false, e
		}
		if i >= n {
			break
		}
		lit, e := a.lemma.read(i)
		if e != nil {
			return false, e
		}
		id := uint32(satLiteral(lit).variable())
		marked, e := a.marked(id)
		if e != nil {
			return false, e
		}
		if marked {
			if e = a.resetMark(id); e != nil {
				return false, e
			}
			level, e := v.levelVar(satBoolVar(id))
			if e != nil {
				return false, e
			}
			if level > newLevel {
				newLevel = level
			}
		} else {
			last, e := a.lemma.read(n - 1)
			if e != nil {
				return false, e
			}
			if e = a.lemma.write(i, last); e != nil {
				return false, e
			}
			if e = a.lemma.shrink(n - 1); e != nil {
				return false, e
			}
			i--
		}
	}
	if newLevel == 0 {
		if e = s.popReinit(v.scopeLevel); e != nil {
			return false, e
		}
	} else {
		if e = s.dynamic.probing.lifecycle.unassignVars(idx); e != nil {
			return false, e
		}
	}
	n, e := a.lemma.size()
	if e != nil {
		return false, e
	}
	_, e = s.dynamic.probing.lifecycle.mkClauseCoreSpan(n, satConflictRuntimeSpan(&a.lemma), true)
	if e != nil {
		return false, e
	}
	p.inconsistent = false
	return true, nil
}

// Explicit checked owner-domain accessor useful for source consumer bindings.
func (s *satSolverConflictResolution) coreLiteral(i uint32) (satLiteral, error) {
	if e := s.open(); e != nil {
		return 0, e
	}
	v, e := s.core.read(i)
	return satLiteral(v), e
}

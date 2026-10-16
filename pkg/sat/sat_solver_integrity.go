// Source: src/sat/sat_solver.cpp:711–716,2845–2850,3053–3075.
package sat

func (l *satSolverClauseLifecycle) bindIntegrity(g *satSolverGC) error {
	if e := l.open(); e != nil {
		return e
	}
	if g == nil || g.self != g || g.lifecycle() != l {
		return ErrSATOwner
	}
	if e := g.open(); e != nil {
		return e
	}
	if l.integrityGC != nil && l.integrityGC != g {
		return ErrSATOwner
	}
	l.integrityGC = g
	return nil
}
func (l *satSolverClauseLifecycle) checkInvariant() error {
	return l.checkInvariantWithActions(false)
}
func (l *satSolverClauseLifecycle) checkInvariantWithActions(actions bool) error {
	if e := l.open(); e != nil {
		return e
	}
	if !l.propagation.limit.Inc() {
		return nil
	}
	// Source constructor only stores a reference; SASSERT(checker()) is erased
	// in release, so no GC/member read is introduced on that profile.
	if !satStorageSourceDebug {
		return nil
	}
	if l.integrityGC == nil {
		return ErrSATMissingIntegrityBinding
	}
	c, e := newSATIntegrityChecker(l, l.integrityGC)
	if e != nil {
		return e
	}
	c.debugActions = actions
	yes, e := c.run()
	if e != nil {
		return e
	}
	if !yes {
		return ErrSATAssertion
	}
	return nil
}
func (l *satSolverClauseLifecycle) propagate(update, checkInvariant, checkMissed bool) (bool, error) {
	result, e := l.propagation.propagateCore(update)
	if e != nil {
		return result, e
	}
	if satStorageSourceDebug && checkInvariant {
		if e = l.checkInvariant(); e != nil {
			return result, e
		}
	}
	if satStorageSourceDebug && checkMissed {
		if e = l.checkMissedPropagation(); e != nil {
			return result, e
		}
	}
	return result, nil
}

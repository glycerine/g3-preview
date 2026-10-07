// Source: src/sat/sat_solver.cpp:3252–3471; src/sat/sat_solver.h:477–503.
// Licensed under the MIT license. Source author: Nikolaj Bjorner.
package sat

import (
	"github.com/glycerine/g3/pkg/util"
)

func (s *Solver) consequenceEntry(out *satSolverLiteralRows) error {
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
func (s *Solver) getConsequences(asms []Literal, vars []BoolVar, out *satSolverLiteralRows) (util.LBool, error) {
	return s.getConsequencesInputs(satConsequenceLiterals(asms), satConsequenceInput{variables: vars, count: uint32(len(vars))}, out)
}
func (s *Solver) getConsequencesInputs(asms, vars satConsequenceInput, out *satSolverLiteralRows) (result util.LBool, err error) {
	if err = s.consequenceEntry(out); err != nil {
		return util.LUndef, err
	}
	s.consequenceRunning = true
	defer func() { s.consequenceRunning = false }()
	f, e := s.consequenceFrame()
	if e != nil {
		return util.LUndef, e
	}
	defer func() { err = s.consequenceDrop(f, err) }()
	if err = f.vector(s, 0, nil); err != nil {
		return util.LUndef, err
	}
	result = util.LTrue
	cfg, e := s.config.State()
	if e != nil {
		return util.LUndef, e
	}
	if cfg.RestartMax != ^uint32(0) && !s.Variables.modelIsCurrent {
		return s.getBoundedConsequencesBody(asms, vars, out)
	}
	if !s.Variables.modelIsCurrent {
		n, e := asms.size()
		if e != nil {
			return util.LUndef, e
		}
		result, err = s.checkSpan(n, asms.cPtr())
		if err != nil {
			return util.LUndef, err
		}
	}
	if result != util.LTrue {
		return result, nil
	}
	if err = f.vector(s, 1, &s.model); err != nil {
		return util.LUndef, err
	}
	for i := uint32(0); ; i++ {
		n, e := vars.size()
		if e != nil {
			return util.LUndef, e
		}
		if i >= n {
			break
		}
		v, e := vars.read(i)
		if e != nil {
			return util.LUndef, e
		}
		// Source get_model()[v] reads the live member, not saved mdl.
		value, e := s.model.Read(v)
		if e != nil {
			return util.LUndef, e
		}
		if util.LBool(int32(value)) == util.LTrue {
			err = f.vectors[0].Push(MakeLiteral(BoolVar(v), false).index())
		} else if util.LBool(int32(value)) == util.LFalse {
			err = f.vectors[0].Push(MakeLiteral(BoolVar(v), true).index())
		}
		if err != nil {
			return util.LUndef, err
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
func (s *Solver) getLiteralConsequences(asms, lits satConsequenceInput, out *satSolverLiteralRows) (result util.LBool, err error) {
	if err = s.consequenceEntry(out); err != nil {
		return util.LUndef, err
	}
	s.consequenceRunning = true
	defer func() { s.consequenceRunning = false }()
	return s.getLiteralConsequencesBody(asms, lits, out)
}
func (s *Solver) consequenceInitialize(asms satConsequenceInput) (result util.LBool, err error) {
	if err = s.popToBase(); err != nil {
		return util.LUndef, err
	}
	if s.propagation.inconsistent {
		return util.LFalse, nil
	}
	if err = s.gc.initSearch(); err != nil {
		return util.LUndef, err
	}
	if _, err = s.Lifecycle.propagate(false, false, false); err != nil {
		return util.LUndef, err
	}
	if s.propagation.inconsistent {
		return util.LFalse, nil
	}
	n, e := asms.size()
	if e != nil {
		return util.LUndef, e
	}
	if n == 0 {
		v, e := s.MkVar(true, false)
		if e != nil {
			return util.LUndef, e
		}
		lit := [1]Literal{MakeLiteral(v, false)}
		err = s.initAssumptionsSpan(1, satSolverLiteralSpan{literals: lit[:]})
	} else {
		err = s.initAssumptionsSpan(n, asms.cPtr())
	}
	if err != nil {
		return util.LUndef, err
	}
	if _, err = s.Lifecycle.propagate(false, false, false); err != nil {
		return util.LUndef, err
	}
	bad, e := s.checkInconsistent()
	if e != nil {
		return util.LUndef, e
	}
	if bad {
		return util.LFalse, nil
	}
	return util.LTrue, nil
}
func (s *Solver) getLiteralConsequencesBody(asms, lits satConsequenceInput, out *satSolverLiteralRows) (result util.LBool, err error) {
	// Source reset precedes actual unfixed_lits, assumptions, bool_var_set ctors.
	if err = s.antecedents.reset(); err != nil {
		return util.LUndef, err
	}
	f, e := s.consequenceFrame()
	if e != nil {
		return util.LUndef, e
	}
	defer func() { err = s.consequenceDrop(f, err) }()
	if err = f.set(s, 0, &lits); err != nil {
		return util.LUndef, err
	}
	if err = f.set(s, 1, &asms); err != nil {
		return util.LUndef, err
	}
	if err = f.set(s, 2, nil); err != nil {
		return util.LUndef, err
	}
	unfixed, assumptions, vars := &f.sets[0], &f.sets[1], &f.sets[2]
	for i := uint32(0); ; i++ {
		n, e := lits.size()
		if e != nil {
			return util.LUndef, e
		}
		if i >= n {
			break
		}
		v, e := lits.read(i)
		if e != nil {
			return util.LUndef, e
		}
		if err = vars.insert(Literal(uint32(Literal(v).variable()))); err != nil {
			return util.LUndef, err
		}
	}
	result, err = s.consequenceInitialize(asms)
	if err != nil || result != util.LTrue {
		return result, err
	}
	iterations := uint32(0)
	if err = s.extractFixedSet(unfixed, assumptions, vars, out); err != nil {
		return util.LUndef, err
	}
	if err = s.updateUnfixed(unfixed, vars); err != nil {
		return util.LUndef, err
	}
	for {
		n, e := unfixed.size()
		if e != nil {
			return util.LUndef, e
		}
		if n == 0 {
			return util.LTrue, nil
		}
		if s.Variables.scopeLevel > 1 {
			if err = s.Lifecycle.pop(s.Variables.scopeLevel - 1); err != nil {
				return util.LUndef, err
			}
		}
		if _, err = s.Lifecycle.propagate(false, false, false); err != nil {
			return util.LUndef, err
		}
		iterations++
		if err = s.propagation.checkpoint(); err != nil {
			return util.LUndef, err
		}
		span := satConflictRuntimeSpan(&unfixed.values)
		n, e = unfixed.size()
		if e != nil {
			return util.LUndef, e
		}
		resolves := uint32(0)
		fixed := uint32(0)
		assigned := uint32(0)
		isSat := util.LTrue
		for i := uint32(0); i < n; i++ {
			lit, e := span.literal(i)
			if e != nil {
				return util.LUndef, e
			}
			value, e := s.Variables.valueLiteral(lit)
			if e != nil {
				return util.LUndef, e
			}
			if value != util.LUndef {
				fixed++
				level, e := s.Variables.levelVar(lit.variable())
				if e != nil {
					return util.LUndef, e
				}
				if level <= 1 {
					value, e = s.Variables.valueLiteral(lit)
					if e != nil {
						return util.LUndef, e
					}
					if value == util.LTrue {
						if err = s.extractFixedConsequence(lit, assumptions, vars, out); err != nil {
							return util.LUndef, err
						}
					}
				}
				continue
			}
			if err = s.Lifecycle.push(); err != nil {
				return util.LUndef, err
			}
			assigned++
			if err = s.propagation.assign(lit.Negated(), satJustification{}); err != nil {
				return util.LUndef, err
			}
			if _, err = s.Lifecycle.propagate(false, false, false); err != nil {
				return util.LUndef, err
			}
			for s.propagation.inconsistent {
				yes, e := s.conflict.resolveConflict()
				if e != nil {
					return util.LUndef, e
				}
				if !yes {
					s.propagation.inconsistent = false
					isSat = util.LUndef
					break
				}
				if _, err = s.Lifecycle.propagate(false, false, false); err != nil {
					return util.LUndef, err
				}
				resolves++
			}
			// Literal source false&&scope_lvl()==1 branch remains unreachable.
		}
		_ = fixed
		_ = assigned // Source verbose-only counters; no fake report output.
		if err = s.extractFixedSet(unfixed, assumptions, vars, out); err != nil {
			return util.LUndef, err
		}
		if isSat == util.LTrue {
			if s.Variables.scopeLevel == 1 && resolves > 0 {
				isSat = util.LUndef
			} else {
				isSat, err = s.boundedSearch()
				if err != nil {
					return util.LUndef, err
				}
				if isSat == util.LUndef {
					if err = s.gc.restart(); err != nil {
						return util.LUndef, err
					}
				}
				if err = s.extractFixedSet(unfixed, assumptions, vars, out); err != nil {
					return util.LUndef, err
				}
			}
		}
		if isSat == util.LFalse {
			s.propagation.inconsistent = false
		}
		if isSat == util.LTrue {
			if err = s.deleteUnfixed(unfixed, vars); err != nil {
				return util.LUndef, err
			}
		}
		if err = s.updateUnfixed(unfixed, vars); err != nil {
			return util.LUndef, err
		}
		n, e = unfixed.size()
		if e != nil {
			return util.LUndef, e
		}
		if n != 0 {
			cfg, e := s.config.State()
			if e != nil {
				return util.LUndef, e
			}
			if cfg.RestartMax <= iterations {
				return util.LUndef, nil
			}
		}
	}
}
func (s *Solver) getBoundedConsequences(asms, vars satConsequenceInput, out *satSolverLiteralRows) (result util.LBool, err error) {
	if err = s.consequenceEntry(out); err != nil {
		return util.LUndef, err
	}
	s.consequenceRunning = true
	defer func() { s.consequenceRunning = false }()
	return s.getBoundedConsequencesBody(asms, vars, out)
}
func (s *Solver) boundedExtract(start *uint32, asms satConsequenceInput, vars *satMUSLiteralSet, out *satSolverLiteralRows) (err error) {
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
func (s *Solver) getBoundedConsequencesBody(asms, vars satConsequenceInput, out *satSolverLiteralRows) (result util.LBool, err error) {
	f, e := s.consequenceFrame()
	if e != nil {
		return util.LUndef, e
	}
	defer func() { err = s.consequenceDrop(f, err) }()
	if err = f.set(s, 0, nil); err != nil {
		return util.LUndef, err
	}
	unfixed := &f.sets[0]
	units, iterations := uint32(0), uint32(0)
	for i := uint32(0); ; i++ {
		n, e := vars.size()
		if e != nil {
			return util.LUndef, e
		}
		if i >= n {
			break
		}
		v, e := vars.read(i)
		if e != nil {
			return util.LUndef, e
		}
		if err = unfixed.insert(Literal(v)); err != nil {
			return util.LUndef, err
		}
	}
	if err = s.antecedents.reset(); err != nil {
		return util.LUndef, err
	}
	result, err = s.consequenceInitialize(asms)
	if err != nil || result != util.LTrue {
		return result, err
	}
	if err = s.boundedExtract(&units, asms, unfixed, out); err != nil {
		return util.LUndef, err
	}
	if err = s.simplifyProblem(); err != nil {
		return util.LUndef, err
	}
	bad, e := s.checkInconsistent()
	if e != nil {
		return util.LUndef, e
	}
	if bad {
		err = s.fixupConsequenceCore()
		return util.LFalse, err
	}
	for {
		iterations++
		if satStorageSourceDebug && s.propagation.inconsistent {
			return util.LUndef, ErrSATAssertion
		}
		result, err = s.boundedSearch()
		if err != nil {
			return util.LUndef, err
		}
		if result != util.LUndef {
			err = s.fixupConsequenceCore()
			return result, err
		}
		if err = s.boundedExtract(&units, asms, unfixed, out); err != nil {
			return util.LUndef, err
		}
		cfg, e := s.config.State()
		if e != nil {
			return util.LUndef, e
		}
		if s.conflict.conflicts > cfg.MaxConflicts {
			return util.LUndef, nil
		}
		if err = s.gc.restart(); err != nil {
			return util.LUndef, err
		}
		if err = s.simplifyProblem(); err != nil {
			return util.LUndef, err
		}
		bad, e = s.checkInconsistent()
		if e != nil {
			return util.LUndef, e
		}
		if bad {
			err = s.fixupConsequenceCore()
			return util.LFalse, err
		}
		if err = s.gc.gc(); err != nil {
			return util.LUndef, err
		}
		cfg, e = s.config.State()
		if e != nil {
			return util.LUndef, e
		}
		if cfg.RestartMax <= iterations {
			return util.LUndef, nil
		}
	}
}

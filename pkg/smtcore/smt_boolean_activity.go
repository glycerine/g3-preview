// Source: src/smt/smt_context.h inc_bvar_activity(),
// decay_bvar_activity(), and src/smt/smt_context.cpp rescale_bool_var_activity().
package smtcore

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
)

const (
	smtBooleanActivityLimit        = 1e100
	smtBooleanInverseActivityLimit = 1e-100
)

func (s *SMTInternalizerState) incrementBooleanActivity(variable int32) error {
	if variable < SMTFirstBoolVar || int(variable) >= len(s.activity) {
		return ast.ErrASTDomain
	}
	s.activity[variable] += s.activityIncrement
	if s.activity[variable] > smtBooleanActivityLimit {
		for index := range s.activity {
			s.activity[index] *= smtBooleanInverseActivityLimit
		}
		s.activityIncrement *= smtBooleanInverseActivityLimit
	}
	return nil
}

func (s *SMTInternalizerState) decayBooleanActivity() {
	s.activityIncrement *= s.activityInverseDecay
}

func (s *SMTInternalizerState) configureBooleanHeuristics(params SMTParamsState) error {
	if params.RandomInitialActivity > 2 || params.CaseSplitStrategy > 6 || params.InvDecay <= 0 {
		return ast.ErrASTDomain
	}
	s.random.SetSeed(params.RandomSeed)
	s.randomInitialActivity = params.RandomInitialActivity
	s.caseSplitStrategy = params.CaseSplitStrategy
	s.activityInverseDecay = params.InvDecay
	return nil
}

func (s *SMTInternalizerState) cacheBooleanActivity(expression ast.ASTHandle, activity float64) error {
	if _, found := s.activityCache[expression]; !found {
		if err := s.manager.IncRef(expression); err != nil {
			return err
		}
	}
	s.activityCache[expression] = activity
	return nil
}

func (s *SMTInternalizerState) releaseBooleanActivityCache() (result error) {
	for expression := range s.activityCache {
		result = errors.Join(result, s.manager.DecRef(expression))
	}
	s.activityCache = nil
	return result
}

// resetBooleanActivityCache implements cact_case_split_queue::init_search_eh:
// cached activity only connects variables deleted and recreated during one
// search. Each subsequent search starts with an empty cache.
func (s *SMTInternalizerState) resetBooleanActivityCache() (result error) {
	for expression := range s.activityCache {
		result = errors.Join(result, s.manager.DecRef(expression))
	}
	s.activityCache = make(map[ast.ASTHandle]float64)
	return result
}

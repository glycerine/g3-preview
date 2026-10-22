// Source: src/smt/smt_context.h inc_bvar_activity(),
// decay_bvar_activity(), and src/smt/smt_context.cpp rescale_bool_var_activity().
package smtcore

import "github.com/glycerine/g3/pkg/ast"

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
	if params.RandomInitialActivity > 2 || params.InvDecay <= 0 {
		return ast.ErrASTDomain
	}
	s.random.SetSeed(params.RandomSeed)
	s.randomInitialActivity = params.RandomInitialActivity
	s.activityInverseDecay = params.InvDecay
	return nil
}

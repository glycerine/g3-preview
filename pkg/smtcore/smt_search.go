// Source: src/smt/smt_context.cpp decide(), bounded_search(), and final_check().
package smtcore

import (
	"errors"
	"fmt"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

type smtSearchProgress struct {
	variables, clauses, assigned int
	equalities, disequalities    int
}

func (s *SMTInternalizerState) searchProgress() smtSearchProgress {
	return smtSearchProgress{
		variables: len(s.boolExpr), clauses: len(s.clauses), assigned: len(s.assignedLiterals),
		equalities: len(s.theoryEqualities), disequalities: len(s.theoryDisequalities),
	}
}

func smtDecisionLiteral(variable int32, phase lbool.LBool) SMTLiteral {
	return NewSMTLiteral(variable, phase == lbool.False)
}

// search performs the retained DPLL control flow over the already ported
// two-watch and theory propagation engines. Conflict analysis, learning, and
// non-chronological backjumping remain separate context work.
func (k *SMTKernel) search(queue *SMTActivityCaseSplitQueue) (lbool.LBool, error) {
	consistent, err := k.internalizer.PropagateCore()
	if err != nil {
		return lbool.Undef, fmt.Errorf("smt search propagate: %w", err)
	}
	if !consistent {
		return lbool.False, nil
	}

	variable, phase, err := queue.NextCaseSplit()
	if err != nil {
		return lbool.Undef, fmt.Errorf("smt search case split: %w", err)
	}
	if variable != SMTNullBoolVar {
		phases := [2]lbool.LBool{phase, phase.Negated()}
		result := lbool.False
		for _, candidate := range phases {
			if err = k.internalizer.PushScope(); err != nil {
				return lbool.Undef, fmt.Errorf("smt search push decision scope: %w", err)
			}
			k.internalizer.statistics.NumDecisions++
			assignErr := k.internalizer.AssignLiteral(smtDecisionLiteral(variable, candidate), NewSMTBooleanAxiom())
			branch := lbool.Undef
			if assignErr == nil {
				branch, assignErr = k.search(queue)
			}
			popErr := k.internalizer.PopScopes(1)
			if assignErr != nil {
				if popErr != nil {
					return lbool.Undef, errors.Join(assignErr, popErr)
				}
				return lbool.Undef, fmt.Errorf("smt search decision: %w", assignErr)
			}
			if popErr != nil {
				return lbool.Undef, fmt.Errorf("smt search pop decision scope: %w", popErr)
			}
			if branch == lbool.True {
				return branch, nil
			}
			if branch == lbool.Undef {
				result = lbool.Undef
			}
		}
		return result, nil
	}

	before := k.internalizer.searchProgress()
	status, err := k.internalizer.FinalCheckTheories()
	if err != nil {
		return lbool.Undef, fmt.Errorf("smt search final check: %w", err)
	}
	switch status {
	case SMTFinalCheckDone:
		return lbool.True, nil
	case SMTFinalCheckGiveUp:
		return lbool.Undef, nil
	case SMTFinalCheckContinue:
		consistent, err = k.internalizer.PropagateCore()
		if err != nil {
			return lbool.Undef, fmt.Errorf("smt search final propagation: %w", err)
		}
		if !consistent {
			return lbool.False, nil
		}
		if k.internalizer.searchProgress() == before {
			return lbool.Undef, nil
		}
		return k.search(queue)
	default:
		return lbool.Undef, ast.ErrASTDomain
	}
}

// Source: src/smt/smt_context.cpp decide(), bounded_search(), resolve_conflict(),
// and final_check().
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

// search runs the retained CDCL control loop. Learned clauses are installed at
// the computed backjump scope and therefore participate in the next propagation
// pass instead of re-exploring the conflicting branch.
func (k *SMTKernel) search(queue *SMTActivityCaseSplitQueue) (result lbool.LBool, resultErr error) {
	baseLevel := k.internalizer.scopeLevel
	resolver, err := NewSMTConflictResolver(k.internalizer)
	if err != nil {
		return lbool.Undef, err
	}
	defer func() {
		if count := k.internalizer.scopeLevel - baseLevel; count != 0 {
			resultErr = errors.Join(resultErr, k.internalizer.PopScopes(count))
			if resultErr != nil {
				result = lbool.Undef
			}
		}
	}()
	for {
		consistent, err := k.internalizer.PropagateCore()
		if err != nil {
			return lbool.Undef, fmt.Errorf("smt search propagate: %w", err)
		}
		if !consistent {
			conflict, found := k.internalizer.Conflict()
			if !found {
				return lbool.Undef, ast.ErrASTDomain
			}
			conflictLiteral := k.internalizer.ConflictLiteral()
			resolution, err := resolver.Resolve(conflict, conflictLiteral, baseLevel)
			if err != nil {
				return lbool.Undef, fmt.Errorf("smt search resolve conflict: %w", err)
			}
			if !resolution.Resolved {
				if err = k.captureUnsatCore(resolver, conflict, conflictLiteral); err != nil {
					return lbool.Undef, err
				}
				return lbool.False, nil
			}
			if resolution.NewScopeLevel < baseLevel || resolution.NewScopeLevel >= k.internalizer.scopeLevel {
				return lbool.Undef, ast.ErrASTDomain
			}
			if err = k.internalizer.PopScopes(k.internalizer.scopeLevel - resolution.NewScopeLevel); err != nil {
				return lbool.Undef, fmt.Errorf("smt search backjump: %w", err)
			}
			if err = k.internalizer.AddLearnedClause(resolution.Lemma); err != nil {
				return lbool.Undef, fmt.Errorf("smt search learn clause: %w", err)
			}
			continue
		}

		variable, phase, err := queue.NextCaseSplit()
		if err != nil {
			return lbool.Undef, fmt.Errorf("smt search case split: %w", err)
		}
		if variable != SMTNullBoolVar {
			if err = k.internalizer.PushScope(); err != nil {
				return lbool.Undef, fmt.Errorf("smt search push decision scope: %w", err)
			}
			k.internalizer.statistics.NumDecisions++
			if err = k.internalizer.AssignLiteral(smtDecisionLiteral(variable, phase), NewSMTBooleanAxiom()); err != nil {
				return lbool.Undef, fmt.Errorf("smt search decision: %w", err)
			}
			continue
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
				continue
			}
			if k.internalizer.searchProgress() == before {
				return lbool.Undef, nil
			}
		default:
			return lbool.Undef, ast.ErrASTDomain
		}
	}
}

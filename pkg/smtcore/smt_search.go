// Source: src/smt/smt_context.cpp decide(), bounded_search(), resolve_conflict(),
// and final_check().
package smtcore

import (
	"errors"
	"fmt"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
	"github.com/glycerine/g3/pkg/util/stopwatch"
)

type smtSearchProgress struct {
	variables, clauses, assigned int
	equalities, disequalities    int
}

type smtRetainedLemmaAtom struct {
	expression ast.ASTHandle
	sign       bool
}

func (s *SMTInternalizerState) retainLemmaAtoms(literals []SMTLiteral) ([]smtRetainedLemmaAtom, error) {
	atoms := make([]smtRetainedLemmaAtom, 0, len(literals))
	for _, literal := range literals {
		expression, err := s.BoolExpression(literal.Var())
		if err != nil {
			_ = s.releaseLemmaAtoms(atoms)
			return nil, err
		}
		if err = s.manager.IncRef(expression); err != nil {
			_ = s.releaseLemmaAtoms(atoms)
			return nil, err
		}
		atoms = append(atoms, smtRetainedLemmaAtom{expression: expression, sign: literal.Sign()})
	}
	return atoms, nil
}

func (s *SMTInternalizerState) releaseLemmaAtoms(atoms []smtRetainedLemmaAtom) (result error) {
	for index := len(atoms) - 1; index >= 0; index-- {
		result = errors.Join(result, s.manager.DecRef(atoms[index].expression))
	}
	return result
}

func (s *SMTInternalizerState) reinternalizeLemmaAtoms(atoms []smtRetainedLemmaAtom) ([]SMTLiteral, error) {
	literals := make([]SMTLiteral, len(atoms))
	for index, atom := range atoms {
		if err := s.Internalize(atom.expression, true); err != nil {
			return nil, err
		}
		literal, err := s.Literal(atom.expression)
		if err != nil {
			return nil, err
		}
		if atom.sign {
			literal = literal.Negated()
		}
		literals[index] = literal
	}
	return literals, nil
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

func (k *SMTKernel) searchResourceLimitExceeded(timer *stopwatch.Timer) (bool, error) {
	timedOut, err := timer.MSTimeout(k.smtState.Timeout)
	if err != nil {
		return false, err
	}
	if timedOut {
		k.lastFailure = SMTFailureTimeout
		return true, nil
	}
	exceeded, err := k.manager.Canceled()
	if err != nil {
		return false, err
	}
	if exceeded {
		if k.manager.Limit.CancelFlag() {
			k.lastFailure = SMTFailureCanceled
		} else {
			k.lastFailure = SMTFailureResourceLimit
		}
		return true, nil
	}
	runtime := k.manager.NumeralService().RuntimeOwner()
	if runtime == nil {
		return false, nil
	}
	above := false
	if process := runtime.ProcessOwner(); process != nil {
		above = process.AboveWatermark()
	} else {
		above, err = runtime.AboveWatermark()
		if err != nil {
			return false, err
		}
	}
	if above {
		k.lastFailure = SMTFailureMemory
		return true, nil
	}
	return false, nil
}

// search runs the retained CDCL control loop. Learned clauses are installed at
// the computed backjump scope and therefore participate in the next propagation
// pass instead of re-exploring the conflicting branch.
func (k *SMTKernel) search(queue *SMTActivityCaseSplitQueue) (result lbool.LBool, resultErr error) {
	baseLevel := k.internalizer.scopeLevel
	conflictBase := k.internalizer.statistics.NumConflicts
	timer, err := stopwatch.NewTimer(k.timerClock)
	if err != nil {
		return lbool.Undef, err
	}
	resolver, err := NewSMTConflictResolver(k.internalizer)
	if err != nil {
		return lbool.Undef, errors.Join(err, timer.Close())
	}
	defer func() {
		resultErr = errors.Join(resultErr, resolver.Close())
		resultErr = errors.Join(resultErr, timer.Close())
		if count := k.internalizer.scopeLevel - baseLevel; count != 0 {
			resultErr = errors.Join(resultErr, k.internalizer.PopScopes(count))
			if resultErr != nil {
				result = lbool.Undef
			}
		}
	}()
	for {
		exceeded, err := k.searchResourceLimitExceeded(timer)
		if err != nil {
			return lbool.Undef, err
		}
		if exceeded {
			return lbool.Undef, nil
		}
		if k.callback != nil {
			k.callback.FastProgressSample()
		}
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
				proof, proofErr := resolver.LemmaProof()
				if proofErr != nil {
					return lbool.Undef, proofErr
				}
				if err = k.manager.SetRef(k.unsatProof, proof); err != nil {
					return lbool.Undef, err
				}
				if err = k.captureUnsatCore(resolver, conflict, conflictLiteral); err != nil {
					return lbool.Undef, err
				}
				return lbool.False, nil
			}
			if resolution.NewScopeLevel < baseLevel || resolution.NewScopeLevel >= k.internalizer.scopeLevel {
				return lbool.Undef, ast.ErrASTDomain
			}
			proof, proofErr := resolver.LemmaProof()
			if proofErr != nil {
				return lbool.Undef, proofErr
			}
			var retained []smtRetainedLemmaAtom
			if resolution.InternalScopeLevel > resolution.NewScopeLevel {
				retained, err = k.internalizer.retainLemmaAtoms(resolution.Lemma)
				if err != nil {
					return lbool.Undef, err
				}
			}
			if err = k.internalizer.PopScopes(k.internalizer.scopeLevel - resolution.NewScopeLevel); err != nil {
				if len(retained) != 0 {
					err = errors.Join(err, k.internalizer.releaseLemmaAtoms(retained))
				}
				return lbool.Undef, fmt.Errorf("smt search backjump: %w", err)
			}
			if len(retained) != 0 {
				resolution.Lemma, err = k.internalizer.reinternalizeLemmaAtoms(retained)
				err = errors.Join(err, k.internalizer.releaseLemmaAtoms(retained))
				if err != nil {
					return lbool.Undef, fmt.Errorf("smt search reinternalize learned clause: %w", err)
				}
			}
			if err = k.internalizer.AddLearnedClauseWithProof(resolution.Lemma, proof); err != nil {
				return lbool.Undef, fmt.Errorf("smt search learn clause: %w", err)
			}
			if err = k.internalizer.recordConflictForClauseMaintenance(); err != nil {
				return lbool.Undef, fmt.Errorf("smt search collect learned clauses: %w", err)
			}
			queue.RecordConflict()
			if k.internalizer.statistics.NumConflicts-conflictBase > k.smtState.MaxConflicts {
				k.lastFailure = SMTFailureConflictLimit
				return lbool.Undef, nil
			}
			continue
		}
		exceeded, err = k.searchResourceLimitExceeded(timer)
		if err != nil {
			return lbool.Undef, err
		}
		if exceeded {
			return lbool.Undef, nil
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
			if err = k.captureRelevantLabels(); err != nil {
				return lbool.Undef, err
			}
			if err = k.captureLiteralResults(baseLevel); err != nil {
				return lbool.Undef, err
			}
			if err = k.buildGeneratedModel(); err != nil {
				return lbool.Undef, fmt.Errorf("smt search build model: %w", err)
			}
			return lbool.True, nil
		case SMTFinalCheckGiveUp:
			k.lastFailure = SMTFailureTheory
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
				k.lastFailure = SMTFailureUnknown
				k.unknown = "theory made no progress"
				return lbool.Undef, nil
			}
		default:
			return lbool.Undef, ast.ErrASTDomain
		}
	}
}

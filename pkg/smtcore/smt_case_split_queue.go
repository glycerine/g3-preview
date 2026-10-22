// Source: src/smt/smt_case_split_queue.cpp activity-based case selection used
// by src/smt/smt_context.cpp decide().
package smtcore

import (
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
	z3rand "github.com/glycerine/g3/pkg/util/random"
)

// SMTActivityCaseSplitQueue is the concrete activity strategy retained by the
// Go search loop. The C++ heap is an implementation detail; scanning the native
// Go slices preserves its choice of a greatest-activity unassigned variable.
type SMTActivityCaseSplitQueue struct {
	state                                 *SMTInternalizerState
	random                                z3rand.Generator
	phaseSelection, cachingOn, cachingOff uint32
	phaseCounter                          uint32
	cacheEnabled, phaseDefault            bool
}

func NewSMTActivityCaseSplitQueue(state *SMTInternalizerState) (*SMTActivityCaseSplitQueue, error) {
	if state == nil || state.open() != nil {
		return nil, ast.ErrASTHandle
	}
	return &SMTActivityCaseSplitQueue{state: state, phaseSelection: 3, cachingOn: 400, cachingOff: 100, cacheEnabled: true}, nil
}

func (q *SMTActivityCaseSplitQueue) ConfigurePhases(state SMTParamsState) error {
	if q == nil || q.state == nil || state.PhaseSelection > 6 {
		return ast.ErrASTDomain
	}
	q.phaseSelection = state.PhaseSelection
	q.cachingOn = state.PhaseCachingOn
	q.cachingOff = state.PhaseCachingOff
	q.random.SetSeed(state.RandomSeed)
	q.phaseCounter = 0
	q.cacheEnabled = true
	q.phaseDefault = false
	return nil
}

func (q *SMTActivityCaseSplitQueue) RecordConflict() {
	q.phaseCounter++
	limit := q.cachingOff
	if q.cacheEnabled {
		limit = q.cachingOn
	}
	if q.phaseCounter < limit {
		return
	}
	q.phaseCounter = 0
	q.cacheEnabled = !q.cacheEnabled
	if q.phaseSelection == 4 {
		q.phaseDefault = !q.phaseDefault
	}
}

func (q *SMTActivityCaseSplitQueue) occurrencePhase(variable int32) bool {
	positive, negative := 0, 0
	for i := range q.state.clauses {
		clause := q.state.clauses[i].clause
		if clause == nil {
			continue
		}
		for _, literal := range clause.literals {
			if literal.Var() != variable {
				continue
			}
			if literal.Sign() {
				negative++
			} else {
				positive++
			}
		}
	}
	return positive > negative
}

// NextCaseSplit returns the highest-activity unassigned Boolean variable and
// the source phase preference. Ties are deterministic in Go variable order.
func (q *SMTActivityCaseSplitQueue) NextCaseSplit() (int32, lbool.LBool, error) {
	if q == nil || q.state == nil {
		return SMTNullBoolVar, lbool.Undef, ast.ErrASTHandle
	}
	if err := q.state.open(); err != nil {
		return SMTNullBoolVar, lbool.Undef, err
	}
	next := SMTNullBoolVar
	best := 0.0
	for variable := int32(SMTFirstBoolVar); int(variable) < len(q.state.boolExpr); variable++ {
		value, err := q.state.Assignment(NewSMTLiteral(variable, false))
		if err != nil {
			return SMTNullBoolVar, lbool.Undef, err
		}
		if value != lbool.Undef {
			continue
		}
		activity := q.state.activity[variable]
		if next == SMTNullBoolVar || activity > best {
			next, best = variable, activity
		}
	}
	if next == SMTNullBoolVar {
		return next, lbool.Undef, nil
	}
	data := &q.state.booleanData[next]
	if data.IsQuantifier() {
		return next, lbool.False, nil
	}
	if data.TryTrueFirst() {
		return next, lbool.True, nil
	}
	phase := false
	switch q.phaseSelection {
	case 0:
	case 1:
		phase = true
	case 2, 3, 4:
		phase = q.phaseDefault
		if q.cacheEnabled && data.PhaseAvailable {
			phase = data.Phase
		}
	case 5:
		phase = q.random.Next()%2 == 0
	case 6:
		phase = q.occurrencePhase(next)
	default:
		return SMTNullBoolVar, lbool.Undef, ast.ErrASTDomain
	}
	if phase {
		return next, lbool.True, nil
	}
	return next, lbool.False, nil
}

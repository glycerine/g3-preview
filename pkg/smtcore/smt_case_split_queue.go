// Source: src/smt/smt_case_split_queue.cpp activity-based case selection used
// by src/smt/smt_context.cpp decide().
package smtcore

import (
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

// SMTActivityCaseSplitQueue is the concrete activity strategy retained by the
// Go search loop. The C++ heap is an implementation detail; scanning the native
// Go slices preserves its choice of a greatest-activity unassigned variable.
type SMTActivityCaseSplitQueue struct {
	state *SMTInternalizerState
}

func NewSMTActivityCaseSplitQueue(state *SMTInternalizerState) (*SMTActivityCaseSplitQueue, error) {
	if state == nil || state.open() != nil {
		return nil, ast.ErrASTHandle
	}
	return &SMTActivityCaseSplitQueue{state: state}, nil
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
	return next, lbool.False, nil
}
